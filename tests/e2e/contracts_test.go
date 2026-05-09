package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	daemonhttp "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func TestContractInputValidationRejectsInvalidInputBeforeProvider(t *testing.T) {
	t.Parallel()

	provider := &countingE2EProvider{}
	stack := newRuntimeStackWithProvider(t, provider)
	postApply(t, stack.server, "contracted-agent.md", contractedE2EDefinition())

	response := postRunRaw(t, stack.server, "contracted-agent", `{"input":{"topic":7}}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid input status: got %d want %d body %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if provider.calls.Load() != 0 {
		t.Fatalf("provider was called for invalid contracted input: %d", provider.calls.Load())
	}
}

func TestContractInputValidationAcceptsValidInput(t *testing.T) {
	t.Parallel()

	provider := &countingE2EProvider{}
	stack := newRuntimeStackWithProvider(t, provider)
	postApply(t, stack.server, "contracted-agent.md", contractedE2EDefinition())

	response := postRunRaw(t, stack.server, "contracted-agent", `{"input":{"topic":"agentd"}}`)
	if response.Code != http.StatusAccepted {
		t.Fatalf("valid input status: got %d want %d body %s", response.Code, http.StatusAccepted, response.Body.String())
	}
	var body model.RunResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	waitForE2ERunStatus(t, stack.runtimeDBs, "contracted-agent", body.RunID, domain.AgentRunStatusCompleted)
	if provider.calls.Load() != 1 {
		t.Fatalf("provider calls: got %d want 1", provider.calls.Load())
	}
}

func postRunRaw(t *testing.T, server *daemonhttp.Server, agentName string, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/v1/agents/"+agentName+"/runs", bytes.NewReader([]byte(body)))
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	return response
}

func contractedE2EDefinition() string {
	return `---
name: contracted-agent
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
contract:
  input: |
    {"type":"object","required":["topic"],"properties":{"topic":{"type":"string"}}}
  output: |
    {"type":"object","required":["summary"],"properties":{"summary":{"type":"string"}}}
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: []
---
Summarize the requested topic.`
}

type countingE2EProvider struct {
	calls atomic.Int64
}

func (p *countingE2EProvider) Name() string {
	return "openai"
}

func (p *countingE2EProvider) Execute(
	_ context.Context,
	request appruntime.ProviderRequest,
) (appruntime.ProviderResponse, error) {
	p.calls.Add(1)

	return appruntime.ProviderResponse{
		RequestID: "provider-" + request.RunID,
		Output:    `{"summary":"done"}`,
	}, nil
}

func (p *countingE2EProvider) Finalize(
	_ context.Context,
	request appruntime.StructuredOutputRequest,
) (appruntime.StructuredOutputResponse, error) {
	return appruntime.StructuredOutputResponse{
		RequestID:  "structured-" + request.RunID,
		OutputJSON: json.RawMessage(`{"summary":"done"}`),
	}, nil
}
