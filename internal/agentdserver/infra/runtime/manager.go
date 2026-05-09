package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	appresult "github.com/vitalii-honchar/agentd/internal/agentdserver/app/result"
	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	goagentagent "github.com/vitalii-honchar/go-agent/pkg/goagent/agent"
	goagentllm "github.com/vitalii-honchar/go-agent/pkg/goagent/llm"

	"github.com/google/uuid"
)

type Manager struct {
	runtimeDBs app.RuntimeDBManager
	logs       app.RunLogFactory
	isolation  *IsolationBuilder
	providers  map[string]appruntime.Provider
	tools      appruntime.ToolExecutor
	now        func() time.Time

	mu     sync.Mutex
	active map[string]*activeRun
}

type activeRun struct {
	run       domain.AgentRun
	cancel    context.CancelFunc
	completed chan struct{}
}

var _ appruntime.Manager = (*Manager)(nil)

func NewManager(
	runtimeDBs app.RuntimeDBManager,
	logs app.RunLogFactory,
	isolation *IsolationBuilder,
	providers []appruntime.Provider,
) (*Manager, error) {
	if runtimeDBs == nil {
		return nil, fmt.Errorf("runtime db manager is required")
	}
	if logs == nil {
		return nil, fmt.Errorf("run log factory is required")
	}
	if isolation == nil {
		return nil, fmt.Errorf("isolation builder is required")
	}

	providerMap := make(map[string]appruntime.Provider, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		providerMap[provider.Name()] = provider
	}

	return &Manager{
		runtimeDBs: runtimeDBs,
		logs:       logs,
		isolation:  isolation,
		providers:  providerMap,
		now:        func() time.Time { return time.Now().UTC() },
		active:     make(map[string]*activeRun),
	}, nil
}

func (m *Manager) SetToolExecutor(executor appruntime.ToolExecutor) {
	m.tools = executor
}

func (m *Manager) Execute(
	ctx context.Context,
	request appruntime.ExecuteRequest,
) (domain.AgentRun, error) {
	provider, ok := m.providers[request.Agent.Vendor.Name]
	if !ok {
		return domain.AgentRun{}, fmt.Errorf("%w: %s", domain.ErrUnsupportedVendor, request.Agent.Vendor.Name)
	}
	if err := m.runtimeDBs.EnsureAgent(ctx, request.Agent.Name); err != nil {
		return domain.AgentRun{}, err
	}

	m.mu.Lock()
	for _, active := range m.active {
		if active.run.AgentName == request.Agent.Name {
			m.mu.Unlock()

			return domain.AgentRun{}, domain.ErrRunAlreadyActive
		}
	}
	m.mu.Unlock()

	runID := uuid.NewString()
	runCtx, cancel := context.WithCancel(context.Background())
	env, err := m.isolation.Build(request.Agent, runID)
	if err != nil {
		cancel()

		return domain.AgentRun{}, err
	}
	logWriter, err := m.logs.Create(ctx, request.Agent.Name, runID)
	if err != nil {
		cancel()

		return domain.AgentRun{}, err
	}

	startedAt := m.now()
	runtimeInput := request.RuntimeInput
	if runtimeInput.LegacyInputs == nil {
		runtimeInput.LegacyInputs = request.Inputs
	}
	run := domain.AgentRun{
		ID:                         runID,
		AgentName:                  request.Agent.Name,
		AgentRevision:              request.Agent.Revision,
		Trigger:                    request.Trigger,
		Status:                     domain.AgentRunStatusRunning,
		StartedAt:                  &startedAt,
		DueAt:                      request.DueAt,
		WorkDir:                    env.WorkDir,
		LogPath:                    logWriter.Path(),
		InputJSON:                  runtimeInputJSONForRun(runtimeInput),
		ProviderName:               request.Agent.Vendor.Name,
		ProviderModel:              request.Agent.Vendor.Model,
		ResultFormat:               domain.ResultFormatText,
		ContractInputSchemaDigest:  contractInputDigest(request.Agent.Contract),
		ContractOutputSchemaDigest: contractOutputDigest(request.Agent.Contract),
	}
	repo := m.runtimeDBs.Runs(request.Agent.Name)
	if repo == nil {
		_ = logWriter.Close()
		cancel()

		return domain.AgentRun{}, fmt.Errorf("run repository is required for agent %s", request.Agent.Name)
	}
	if err := repo.Create(ctx, run); err != nil {
		_ = logWriter.Close()
		cancel()

		return domain.AgentRun{}, err
	}
	if request.Agent.Contract != nil {
		m.appendRunEvent(run, domain.RunActionContractInputValidated, domain.EventLevelInfo, "contract input validated")
	}

	active := &activeRun{run: run, cancel: cancel, completed: make(chan struct{})}
	m.mu.Lock()
	m.active[run.ID] = active
	m.mu.Unlock()

	go m.runProvider(runCtx, provider, request.Agent, request.Revision, run, runtimeInput, logWriter, active)

	return run, nil
}

