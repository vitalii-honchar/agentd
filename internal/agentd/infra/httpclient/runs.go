package httpclient

import (
	"context"
	"fmt"
	stdhttp "net/http"

	"github.com/vitalii-honchar/agentd/internal/agentd/app"
)

func (c *Client) Execute(ctx context.Context, agentName string) (app.RunResponse, error) {
	var response app.RunResponse
	if err := c.doJSON(ctx, stdhttp.MethodPost, fmt.Sprintf("/v1/agents/%s/runs", agentName), nil, &response); err != nil {
		return app.RunResponse{}, err
	}

	return response, nil
}

func (c *Client) Stop(ctx context.Context, request app.StopRequest) (app.RunResponse, error) {
	path := fmt.Sprintf("/v1/agents/%s/runs/stop", request.AgentName)
	if request.RunID != "" {
		path = fmt.Sprintf("/v1/agents/%s/runs/%s/stop", request.AgentName, request.RunID)
	}
	var response app.RunResponse
	if err := c.doJSON(ctx, stdhttp.MethodPost, path, nil, &response); err != nil {
		return app.RunResponse{}, err
	}

	return response, nil
}
