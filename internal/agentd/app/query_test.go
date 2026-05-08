package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
)

var testLogTime = time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)

func TestListCommandCallsClient(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{listResponse: ListResponse{Agents: []AgentSummary{{
		Name:         "release-notes-helper",
		Enabled:      true,
		Status:       "active",
		ScheduleType: "manual",
	}}}}
	var out bytes.Buffer
	cmd := NewListCommand(client, NewOutput(config.OutputText, &out))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !client.listCalled {
		t.Fatal("list client was not called")
	}
	if !strings.Contains(out.String(), "release-notes-helper") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestInspectCommandCallsClient(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{agent: AgentDetail{
		Name:         "release-notes-helper",
		Status:       "active",
		ScheduleType: "manual",
		Revision:     "rev-1",
		VendorName:   "openai",
		VendorModel:  "gpt-5",
	}}
	var out bytes.Buffer
	cmd := NewInspectCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"release-notes-helper"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.inspectAgent != "release-notes-helper" {
		t.Fatalf("inspect agent: got %q", client.inspectAgent)
	}
	if !strings.Contains(out.String(), "openai/gpt-5") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestLogsCommandCallsClientWithRunAndTail(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{logsResponse: LogsResponse{
		AgentName: "release-notes-helper",
		RunID:     "run-1",
		Entries:   []LogEntry{{RunID: "run-1", Line: "completed"}},
	}}
	var out bytes.Buffer
	cmd := NewLogsCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"release-notes-helper", "--run", "run-1", "--tail", "25"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.logsRequest.AgentName != "release-notes-helper" ||
		client.logsRequest.RunID != "run-1" ||
		client.logsRequest.Tail != 25 {
		t.Fatalf("logs request: %#v", client.logsRequest)
	}
	if !strings.Contains(out.String(), "completed") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestLogsCommandFormatsActionLogs(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{logsResponse: LogsResponse{
		AgentName: "release-notes-helper",
		RunID:     "run-1",
		Entries: []LogEntry{{
			Timestamp: testLogTime,
			RunID:     "run-1",
			Action:    "llm.prompt.send",
			Message:   "sent prompt",
		}},
	}}
	var out bytes.Buffer
	cmd := NewLogsCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"release-notes-helper", "--run", "run-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "2026-05-08T10:00:00Z run-1 llm.prompt.send sent prompt") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestRootCommandWiresQueryCommands(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{}
	cmd := NewRootCommand(RootOptions{
		Config: &config.Config{
			ServerURL:      config.DefaultServerURL,
			OutputFormat:   config.OutputText,
			RequestTimeout: config.DefaultRequestTimeout,
		},
		QueryClient: client,
		Out:         &bytes.Buffer{},
		Err:         &bytes.Buffer{},
	})

	requireCommand(t, cmd, "list")
	requireCommand(t, cmd, "inspect")
	requireCommand(t, cmd, "ps")
	requireCommand(t, cmd, "result")
	requireCommand(t, cmd, "logs")
}

type fakeQueryClient struct {
	listCalled         bool
	inspectAgent       string
	logsRequest        LogsRequest
	listRunsIncludeAll bool
	resultAgentName    string
	resultRunID        string

	listResponse         ListResponse
	runsResponse         RunListResponse
	agentResultsResponse AgentResultsResponse
	runResult            RunResult
	agent                AgentDetail
	logsResponse         LogsResponse
	err                  error
}

func (f *fakeQueryClient) List(ctx context.Context) (ListResponse, error) {
	f.listCalled = true
	if f.err != nil {
		return ListResponse{}, f.err
	}

	return f.listResponse, nil
}

func (f *fakeQueryClient) Inspect(_ context.Context, agentName string) (AgentDetail, error) {
	f.inspectAgent = agentName
	if f.err != nil {
		return AgentDetail{}, f.err
	}

	return f.agent, nil
}

func (f *fakeQueryClient) ListRuns(_ context.Context, includeAll bool) (RunListResponse, error) {
	f.listRunsIncludeAll = includeAll
	if f.err != nil {
		return RunListResponse{}, f.err
	}

	return f.runsResponse, nil
}

func (f *fakeQueryClient) ResultsByAgent(_ context.Context, agentName string) (AgentResultsResponse, error) {
	f.resultAgentName = agentName
	if f.err != nil {
		return AgentResultsResponse{}, f.err
	}

	return f.agentResultsResponse, nil
}

func (f *fakeQueryClient) ResultByRunID(_ context.Context, runID string) (RunResult, error) {
	f.resultRunID = runID
	if f.err != nil {
		return RunResult{}, f.err
	}

	return f.runResult, nil
}

func (f *fakeQueryClient) Logs(_ context.Context, request LogsRequest) (LogsResponse, error) {
	f.logsRequest = request
	if f.err != nil {
		return LogsResponse{}, f.err
	}

	return f.logsResponse, nil
}