func (m *Manager) Stop(ctx context.Context, request appruntime.StopRequest) (domain.AgentRun, error) {
	active, err := m.findActive(request.AgentName, request.RunID)
	if err != nil {
		return domain.AgentRun{}, err
	}

	stopAt := m.now()
	run := active.run
	run.Status = domain.AgentRunStatusStopping
	run.StopRequestedAt = &stopAt
	m.setActiveRun(run)
	if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
		if err := repo.Update(ctx, run); err != nil {
			return domain.AgentRun{}, err
		}
	}
	active.cancel()

	return run, nil
}

func (m *Manager) Recover(ctx context.Context) (appruntime.RecoveryResult, error) {
	now := m.now()
	activeRuns := m.activeRunsSnapshot()
	for _, active := range activeRuns {
		active.cancel()
		select {
		case <-active.completed:
		case <-ctx.Done():
			return appruntime.RecoveryResult{}, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		run := active.run
		run.Status = domain.AgentRunStatusInterrupted
		run.CompletedAt = &now
		if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
			if err := repo.Update(ctx, run); err != nil {
				return appruntime.RecoveryResult{}, err
			}
		}
		m.removeActive(run.ID)
	}

	interrupted := make([]domain.AgentRun, 0, len(activeRuns))
	for _, active := range activeRuns {
		run := active.run
		run.Status = domain.AgentRunStatusInterrupted
		run.CompletedAt = &now
		interrupted = append(interrupted, run)
	}

	return appruntime.RecoveryResult{InterruptedRuns: interrupted, RecoveredAt: now}, nil
}

func (m *Manager) ActiveRuns(context.Context) ([]domain.AgentRun, error) {
	return m.ActiveRunsSnapshot(), nil
}

func (m *Manager) ActiveRunsSnapshot() []domain.AgentRun {
	activeRuns := m.activeRunsSnapshot()
	runs := make([]domain.AgentRun, 0, len(activeRuns))
	for _, active := range activeRuns {
		runs = append(runs, active.run)
	}

	return runs
}

func (m *Manager) activeRunsSnapshot() []*activeRun {
	m.mu.Lock()
	defer m.mu.Unlock()

	runs := make([]*activeRun, 0, len(m.active))
	for _, active := range m.active {
		runs = append(runs, active)
	}

	return runs
}

