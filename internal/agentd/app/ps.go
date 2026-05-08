package app

import (
	"fmt"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"

	"github.com/spf13/cobra"
)

func NewPSCommand(client QueryClient, output Output) *cobra.Command {
	var includeAll bool
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List Agent Runs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if client == nil {
				return fmt.Errorf("query client is required")
			}
			response, err := client.ListRuns(cmd.Context(), includeAll)
			if err != nil {
				return err
			}
			if output.format == config.OutputJSON {
				return output.Write(response)
			}

			rows := make([][]string, 0, len(response.Runs))
			for _, run := range response.Runs {
				rows = append(rows, []string{
					TrimTableCell(run.RunID, 36),
					TrimTableCell(run.AgentName, 32),
					run.Status,
					run.Trigger,
					formatOptionalTime(run.StartedAt),
					formatOptionalTime(run.CompletedAt),
				})
			}

			return output.WriteTable(
				[]string{"RUN ID", "AGENT", "STATUS", "TRIGGER", "STARTED", "COMPLETED"},
				rows,
			)
		},
	}
	cmd.Flags().BoolVarP(&includeAll, "all", "a", false, "show all runs, including finished runs")

	return cmd
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return "-"
	}

	return value.UTC().Format(time.RFC3339)
}
