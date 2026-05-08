package runtime

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestProcessToolExecutorCancelsProcessGroupOnUnix(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("process group cancellation is verified on Linux and macOS")
	}

	workDir := t.TempDir()
	script := writeToolScript(t, workDir, "spawns-child.sh", "(sleep 1; echo late > marker.txt) & wait")
	executor := NewProcessToolExecutor(50 * time.Millisecond)

	result, err := executor.Execute(context.Background(), toolRequest(workDir, script))
	if err == nil {
		t.Fatal("Execute error is nil")
	}
	if !result.TimedOut {
		t.Fatalf("timed out: %#v", result)
	}

	time.Sleep(1200 * time.Millisecond)
	if _, err := os.Stat(workDir + "/marker.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("child process was not cancelled; marker stat error: %v", err)
	}
}
