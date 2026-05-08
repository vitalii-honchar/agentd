package agentdclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) Logs(ctx context.Context, query LogsQuery) (LogsResult, error) {
	values := url.Values{}
	if query.RunID != "" {
		values.Set("run_id", query.RunID)
	}
	if query.Tail > 0 {
		values.Set("tail", strconv.Itoa(query.Tail))
	}
	path := fmt.Sprintf("/v1/agents/%s/logs", query.AgentName)
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var response LogsResult
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return LogsResult{}, err
	}

	return response, nil
}
