package agentdclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) Apply(ctx context.Context, request ApplyRequest) (ApplyResponse, error) {
	var response ApplyResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/agents/apply", request, &response); err != nil {
		return ApplyResponse{}, err
	}

	return response, nil
}

func (c *Client) ListAgents(ctx context.Context) ([]AgentSummary, error) {
	var response AgentListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/agents", nil, &response); err != nil {
		return nil, err
	}

	return response.Agents, nil
}

func (c *Client) InspectAgent(ctx context.Context, name string) (AgentDetail, error) {
	var response AgentDetail
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/v1/agents/%s", url.PathEscape(name)), nil, &response); err != nil {
		return AgentDetail{}, err
	}

	return response, nil
}
