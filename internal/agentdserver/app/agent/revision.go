package agent

import (
	"context"
	"fmt"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type RevisionUseCase struct {
	revisions app.AgentRevisionRepository
}

func NewRevisionUseCase(revisions app.AgentRevisionRepository) (*RevisionUseCase, error) {
	if revisions == nil {
		return nil, fmt.Errorf("agent revision repository is required")
	}

	return &RevisionUseCase{revisions: revisions}, nil
}

func (u *RevisionUseCase) ListRevisions(ctx context.Context, agentName string) ([]domain.AgentRevision, error) {
	return u.revisions.ListRevisions(ctx, agentName)
}

func (u *RevisionUseCase) InspectRevision(
	ctx context.Context,
	agentName string,
	revisionID string,
) (domain.AgentRevision, error) {
	revision, err := u.revisions.FindRevisionByID(ctx, agentName, revisionID)
	if err != nil {
		return domain.AgentRevision{}, err
	}
	for index := range revision.Environment {
		if revision.Environment[index].Masked {
			revision.Environment[index].Value = "********"
		}
	}

	return revision, nil
}
