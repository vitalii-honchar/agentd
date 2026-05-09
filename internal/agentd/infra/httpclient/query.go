package httpclient

import (
	"context"

	"github.com/vitalii-honchar/agentd/internal/agentd/app"
	"github.com/vitalii-honchar/agentd/pkg/agentdclient"
)

func (c *Client) List(ctx context.Context) (app.ListResponse, error) {
	agents, err := c.client.ListAgents(ctx)
	if err != nil {
		return app.ListResponse{}, err
	}
	response := app.ListResponse{Agents: make([]app.AgentSummary, 0, len(agents))}
	for _, agent := range agents {
		response.Agents = append(response.Agents, toAppAgentSummary(agent))
	}

	return response, nil
}

func (c *Client) Inspect(ctx context.Context, agentName string) (app.AgentDetail, error) {
	response, err := c.client.InspectAgent(ctx, agentName)
	if err != nil {
		return app.AgentDetail{}, err
	}

	return toAppAgentDetail(response), nil
}

func (c *Client) ListRevisions(ctx context.Context, agentName string) (app.RevisionListResponse, error) {
	revisions, err := c.client.ListRevisions(ctx, agentName)
	if err != nil {
		return app.RevisionListResponse{}, err
	}
	response := app.RevisionListResponse{Revisions: make([]app.RevisionSummary, 0, len(revisions))}
	for _, revision := range revisions {
		response.Revisions = append(response.Revisions, toAppRevisionSummary(revision))
	}

	return response, nil
}

func (c *Client) InspectRevision(
	ctx context.Context,
	agentName string,
	revisionID string,
) (app.RevisionInspectResponse, error) {
	revision, err := c.client.InspectRevision(ctx, agentName, revisionID)
	if err != nil {
		return app.RevisionInspectResponse{}, err
	}

	return app.RevisionInspectResponse{Revision: toAppRevisionDetail(revision)}, nil
}

func (c *Client) Logs(ctx context.Context, request app.LogsRequest) (app.LogsResponse, error) {
	response, err := c.client.Logs(ctx, agentdclient.LogsQuery{
		RunID: request.RunID,
		Tail:  request.Tail,
	})
	if err != nil {
		return app.LogsResponse{}, err
	}

	entries := make([]app.LogEntry, 0, len(response.Entries))
	for _, entry := range response.Entries {
		entries = append(entries, app.LogEntry{
			Timestamp: entry.Timestamp,
			RunID:     entry.RunID,
			Action:    entry.Action,
			Message:   entry.Message,
			Line:      entry.Line,
		})
	}

	return app.LogsResponse{
		AgentName: response.AgentName,
		RunID:     response.RunID,
		Entries:   entries,
	}, nil
}
