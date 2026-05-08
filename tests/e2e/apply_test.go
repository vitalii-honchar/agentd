package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	appagent "github.com/vitalii-honchar/agentd/internal/agentdserver/app/agent"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db/repository"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/definition"
	daemonhttp "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func TestApplySmokeCreatedThenUnchanged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	settingsDB, err := db.New("settings", db.Config{
		Path:         filepath.Join(dir, "settings.db"),
		MaxOpenConns: 1,
		Pragmas:      db.PragmasSettings,
	})
	if err != nil {
		t.Fatalf("New settings DB: %v", err)
	}
	t.Cleanup(func() {
		if err := settingsDB.Stop(context.Background()); err != nil {
			t.Fatalf("Stop settings DB: %v", err)
		}
	})
	if err := settingsDB.Start(context.Background()); err != nil {
		t.Fatalf("Start settings DB: %v", err)
	}

	agentRepo, err := repository.NewAgentRepository(settingsDB)
	if err != nil {
		t.Fatalf("NewAgentRepository: %v", err)
	}
	runtimeDir := filepath.Join(dir, "agents")
	runtimeDBs, err := repository.NewRuntimeDBManager(runtimeDir, 1)
	if err != nil {
		t.Fatalf("NewRuntimeDBManager: %v", err)
	}
	t.Cleanup(func() {
		if err := runtimeDBs.Close(context.Background()); err != nil {
			t.Fatalf("Close runtime DBs: %v", err)
		}
	})

	applyUC, err := appagent.NewApplyUseCase(
		appagent.ParserFunc(definition.ParseMarkdown),
		agentRepo,
		runtimeDBs,
	)
	if err != nil {
		t.Fatalf("NewApplyUseCase: %v", err)
	}
	server := daemonhttp.NewServer(daemonhttp.Config{}, daemonhttp.WithApplyUseCase(applyUC))

	first := postApply(t, server, "agent.md", releaseNotesDefinition())
	if first.Outcome != "created" {
		t.Fatalf("first outcome: got %q want created", first.Outcome)
	}
	if first.Agent.Name != "release-notes-helper" {
		t.Fatalf("agent name: got %q", first.Agent.Name)
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "release-notes-helper.db")); err != nil {
		t.Fatalf("runtime DB was not created: %v", err)
	}

	second := postApply(t, server, "agent.md", releaseNotesDefinition())
	if second.Outcome != "unchanged" {
		t.Fatalf("second outcome: got %q want unchanged", second.Outcome)
	}
}

func postApply(
	t *testing.T,
	server *daemonhttp.Server,
	sourcePath string,
	markdown string,
) model.ApplyResponse {
	t.Helper()

	payload, err := json.Marshal(map[string]string{
		"source_path": sourcePath,
		"markdown":    markdown,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(stdhttp.MethodPost, "/v1/agents/apply", bytes.NewReader(payload))
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	if response.Code != stdhttp.StatusOK {
		t.Fatalf("apply status: got %d body %s", response.Code, response.Body.String())
	}

	var body model.ApplyResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return body
}

func releaseNotesDefinition() string {
	return `---
name: release-notes-helper
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: ["api.openai.com"]
---
You summarize recent project changes into concise release notes.`
}