func (m *Manager) runProvider(
	ctx context.Context,
	provider appruntime.Provider,
	agent domain.Agent,
	revision domain.AgentRevision,
	run domain.AgentRun,
	input domain.RuntimeInput,
	logWriter app.RunLogWriter,
	active *activeRun,
) {
	defer close(active.completed)
	defer logWriter.Close()
	defer m.removeActive(run.ID)

	preparedAgent, inputErr := applyRunInputs(agent, input.LegacyInputs)
	if inputErr != nil {
		completedAt := m.now()
		run.CompletedAt = &completedAt
		run.Status = domain.AgentRunStatusFailed
		run.ErrorCode = "missing_input"
		run.ErrorMessage = inputErr.Error()
		run.Result = fmt.Sprintf("run failed: %s", inputErr.Error())
		run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
		if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
			_ = repo.Update(context.Background(), run)
		}
		m.appendRunEvent(run, domain.RunActionResultPersisted, domain.EventLevelInfo, "persisted run result")
		m.appendRunEvent(run, domain.RunActionFail, domain.EventLevelError, "run failed")

		return
	}

	prompt := preparedAgent.Prompt
	if reactProvider, ok := provider.(appruntime.ReActProvider); ok {
		m.runReActProvider(ctx, reactProvider, preparedAgent, revision, &run, input, logWriter)
		if run.Status == domain.AgentRunStatusCompleted &&
			preparedAgent.Contract != nil &&
			preparedAgent.Contract.OutputSchemaRaw != "" {
			if err := m.finalizeContractedOutput(ctx, provider, preparedAgent, &run, prompt, run.Result, logWriter); err != nil {
				run.Status = domain.AgentRunStatusFailed
				run.ErrorCode = "contract_output_invalid"
				run.ErrorMessage = err.Error()
				run.Result = fmt.Sprintf("run failed: %s", err.Error())
				run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
			}
		}
		if run.Status == domain.AgentRunStatusStopped && run.Result == "" {
			run.Result = "run stopped before completion"
			run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
		}
		if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
			_ = repo.Update(context.Background(), run)
		}
		m.appendRunEvent(run, domain.RunActionResultPersisted, domain.EventLevelInfo, "persisted run result")
		if run.Status == domain.AgentRunStatusCompleted {
			m.appendRunEvent(run, domain.RunActionComplete, domain.EventLevelInfo, "run completed")
		} else {
			m.appendRunEvent(run, domain.RunActionFail, domain.EventLevelError, "run failed or stopped")
		}

		return
	}

	toolOutput, toolErr := m.executeDeclaredTools(ctx, preparedAgent, revision, run)
	if toolErr != nil {
		completedAt := m.now()
		run.CompletedAt = &completedAt
		run.Status = domain.AgentRunStatusFailed
		run.ErrorCode = "tool_failed"
		run.ErrorMessage = toolErr.Error()
		run.Result = fmt.Sprintf("run failed: %s", toolErr.Error())
		run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
		if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
			_ = repo.Update(context.Background(), run)
		}
		m.appendRunEvent(run, domain.RunActionResultPersisted, domain.EventLevelInfo, "persisted run result")
		m.appendRunEvent(run, domain.RunActionFail, domain.EventLevelError, "run failed")

		return
	}
	if toolOutput != "" {
		prompt += "\n\nTool results:\n" + toolOutput
	}

	m.appendRunEvent(run, domain.RunActionLLMPromptSend, domain.EventLevelInfo, "send LLM prompt to provider")
	response, err := provider.Execute(ctx, appruntime.ProviderRequest{
		RunID:     run.ID,
		AgentName: agent.Name,
		Model:     agent.Vendor.Model,
		Prompt:    prompt,
	})
	completedAt := m.now()
	run.CompletedAt = &completedAt
	if err != nil {
		if errors.Is(err, context.Canceled) {
			run.Status = domain.AgentRunStatusStopped
		} else {
			run.Status = domain.AgentRunStatusFailed
			run.ErrorCode = "provider_error"
			run.ErrorMessage = err.Error()
			run.Result = fmt.Sprintf("run failed: %s", err.Error())
			run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
			m.appendRunEvent(run, domain.RunActionProviderFail, domain.EventLevelError, err.Error())
		}
	} else {
		run.Status = domain.AgentRunStatusCompleted
		run.ProviderRequestID = response.RequestID
		m.appendRunEvent(run, domain.RunActionLLMResponseReceive, domain.EventLevelInfo, "received LLM provider response")
		if response.Output != "" {
			if agent.Contract != nil && agent.Contract.OutputSchemaRaw != "" {
				run.ResultFormat = domain.ResultFormatJSON
				if err := m.finalizeContractedOutput(ctx, provider, agent, &run, prompt, response.Output, logWriter); err != nil {
					run.Status = domain.AgentRunStatusFailed
					run.ErrorCode = "contract_output_invalid"
					run.ErrorMessage = err.Error()
					run.Result = fmt.Sprintf("run failed: %s", err.Error())
					run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
				}
			} else {
				run.Result = response.Output
				run.ResultSummary = appresult.Summarize(response.Output, appresult.DefaultSummaryLimit)
				_, _ = io.WriteString(logWriter, response.Output)
			}
		}
	}
	if run.Status == domain.AgentRunStatusStopped && run.Result == "" {
		run.Result = "run stopped before completion"
		run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
	}
	if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
		_ = repo.Update(context.Background(), run)
	}
	m.appendRunEvent(run, domain.RunActionResultPersisted, domain.EventLevelInfo, "persisted run result")
	if run.Status == domain.AgentRunStatusCompleted {
		m.appendRunEvent(run, domain.RunActionComplete, domain.EventLevelInfo, "run completed")
	} else {
		m.appendRunEvent(run, domain.RunActionFail, domain.EventLevelError, "run failed or stopped")
	}
}

