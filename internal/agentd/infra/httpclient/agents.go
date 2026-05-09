package httpclient

import (
	"context"

	"github.com/vitalii-honchar/agentd/internal/agentd/app"
	"github.com/vitalii-honchar/agentd/pkg/agentdclient"
)

func (c *Client) Apply(ctx context.Context, request app.ApplyRequest) (app.ApplyResponse, error) {
	response, err := c.client.Apply(ctx, agentdclient.ApplyRequest{
		SourcePath: request.SourcePath,
		Markdown:   request.Markdown,
	})
	if err != nil {
		return app.ApplyResponse{}, err
	}

	return app.ApplyResponse{
		Outcome:        response.Outcome,
		Agent:          toAppAgentDetail(response.Agent),
		RevisionID:     response.RevisionID,
		ArtifactPath:   response.ArtifactPath,
		RevisionStatus: response.RevisionStatus,
		RevisionReused: response.RevisionReused,
	}, nil
}
