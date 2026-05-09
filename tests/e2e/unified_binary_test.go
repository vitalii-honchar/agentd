package e2e

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUnifiedAgentdBinaryAdvertisesDaemonMode(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	cmd := exec.Command("go", "run", "./cmd/agentd", "--help")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agentd --help: %v\n%s", err, output)
	}
	help := string(output)
	for _, want := range []string{"--daemon", "-d", "--deamon"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q:\n%s", want, help)
		}
	}
}

func TestUnifiedAgentdBinaryRejectsDaemonSubcommand(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("agentd daemon mode is supported on Linux and macOS")
	}

	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	cmd := exec.Command("go", "run", "./cmd/agentd", "--daemon", "list")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("agentd --daemon list succeeded unexpectedly:\n%s", output)
	}
	if !strings.Contains(string(output), "daemon mode cannot be combined with a client subcommand") {
		t.Fatalf("unexpected error output:\n%s", output)
	}
}
