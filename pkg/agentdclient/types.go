package agentdclient

import (
	"fmt"
	"time"
)

const (
	ErrorCodeAgentDisabled         = "agent_disabled"
	ErrorCodeAgentNotFound         = "agent_not_found"
	ErrorCodeAgentRunFailed        = "agent_run_failed"
	ErrorCodeDaemonError           = "daemon_error"
	ErrorCodeDaemonUnavailable     = "daemon_unavailable"
	ErrorCodeInvalidQuery          = "invalid_query"
	ErrorCodeInvalidState          = "invalid_state"
	ErrorCodeRemoteClientForbidden = "remote_client_forbidden"
	ErrorCodeRunAlreadyActive      = "run_already_active"
	ErrorCodeRunNotFound           = "run_not_found"
	ErrorCodeRunNotTerminal        = "run_not_terminal"
	ErrorCodeValidationFailed      = "validation_failed"
)

type Error struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Code == "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) IsCode(code string) bool {
	return e != nil && e.Code == code
}

type ApplyRequest struct {
	SourcePath string `json:"source_path"`
	Markdown   string `json:"markdown"`
}

type ApplyResponse struct {
	Outcome        string      `json:"outcome"`
	Agent          AgentDetail `json:"agent"`
	RevisionID     string      `json:"revision_id,omitempty"`
	ArtifactPath   string      `json:"artifact_path,omitempty"`
	RevisionStatus string      `json:"revision_status,omitempty"`
	RevisionReused bool        `json:"revision_reused"`
}

type AgentSummary struct {
	Name          string     `json:"name"`
	Enabled       bool       `json:"enabled"`
	Status        string     `json:"status"`
	ScheduleType  string     `json:"schedule_type"`
	NextRunAt     *time.Time `json:"next_run_at,omitempty"`
	LastRunStatus string     `json:"last_run_status,omitempty"`
}

type AgentDetail struct {
	AgentSummary
	Revision    string `json:"revision"`
	VendorName  string `json:"vendor_name"`
	VendorModel string `json:"vendor_model"`
	LastRunID   string `json:"last_run_id,omitempty"`
	RecentError string `json:"recent_error,omitempty"`
}

type AgentListResponse struct {
	Agents []AgentSummary `json:"agents"`
}

type RunSummary struct {
	RunID       string     `json:"run_id"`
	AgentName   string     `json:"agent_name"`
	Status      string     `json:"status"`
	Trigger     string     `json:"trigger,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type RunListResponse struct {
	Runs []RunSummary `json:"runs"`
}

type RunResult struct {
	RunSummary
	Result        string   `json:"result,omitempty"`
	ResultSummary string   `json:"result_summary,omitempty"`
	Failure       *Failure `json:"failure,omitempty"`
}

type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AgentResultsResponse struct {
	AgentName string      `json:"agent_name"`
	Results   []RunResult `json:"results"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	RunID     string    `json:"run_id,omitempty"`
	Action    string    `json:"action,omitempty"`
	Line      string    `json:"line,omitempty"`
	Message   string    `json:"message,omitempty"`
}

type LogsQuery struct {
	AgentName string
	RunID     string
	Tail      int
}

type LogsResult struct {
	AgentName string     `json:"agent_name"`
	RunID     string     `json:"run_id,omitempty"`
	Entries   []LogEntry `json:"entries"`
}
