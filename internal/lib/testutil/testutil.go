package testutil

import (
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
