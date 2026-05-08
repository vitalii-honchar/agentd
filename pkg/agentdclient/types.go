package agentdclient

import (
	"fmt"
	"time"
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

type ApplyRequest struct {
	SourcePath string `json:"source_path"`
	Markdown   string `json:"markdown"`
}

type ApplyResponse struct {
	Outcome string      `json:"outcome"`
	Agent   AgentDetail `json:"agent"`
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
