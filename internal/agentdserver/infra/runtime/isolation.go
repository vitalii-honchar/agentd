package runtime

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type IsolationBuilder struct {
	workDir string
}

type RunEnvironment struct {
	WorkDir string
	Env     []string
}

func NewIsolationBuilder(workDir string) (*IsolationBuilder, error) {
	if workDir == "" {
		return nil, fmt.Errorf("work dir is required")
	}

	return &IsolationBuilder{workDir: workDir}, nil
}

func (b *IsolationBuilder) Build(agent domain.Agent, runID string) (RunEnvironment, error) {
	if !domain.IsValidAgentName(agent.Name) {
		return RunEnvironment{}, fmt.Errorf("%w: invalid agent name %q", domain.ErrInvalidDefinition, agent.Name)
	}
	if runID == "" {
		return RunEnvironment{}, fmt.Errorf("run id is required")
	}

	runDir := filepath.Join(b.workDir, agent.Name, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return RunEnvironment{}, fmt.Errorf("create run work dir: %w", err)
	}

	return RunEnvironment{WorkDir: runDir, Env: []string{}}, nil
}