func (m *Manager) finalizeContractedOutput(
	ctx context.Context,
	provider appruntime.Provider,
	agent domain.Agent,
	run *domain.AgentRun,
	prompt string,
	plainTextResult string,
	logWriter app.RunLogWriter,
) error {
	structuredProvider, ok := provider.(appruntime.StructuredOutputProvider)
	if !ok {
		return fmt.Errorf("provider %q does not support structured output", provider.Name())
	}
	m.appendRunEvent(*run, domain.RunActionOutputFinalizeStart, domain.EventLevelInfo, "finalize structured output")
	finalizer := appruntime.NewOutputFinalizer(structuredProvider, NewContractValidator(), 1)
	result, err := finalizer.Finalize(ctx, appruntime.OutputFinalizationRequest{
		RunID:           run.ID,
		AgentName:       agent.Name,
		RevisionID:      agent.Revision,
		Model:           agent.Vendor.Model,
		OutputSchemaRaw: agent.Contract.OutputSchemaRaw,
		PlainTextResult: plainTextResult,
		History: []appruntime.ProviderMessage{
			{Role: appruntime.ProviderRoleUser, Content: prompt},
			{Role: appruntime.ProviderRoleAssistant, Content: plainTextResult},
		},
	})
	if err != nil {
		m.appendRunEvent(*run, domain.RunActionOutputFinalizeFail, domain.EventLevelError, err.Error())

		return err
	}
	run.ResultFormat = domain.ResultFormatJSON
	run.Result = string(result.OutputJSON)
	run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
	_, _ = io.WriteString(logWriter, run.Result)
	m.appendRunEvent(*run, domain.RunActionOutputFinalizeDone, domain.EventLevelInfo, "structured output finalized")

	return nil
}

type reActFinalOutput struct {
	FinalText string `json:"final_text"`
}

func (m *Manager) runReActProvider(
	ctx context.Context,
	provider appruntime.ReActProvider,
	agent domain.Agent,
	revision domain.AgentRevision,
	run *domain.AgentRun,
	input domain.RuntimeInput,
	logWriter app.RunLogWriter,
) {
	adapter := appruntime.NewReActAdapterWithRequest(provider, appruntime.ReActRequest{
		RunID:      run.ID,
		AgentName:  agent.Name,
		RevisionID: agent.Revision,
		Model:      agent.Vendor.Model,
		Prompt:     agent.Prompt,
		MCPServers: agent.MCPServers,
		Access:     domain.AccessPolicy{},
		MaxSteps:   8,
	})
	dynamicTools := m.dynamicToolsForRun(agent, revision, run)
	inputJSON := runtimeInputJSONForRun(input)
	if len(inputJSON) == 0 {
		inputJSON = json.RawMessage(`{}`)
	}
	runner, err := goagentagent.NewDynamicAgent(goagentagent.DynamicAgentConfig{
		Name:            dynamicAgentName(agent.Name),
		Behavior:        agent.Prompt,
		LLM:             adapter,
		Tools:           dynamicTools,
		OutputSchemaRaw: json.RawMessage(`{"type":"object","required":["final_text"],"properties":{"final_text":{"type":"string"}}}`),
		MaxSteps:        8,
	})
	if err != nil {
		m.failRun(run, "react_configuration_invalid", err)

		return
	}
	result, err := runner.Run(ctx, inputJSON)
	if err != nil {
		switch {
		case errors.Is(err, goagentagent.ErrToolNotFound):
			m.failRun(run, "tool_denied", err)
		case errors.Is(err, goagentagent.ErrLimitReached):
			m.failRun(run, "tool_limit_reached", err)
		case errors.Is(err, appruntime.ErrReActFailed):
			m.failRun(run, "react_failed", err)
		case errors.Is(err, context.Canceled):
			run.Status = domain.AgentRunStatusStopped
		default:
			m.failRun(run, "react_failed", err)
		}
		m.appendRunEvent(*run, domain.RunActionProviderFail, domain.EventLevelError, err.Error())

		return
	}
	var output reActFinalOutput
	if err := json.Unmarshal(result.OutputJSON, &output); err != nil {
		m.failRun(run, "react_failed", err)

		return
	}
	run.Status = domain.AgentRunStatusCompleted
	run.ResultFormat = domain.ResultFormatText
	run.Result = output.FinalText
	run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
	_, _ = io.WriteString(logWriter, run.Result)
	m.appendRunEvent(*run, domain.RunActionReActStep, domain.EventLevelInfo, "react loop completed")
}

