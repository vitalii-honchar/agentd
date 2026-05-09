package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

const ProviderName = "codex"

const maxProcessOutputBytes = 1 << 20

type Config struct {
	Command string
	Model   string
	Profile string
	Timeout time.Duration
}

type ProcessResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
}

type ProcessRunner interface {
	Run(ctx context.Context, command string, args []string, stdin string) (ProcessResult, error)
}

type Provider struct {
	cfg    Config
	runner ProcessRunner
}

var _ appruntime.Provider = (*Provider)(nil)
var _ appruntime.ReActProvider = (*Provider)(nil)
var _ appruntime.StructuredOutputProvider = (*Provider)(nil)

func NewProvider(cfg Config) *Provider {
	return NewProviderWithRunner(cfg, processRunner{})
}

func NewProviderWithRunner(cfg Config, runner ProcessRunner) *Provider {
	if strings.TrimSpace(cfg.Command) == "" {
		cfg.Command = ProviderName
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Minute
	}
	if runner == nil {
		runner = processRunner{}
	}

	return &Provider{cfg: cfg, runner: runner}
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) Execute(
	ctx context.Context,
	request appruntime.ProviderRequest,
) (appruntime.ProviderResponse, error) {
	if strings.TrimSpace(request.Prompt) == "" {
		return appruntime.ProviderResponse{}, fmt.Errorf("codex prompt is required")
	}
	model := p.model(request.Model)
	if strings.TrimSpace(model) == "" {
		return appruntime.ProviderResponse{}, fmt.Errorf("codex model is required")
	}

	output, err := p.run(ctx, model, request.Prompt)
	if err != nil {
		return appruntime.ProviderResponse{}, err
	}

	return appruntime.ProviderResponse{
		RequestID: request.RunID,
		Output:    output,
	}, nil
}

func (p *Provider) Decide(
	ctx context.Context,
	request appruntime.ReActRequest,
) (appruntime.ReActResponse, error) {
	prompt := buildReActPrompt(request)
	output, err := p.run(ctx, p.model(request.Model), prompt)
	if err != nil {
		return appruntime.ReActResponse{}, err
	}

	decision := parseReActDecision(output)
	decision.RequestID = request.RunID

	return decision, nil
}

func (p *Provider) Finalize(
	ctx context.Context,
	request appruntime.StructuredOutputRequest,
) (appruntime.StructuredOutputResponse, error) {
	if strings.TrimSpace(request.OutputSchemaRaw) == "" {
		return appruntime.StructuredOutputResponse{}, fmt.Errorf("%w: contract.output is required", domain.ErrInvalidContractSchema)
	}
	schemaPath, finalTextPath, cleanup, err := writeStructuredOutputFiles(
		request.OutputSchemaRaw,
		request.PlainTextResult,
	)
	if err != nil {
		return appruntime.StructuredOutputResponse{}, err
	}
	defer cleanup()

	prompt := strings.Join([]string{
		"Return only JSON that matches this JSON Schema.",
		"Schema file:",
		schemaPath,
		"Conversation final text file:",
		finalTextPath,
		"Schema:",
		request.OutputSchemaRaw,
		"Conversation final text:",
		request.PlainTextResult,
	}, "\n")
	output, err := p.run(ctx, p.model(request.Model), prompt)
	if err != nil {
		return appruntime.StructuredOutputResponse{}, err
	}
	raw := json.RawMessage(strings.TrimSpace(output))
	if !json.Valid(raw) {
		return appruntime.StructuredOutputResponse{}, fmt.Errorf("codex structured output was not valid JSON")
	}

	return appruntime.StructuredOutputResponse{
		RequestID:  request.RunID,
		OutputJSON: raw,
	}, nil
}

