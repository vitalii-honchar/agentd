package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type ExecuteUseCase struct {
	agents    app.AgentRepository
	revisions app.AgentRevisionRepository
	manager   Manager
}

func NewExecuteUseCase(agents app.AgentRepository, manager Manager) *ExecuteUseCase {
	revisions, _ := agents.(app.AgentRevisionRepository)

	return &ExecuteUseCase{agents: agents, revisions: revisions, manager: manager}
}

func (u *ExecuteUseCase) Execute(ctx context.Context, agentSelector string, inputs map[string]string) (domain.AgentRun, error) {
	agentName, revisionID := splitAgentRevisionSelector(agentSelector)
	agent, err := u.agents.FindByName(ctx, agentName)
	if err != nil {
		return domain.AgentRun{}, err
	}
	if err := agent.CanExecute(); err != nil {
		return domain.AgentRun{}, err
	}
	revision, hasRevision, err := u.resolveRevision(ctx, agent.Name, revisionID)
	if err != nil {
		return domain.AgentRun{}, err
	}
	if hasRevision {
		agent = agentFromRevision(agent, revision)
	}

	return u.manager.Execute(ctx, ExecuteRequest{
		Agent:    agent,
		Revision: revision,
		Trigger:  domain.RunTriggerManual,
		Inputs:   inputs,
	})
}

func (u *ExecuteUseCase) resolveRevision(
	ctx context.Context,
	agentName string,
	revisionID string,
) (domain.AgentRevision, bool, error) {
	if u.revisions == nil {
		return domain.AgentRevision{}, false, nil
	}
	var (
		revision domain.AgentRevision
		err      error
	)
	if revisionID != "" {
		revision, err = u.revisions.FindRevisionByID(ctx, agentName, revisionID)
	} else {
		revision, err = u.revisions.FindLatestFinalizedRevision(ctx, agentName)
	}
	if err != nil {
		if revisionID == "" && errors.Is(err, domain.ErrNotFound) {
			return domain.AgentRevision{}, false, nil
		}

		return domain.AgentRevision{}, false, err
	}
	if revision.Status != domain.AgentRevisionStatusFinalized {
		return domain.AgentRevision{}, false, fmt.Errorf("%w: revision %s is %s", domain.ErrInvalidState, revision.RevisionID, revision.Status)
	}

	return revision, true, nil
}

func splitAgentRevisionSelector(selector string) (string, string) {
	agentName, revisionID, ok := strings.Cut(selector, ":")
	if !ok {
		return selector, ""
	}

	return agentName, revisionID
}

func agentFromRevision(agent domain.Agent, revision domain.AgentRevision) domain.Agent {
	agent.Revision = revision.RevisionID
	agent.Prompt = revision.Prompt
	agent.Vendor = revision.Vendor
	agent.Schedule = revision.Schedule
	agent.Tools = toolsFromRevision(revision.Tools)

	return agent
}

func toolsFromRevision(revisionTools []domain.RevisionTool) []domain.ToolPermission {
	tools := make([]domain.ToolPermission, 0, len(revisionTools))
	for _, revisionTool := range revisionTools {
		command := revisionTool.OriginalCommand
		if revisionTool.Kind == domain.ToolKindCustomTool && revisionTool.RewrittenCommand != "" {
			command = revisionTool.RewrittenCommand
		}
		if revisionTool.Kind == domain.ToolKindHostTool && revisionTool.HostCommand != "" {
			command = revisionTool.HostCommand
		}
		tools = append(tools, domain.ToolPermission{
			AgentName:    revisionTool.AgentName,
			Kind:         revisionTool.Kind,
			Name:         revisionTool.Name,
			Command:      command,
			Args:         append([]string(nil), revisionTool.Args...),
			Env:          append([]string(nil), revisionTool.Env...),
			Timeout:      revisionTool.Timeout,
			ReadPaths:    append([]string(nil), revisionTool.ReadPaths...),
			WritePaths:   append([]string(nil), revisionTool.WritePaths...),
			NetworkAllow: append([]string(nil), revisionTool.NetworkAllow...),
		})
	}

	return tools
}
