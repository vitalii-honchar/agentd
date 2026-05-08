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

type RunSummary struct {
	RunID       string     `json:"run_id"`
	AgentName   string     `json:"agent_name"`
	Status      string     `json:"status"`
	Trigger     string     `json:"trigger"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
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
