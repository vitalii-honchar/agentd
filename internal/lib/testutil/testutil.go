package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "agentd-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

func TempPath(t *testing.T, name string) string {
	t.Helper()

	return filepath.Join(TempDir(t), name)
}

func RequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func HostToolDefinitionMarkdown(agentName string) string {
	if agentName == "" {
		agentName = "host-tool-agent"
	}

	return fmt.Sprintf(`---
name: %s
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools:
  - name: github_api
    kind: host_tool
    command: gh
    args: ["api", "search/repositories"]
access:
  filesystem:
    read: []
    write: []
  network:
    allow: ["api.github.com"]
---
Use the host GitHub CLI to inspect public repositories.`, agentName)
}
