package e2e

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cliapp "github.com/vitalii-honchar/agentd/internal/agentd/app"
	cliconfig "github.com/vitalii-honchar/agentd/internal/agentd/config"
	"github.com/vitalii-honchar/agentd/internal/agentd/infra/httpclient"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestCLIOperationsAgainstDaemonAPI(t *testing.T) {
	t.Parallel()

	stack := newRuntimeStackWithProvider(t, outputE2EProvider{})
	httpServer := httptest.NewServer(stack.server.Handler())
	t.Cleanup(httpServer.Close)

	cfg := &cliconfig.Config{
		ServerURL:      httpServer.URL,
		OutputFormat:   cliconfig.OutputText,
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

	applyOut := runCLI(t, cfg, client, "apply", definitionPath)
	if !strings.Contains(applyOut, "created release-notes-helper") {
		t.Fatalf("apply output: %q", applyOut)
	}
	listOut := runCLI(t, cfg, client, "list")
	if !strings.Contains(listOut, "release-notes-helper") {
		t.Fatalf("list output: %q", listOut)
	}
	inspectOut := runCLI(t, cfg, client, "inspect", "release-notes-helper")
	if !strings.Contains(inspectOut, "openai/gpt-5") {
		t.Fatalf("inspect output: %q", inspectOut)
	}

	executeOut := runCLI(t, cfg, client, "execute", "release-notes-helper")
	if !strings.Contains(executeOut, "running release-notes-helper") {
		t.Fatalf("execute output: %q", executeOut)
	}
	run, err := stack.runtimeDBs.Runs("release-notes-helper").FindLatest(context.Background())
	if err != nil {
		t.Fatalf("FindLatest: %v", err)
	}
	waitForE2ERunStatus(
		t,
		stack.runtimeDBs,
		"release-notes-helper",
		run.ID,
		domain.AgentRunStatusCompleted,
	)

	logsOut := runCLI(t, cfg, client, "logs", "release-notes-helper")
	if !strings.Contains(logsOut, "output for release-notes-helper") {
		t.Fatalf("logs output: %q", logsOut)
	}
}

func runCLI(
	t *testing.T,
	cfg *cliconfig.Config,
	client interface {
		cliapp.ApplyClient
		cliapp.ExecuteClient
		cliapp.StopClient
		cliapp.QueryClient
	},
	args ...string,
) string {
	t.Helper()

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := cliapp.NewRootCommand(cliapp.RootOptions{
		Config:        cfg,
		Client:        client,
		ExecuteClient: client,
		StopClient:    client,
		QueryClient:   client,
		Out:           &out,
		Err:           &errOut,
	})
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agentd %v: %v stderr=%s", args, err, errOut.String())
	}

	return out.String()
}
