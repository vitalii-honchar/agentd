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

func (c *Client) ResultsByAgent(ctx context.Context, agentName string) (app.AgentResultsResponse, error) {
	results, err := c.client.ResultsByAgent(ctx, agentName)
	if err != nil {
		return app.AgentResultsResponse{}, err
	}
	response := app.AgentResultsResponse{
		AgentName: agentName,
		Results:   make([]app.RunResult, 0, len(results)),
	}
	for _, result := range results {
		response.Results = append(response.Results, toAppRunResult(result))
	}

	return response, nil
}

func (c *Client) ResultByRunID(ctx context.Context, runID string) (app.RunResult, error) {
	result, err := c.client.ResultByRunID(ctx, runID)
	if err != nil {
		return app.RunResult{}, err
	}

	return toAppRunResult(result), nil
}
