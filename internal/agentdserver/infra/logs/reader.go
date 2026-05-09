package logs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type RunLogReader struct {
	baseDir string
}

var _ app.RunLogReader = (*RunLogReader)(nil)

func NewRunLogReader(baseDir string) (*RunLogReader, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("run log dir is required")
	}

	return &RunLogReader{baseDir: baseDir}, nil
}

func (r *RunLogReader) Read(ctx context.Context, query app.LogQuery) ([]app.LogEntry, error) {
	if !domain.IsValidAgentName(query.AgentName) {
		return nil, fmt.Errorf("%w: invalid agent name %q", domain.ErrInvalidDefinition, query.AgentName)
	}
	if query.RunID == "" {
		return nil, fmt.Errorf("run id is required")
	}

	path := query.LogPath
	if path == "" {
		path = filepath.Join(r.baseDir, query.AgentName, query.RunID+".log")
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrNotFound
		}

		return nil, fmt.Errorf("open run log: %w", err)
	}
	defer file.Close()

	return scanLogEntries(ctx, file, query.RunID, query.Tail)
}

func scanLogEntries(ctx context.Context, file *os.File, runID string, tail int) ([]app.LogEntry, error) {
	scanner := bufio.NewScanner(file)
	var entries []app.LogEntry
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		entries = append(entries, parseLogLine(runID, scanner.Text()))
		if tail > 0 && len(entries) > tail {
			copy(entries, entries[len(entries)-tail:])
			entries = entries[:tail]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan run log: %w", err)
	}

	return entries, nil
}

func parseLogLine(runID string, line string) app.LogEntry {
	var structured logLine
	if err := json.Unmarshal([]byte(line), &structured); err == nil && isStructuredLogLine(structured) {
		entry := app.LogEntry{
			RunID:   firstNonEmpty(structured.RunID, runID),
			Action:  firstNonEmpty(structured.Action, structured.Event),
			Message: structured.Message,
			Line:    structured.Line,
		}
		if entry.Line == "" {
			entry.Line = entry.Message
		}
		if structured.Timestamp != "" {
			if parsed, err := time.Parse(time.RFC3339Nano, structured.Timestamp); err == nil {
				entry.Timestamp = parsed
			}
		}

		return entry
	}

	return app.LogEntry{RunID: runID, Line: line}
}

func isStructuredLogLine(line logLine) bool {
	return line.Timestamp != "" ||
		line.RunID != "" ||
		line.Action != "" ||
		line.Event != "" ||
		line.Message != ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
