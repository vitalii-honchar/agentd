package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewInspectCommand(client QueryClient, output Output) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <agent_name>",
		Short: "Inspect an Agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("query client is required")
			}
			agent, err := client.Inspect(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(agent)
			}

			return output.Write(fmt.Sprintf(
				"name: %s\nstatus: %s\nschedule: %s\nvendor: %s/%s\nrevision: %s\nlast_run: %s",
				agent.Name,
				agent.Status,
				agent.ScheduleType,
				agent.VendorName,
				agent.VendorModel,
				agent.Revision,
				agent.LastRunID,
			))
		},
	}
}
