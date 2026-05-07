package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

type ExecuteClient interface {
	Execute(context.Context, string) (RunResponse, error)
}

type RunResponse struct {
	RunID     string `json:"run_id"`
	AgentName string `json:"agent_name"`
	Status    string `json:"status"`
}

func NewExecuteCommand(client ExecuteClient, output Output) *cobra.Command {
	return &cobra.Command{
		Use:   "execute <agent_name>",
		Short: "Execute an Agent immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("execute client is required")
			}
			response, err := client.Execute(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(response)
			}

			return output.Write(fmt.Sprintf("%s %s %s", response.Status, response.AgentName, response.RunID))
		},
	}
}
