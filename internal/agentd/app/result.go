package app

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
	"github.com/vitalii-honchar/agentd/pkg/agentdclient"

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
					return mapResultError(err)
				}
				if output.format == config.OutputJSON {
					if err := output.Write(result); err != nil {
						return err
					}
				} else if err := output.Write(result.Result); err != nil {
					return err
				}
				if result.Status == "failed" {
					return ExitError{Code: 5, Err: fmt.Errorf("agent run failed")}
				}

				return nil
			}

			response, err := client.ResultsByAgent(cmd.Context(), target)
			if err != nil {
				return mapResultError(err)
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

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}

	return e.Err.Error()
}

func (e ExitError) Unwrap() error {
	return e.Err
}

func ExitCode(err error) int {
	var exitErr ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}

	return 1
}

func mapResultError(err error) error {
	var daemonErr *agentdclient.Error
	if !errors.As(err, &daemonErr) {
		return err
	}
	switch daemonErr.Code {
	case agentdclient.ErrorCodeAgentNotFound:
		return ExitError{Code: 2, Err: err}
	case agentdclient.ErrorCodeRunNotFound:
		return ExitError{Code: 3, Err: err}
	case agentdclient.ErrorCodeRunNotTerminal:
		return ExitError{Code: 4, Err: err}
	case agentdclient.ErrorCodeDaemonUnavailable:
		return ExitError{Code: 10, Err: err}
	default:
		return err
	}
}
