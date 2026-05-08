package agentdclient

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) Execute(ctx context.Context, name string, inputs map[string]string) (RunSummary, error) {
	var response RunSummary
	var body any
	if len(inputs) > 0 {
		body = map[string]map[string]string{"inputs": inputs}
	}
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/v1/agents/%s/runs", name), body, &response); err != nil {
		return RunSummary{}, err
	}

	return response, nil
}

func (c *Client) Stop(ctx context.Context, agentName string, runID string) (RunSummary, error) {
	path := fmt.Sprintf("/v1/agents/%s/runs/stop", agentName)
	if runID != "" {
		path = fmt.Sprintf("/v1/agents/%s/runs/%s/stop", agentName, runID)
	}
	var response RunSummary
	if err := c.doJSON(ctx, http.MethodPost, path, nil, &response); err != nil {
		return RunSummary{}, err
	}

	return response, nil
}

func (c *Client) ListRuns(ctx context.Context, includeAll bool) ([]RunSummary, error) {
	path := "/v1/runs"
	if includeAll {
		path += "?all=true"
	}
	var response RunListResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}

	return response.Runs, nil
}

func (c *Client) ResultsByAgent(ctx context.Context, name string) ([]RunResult, error) {
	var response AgentResultsResponse
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/v1/agents/%s/results", name), nil, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (c *Client) ResultByRunID(ctx context.Context, runID string) (RunResult, error) {
	var response RunResult
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/v1/runs/%s/result", runID), nil, &response); err != nil {
		return RunResult{}, err
	}

	return response, nil
}
