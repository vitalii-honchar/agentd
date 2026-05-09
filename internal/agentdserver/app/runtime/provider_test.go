package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

var (
	_ Provider                 = (*compileProvider)(nil)
	_ ReActProvider            = (*compileProvider)(nil)
	_ StructuredOutputProvider = (*compileProvider)(nil)
)

func TestProviderRequestShapes(t *testing.T) {
	t.Parallel()

	plain := ProviderRequest{
		RunID:     "run-1",
		AgentName: "release-notes-helper",
		Model:     "gpt-5",
		Prompt:    "Summarize changes.",
	}
	step := ReActRequest{
		RunID:     plain.RunID,
		AgentName: plain.AgentName,
		Model:     plain.Model,
		History: []ProviderMessage{{
			Role:    ProviderRoleUser,
			Content: "Need the latest commits.",
		}},
		Tools: []domain.ToolPermission{{Name: "fetch_changes", Kind: domain.ToolKindCustomTool}},
	}
	output := StructuredOutputRequest{
		RunID:           plain.RunID,
		AgentName:       plain.AgentName,
		Model:           plain.Model,
		OutputSchemaRaw: `{"type":"object","required":["summary"]}`,
		History:         step.History,
	}

	if step.History[0].Role != ProviderRoleUser || step.Tools[0].Name != "fetch_changes" {
		t.Fatalf("step request: %#v", step)
	}
	if output.OutputSchemaRaw == "" || len(output.History) != 1 {
		t.Fatalf("structured output request: %#v", output)
	}
}

func TestProviderReActAndStructuredResponses(t *testing.T) {
	t.Parallel()

	step := ReActResponse{
		RequestID:    "request-1",
		Decision:     domain.ReActDecisionToolCall,
		Message:      ProviderMessage{Role: ProviderRoleAssistant, Content: "Fetching changes."},
		ToolName:     "fetch_changes",
		ToolArgsJSON: `{"limit":3}`,
	}
	output := StructuredOutputResponse{
		RequestID:  "request-2",
		OutputJSON: json.RawMessage(`{"summary":"done"}`),
	}

	if step.Decision != domain.ReActDecisionToolCall || step.ToolName == "" {
		t.Fatalf("step response: %#v", step)
	}
	if !json.Valid(output.OutputJSON) {
		t.Fatalf("structured output response should be JSON: %s", output.OutputJSON)
	}
}

type compileProvider struct{}

func (compileProvider) Name() string {
	return "compile"
}

func (compileProvider) Execute(context.Context, ProviderRequest) (ProviderResponse, error) {
	return ProviderResponse{}, nil
}

func (compileProvider) Decide(context.Context, ReActRequest) (ReActResponse, error) {
	return ReActResponse{}, nil
}

func (compileProvider) Finalize(context.Context, StructuredOutputRequest) (StructuredOutputResponse, error) {
	return StructuredOutputResponse{}, nil
}
