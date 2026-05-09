package agentdclient

import (
	"encoding/json"
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
	Name          string           `json:"name"`
	Enabled       bool             `json:"enabled"`
	Status        string           `json:"status"`
	ScheduleType  string           `json:"schedule_type"`
	NextRunAt     *time.Time       `json:"next_run_at,omitempty"`
	LastRunStatus string           `json:"last_run_status,omitempty"`
	Contract      *ContractSummary `json:"contract,omitempty"`
}

type AgentDetail struct {
	AgentSummary
	Revision    string `json:"revision"`
	VendorName  string `json:"vendor_name"`
	VendorModel string `json:"vendor_model"`
	LastRunID   string `json:"last_run_id,omitempty"`
	RecentError string `json:"recent_error,omitempty"`
}

type ContractSummary struct {
	InputSchemaDigest  string `json:"input_schema_digest,omitempty"`
	OutputSchemaDigest string `json:"output_schema_digest,omitempty"`
}

type AgentListResponse struct {
	Agents []AgentSummary `json:"agents"`
}

type RevisionSummary struct {
	RevisionID   string     `json:"revision_id"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	Latest       bool       `json:"latest"`
	SourcePath   string     `json:"source_path,omitempty"`
	ArtifactPath string     `json:"artifact_path,omitempty"`
	FinalizedAt  *time.Time `json:"finalized_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

type RevisionListResponse struct {
	Revisions []RevisionSummary `json:"revisions"`
}

type RevisionInspectResponse struct {
	Revision RevisionDetail `json:"revision"`
}

type RevisionDetail struct {
	RevisionSummary
	Prompt        string                 `json:"prompt,omitempty"`
	Tools         []RevisionTool         `json:"tools,omitempty"`
	ArtifactFiles []RevisionArtifactFile `json:"artifact_files,omitempty"`
	Environment   []RevisionEnvironment  `json:"environment,omitempty"`
}

type RevisionTool struct {
	Name             string   `json:"name"`
	Kind             string   `json:"kind"`
	OriginalCommand  string   `json:"original_command,omitempty"`
	RewrittenCommand string   `json:"rewritten_command,omitempty"`
	HostCommand      string   `json:"host_command,omitempty"`
	CopiedFiles      []string `json:"copied_files,omitempty"`
}

type RevisionArtifactFile struct {
	Path       string `json:"path"`
	SourcePath string `json:"source_path,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	SizeBytes  int64  `json:"size_bytes,omitempty"`
}

type RevisionEnvironment struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Source string `json:"source,omitempty"`
	Masked bool   `json:"masked"`
}

type RunSummary struct {
	RunID         string     `json:"run_id"`
	AgentName     string     `json:"agent_name"`
	AgentRevision string     `json:"agent_revision,omitempty"`
	Status        string     `json:"status"`
	Trigger       string     `json:"trigger,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type RunInput struct {
	Input        json.RawMessage
	LegacyInputs map[string]string
}

type RunListResponse struct {
	Runs []RunSummary `json:"runs"`
}

type RunResult struct {
	RunSummary
	ResultFormat  string          `json:"result_format,omitempty"`
	Result        string          `json:"result,omitempty"`
	ResultJSON    json.RawMessage `json:"result_json,omitempty"`
	ResultSummary string          `json:"result_summary,omitempty"`
	Failure       *Failure        `json:"failure,omitempty"`
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