func (m *Manager) failRun(run *domain.AgentRun, code string, err error) {
	run.Status = domain.AgentRunStatusFailed
	run.ErrorCode = code
	run.ErrorMessage = err.Error()
	run.Result = fmt.Sprintf("run failed: %s", err.Error())
	run.ResultSummary = appresult.Summarize(run.Result, appresult.DefaultSummaryLimit)
}

func (m *Manager) dynamicToolsForRun(
	agent domain.Agent,
	revision domain.AgentRevision,
	run *domain.AgentRun,
) []goagentllm.DynamicTool {
	tools := make([]goagentllm.DynamicTool, 0, len(agent.Tools))
	usage := make(map[string]int)
	for _, tool := range agent.Tools {
		toolForRun := tool
		toolForRun.Command = resolveToolCommandForRun(agent, revision, tool)
		toolForRun.Env = buildToolProcessEnv(revision.Environment, tool.Env)
		tools = append(tools, goagentllm.DynamicTool{
			Name:                tool.Name,
			Description:         "Execute " + tool.Name,
			ParametersSchemaRaw: json.RawMessage(`{"type":"object","additionalProperties":true}`),
			Callback: func(ctx context.Context, call goagentllm.DynamicToolCall) (goagentllm.DynamicToolResult, error) {
				usage[call.ToolName]++
				if usage[call.ToolName] > 1 {
					return goagentllm.DynamicToolResult{}, goagentagent.ErrLimitReached
				}
				result, err := m.executeReActTool(ctx, agent, revision, *run, toolForRun)
				if err != nil {
					return goagentllm.DynamicToolResult{}, err
				}
				body, err := ToolResultObservationJSON(result)
				if err != nil {
					return goagentllm.DynamicToolResult{}, err
				}

				return goagentllm.DynamicToolResult{
					ToolCallID:  call.ID,
					ContentJSON: body,
				}, nil
			},
		})
	}

	return tools
}

