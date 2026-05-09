package logs

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestRunLogReaderParsesStructuredEventLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	factory, err := NewRunLogFactory(dir)
	if err != nil {
		t.Fatalf("NewRunLogFactory: %v", err)
	}
	writer, err := factory.Create(context.Background(), "react-agent", "run-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	structured, ok := writer.(*RunLogWriter)
	if !ok {
		t.Fatalf("writer type: %T", writer)
	}
	timestamp := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	if err := structured.WriteEntry(app.LogEntry{
		Timestamp: timestamp,
		RunID:     "run-1",
		Action:    domain.RunActionReActStep,
		Message:   "react step completed",
	}); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}
	if _, err := structured.Write([]byte("plain provider output\n")); err != nil {
		t.Fatalf("Write plain line: %v", err)
	}
	if err := structured.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reader, err := NewRunLogReader(dir)
	if err != nil {
		t.Fatalf("NewRunLogReader: %v", err)
	}
	entries, err := reader.Read(context.Background(), app.LogQuery{
		AgentName: "react-agent",
		RunID:     "run-1",
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries: %#v", entries)
	}
	if entries[0].Action != domain.RunActionReActStep ||
		entries[0].Message != "react step completed" ||
		!entries[0].Timestamp.Equal(timestamp) {
		t.Fatalf("structured entry: %#v", entries[0])
	}
	if entries[1].Line != "plain provider output" || entries[1].Action != "" {
		t.Fatalf("plain entry: %#v", entries[1])
	}
}

func TestRunLogReaderParsesExternalJSONEventLines(t *testing.T) {
	t.Parallel()

	file, err := os.CreateTemp(t.TempDir(), "run-*.log")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := file.WriteString(`{"event":"provider.fail","message":"provider failed","run_id":"run-2"}` + "\n"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("Seek: %v", err)
	}

	entries, err := scanLogEntries(context.Background(), file, "fallback-run", 0)
	if err != nil {
		t.Fatalf("scanLogEntries: %v", err)
	}
	if len(entries) != 1 || entries[0].RunID != "run-2" || entries[0].Action != domain.RunActionProviderFail {
		t.Fatalf("entry: %#v", entries)
	}
}
