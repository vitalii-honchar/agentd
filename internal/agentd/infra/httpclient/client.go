package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"

	"agentd/internal/agentd/config"
)

type Client struct {
	baseURL *url.URL
	http    *stdhttp.Client
}

func New(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("parse server url: %w", err)
	}

	return &Client{
		baseURL: baseURL,
		http: &stdhttp.Client{
			Timeout: cfg.RequestTimeout,
		},
	}, nil
}

func (c *Client) Health(ctx context.Context) error {
	var response struct {
		Status string `json:"status"`
	}
	if err := c.doJSON(ctx, stdhttp.MethodGet, "/health", nil, &response); err != nil {
		return err
	}
	if response.Status != "ok" {
		return fmt.Errorf("daemon health status %q", response.Status)
	}

	return nil
}

func (c *Client) doJSON(
	ctx context.Context,
	method string,
	path string,
	body any,
	out any,
) error {
	request, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("daemon request %s %s: %w", method, path, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return decodeError(response)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, response.Body)

		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("decode daemon response: %w", err)
	}

	return nil
}

func (c *Client) newRequest(
	ctx context.Context,
	method string,
	path string,
	body any,
) (*stdhttp.Request, error) {
	target := c.baseURL.ResolveReference(&url.URL{Path: strings.TrimPrefix(path, "/")})

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode daemon request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	request, err := stdhttp.NewRequestWithContext(ctx, method, target.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("new daemon request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func decodeError(response *stdhttp.Response) error {
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return fmt.Errorf("daemon returned %s", response.Status)
	}
	if payload.Error.Message == "" {
		return fmt.Errorf("daemon returned %s", response.Status)
	}
	if payload.Error.Code == "" {
		return fmt.Errorf("daemon error: %s", payload.Error.Message)
	}

	return fmt.Errorf("daemon error %s: %s", payload.Error.Code, payload.Error.Message)
}
