package httpclient

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"strconv"

	"agentd/internal/agentd/app"
)

func (c *Client) List(ctx context.Context) (app.ListResponse, error) {
	var response app.ListResponse
	if err := c.doJSON(ctx, stdhttp.MethodGet, "/v1/agents", nil, &response); err != nil {
		return app.ListResponse{}, err
	}

	return response, nil
}

func (c *Client) Inspect(ctx context.Context, agentName string) (app.AgentDetail, error) {
	var response app.AgentDetail
	path := fmt.Sprintf("/v1/agents/%s", url.PathEscape(agentName))
	if err := c.doJSON(ctx, stdhttp.MethodGet, path, nil, &response); err != nil {
		return app.AgentDetail{}, err
	}

	return response, nil
}

func (c *Client) Logs(ctx context.Context, request app.LogsRequest) (app.LogsResponse, error) {
	query := url.Values{}
	if request.RunID != "" {
		query.Set("run_id", request.RunID)
	}
	if request.Tail > 0 {
		query.Set("tail", strconv.Itoa(request.Tail))
	}
	path := fmt.Sprintf("/v1/agents/%s/logs", url.PathEscape(request.AgentName))
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var response app.LogsResponse
	if err := c.doJSON(ctx, stdhttp.MethodGet, path, nil, &response); err != nil {
		return app.LogsResponse{}, err
	}

	return response, nil
}
