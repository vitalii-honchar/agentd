package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeProcess struct {
	BinDir string
	Path   string
	Log    string
}

func newFakeProcess(t *testing.T, name string, body string) fakeProcess {
	t.Helper()

	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake process bin dir: %v", err)
	}
	logPath := filepath.Join(dir, name+".log")
	path := filepath.Join(binDir, name)
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$0 $*\" >> " + shellQuote(logPath) + "\n" +
		body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake process %q: %v", name, err)
	}

	return fakeProcess{BinDir: binDir, Path: path, Log: logPath}
}

func prependFakeProcessPath(t *testing.T, dir string) {
	t.Helper()

	path := dir
	if existing := os.Getenv("PATH"); existing != "" {
		path += string(os.PathListSeparator) + existing
	}
	t.Setenv("PATH", path)
}

func readFakeProcessLog(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fake process log: %v", err)
	}
	return string(data)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
