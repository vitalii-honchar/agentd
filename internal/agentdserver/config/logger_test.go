package config

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestConfigureLoggerIncludesTimestamp(t *testing.T) {
	originalLogger := slog.Default()
	originalStdout := os.Stdout
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
		os.Stdout = originalStdout
	})

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stdout = writer

	ConfigureLogger(&Config{})
	slog.Info("logger smoke")
	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}
	var output bytes.Buffer
	if _, err := io.Copy(&output, reader); err != nil {
		t.Fatalf("Copy log output: %v", err)
	}

	if !strings.Contains(output.String(), "time=") {
		t.Fatalf("log output does not include timestamp: %q", output.String())
	}
	if !strings.Contains(output.String(), `msg="logger smoke"`) {
		t.Fatalf("log output does not include message: %q", output.String())
	}
}
