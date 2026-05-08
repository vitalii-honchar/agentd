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

func (c *Client) ListRuns(ctx context.Context, includeAll bool) (app.RunListResponse, error) {
	runs, err := c.client.ListRuns(ctx, includeAll)
	if err != nil {
		return app.RunListResponse{}, err
	}
	response := app.RunListResponse{Runs: make([]app.RunSummary, 0, len(runs))}
	for _, run := range runs {
		response.Runs = append(response.Runs, toAppRunSummary(run))
	}

	return response, nil
}