func writeStructuredOutputFiles(
	outputSchemaRaw string,
	plainTextResult string,
) (schemaPath string, finalTextPath string, cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "agentd-codex-output-*")
	if err != nil {
		return "", "", func() {}, fmt.Errorf("create codex structured output temp dir: %w", err)
	}
	cleanup = func() {
		_ = os.RemoveAll(dir)
	}
	schemaPath = filepath.Join(dir, "output_schema.json")
	finalTextPath = filepath.Join(dir, "final_message.txt")
	if err := os.WriteFile(schemaPath, []byte(outputSchemaRaw), 0o600); err != nil {
		cleanup()

		return "", "", func() {}, fmt.Errorf("write codex output schema: %w", err)
	}
	if err := os.WriteFile(finalTextPath, []byte(plainTextResult), 0o600); err != nil {
		cleanup()

		return "", "", func() {}, fmt.Errorf("write codex final message: %w", err)
	}

	return schemaPath, finalTextPath, cleanup, nil
}

func (p *Provider) model(requestModel string) string {
	if strings.TrimSpace(p.cfg.Model) != "" {
		return p.cfg.Model
	}

	return requestModel
}

func (p *Provider) run(ctx context.Context, model string, prompt string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, p.cfg.Timeout)
	defer cancel()

	args := []string{"exec", "--json", "--model", model}
	if strings.TrimSpace(p.cfg.Profile) != "" {
		args = append(args, "--profile", p.cfg.Profile)
	}
	result, err := p.runner.Run(runCtx, p.cfg.Command, args, prompt)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("codex CLI not found: %s", p.cfg.Command)
		}
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			return "", fmt.Errorf("codex CLI not found: %s", p.cfg.Command)
		}

		return "", fmt.Errorf("run codex CLI: %w", err)
	}
	if result.TimedOut {
		return "", fmt.Errorf("codex CLI timed out after %s", p.cfg.Timeout)
	}
	if result.ExitCode != 0 {
		return "", codexExitError(result)
	}

	output, err := parseCodexJSONEvents(result.Stdout)
	if err != nil {
		return "", err
	}
	if output == "" {
		return "", fmt.Errorf("codex CLI produced no final output")
	}

	return output, nil
}

func codexExitError(result ProcessResult) error {
	stderr := strings.TrimSpace(result.Stderr)
	if isAuthError(stderr) {
		return fmt.Errorf("run codex login before using vendor.name: codex")
	}
	if stderr == "" {
		return fmt.Errorf("codex CLI exited with status %d", result.ExitCode)
	}

	return fmt.Errorf("codex CLI exited with status %d: %s", result.ExitCode, truncate(stderr, 2000))
}

func isAuthError(text string) bool {
	lower := strings.ToLower(text)

	return strings.Contains(lower, "not logged in") ||
		strings.Contains(lower, "unauthenticated") ||
		strings.Contains(lower, "authentication") ||
		strings.Contains(lower, "codex login")
}

type codexEvent struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
	Content json.RawMessage `json:"content"`
	Output  json.RawMessage `json:"output"`
	Text    json.RawMessage `json:"text"`
	Error   json.RawMessage `json:"error"`
}

func parseCodexJSONEvents(stdout string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	scanner.Buffer(make([]byte, 0, 64*1024), maxProcessOutputBytes)
	var final string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event codexEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return "", fmt.Errorf("malformed codex JSON event: %w", err)
		}
		if len(event.Error) > 0 && string(event.Error) != "null" {
			return "", fmt.Errorf("codex JSON event error: %s", string(event.Error))
		}
		value := firstEventString(event.Message, event.Content, event.Output, event.Text)
		switch event.Type {
		case "message", "assistant_message", "final", "result", "output":
			if value != "" {
				final = value
			}
		default:
			if value != "" {
				final = value
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read codex JSON events: %w", err)
	}

	return strings.TrimSpace(final), nil
}

