package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
	codexadapter "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/llm/codex"
)

func TestCodexProviderFixtureWithFakeCLI(t *testing.T) {
	t.Parallel()

	fakeCodex := writeFakeCodexCLI(t)
	provider := codexadapter.NewProvider(codexadapter.Config{
		Command: fakeCodex,
		Timeout: 2 * time.Second,
	})
	stack := newRuntimeStackWithProvider(t, provider)

	fixturePath := filepath.Clean("../fixtures/codex-provider-agent.md")
	body, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("ReadFile fixture: %v", err)
	}
	postApply(t, stack.server, fixturePath, string(body))
	response := postRunRaw(t, stack.server, "codex-provider-agent", `{"input":{"topic":"agentd"}}`)
	if response.Code != http.StatusAccepted {
		t.Fatalf("run status: got %d body %s", response.Code, response.Body.String())
	}
	var runResponse model.RunResponse
	if err := json.NewDecoder(response.Body).Decode(&runResponse); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	waitForE2ERunStatus(
		t,
		stack.runtimeDBs,
		"codex-provider-agent",
		runResponse.RunID,
		domain.AgentRunStatusCompleted,
	)

	run, err := stack.runtimeDBs.Runs("codex-provider-agent").FindByID(context.Background(), runResponse.RunID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if run.ProviderName != "codex" || run.ResultFormat != domain.ResultFormatJSON {
		t.Fatalf("run metadata: %#v", run)
	}
	if run.Result != `{"summary":"codex fixture done"}` {
		t.Fatalf("result: %q", run.Result)
	}
}

func writeFakeCodexCLI(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "codex")
	script := `#!/bin/sh
input=$(cat)
case "$input" in
  *"Return only JSON"*)
    printf '%s\n' '{"type":"message","message":"{\"summary\":\"codex fixture done\"}"}'
    ;;
  *)
    printf '%s\n' '{"type":"message","message":"plain codex fixture result"}'
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("WriteFile fake codex: %v", err)
	}

	return path
}
