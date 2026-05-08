# Contract: Public Go Client

Package path: `github.com/vitalii-honchar/agentd/pkg/agentdclient`

The public client wraps the same local daemon REST contract used by the CLI.
It is intended for local Go integrations that execute agents and retrieve
results programmatically.

## Client Construction

```go
client := agentdclient.New(agentdclient.Config{
    ServerURL: "http://127.0.0.1:18080",
    Timeout:   30 * time.Second,
})
```

## Core Types

```go
type AgentSummary struct {
    Name          string
    Enabled       bool
    Status        string
    ScheduleType  string
    NextRunAt     *time.Time
    LastRunStatus string
}

type RunSummary struct {
    RunID       string
    AgentName   string
    Status      string
    Trigger     string
    StartedAt   *time.Time
    CompletedAt *time.Time
}

type RunResult struct {
    RunID         string
    AgentName     string
    Status        string
    Trigger       string
    StartedAt     *time.Time
    CompletedAt   *time.Time
    Result        string
    ResultSummary string
    Failure       *Failure
}

type Failure struct {
    Code    string
    Message string
}
```

## Operations

```go
Apply(ctx context.Context, sourcePath string, markdown []byte) (ApplyResult, error)
ListAgents(ctx context.Context) ([]AgentSummary, error)
InspectAgent(ctx context.Context, name string) (AgentDetail, error)
Execute(ctx context.Context, name string, inputs map[string]string) (RunSummary, error)
ListRuns(ctx context.Context, includeAll bool) ([]RunSummary, error)
ResultsByAgent(ctx context.Context, name string) ([]RunResult, error)
ResultByRunID(ctx context.Context, runID string) (RunResult, error)
Stop(ctx context.Context, agentName string, runID string) (RunSummary, error)
Logs(ctx context.Context, query LogsQuery) (LogsResult, error)
```

## Error Contract

Errors expose stable daemon error codes:

```go
type Error struct {
    Code       string
    Message    string
    HTTPStatus int
}
```

Required codes:
- `agent_not_found`
- `run_not_found`
- `run_not_terminal`
- `agent_run_failed`
- `validation_failed`
- `daemon_unavailable`
- `remote_client_forbidden`

## Compatibility Rules

- The package must not import `internal/agentdserver`.
- The CLI should use the same types or conversion layer so behavior is
  consistent.
- The client should support minor daemon response additions without breaking.
