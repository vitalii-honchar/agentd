package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

type StopClient interface {
	Stop(context.Context, StopRequest) (RunResponse, error)
}

type StopRequest struct {
	AgentName string
	RunID     string
}

func NewStopCommand(client StopClient, output Output) *cobra.Command {
	var runID string
	cmd := &cobra.Command{
		Use:   "stop <agent_name>",
		Short: "Stop a running Agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("stop client is required")
			}
			response, err := client.Stop(cmd.Context(), StopRequest{
				AgentName: args[0],
				RunID:     runID,
			})
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(response)
			}

			return output.Write(fmt.Sprintf("%s %s %s", response.Status, response.AgentName, response.RunID))
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "run ID to stop")

	return cmd
}
