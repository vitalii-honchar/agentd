package app

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

type LogsRequest struct {
	AgentName string
	RunID     string
	Tail      int
}

type LogsResponse struct {
	AgentName string     `json:"agent_name"`
	RunID     string     `json:"run_id,omitempty"`
	Entries   []LogEntry `json:"entries"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	RunID     string    `json:"run_id,omitempty"`
	Action    string    `json:"action,omitempty"`
	Message   string    `json:"message,omitempty"`
	Line      string    `json:"line"`
}

func NewLogsCommand(client QueryClient, output Output) *cobra.Command {
	var tail int
	cmd := &cobra.Command{
		Use:   "logs <run_id>",
		Short: "Read logs for a single Agent run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("query client is required")
			}
			runID := args[0]
			if !uuidPattern.MatchString(runID) {
				return fmt.Errorf("logs require an agent run ID")
			}
			response, err := client.Logs(cmd.Context(), LogsRequest{
				RunID: runID,
				Tail:  tail,
			})
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(response)
			}
			for _, entry := range response.Entries {
				line := entry.Line
				if entry.Action != "" {
					if entry.RunID != "" {
						line = fmt.Sprintf("%s %s %s %s", entry.Timestamp.Format(time.RFC3339), entry.RunID, entry.Action, entry.Message)
					} else {
						line = fmt.Sprintf("%s %s %s", entry.Timestamp.Format(time.RFC3339), entry.Action, entry.Message)
					}
				}
				if _, err := fmt.Fprintln(output.writer, line); err != nil {
					return err
				}
			}

			return nil
		},
	}
	cmd.Flags().IntVar(&tail, "tail", 0, "number of log lines to read")

	return cmd
}
