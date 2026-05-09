package agentdclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) Logs(ctx context.Context, query LogsQuery) (LogsResult, error) {
	if query.RunID == "" {
		return LogsResult{}, fmt.Errorf("run ID is required")
	}

	values := url.Values{}
	if query.Tail > 0 {
		values.Set("tail", strconv.Itoa(query.Tail))
	}
	path := fmt.Sprintf("/v1/runs/%s/logs", url.PathEscape(query.RunID))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var response LogsResult
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return LogsResult{}, err
	}

	return response, nil
}
