package codex

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
)

func TestProviderExecuteParsesJSONEvents(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{result: ProcessResult{
		Stdout: `{"type":"message","message":"structured answer"}` + "\n",
	}}
	provider := NewProviderWithRunner(Config{
		Command: "codex",
		Model:   "gpt-5.4-mini",
		Profile: "agentd",
		Timeout: time.Minute,
	}, runner)

	response, err := provider.Execute(context.Background(), appruntime.ProviderRequest{
		RunID:     "run-1",
		AgentName: "codex-agent",
		Model:     "gpt-5.4-mini",
		Prompt:    "Say done",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if response.Output != "structured answer" {
		t.Fatalf("output: got %q", response.Output)
	}
	if runner.command != "codex" {
		t.Fatalf("command: got %q", runner.command)
	}
	if !containsArg(runner.args, "exec") || !containsArg(runner.args, "--json") ||
		!containsArg(runner.args, "--model") || !containsArg(runner.args, "gpt-5.4-mini") ||
		!containsArg(runner.args, "--profile") || !containsArg(runner.args, "agentd") {
		t.Fatalf("args: %#v", runner.args)
	}
	if !strings.Contains(runner.stdin, "Say done") {
		t.Fatalf("stdin did not include prompt: %q", runner.stdin)
	}
}

func TestProviderExecuteReportsMissingCLI(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(Config{Command: "codex", Timeout: time.Minute}, &fakeRunner{
		err: exec.ErrNotFound,
	})
	_, err := provider.Execute(context.Background(), appruntime.ProviderRequest{
		Model: "gpt-5.4-mini", Prompt: "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "codex CLI not found") {
		t.Fatalf("error: %v", err)
	}
}

func TestProviderExecuteReportsUnauthenticatedCodex(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(Config{Command: "codex", Timeout: time.Minute}, &fakeRunner{
		result: ProcessResult{Stderr: "not logged in; run codex login", ExitCode: 1},
	})
	_, err := provider.Execute(context.Background(), appruntime.ProviderRequest{
		Model: "gpt-5.4-mini", Prompt: "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "run codex login") {
		t.Fatalf("error: %v", err)
	}
}

func TestProviderExecuteRejectsMalformedJSONEvents(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(Config{Command: "codex", Timeout: time.Minute}, &fakeRunner{
		result: ProcessResult{Stdout: "{not-json\n"},
	})
	_, err := provider.Execute(context.Background(), appruntime.ProviderRequest{
		Model: "gpt-5.4-mini", Prompt: "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "malformed codex JSON event") {
		t.Fatalf("error: %v", err)
	}
}

func TestProviderExecuteReportsTimeout(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(Config{Command: "codex", Timeout: time.Nanosecond}, &fakeRunner{
		result: ProcessResult{TimedOut: true},
	})
	_, err := provider.Execute(context.Background(), appruntime.ProviderRequest{
		Model: "gpt-5.4-mini", Prompt: "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("error: %v", err)
	}
}

func TestProviderExecutePropagatesCancellation(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(Config{Command: "codex", Timeout: time.Minute}, &fakeRunner{
		err: context.Canceled,
	})
	_, err := provider.Execute(context.Background(), appruntime.ProviderRequest{
		Model: "gpt-5.4-mini", Prompt: "hello",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error: got %v want context.Canceled", err)
	}
}

func TestProviderFinalizeUsesOutputSchema(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{result: ProcessResult{
		Stdout: `{"type":"message","message":"{\"summary\":\"done\"}"}` + "\n",
	}}
	provider := NewProviderWithRunner(Config{Command: "codex", Timeout: time.Minute}, runner)
	response, err := provider.Finalize(context.Background(), appruntime.StructuredOutputRequest{
		Model:           "gpt-5.4-mini",
		OutputSchemaRaw: `{"type":"object","required":["summary"],"properties":{"summary":{"type":"string"}}}`,
		PlainTextResult: "done",
	})
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if string(response.OutputJSON) != `{"summary":"done"}` {
		t.Fatalf("output json: %s", response.OutputJSON)
	}
	if !strings.Contains(runner.stdin, "Return only JSON") ||
		!strings.Contains(runner.stdin, "summary") {
		t.Fatalf("stdin did not include schema instructions: %q", runner.stdin)
	}
}

type fakeRunner struct {
	command string
	args    []string
	stdin   string
	result  ProcessResult
	err     error
}

func (r *fakeRunner) Run(_ context.Context, command string, args []string, stdin string) (ProcessResult, error) {
	r.command = command
	r.args = append([]string(nil), args...)
	r.stdin = stdin

	return r.result, r.err
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}

	return false
}
