package logs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
		entries = append(entries, app.LogEntry{
			RunID: runID,
			Line:  scanner.Text(),
		})
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