func (m *Manager) executeReActTool(
	ctx context.Context,
	agent domain.Agent,
	revision domain.AgentRevision,
	run domain.AgentRun,
	tool domain.ToolPermission,
) (appruntime.ToolResult, error) {
	if m.tools == nil {
		return appruntime.ToolResult{}, fmt.Errorf("tool executor is required")
	}
	if err := validateCustomToolArtifact(revision, tool); err != nil {
		return appruntime.ToolResult{}, err
	}
	if err := validateHostToolExecutable(tool); err != nil {
		return appruntime.ToolResult{}, err
	}
	m.appendRunEvent(run, domain.RunActionToolExecuteStart, domain.EventLevelInfo, "execute tool "+tool.Name)
	startedAt := m.now()
	result, err := m.tools.Execute(ctx, appruntime.ToolRequest{
		RunID:   run.ID,
		Agent:   agent,
		Tool:    tool,
		WorkDir: run.WorkDir,
	})
	completedAt := m.now()
	if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
		_ = repo.CreateToolExecution(context.Background(), domain.ToolExecution{
			ID:             uuid.NewString(),
			RunID:          run.ID,
			AgentName:      run.AgentName,
			ToolName:       tool.Name,
			CommandSummary: tool.Command,
			StartedAt:      startedAt,
			CompletedAt:    &completedAt,
			ExitCode:       result.ExitCode,
			TimedOut:       result.TimedOut,
			StdoutSummary:  result.StdoutSummary,
			StderrSummary:  result.StderrSummary,
		})
	}
	if err != nil {
		emitToolExecutionLog(run, tool, result, err)
		m.appendRunEvent(run, domain.RunActionToolExecuteFail, domain.EventLevelError, toolLogMessage("tool failed", tool.Name, result, err))

		return result, err
	}
	emitToolExecutionLog(run, tool, result, nil)
	m.appendRunEvent(run, domain.RunActionToolExecuteComplete, domain.EventLevelInfo, toolLogMessage("tool completed", tool.Name, result, nil))
	m.appendRunEvent(run, domain.RunActionReActToolObservation, domain.EventLevelInfo, "react observed tool "+tool.Name)

	return result, nil
}

func dynamicAgentName(name string) string {
	var builder strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "agent"
	}

	return result
}

func runtimeInputJSONForRun(input domain.RuntimeInput) json.RawMessage {
	if len(input.RawJSON) > 0 {
		return append(json.RawMessage(nil), input.RawJSON...)
	}
	if input.LegacyInputs != nil {
		body, err := json.Marshal(input.LegacyInputs)
		if err == nil {
			return body
		}
	}

	return nil
}

func contractInputDigest(contract *domain.AgentContract) string {
	if contract == nil {
		return ""
	}

	return contract.InputSchemaDigest
}

func contractOutputDigest(contract *domain.AgentContract) string {
	if contract == nil {
		return ""
	}

	return contract.OutputSchemaDigest
}

var inputPlaceholderPattern = regexp.MustCompile(`\{\{inputs\.([A-Za-z0-9_-]+)\}\}`)

func applyRunInputs(agent domain.Agent, inputs map[string]string) (domain.Agent, error) {
	var err error
	if agent.Prompt, err = expandRunInputs(agent.Prompt, inputs); err != nil {
		return domain.Agent{}, err
	}
	for i := range agent.Tools {
		if agent.Tools[i].Command, err = expandRunInputs(agent.Tools[i].Command, inputs); err != nil {
			return domain.Agent{}, err
		}
		if agent.Tools[i].Args, err = expandRunInputList(agent.Tools[i].Args, inputs); err != nil {
			return domain.Agent{}, err
		}
		if agent.Tools[i].Env, err = expandRunInputList(agent.Tools[i].Env, inputs); err != nil {
			return domain.Agent{}, err
		}
	}

	return agent, nil
}

func expandRunInputList(values []string, inputs map[string]string) ([]string, error) {
	expanded := make([]string, 0, len(values))
	for _, value := range values {
		next, err := expandRunInputs(value, inputs)
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, next)
	}

	return expanded, nil
}

func expandRunInputs(value string, inputs map[string]string) (string, error) {
	var missing string
	expanded := inputPlaceholderPattern.ReplaceAllStringFunc(value, func(match string) string {
		key := inputPlaceholderPattern.FindStringSubmatch(match)[1]
		replacement, ok := inputs[key]
		if !ok {
			missing = key

			return match
		}

		return replacement
	})
	if missing != "" {
		return "", fmt.Errorf("missing required input %q", missing)
	}

	return expanded, nil
}

