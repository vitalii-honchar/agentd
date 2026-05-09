package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestAgentDefinitionValidateAcceptsManualDefinition(t *testing.T) {
	t.Parallel()

	definition := AgentDefinition{
		Name:     "release-notes-helper",
		Enabled:  true,
		Schedule: Schedule{Type: ScheduleTypeManual},
		Vendor:   Vendor{Name: "openai", Model: "gpt-5"},
		Prompt:   "Summarize changes.",
	}

	if err := definition.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestAgentDefinitionValidateCollectsIssues(t *testing.T) {
	t.Parallel()

	definition := AgentDefinition{
		Name:     "Bad Name",
		Schedule: Schedule{Type: ScheduleTypeManual, Expression: "0 9 * * *"},
		Vendor:   Vendor{Name: "", Model: ""},
		Prompt:   "",
	}

	err := definition.Validate()
	if err == nil {
		t.Fatal("Validate returned nil error")
	}
	if !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("Validate error %v does not match ErrInvalidDefinition", err)
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validate error %T is not ValidationError", err)
	}
	if len(validationErr.Issues) != 5 {
		t.Fatalf("issues: got %d want 5: %#v", len(validationErr.Issues), validationErr.Issues)
	}
}

func TestAgentDefinitionValidateRequiresCronExpression(t *testing.T) {
	t.Parallel()

	definition := AgentDefinition{
		Name:     "daily-pr-review",
		Schedule: Schedule{Type: ScheduleTypeCron},
		Vendor:   Vendor{Name: "openai", Model: "gpt-5"},
		Prompt:   "Review pull requests.",
	}

	err := definition.Validate()
	if err == nil {
		t.Fatal("Validate returned nil error")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validate error %T is not ValidationError", err)
	}
	if validationErr.Issues[0].Field != "schedule.expression" {
		t.Fatalf("first field: got %q want schedule.expression", validationErr.Issues[0].Field)
	}
}

func TestAgentCanExecute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent Agent
		want  error
	}{
		"active enabled": {
			agent: Agent{Enabled: true, Status: AgentStatusActive},
		},
		"disabled flag": {
			agent: Agent{Enabled: false, Status: AgentStatusActive},
			want:  ErrAgentDisabled,
		},
		"disabled status": {
			agent: Agent{Enabled: true, Status: AgentStatusDisabled},
			want:  ErrAgentDisabled,
		},
		"invalid": {
			agent: Agent{Enabled: true, Status: AgentStatusInvalid},
			want:  ErrInvalidState,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tt.agent.CanExecute()
			if !errors.Is(err, tt.want) {
				t.Fatalf("CanExecute error: got %v want %v", err, tt.want)
			}
		})
	}
}

func TestAgentRunStatusHelpers(t *testing.T) {
	t.Parallel()

	active := AgentRun{Status: AgentRunStatusRunning}
	if !active.IsActive() {
		t.Fatal("running run should be active")
	}
	if active.IsTerminal() {
		t.Fatal("running run should not be terminal")
	}

	terminal := AgentRun{Status: AgentRunStatusInterrupted}
	if terminal.IsActive() {
		t.Fatal("interrupted run should not be active")
	}
	if !terminal.IsTerminal() {
		t.Fatal("interrupted run should be terminal")
	}
}

func TestAgentRunStoresTerminalResult(t *testing.T) {
	t.Parallel()

	run := AgentRun{
		Status:        AgentRunStatusCompleted,
		Result:        "Full untrimmed result",
		ResultSummary: "Full untrimmed...",
	}
	if !run.IsTerminal() {
		t.Fatal("completed run should be terminal")
	}
	if run.Result == "" {
		t.Fatal("terminal run result should be stored")
	}
	if run.ResultSummary == "" {
		t.Fatal("terminal run result summary should be stored")
	}
}

func TestAgentContractCapturesSchemasAndDigests(t *testing.T) {
	t.Parallel()

	contract := AgentContract{
		InputSchemaRaw:      `{"type":"object","required":["topic"]}`,
		OutputSchemaRaw:     `{"type":"object","required":["summary"]}`,
		InputSchemaDigest:   "sha256:input",
		OutputSchemaDigest:  "sha256:output",
		CreatedFromRevision: "revision-1",
	}
	definition := AgentDefinition{Contract: &contract}
	agent := Agent{Contract: &contract}
	revision := AgentRevision{
		ContractInputSchemaRaw:     contract.InputSchemaRaw,
		ContractOutputSchemaRaw:    contract.OutputSchemaRaw,
		ContractInputSchemaDigest:  contract.InputSchemaDigest,
		ContractOutputSchemaDigest: contract.OutputSchemaDigest,
		ContractDigest:             "sha256:contract",
	}

	if definition.Contract == nil || agent.Contract == nil {
		t.Fatal("contract should be optional but retained when provided")
	}
	if revision.ContractInputSchemaDigest != "sha256:input" ||
		revision.ContractOutputSchemaDigest != "sha256:output" ||
		revision.ContractDigest != "sha256:contract" {
		t.Fatalf("revision contract metadata: %#v", revision)
	}
}

