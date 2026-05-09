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

	logsA := getLogsByRunID(t, stack.server, runA.RunID)
	logsB := getLogsByRunID(t, stack.server, runB.RunID)

	if !logsContain(logsA.Entries, "output for agent-a") {
		t.Fatalf("agent-a logs: %#v", logsA.Entries)
	}
	if logsContain(logsA.Entries, "output for agent-b") {
		t.Fatalf("agent-a logs include agent-b output: %#v", logsA.Entries)
	}
	if !logsContain(logsB.Entries, "output for agent-b") {
		t.Fatalf("agent-b logs: %#v", logsB.Entries)
	}
}

func TestLogsAreIsolatedAcrossRunsForSameAgent(t *testing.T) {
	t.Parallel()

	stack := newRuntimeStackWithProvider(t, outputE2EProvider{})
	postApply(t, stack.server, "agent-a.md", runtimeDefinition("agent-a"))

	first := postRun(t, stack.server, "agent-a")
	waitForE2ERunStatus(t, stack.runtimeDBs, "agent-a", first.RunID, domain.AgentRunStatusCompleted)
	second := postRun(t, stack.server, "agent-a")
	waitForE2ERunStatus(t, stack.runtimeDBs, "agent-a", second.RunID, domain.AgentRunStatusCompleted)

	firstLogs := getLogsByRunID(t, stack.server, first.RunID)
	secondLogs := getLogsByRunID(t, stack.server, second.RunID)

	if firstLogs.RunID != first.RunID {
		t.Fatalf("first logs run id: got %q want %q", firstLogs.RunID, first.RunID)
	}
	if secondLogs.RunID != second.RunID {
		t.Fatalf("second logs run id: got %q want %q", secondLogs.RunID, second.RunID)
	}
	if !logsContain(firstLogs.Entries, first.RunID) || logsContain(firstLogs.Entries, second.RunID) {
		t.Fatalf("first run logs are not isolated: %#v", firstLogs.Entries)
	}
	if !logsContain(secondLogs.Entries, second.RunID) || logsContain(secondLogs.Entries, first.RunID) {
		t.Fatalf("second run logs are not isolated: %#v", secondLogs.Entries)
	}
}

func logsContain(entries []model.LogEntry, text string) bool {
	for _, entry := range entries {
		if strings.Contains(entry.Line, text) || strings.Contains(entry.Message, text) {
			return true
		}
	}

	return false
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
		Output:    "output for " + request.AgentName + " run " + request.RunID,
	}, nil
}

func getLogsByRunID(t *testing.T, server interface{ Handler() http.Handler }, runID string) model.LogsResponse {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID+"/logs", nil)
	request.RemoteAddr = "127.0.0.1:12345"
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