func (m *Manager) executeDeclaredTools(
	ctx context.Context,
	agent domain.Agent,
	revision domain.AgentRevision,
	run domain.AgentRun,
) (string, error) {
	if m.tools == nil || len(agent.Tools) == 0 {
		return "", nil
	}
	var outputs []string
	for _, tool := range agent.Tools {
		if tool.Kind != domain.ToolKindLocalTool &&
			tool.Kind != domain.ToolKindCustomTool &&
			tool.Kind != domain.ToolKindHostTool {
			continue
		}
		startedAt := m.now()
		m.appendRunEvent(run, domain.RunActionToolExecuteStart, domain.EventLevelInfo, "execute tool "+tool.Name)
		toolForRun := tool
		toolForRun.Command = resolveToolCommandForRun(agent, revision, tool)
		toolForRun.Env = buildToolProcessEnv(revision.Environment, tool.Env)
		if err := validateCustomToolArtifact(revision, toolForRun); err != nil {
			m.appendRunEvent(run, domain.RunActionToolExecuteFail, domain.EventLevelError, err.Error())

			return "", err
		}
		if err := validateHostToolExecutable(toolForRun); err != nil {
			m.appendRunEvent(run, domain.RunActionToolExecuteFail, domain.EventLevelError, err.Error())

			return "", err
		}
		result, err := m.tools.Execute(ctx, appruntime.ToolRequest{
			RunID:   run.ID,
			Agent:   agent,
			Tool:    toolForRun,
			WorkDir: run.WorkDir,
		})
		completedAt := m.now()
		execution := domain.ToolExecution{
			ID:             uuid.NewString(),
			RunID:          run.ID,
			AgentName:      run.AgentName,
			ToolName:       tool.Name,
			CommandSummary: toolForRun.Command,
			StartedAt:      startedAt,
			CompletedAt:    &completedAt,
			ExitCode:       result.ExitCode,
			TimedOut:       result.TimedOut,
			StdoutSummary:  result.StdoutSummary,
			StderrSummary:  result.StderrSummary,
		}
		if err != nil {
			execution.ErrorMessage = err.Error()
		}
		if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
			_ = repo.CreateToolExecution(context.Background(), execution)
		}
		if err != nil {
			emitToolExecutionLog(run, tool, result, err)
			m.appendRunEvent(
				run,
				domain.RunActionToolExecuteFail,
				domain.EventLevelError,
				toolLogMessage("tool failed", tool.Name, result, err),
			)

			return strings.Join(outputs, "\n"), err
		}
		emitToolExecutionLog(run, tool, result, nil)
		m.appendRunEvent(
			run,
			domain.RunActionToolExecuteComplete,
			domain.EventLevelInfo,
			toolLogMessage("tool completed", tool.Name, result, nil),
		)
		if result.StdoutSummary != "" {
			outputs = append(outputs, fmt.Sprintf("%s stdout: %s", tool.Name, result.StdoutSummary))
		}
		if result.StderrSummary != "" {
			outputs = append(outputs, fmt.Sprintf("%s stderr: %s", tool.Name, result.StderrSummary))
		}
	}

	return strings.Join(outputs, "\n"), nil
}

func toolLogMessage(prefix string, toolName string, result appruntime.ToolResult, err error) string {
	parts := []string{prefix + " " + toolName}
	if result.StdoutSummary != "" {
		parts = append(parts, "stdout: "+result.StdoutSummary)
	}
	if result.StderrSummary != "" {
		parts = append(parts, "stderr: "+result.StderrSummary)
	}
	if result.ResultSummary != "" {
		parts = append(parts, "result: "+result.ResultSummary)
	}
	parts = append(parts, fmt.Sprintf("exit_code: %d", result.ExitCode))
	if result.TimedOut {
		parts = append(parts, "timed_out: true")
	}
	if err != nil {
		parts = append(parts, "error: "+err.Error())
	}

	return strings.Join(parts, " | ")
}

func emitToolExecutionLog(
	run domain.AgentRun,
	tool domain.ToolPermission,
	result appruntime.ToolResult,
	err error,
) {
	event := domain.RunActionToolExecuteComplete
	level := slog.LevelInfo
	attrs := []any{
		"event", event,
		"agent", run.AgentName,
		"run_id", run.ID,
		"revision", run.AgentRevision,
		"tool", tool.Name,
		"tool_kind", string(tool.Kind),
		"stdout", result.StdoutSummary,
		"stderr", result.StderrSummary,
		"result", result.ResultSummary,
		"exit_code", result.ExitCode,
		"timed_out", result.TimedOut,
	}
	if err != nil {
		event = domain.RunActionToolExecuteFail
		level = slog.LevelError
		attrs[1] = event
		attrs = append(attrs, "error", err.Error())
	}
	slog.Log(context.Background(), level, event, attrs...)
}

