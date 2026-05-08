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
	Line      string    `json:"line"`
}

func NewLogsCommand(client QueryClient, output Output) *cobra.Command {
	var runID string
	var tail int
	cmd := &cobra.Command{
		Use:   "logs <agent_name>",
		Short: "Read isolated Agent logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("query client is required")
			}
			response, err := client.Logs(cmd.Context(), LogsRequest{
				AgentName: args[0],
				RunID:     runID,
				Tail:      tail,
			})
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(response)
			}
			for _, entry := range response.Entries {
				if _, err := fmt.Fprintln(output.writer, entry.Line); err != nil {
					return err
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "run ID to read logs for")
	cmd.Flags().IntVar(&tail, "tail", 0, "number of log lines to read")

	return cmd
}