func firstEventString(values ...json.RawMessage) string {
	for _, value := range values {
		if len(value) == 0 || string(value) == "null" {
			continue
		}
		var s string
		if err := json.Unmarshal(value, &s); err == nil {
			return s
		}
		var structured any
		if err := json.Unmarshal(value, &structured); err == nil {
			encoded, _ := json.Marshal(structured)

			return string(encoded)
		}
	}

	return ""
}

type reActDecisionOutput struct {
	Decision     string          `json:"decision"`
	ToolName     string          `json:"tool_name"`
	ToolArgsJSON json.RawMessage `json:"tool_args_json"`
	FinalText    string          `json:"final_text"`
	Failure      string          `json:"failure"`
}

func buildReActPrompt(request appruntime.ReActRequest) string {
	var builder strings.Builder
	builder.WriteString(request.Prompt)
	builder.WriteString("\n\nReturn only JSON with one of these decisions: tool_call, final, fail.")
	builder.WriteString("\nFields: decision, tool_name, tool_args_json, final_text, failure.")
	if len(request.Tools) > 0 {
		builder.WriteString("\nAvailable tools:")
		for _, tool := range request.Tools {
			builder.WriteString("\n- ")
			builder.WriteString(tool.Name)
		}
	}
	if len(request.History) > 0 {
		builder.WriteString("\n\nHistory:")
		for _, message := range request.History {
			builder.WriteString("\n")
			builder.WriteString(string(message.Role))
			builder.WriteString(": ")
			builder.WriteString(message.Content)
		}
	}

	return builder.String()
}

func parseReActDecision(output string) appruntime.ReActResponse {
	var parsed reActDecisionOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil || parsed.Decision == "" {
		return appruntime.ReActResponse{
			Decision:  domain.ReActDecisionFinal,
			FinalText: output,
			Message:   appruntime.ProviderMessage{Role: appruntime.ProviderRoleAssistant, Content: output},
		}
	}
	response := appruntime.ReActResponse{
		Decision:     domainReActDecision(parsed.Decision),
		ToolName:     parsed.ToolName,
		ToolArgsJSON: string(parsed.ToolArgsJSON),
		FinalText:    parsed.FinalText,
		Failure:      parsed.Failure,
		Message:      appruntime.ProviderMessage{Role: appruntime.ProviderRoleAssistant, Content: output},
	}
	if response.Decision == domain.ReActDecisionFinal && response.FinalText == "" {
		response.FinalText = output
	}

	return response
}

func domainReActDecision(value string) domain.ReActDecision {
	switch strings.TrimSpace(value) {
	case string(domain.ReActDecisionToolCall), string(domain.ReActDecisionFinal), string(domain.ReActDecisionFail):
		return domain.ReActDecision(value)
	default:
		return domain.ReActDecisionFinal
	}
}

type processRunner struct{}

func (processRunner) Run(ctx context.Context, command string, args []string, stdin string) (ProcessResult, error) {
	cmd := exec.Command(command, args...)
	configureProcessGroup(cmd)
	cmd.Stdin = strings.NewReader(stdin)
	stdout := newBoundedBuffer(maxProcessOutputBytes)
	stderr := newBoundedBuffer(maxProcessOutputBytes)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return ProcessResult{}, err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	timedOut := false
	select {
	case err = <-done:
	case <-ctx.Done():
		timedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		killProcessGroup(cmd)
		err = <-done
	}
	result := ProcessResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		TimedOut: timedOut,
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()

			return result, nil
		}

		return result, err
	}

	return result, nil
}

type boundedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func newBoundedBuffer(limit int) boundedBuffer {
	return boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 || b.buffer.Len() >= b.limit {
		return len(p), nil
	}
	remaining := b.limit - b.buffer.Len()
	if len(p) > remaining {
		_, _ = b.buffer.Write(p[:remaining])

		return len(p), nil
	}
	_, _ = b.buffer.Write(p)

	return len(p), nil
}

func (b *boundedBuffer) String() string {
	return b.buffer.String()
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}

	return value[:limit]
}
