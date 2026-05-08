package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type ScheduleType string

const (
	ScheduleTypeCron   ScheduleType = "cron"
	ScheduleTypeManual ScheduleType = "manual"
)

type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"
	AgentStatusDisabled AgentStatus = "disabled"
	AgentStatusInvalid  AgentStatus = "invalid"
)

type RunTrigger string

const (
	RunTriggerSchedule RunTrigger = "schedule"
	RunTriggerManual   RunTrigger = "manual"
	RunTriggerRecovery RunTrigger = "recovery"
)

type AgentRunStatus string

const (
	AgentRunStatusQueued      AgentRunStatus = "queued"
	AgentRunStatusRunning     AgentRunStatus = "running"
	AgentRunStatusCompleted   AgentRunStatus = "completed"
	AgentRunStatusFailed      AgentRunStatus = "failed"
	AgentRunStatusStopping    AgentRunStatus = "stopping"
	AgentRunStatusStopped     AgentRunStatus = "stopped"
	AgentRunStatusInterrupted AgentRunStatus = "interrupted"
)

type ToolKind string

const (
	ToolKindLocalTool ToolKind = "local_tool"
	ToolKindMCPServer ToolKind = "mcp_server"
)

type EventLevel string

const (
	EventLevelDebug EventLevel = "debug"
	EventLevelInfo  EventLevel = "info"
	EventLevelWarn  EventLevel = "warn"
	EventLevelError EventLevel = "error"
)

const (
	RunActionLLMPromptSend       = "llm.prompt.send"
	RunActionLLMResponseReceive  = "llm.response.receive"
	RunActionToolExecuteStart    = "tool.execute.start"
	RunActionToolExecuteComplete = "tool.execute.complete"
	RunActionToolExecuteFail     = "tool.execute.fail"
	RunActionResultPersisted     = "run.result.persisted"
	RunActionComplete            = "run.complete"
	RunActionFail                = "run.fail"
)

var agentNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

type AgentDefinition struct {
	Name        string
	Enabled     bool
	Schedule    Schedule
	Vendor      Vendor
	Inputs      []InputDefinition
	Tools       []ToolPermission
	MCPServers  []ToolPermission
	Access      AccessPolicy
	Prompt      string
	SourcePath  string
	RawMarkdown string
}

func (d AgentDefinition) Validate() error {
	var issues []ValidationIssue
	if !IsValidAgentName(d.Name) {
		issues = append(issues, ValidationIssue{
			Field:   "name",
			Message: "must contain lowercase letters, numbers, hyphen, underscore, or dot",
		})
	}
	if strings.TrimSpace(string(d.Schedule.Type)) == "" {
		issues = append(issues, ValidationIssue{Field: "schedule.type", Message: "is required"})
	} else if d.Schedule.Type != ScheduleTypeCron && d.Schedule.Type != ScheduleTypeManual {
		issues = append(issues, ValidationIssue{Field: "schedule.type", Message: "must be cron or manual"})
	}
	if d.Schedule.Type == ScheduleTypeCron && strings.TrimSpace(d.Schedule.Expression) == "" {
		issues = append(issues, ValidationIssue{
			Field:   "schedule.expression",
			Message: "is required for cron schedules",
		})
	}
	if d.Schedule.Type == ScheduleTypeManual && strings.TrimSpace(d.Schedule.Expression) != "" {
		issues = append(issues, ValidationIssue{
			Field:   "schedule.expression",
			Message: "must be omitted for manual schedules",
		})
	}
	if strings.TrimSpace(d.Vendor.Name) == "" {
		issues = append(issues, ValidationIssue{Field: "vendor.name", Message: "is required"})
	}
	if strings.TrimSpace(d.Vendor.Model) == "" {
		issues = append(issues, ValidationIssue{Field: "vendor.model", Message: "is required"})
	}
	if strings.TrimSpace(d.Prompt) == "" {
		issues = append(issues, ValidationIssue{Field: "prompt", Message: "is required"})
	}
	if len(issues) > 0 {
		return NewValidationError(issues)
	}

	return nil
}

type Schedule struct {
	Type       ScheduleType
	Expression string
}

type Vendor struct {
	Name  string
	Model string
}

type AccessPolicy struct {
	Filesystem FilesystemAccess
	Network    NetworkAccess
}

type FilesystemAccess struct {
	Read  []string
	Write []string
}

type NetworkAccess struct {
	Allow []string
}

type InputDefinition struct {
	Name        string
	Required    bool
	Description string
}

type Agent struct {
	Name               string
	Revision           string
	DefinitionSource   string
	DefinitionMarkdown string
	Prompt             string
	Enabled            bool
	Vendor             Vendor
	Schedule           Schedule
	NextRunAt          *time.Time
	Status             AgentStatus
	LastRunID          string
	LastError          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	AppliedAt          time.Time
}

func (a Agent) CanExecute() error {
	if !a.Enabled || a.Status == AgentStatusDisabled {
		return ErrAgentDisabled
	}
	if a.Status != AgentStatusActive {
		return fmt.Errorf("%w: agent status %q", ErrInvalidState, a.Status)
	}

	return nil
}

type AgentRun struct {
	ID                string
	AgentName         string
	AgentRevision     string
	Trigger           RunTrigger
	Status            AgentRunStatus
	StartedAt         *time.Time
	CompletedAt       *time.Time
	DueAt             *time.Time
	WorkDir           string
	LogPath           string
	ProviderRequestID string
	Result            string
	ResultSummary     string
	ErrorCode         string
	ErrorMessage      string
	StopRequestedAt   *time.Time
}

type ToolExecution struct {
	ID             string
	RunID          string
	AgentName      string
	ToolName       string
	CommandSummary string
	StartedAt      time.Time
	CompletedAt    *time.Time
	ExitCode       int
	TimedOut       bool
	StdoutSummary  string
	StderrSummary  string
	ErrorMessage   string
}

func (r AgentRun) IsActive() bool {
	switch r.Status {
	case AgentRunStatusQueued, AgentRunStatusRunning, AgentRunStatusStopping:
		return true
	default:
		return false
	}
}

func (r AgentRun) IsTerminal() bool {
	switch r.Status {
	case AgentRunStatusCompleted,
		AgentRunStatusFailed,
		AgentRunStatusStopped,
		AgentRunStatusInterrupted:
		return true
	default:
		return false
	}
}

func (r AgentRun) CanTransitionTo(next AgentRunStatus) bool {
	switch r.Status {
	case AgentRunStatusQueued:
		return next == AgentRunStatusRunning || next == AgentRunStatusInterrupted
	case AgentRunStatusRunning:
		return next == AgentRunStatusCompleted ||
			next == AgentRunStatusFailed ||
			next == AgentRunStatusStopping ||
			next == AgentRunStatusInterrupted
	case AgentRunStatusStopping:
		return next == AgentRunStatusStopped ||
			next == AgentRunStatusFailed ||
			next == AgentRunStatusCompleted ||
			next == AgentRunStatusInterrupted
	default:
		return false
	}
}

type ToolPermission struct {
	AgentName    string
	Kind         ToolKind
	Name         string
	Command      string
	Args         []string
	Env          []string
	Timeout      string
	ReadPaths    []string
	WritePaths   []string
	NetworkAllow []string
}

type RuntimeEvent struct {
	ID             string
	AgentName      string
	RunID          string
	EventType      string
	Level          EventLevel
	Message        string
	AttributesJSON string
	CreatedAt      time.Time
}

func IsValidAgentName(name string) bool {
	return agentNamePattern.MatchString(name)
}
