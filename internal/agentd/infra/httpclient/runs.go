package httpclient

import (
	"context"

	"github.com/vitalii-honchar/agentd/internal/agentd/app"
)

func (c *Client) Execute(ctx context.Context, agentName string) (app.RunResponse, error) {
	response, err := c.client.Execute(ctx, agentName, nil)
	if err != nil {
		return app.RunResponse{}, err
	}

	return toAppRunResponse(response), nil
}

func (c *Client) Stop(ctx context.Context, request app.StopRequest) (app.RunResponse, error) {
	response, err := c.client.Stop(ctx, request.AgentName, request.RunID)
	if err != nil {
		return app.RunResponse{}, err
	}

	return toAppRunResponse(response), nil
}
