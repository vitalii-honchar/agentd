package agentdclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultServerURL = "http://127.0.0.1:18080"

type Config struct {
	ServerURL  string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Client struct {
	baseURL *url.URL
	http    *http.Client
}

func New(cfg Config) (*Client, error) {
	serverURL := strings.TrimSpace(cfg.ServerURL)
	if serverURL == "" {
		serverURL = DefaultServerURL
	}
	baseURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("parse server url: %w", err)
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{baseURL: baseURL, http: httpClient}, nil
}

func (c *Client) Health(ctx context.Context) error {
	var response struct {
		Status string `json:"status"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/health", nil, &response); err != nil {
		return err
	}
	if response.Status != "ok" {
		return fmt.Errorf("daemon health status %q", response.Status)
	}

	return nil
}

func (c *Client) doJSON(ctx context.Context, method string, path string, body any, out any) error {
	request, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return &Error{
			Code:    ErrorCodeDaemonUnavailable,
			Message: fmt.Sprintf("daemon request %s %s: %v", method, path, err),
		}
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

func (c *Client) newRequest(ctx context.Context, method string, path string, body any) (*http.Request, error) {
	relative, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse daemon request path: %w", err)
	}
	target := c.baseURL.ResolveReference(relative)
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode daemon request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, target.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("new daemon request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func decodeError(response *http.Response) error {
	var payload struct {
		Error Error `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return &Error{Code: ErrorCodeDaemonError, Message: response.Status, HTTPStatus: response.StatusCode}
	}
	payload.Error.HTTPStatus = response.StatusCode
	if payload.Error.Message == "" {
		payload.Error.Message = response.Status
	}
	if payload.Error.Code == "" {
		payload.Error.Code = ErrorCodeDaemonError
	}

	return &payload.Error
}