func resolveToolCommand(sourcePath, command string) string {
	if filepath.IsAbs(command) || sourcePath == "" || filepath.Base(command) == command {
		return command
	}

	resolved := filepath.Join(filepath.Dir(sourcePath), command)
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return resolved
	}

	return absolute
}

func resolveToolCommandForRun(agent domain.Agent, revision domain.AgentRevision, tool domain.ToolPermission) string {
	if tool.Kind == domain.ToolKindCustomTool && revision.ArtifactPath != "" {
		command := tool.Command
		if !filepath.IsAbs(command) && !commandWithinArtifact(revision.ArtifactPath, command) {
			command = filepath.Join(revision.ArtifactPath, command)
		}

		return absolutePath(command)
	}

	return resolveToolCommand(agent.DefinitionSource, tool.Command)
}

func absolutePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	return absolute
}

func commandWithinArtifact(artifactPath, command string) bool {
	if artifactPath == "" || command == "" {
		return false
	}
	relative, err := filepath.Rel(filepath.Clean(artifactPath), filepath.Clean(command))
	if err != nil {
		return false
	}

	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func validateCustomToolArtifact(revision domain.AgentRevision, tool domain.ToolPermission) error {
	if tool.Kind != domain.ToolKindCustomTool || revision.ArtifactPath == "" {
		return nil
	}
	if _, err := os.Stat(tool.Command); err != nil {
		return fmt.Errorf("custom tool artifact %q is not executable from revision %s: %w", tool.Command, revision.RevisionID, err)
	}

	return nil
}

func validateHostToolExecutable(tool domain.ToolPermission) error {
	if tool.Kind != domain.ToolKindHostTool {
		return nil
	}
	if strings.TrimSpace(tool.Command) == "" {
		return fmt.Errorf("host tool executable is required for %s", tool.Name)
	}
	if filepath.IsAbs(tool.Command) {
		if _, err := os.Stat(tool.Command); err != nil {
			return fmt.Errorf("host tool executable %q is not available: %w", tool.Command, err)
		}

		return nil
	}
	if _, err := exec.LookPath(tool.Command); err != nil {
		return fmt.Errorf("host tool executable %q is not available: %w", tool.Command, err)
	}

	return nil
}

func buildToolProcessEnv(revisionEnv []domain.RevisionEnvironment, toolEnv []string) []string {
	values := make(map[string]string, len(revisionEnv)+len(toolEnv))
	for _, entry := range revisionEnv {
		values[entry.Key] = entry.Value
	}
	for _, entry := range toolEnv {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		values[key] = value
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}

	return env
}

func (m *Manager) appendRunEvent(
	run domain.AgentRun,
	action string,
	level domain.EventLevel,
	message string,
) {
	repo := m.runtimeDBs.Events(run.AgentName)
	if repo == nil {
		return
	}
	_ = repo.Append(context.Background(), domain.RuntimeEvent{
		ID:             uuid.NewString(),
		AgentName:      run.AgentName,
		RunID:          run.ID,
		EventType:      action,
		Level:          level,
		Message:        message,
		AttributesJSON: "{}",
		CreatedAt:      m.now(),
	})
}

func (m *Manager) findActive(agentName, runID string) (*activeRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, active := range m.active {
		if runID != "" && active.run.ID != runID {
			continue
		}
		if agentName != "" && active.run.AgentName != agentName {
			continue
		}

		return active, nil
	}

	return nil, domain.ErrNotFound
}

func (m *Manager) setActiveRun(run domain.AgentRun) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if active := m.active[run.ID]; active != nil {
		active.run = run
	}
}

func (m *Manager) removeActive(runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, runID)
}
