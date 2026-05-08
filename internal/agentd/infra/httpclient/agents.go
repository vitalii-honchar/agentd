package httpclient

import (
	"context"
	stdhttp "net/http"

	"github.com/vitalii-honchar/agentd/internal/agentd/app"
)

func (c *Client) Apply(ctx context.Context, request app.ApplyRequest) (app.ApplyResponse, error) {
	var response app.ApplyResponse
	if err := c.doJSON(ctx, stdhttp.MethodPost, "/v1/agents/apply", applyRequest{
		SourcePath: request.SourcePath,
		Markdown:   request.Markdown,
	}, &response); err != nil {
		return app.ApplyResponse{}, err
	}

	return response, nil
}

type applyRequest struct {
	SourcePath string `json:"source_path"`
	Markdown   string `json:"markdown"`
}
