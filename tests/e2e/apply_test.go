package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	appagent "github.com/vitalii-honchar/agentd/internal/agentdserver/app/agent"
	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db/repository"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/definition"
	daemonhttp "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
	infraruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/runtime"
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

func TestApplyRevisionRunsAfterSourceMutationAndDeletion(t *testing.T) {
	t.Parallel()

	provider := &recordingPromptProvider{}
	stack := newRuntimeStackWithProvider(t, provider)
	stack.manager.SetToolExecutor(infraruntime.NewProcessToolExecutor(5 * time.Second))
	sourceDir := t.TempDir()
	toolsDir := filepath.Join(sourceDir, "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll tools: %v", err)
	}
	toolPath := filepath.Join(toolsDir, "fetch.sh")
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\necho tool-version-one\n"), 0o755); err != nil {
		t.Fatalf("WriteFile v1 tool: %v", err)
	}
	sourcePath := filepath.Join(sourceDir, "agent.md")
	first := postApply(t, stack.server, sourcePath, immutableToolDefinition("Prompt version one."))
	if first.RevisionID == "" {
		t.Fatal("first apply did not return revision ID")
	}
	if _, err := os.Stat(filepath.Join(first.ArtifactPath, "tools", "fetch.sh")); err != nil {
		t.Fatalf("first revision did not copy tool: %v", err)
	}

	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\necho tool-version-two\n"), 0o755); err != nil {
		t.Fatalf("WriteFile v2 tool: %v", err)
	}
	second := postApply(t, stack.server, sourcePath, immutableToolDefinition("Prompt version two."))
	if second.RevisionID == "" || second.RevisionID == first.RevisionID {
		t.Fatalf("second revision: first=%q second=%q", first.RevisionID, second.RevisionID)
	}
	if err := os.RemoveAll(sourceDir); err != nil {
		t.Fatalf("RemoveAll source: %v", err)
	}

	run := postRun(t, stack.server, "immutable-e2e-agent:"+first.RevisionID)
	if run.AgentRevision != first.RevisionID {
		t.Fatalf("run revision: got %q want %q", run.AgentRevision, first.RevisionID)
	}
	waitForE2ERunStatus(t, stack.runtimeDBs, "immutable-e2e-agent", run.RunID, domain.AgentRunStatusCompleted)
	prompt := provider.promptForRun(run.RunID)
	if !strings.Contains(prompt, "Prompt version one.") ||
		!strings.Contains(prompt, "tool-version-one") {
		t.Fatalf("prompt did not use first revision artifact: %q", prompt)
	}
	if strings.Contains(prompt, "Prompt version two.") || strings.Contains(prompt, "tool-version-two") {
		t.Fatalf("prompt used mutated source instead of immutable revision: %q", prompt)
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

func immutableToolDefinition(prompt string) string {
	return `---
name: immutable-e2e-agent
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools:
  - name: fetch
    kind: custom_tool
    command: tools/fetch.sh
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: []
---
` + prompt
}

type recordingPromptProvider struct {
	mu      sync.Mutex
	prompts map[string]string
}

func (p *recordingPromptProvider) Name() string {
	return "openai"
}

func (p *recordingPromptProvider) Execute(
	_ context.Context,
	request appruntime.ProviderRequest,
) (appruntime.ProviderResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.prompts == nil {
		p.prompts = make(map[string]string)
	}
	p.prompts[request.RunID] = request.Prompt

	return appruntime.ProviderResponse{RequestID: "request-" + request.RunID, Output: "ok"}, nil
}

func (p *recordingPromptProvider) promptForRun(runID string) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.prompts[runID]
}
