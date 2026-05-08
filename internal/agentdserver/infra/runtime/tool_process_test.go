package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestProcessToolExecutorSuccess(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	script := writeToolScript(t, workDir, "ok.sh", "echo tool-output")
	executor := NewProcessToolExecutor(2 * time.Second)

	result, err := executor.Execute(context.Background(), toolRequest(workDir, script))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.ExitCode != 0 || !strings.Contains(result.StdoutSummary, "tool-output") {
		t.Fatalf("result: %#v", result)
	}
}

func TestProcessToolExecutorNonZeroExit(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	script := writeToolScript(t, workDir, "fail.sh", "echo bad >&2\nexit 7")
	executor := NewProcessToolExecutor(2 * time.Second)

	result, err := executor.Execute(context.Background(), toolRequest(workDir, script))
	if err == nil {
		t.Fatal("Execute error is nil")
	}
	if result.ExitCode != 7 || !strings.Contains(result.StderrSummary, "bad") {
		t.Fatalf("result: %#v", result)
	}
}

func TestProcessToolExecutorTimeout(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	script := writeToolScript(t, workDir, "slow.sh", "sleep 2")
	executor := NewProcessToolExecutor(50 * time.Millisecond)

	result, err := executor.Execute(context.Background(), toolRequest(workDir, script))
	if err == nil {
		t.Fatal("Execute error is nil")
	}
	if !result.TimedOut {
		t.Fatalf("timed out: %#v", result)
	}
}

func TestProcessToolExecutorScopedEnv(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	script := writeToolScript(t, workDir, "env.sh", "echo ${VISIBLE_ENV:-missing}:${SECRET_ENV:-missing}")
	request := toolRequest(workDir, script)
	request.Tool.Env = []string{"VISIBLE_ENV=shown"}
	executor := NewProcessToolExecutor(2 * time.Second)

	result, err := executor.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.StdoutSummary, "shown:missing") {
		t.Fatalf("stdout: %q", result.StdoutSummary)
	}
}

func writeToolScript(t *testing.T, dir, name, body string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	return path
}

func toolRequest(workDir, command string) appruntime.ToolRequest {
	return appruntime.ToolRequest{
		RunID:   "run-1",
		Agent:   domain.Agent{Name: "agent-a"},
		WorkDir: workDir,
		Tool: domain.ToolPermission{
			Name:    "tool",
			Kind:    domain.ToolKindLocalTool,
			Command: command,
		},
	}
}
