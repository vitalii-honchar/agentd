package e2e

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cliconfig "github.com/vitalii-honchar/agentd/internal/agentd/config"
	"github.com/vitalii-honchar/agentd/internal/agentd/infra/httpclient"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestResultAutomationScenario(t *testing.T) {
	t.Parallel()

	stack := newRuntimeStackWithProvider(t, outputE2EProvider{})
	httpServer := httptest.NewServer(stack.server.Handler())
	t.Cleanup(httpServer.Close)

	cfg := &cliconfig.Config{
		ServerURL:      httpServer.URL,
		OutputFormat:   cliconfig.OutputJSON,
		RequestTimeout: 2 * time.Second,
	}
	client, err := httpclient.New(cfg)
	if err != nil {
		t.Fatalf("New HTTP client: %v", err)
	}

	definitionPath := filepath.Join(t.TempDir(), "release-notes-helper.md")
	if err := os.WriteFile(definitionPath, []byte(releaseNotesDefinition()), 0o600); err != nil {
		t.Fatalf("write definition: %v", err)
	}
	runCLI(t, cfg, client, "apply", definitionPath)

	executeOut := runCLI(t, cfg, client, "execute", "release-notes-helper")
	var executeBody struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(executeOut), &executeBody); err != nil {
		t.Fatalf("decode execute output: %v output=%s", err, executeOut)
	}
	if executeBody.RunID == "" {
		t.Fatalf("execute output missing run_id: %s", executeOut)
	}
	waitForE2ERunStatus(
		t,
		stack.runtimeDBs,
		"release-notes-helper",
		executeBody.RunID,
		domain.AgentRunStatusCompleted,
	)

	resultOut := runCLI(t, cfg, client, "result", executeBody.RunID)
	if !strings.Contains(resultOut, "output for release-notes-helper") {
		t.Fatalf("result output: %q", resultOut)
	}
	var resultBody struct {
		RunID  string `json:"run_id"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(resultOut), &resultBody); err != nil {
		t.Fatalf("decode result output: %v output=%s", err, resultOut)
	}
	if resultBody.RunID != executeBody.RunID {
		t.Fatalf("run id: got %q want %q", resultBody.RunID, executeBody.RunID)
	}
}
