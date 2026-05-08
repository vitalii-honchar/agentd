package app

import (
	"fmt"
	"regexp"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"

	"github.com/spf13/cobra"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func NewResultCommand(client QueryClient, output Output) *cobra.Command {
	return &cobra.Command{
		Use:   "result <agent-name|run-id>",
		Short: "Read Agent Run results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("query client is required")
			}
			target := args[0]
			if uuidPattern.MatchString(target) {
				result, err := client.ResultByRunID(cmd.Context(), target)
				if err != nil {
					return err
				}
				if output.format == config.OutputJSON {
					return output.Write(result)
				}

				return output.Write(result.Result)
			}

			response, err := client.ResultsByAgent(cmd.Context(), target)
			if err != nil {
				return err
			}
			if output.format == config.OutputJSON {
				return output.Write(response)
			}
			rows := make([][]string, 0, len(response.Results))
			for _, result := range response.Results {
				rows = append(rows, []string{
					TrimTableCell(result.RunID, 36),
					result.Status,
					formatOptionalTime(result.CompletedAt),
					TrimTableCell(result.ResultSummary, DefaultTableCellLimit),
				})
			}

			return output.WriteTable([]string{"RUN ID", "STATUS", "COMPLETED", "RESULT"}, rows)
		},
	}
}
