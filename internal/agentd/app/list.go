package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

type QueryClient interface {
	List(context.Context) (ListResponse, error)
	Inspect(context.Context, string) (AgentDetail, error)
	ListRevisions(context.Context, string) (RevisionListResponse, error)
	InspectRevision(context.Context, string, string) (RevisionInspectResponse, error)
	ListRuns(context.Context, bool) (RunListResponse, error)
	ResultsByAgent(context.Context, string) (AgentResultsResponse, error)
	ResultByRunID(context.Context, string) (RunResult, error)
	Logs(context.Context, LogsRequest) (LogsResponse, error)
}

type AgentSummary struct {
	Name          string `json:"name"`
	Enabled       bool   `json:"enabled"`
	Status        string `json:"status"`
	ScheduleType  string `json:"schedule_type"`
	LastRunStatus string `json:"last_run_status,omitempty"`
}

type ListResponse struct {
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
	Trigger       string     `json:"trigger"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type RunListResponse struct {
	Runs []RunSummary `json:"runs"`
}

type AgentResultsResponse struct {
	AgentName string      `json:"agent_name"`
	Results   []RunResult `json:"results"`
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

func NewListCommand(client QueryClient, output Output) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List applied Agents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if client == nil {
				return fmt.Errorf("query client is required")
			}
			response, err := client.List(cmd.Context())
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(response)
			}
			for _, agent := range response.Agents {
				if _, err := fmt.Fprintf(
					output.writer,
					"%s\t%s\t%s\t%t\n",
					agent.Name,
					agent.Status,
					agent.ScheduleType,
					agent.Enabled,
				); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func NewRevisionsCommand(client QueryClient, output Output) *cobra.Command {
	return &cobra.Command{
		Use:   "revisions <agent_name>",
		Short: "List immutable Agent revisions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("query client is required")
			}
			response, err := client.ListRevisions(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(response)
			}
			for _, revision := range response.Revisions {
				created := ""
				if !revision.CreatedAt.IsZero() {
					created = revision.CreatedAt.Format(time.RFC3339)
				}
				if _, err := fmt.Fprintf(
					output.writer,
					"%s\t%s\t%s\t%t\t%s\t%s\n",
					revision.RevisionID,
					revision.Status,
					created,
					revision.Latest,
					revision.SourcePath,
					revision.ArtifactPath,
				); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
