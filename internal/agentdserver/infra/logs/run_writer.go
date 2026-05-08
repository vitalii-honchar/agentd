package logs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type RunLogFactory struct {
	baseDir string
}

type RunLogWriter struct {
	*os.File
	path string
}

var _ app.RunLogFactory = (*RunLogFactory)(nil)
var _ app.RunLogWriter = (*RunLogWriter)(nil)

func NewRunLogFactory(baseDir string) (*RunLogFactory, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("run log dir is required")
	}

	return &RunLogFactory{baseDir: baseDir}, nil
}

func (f *RunLogFactory) Create(_ context.Context, agentName, runID string) (app.RunLogWriter, error) {
	if !domain.IsValidAgentName(agentName) {
		return nil, fmt.Errorf("%w: invalid agent name %q", domain.ErrInvalidDefinition, agentName)
	}
	if runID == "" {
		return nil, fmt.Errorf("run id is required")
	}

	dir := filepath.Join(f.baseDir, agentName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create run log dir: %w", err)
	}
	path := filepath.Join(dir, runID+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open run log: %w", err)
	}

	return &RunLogWriter{File: file, path: path}, nil
}

func (w *RunLogWriter) Path() string {
	return w.path
}
