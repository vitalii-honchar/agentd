package runtime

import (
	"fmt"
	"path/filepath"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type RevisionArtifactService struct {
	workRoot string
}

type RevisionArtifactRequest struct {
	Definition domain.AgentDefinition
	RevisionID  string
}

type RevisionArtifactResult struct {
	Revision domain.AgentRevision
}

func NewRevisionArtifactService(workRoot string) (*RevisionArtifactService, error) {
	if workRoot == "" {
		return nil, fmt.Errorf("revision artifact work root is required")
	}

	return &RevisionArtifactService{workRoot: workRoot}, nil
}

func (s *RevisionArtifactService) ArtifactPath(agentName, revisionID string) (string, error) {
	if !domain.IsValidAgentName(agentName) {
		return "", fmt.Errorf("%w: invalid agent name %q", domain.ErrInvalidDefinition, agentName)
	}
	if revisionID == "" {
		return "", fmt.Errorf("revision id is required")
	}

	return filepath.Join(s.workRoot, agentName, revisionID), nil
}

func (s *RevisionArtifactService) ExecutionWorkDirPath(agentName, executionID string) (string, error) {
	if !domain.IsValidAgentName(agentName) {
		return "", fmt.Errorf("%w: invalid agent name %q", domain.ErrInvalidDefinition, agentName)
	}
	if executionID == "" {
		return "", fmt.Errorf("execution id is required")
	}

	return filepath.Join(s.workRoot, agentName, "executions", executionID), nil
}
