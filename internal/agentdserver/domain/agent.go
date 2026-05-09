package domain

import (
	"encoding/json"
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

type AgentRevisionStatus string

const (
	AgentRevisionStatusPending   AgentRevisionStatus = "pending"
	AgentRevisionStatusFinalized AgentRevisionStatus = "finalized"
	AgentRevisionStatusCorrupt   AgentRevisionStatus = "corrupt"
)

type ResultFormat string

const (
	ResultFormatText ResultFormat = "text"
	ResultFormatJSON ResultFormat = "json"
)

type RuntimeInputSource string

const (
	RuntimeInputSourceCLI          RuntimeInputSource = "cli"
	RuntimeInputSourceFile         RuntimeInputSource = "file"
	RuntimeInputSourcePublicClient RuntimeInputSource = "public_client"
	RuntimeInputSourceSchedule     RuntimeInputSource = "schedule"
	RuntimeInputSourceInternalTest RuntimeInputSource = "internal_test"
)

type ReActDecision string

const (
	ReActDecisionToolCall ReActDecision = "tool_call"
	ReActDecisionFinal    ReActDecision = "final"
	ReActDecisionFail     ReActDecision = "fail"
)

type RevisionEnvironmentSource string

const (
	RevisionEnvironmentSourceLiteral RevisionEnvironmentSource = "literal"
	RevisionEnvironmentSourceEnvFile RevisionEnvironmentSource = "env_file"
	RevisionEnvironmentSourceToolEnv RevisionEnvironmentSource = "tool_env"
)

type ToolExecutionEventType string

const (
	ToolExecutionEventStart    ToolExecutionEventType = "tool.execute.start"
	ToolExecutionEventComplete ToolExecutionEventType = "tool.execute.complete"
	ToolExecutionEventFail     ToolExecutionEventType = "tool.execute.fail"
)

type ToolKind string

const (
	ToolKindCustomTool ToolKind = "custom_tool"
	ToolKindHostTool   ToolKind = "host_tool"
	ToolKindLocalTool  ToolKind = "local_tool"
	ToolKindMCPServer  ToolKind = "mcp_server"
)

type EventLevel string

const (
	EventLevelDebug EventLevel = "debug"
	EventLevelInfo  EventLevel = "info"
	EventLevelWarn  EventLevel = "warn"
	EventLevelError EventLevel = "error"
)

const (
	RunActionLLMPromptSend          = "llm.prompt.send"
	RunActionLLMResponseReceive     = "llm.response.receive"
	RunActionProviderFail           = "provider.fail"
	RunActionContractInputValidated = "contract.input.validated"
	RunActionContractInputRejected  = "contract.input.rejected"
	RunActionReActStep              = "react.step"
	RunActionReActToolObservation   = "react.tool.observation"
	RunActionToolExecuteStart       = "tool.execute.start"
	RunActionToolExecuteComplete    = "tool.execute.complete"
	RunActionToolExecuteFail        = "tool.execute.fail"
	RunActionOutputFinalizeStart    = "output.finalize.start"
	RunActionOutputFinalizeDone     = "output.finalize.complete"
	RunActionOutputFinalizeFail     = "output.finalize.fail"
	RunActionResultPersisted        = "run.result.persisted"
	RunActionComplete               = "run.complete"
	RunActionFail                   = "run.fail"
)

var agentNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

type AgentDefinition struct {
	Name        string
	Enabled     bool
	Schedule    Schedule
	Vendor      Vendor
	Environment DefinitionEnvironment
	Inputs      []InputDefinition
	Contract    *AgentContract
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

type ProviderMetadata struct {
	Name                 string
	Model                string
	RequiresDirectAPIKey bool
	ConfigJSON           string
}

type AgentContract struct {
	InputSchemaRaw      string
	OutputSchemaRaw     string
	InputSchemaDigest   string
	OutputSchemaDigest  string
	CreatedFromRevision string
}

type RuntimeInput struct {
	RawJSON      json.RawMessage
	LegacyInputs map[string]string
	Source       RuntimeInputSource
}

type ReActStep struct {
	StepIndex          int
	RunID              string
	AgentName          string
	RevisionID         string
	ModelMessage       string
	Decision           ReActDecision
	ToolName           string
	ToolArgsJSON       string
	ObservationSummary string
	StartedAt          time.Time
	CompletedAt        *time.Time
	ErrorMessage       string
}

type DefinitionEnvironment struct {
	Variables map[string]string
	Files     []string
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
	Contract           *AgentContract
	Schedule           Schedule
	Tools              []ToolPermission
	MCPServers         []ToolPermission
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
	ID                         string
	AgentName                  string
	AgentRevision              string
	Trigger                    RunTrigger
	Status                     AgentRunStatus
	StartedAt                  *time.Time
	CompletedAt                *time.Time
	DueAt                      *time.Time
	WorkDir                    string
	LogPath                    string
	InputJSON                  json.RawMessage
	ContractInputSchemaDigest  string
	ContractOutputSchemaDigest string
	ProviderName               string
	ProviderModel              string
	ProviderRequestID          string
	ResultFormat               ResultFormat
	Result                     string
	ResultSummary              string
	ErrorCode                  string
	ErrorMessage               string
	StopRequestedAt            *time.Time
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

type AgentRevision struct {
	AgentName                  string
	RevisionID                 string
	ContentDigest              string
	SourcePath                 string
	ArtifactPath               string
	EnvironmentJSON            string
	Prompt                     string
	Vendor                     Vendor
	Schedule                   Schedule
	ContractInputSchemaRaw     string
	ContractOutputSchemaRaw    string
	ContractInputSchemaDigest  string
	ContractOutputSchemaDigest string
	ContractDigest             string
	Status                     AgentRevisionStatus
	CreatedAt                  time.Time
	FinalizedAt                *time.Time
	ErrorMessage               string
	Tools                      []RevisionTool
	ArtifactFiles              []RevisionArtifactFile
	Environment                []RevisionEnvironment
	IsLatestFinalized          bool
}

type RevisionTool struct {
	AgentName        string
	RevisionID       string
	Name             string
	Kind             ToolKind
	OriginalCommand  string
	RewrittenCommand string
	HostCommand      string
	Args             []string
	Env              []string
	Timeout          string
	ReadPaths        []string
	WritePaths       []string
	NetworkAllow     []string
	CopiedFiles      []string
	CreatedAt        time.Time
}

type RevisionArtifactFile struct {
	AgentName            string
	RevisionID           string
	ArtifactRelativePath string
	SourcePath           string
	SHA256               string
	Mode                 int64
	SizeBytes            int64
	CopiedAt             time.Time
}

type RevisionEnvironment struct {
	AgentName            string
	RevisionID           string
	Key                  string
	Value                string
	Source               RevisionEnvironmentSource
	SourcePath           string
	ArtifactRelativePath string
	Masked               bool
	CreatedAt            time.Time
}

type ExecutionWorkDir struct {
	AgentName   string
	ExecutionID string
	Path        string
	RevisionID  string
	CreatedAt   time.Time
}

type ToolExecutionLog struct {
	AgentName     string
	RunID         string
	RevisionID    string
	ToolName      string
	ToolKind      ToolKind
	EventType     ToolExecutionEventType
	StdoutSummary string
	StderrSummary string
	ResultSummary string
	ExitCode      int
	TimedOut      bool
	ErrorMessage  string
	CreatedAt     time.Time
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