func TestRuntimeInputCapturesStructuredAndLegacyInputs(t *testing.T) {
	t.Parallel()

	input := RuntimeInput{
		RawJSON:      json.RawMessage(`{"topic":"agentd","limit":3}`),
		LegacyInputs: map[string]string{"topic": "agentd"},
		Source:       RuntimeInputSourceCLI,
	}

	if !json.Valid(input.RawJSON) {
		t.Fatalf("runtime input JSON should be valid: %s", input.RawJSON)
	}
	if input.LegacyInputs["topic"] != "agentd" {
		t.Fatalf("legacy inputs: %#v", input.LegacyInputs)
	}
	if input.Source != RuntimeInputSourceCLI {
		t.Fatalf("source: got %q", input.Source)
	}
}

func TestReActStepCapturesModelDecisionAndObservation(t *testing.T) {
	t.Parallel()

	started := time.Now()
	completed := started.Add(time.Second)
	step := ReActStep{
		StepIndex:          2,
		RunID:              "run-1",
		AgentName:          "release-notes-helper",
		RevisionID:         "revision-1",
		ModelMessage:       "Need repository data.",
		Decision:           ReActDecisionToolCall,
		ToolName:           "fetch_changes",
		ToolArgsJSON:       `{"repo":"agentd"}`,
		ObservationSummary: "3 commits",
		StartedAt:          started,
		CompletedAt:        &completed,
	}

	if step.Decision != ReActDecisionToolCall || step.ToolName != "fetch_changes" {
		t.Fatalf("step decision: %#v", step)
	}
	if !json.Valid([]byte(step.ToolArgsJSON)) {
		t.Fatalf("tool args should be JSON: %s", step.ToolArgsJSON)
	}
	if step.CompletedAt == nil {
		t.Fatal("completed ReAct step should capture completion time")
	}
}

func TestProviderMetadataAndRunResultFormat(t *testing.T) {
	t.Parallel()

	provider := ProviderMetadata{
		Name:                 "codex",
		Model:                "gpt-5.2",
		RequiresDirectAPIKey: false,
		ConfigJSON:           `{"command":"codex","timeout_seconds":120}`,
	}
	run := AgentRun{
		ID:                         "run-1",
		ProviderName:               provider.Name,
		ProviderModel:              provider.Model,
		ProviderRequestID:          "codex-process-1",
		ResultFormat:               ResultFormatJSON,
		Result:                     `{"summary":"done"}`,
		ContractInputSchemaDigest:  "sha256:input",
		ContractOutputSchemaDigest: "sha256:output",
	}

	if provider.RequiresDirectAPIKey {
		t.Fatal("codex provider should not require a direct API key")
	}
	if run.ResultFormat != ResultFormatJSON || !json.Valid([]byte(run.Result)) {
		t.Fatalf("run result: %#v", run)
	}
	if run.ProviderName != "codex" || run.ProviderModel == "" {
		t.Fatalf("provider metadata was not copied to run: %#v", run)
	}
}

func TestToolExecutionCapturesProcessOutcome(t *testing.T) {
	t.Parallel()

	started := time.Now()
	completed := started.Add(time.Second)
	execution := ToolExecution{
		ID:             "tool-run-1",
		RunID:          "run-1",
		AgentName:      "agent",
		ToolName:       "fetch",
		CommandSummary: "tools/fetch.py",
		StartedAt:      started,
		CompletedAt:    &completed,
		ExitCode:       0,
		StdoutSummary:  "ok",
	}
	if execution.RunID != "run-1" || execution.ToolName != "fetch" {
		t.Fatalf("execution identity: %#v", execution)
	}
	if execution.CompletedAt == nil {
		t.Fatal("completed tool execution should capture completion time")
	}
}

func TestAgentRunCanTransitionTo(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		from AgentRunStatus
		to   AgentRunStatus
		want bool
	}{
		"queued to running": {
			from: AgentRunStatusQueued,
			to:   AgentRunStatusRunning,
			want: true,
		},
		"running to completed": {
			from: AgentRunStatusRunning,
			to:   AgentRunStatusCompleted,
			want: true,
		},
		"running to stopped requires stopping": {
			from: AgentRunStatusRunning,
			to:   AgentRunStatusStopped,
		},
		"stopping to completed race": {
			from: AgentRunStatusStopping,
			to:   AgentRunStatusCompleted,
			want: true,
		},
		"terminal cannot transition": {
			from: AgentRunStatusCompleted,
			to:   AgentRunStatusRunning,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			run := AgentRun{Status: tt.from}
			if got := run.CanTransitionTo(tt.to); got != tt.want {
				t.Fatalf("CanTransitionTo: got %v want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidAgentName(t *testing.T) {
	t.Parallel()

	valid := []string{"agent", "agent-1", "agent_1", "agent.1"}
	for _, name := range valid {
		if !IsValidAgentName(name) {
			t.Fatalf("IsValidAgentName(%q) = false", name)
		}
	}

	invalid := []string{"", "Agent", "-agent", "agent name", "agent/"}
	for _, name := range invalid {
		if IsValidAgentName(name) {
			t.Fatalf("IsValidAgentName(%q) = true", name)
		}
	}
}
