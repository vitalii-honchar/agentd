package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func TestLogsAreIsolatedAcrossConcurrentAgents(t *testing.T) {
	t.Parallel()

	stack := newRuntimeStackWithProvider(t, outputE2EProvider{})
	postApply(t, stack.server, "agent-a.md", runtimeDefinition("agent-a"))
	postApply(t, stack.server, "agent-b.md", runtimeDefinition("agent-b"))

	runA := postRun(t, stack.server, "agent-a")
	runB := postRun(t, stack.server, "agent-b")
	waitForE2ERunStatus(t, stack.runtimeDBs, "agent-a", runA.RunID, domain.AgentRunStatusCompleted)
	waitForE2ERunStatus(t, stack.runtimeDBs, "agent-b", runB.RunID, domain.AgentRunStatusCompleted)

	logsA := getLogs(t, stack.server, "agent-a")
	logsB := getLogs(t, stack.server, "agent-b")

	if len(logsA.Entries) != 1 || !strings.Contains(logsA.Entries[0].Line, "agent-a") {
		t.Fatalf("agent-a logs: %#v", logsA.Entries)
	}
	if strings.Contains(logsA.Entries[0].Line, "agent-b") {
		t.Fatalf("agent-a logs include agent-b output: %#v", logsA.Entries)
	}
	if len(logsB.Entries) != 1 || !strings.Contains(logsB.Entries[0].Line, "agent-b") {
		t.Fatalf("agent-b logs: %#v", logsB.Entries)
	}
}

type outputE2EProvider struct{}

func (outputE2EProvider) Name() string {
	return "openai"
}

func (outputE2EProvider) Execute(
	_ context.Context,
	request appruntime.ProviderRequest,
) (appruntime.ProviderResponse, error) {
	return appruntime.ProviderResponse{
		RequestID: "provider-" + request.RunID,
		Output:    "output for " + request.AgentName,
	}, nil
}

func getLogs(t *testing.T, server interface{ Handler() http.Handler }, agentName string) model.LogsResponse {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/v1/agents/"+agentName+"/logs", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("logs status: got %d body %s", response.Code, response.Body.String())
	}

	var body model.LogsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode logs response: %v", err)
	}

	return body
}
