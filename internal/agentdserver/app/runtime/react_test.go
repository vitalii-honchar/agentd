package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	goagentllm "github.com/vitalii-honchar/go-agent/pkg/goagent/llm"
)

func TestReActAdapterMapsToolCallDecision(t *testing.T) {
	t.Parallel()

	provider := &fakeReActProvider{response: ReActResponse{
		Decision:     domain.ReActDecisionToolCall,
		ToolName:     "lookup",
		ToolArgsJSON: `{"topic":"agentd"}`,
		Message:      ProviderMessage{Role: ProviderRoleAssistant, Content: "Need evidence."},
	}}
	adapter := NewReActAdapter(provider)

	message, err := adapter.CallDynamic(context.Background(), []goagentllm.DynamicMessage{
		{Role: goagentllm.DynamicRoleSystem, Content: "system"},
		{Role: goagentllm.DynamicRoleUser, Content: `{"topic":"agentd"}`},
	}, nil)
	if err != nil {
		t.Fatalf("CallDynamic: %v", err)
	}
	if len(message.ToolCalls) != 1 {
		t.Fatalf("tool calls: %#v", message.ToolCalls)
	}
	if message.ToolCalls[0].ToolName != "lookup" || string(message.ToolCalls[0].ArgsJSON) != `{"topic":"agentd"}` {
		t.Fatalf("tool call: %#v", message.ToolCalls[0])
	}
	if message.End {
		t.Fatal("tool-call decision should not end the loop")
	}
}

func TestReActAdapterMapsFinalDecision(t *testing.T) {
	t.Parallel()

	provider := &fakeReActProvider{response: ReActResponse{
		Decision:  domain.ReActDecisionFinal,
		FinalText: "done",
		Message:   ProviderMessage{Role: ProviderRoleAssistant, Content: "done"},
	}}
	adapter := NewReActAdapter(provider)

	message, err := adapter.CallDynamic(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("CallDynamic: %v", err)
	}
	if !message.End {
		t.Fatal("final decision should end the loop")
	}
	if message.Content != "done" {
		t.Fatalf("content: got %q", message.Content)
	}
}

func TestReActAdapterMapsFailDecision(t *testing.T) {
	t.Parallel()

	provider := &fakeReActProvider{response: ReActResponse{
		Decision: domain.ReActDecisionFail,
		Failure:  "tool denied",
	}}
	adapter := NewReActAdapter(provider)

	_, err := adapter.CallDynamic(context.Background(), nil, nil)
	if !errors.Is(err, ErrReActFailed) {
		t.Fatalf("CallDynamic error: got %v want %v", err, ErrReActFailed)
	}
}

type fakeReActProvider struct {
	response ReActResponse
	request  ReActRequest
	err      error
}

func (p *fakeReActProvider) Name() string {
	return "fake"
}

func (p *fakeReActProvider) Decide(_ context.Context, request ReActRequest) (ReActResponse, error) {
	p.request = request
	if p.err != nil {
		return ReActResponse{}, p.err
	}

	return p.response, nil
}
