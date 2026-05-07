package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type ApplyClient interface {
	Apply(context.Context, ApplyRequest) (ApplyResponse, error)
}

type ApplyRequest struct {
	SourcePath string
	Markdown   string
}

type ApplyResponse struct {
	Outcome string      `json:"outcome"`
	Agent   AgentDetail `json:"agent"`
}

type AgentDetail struct {
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	Status       string `json:"status"`
	ScheduleType string `json:"schedule_type"`
	Revision     string `json:"revision"`
	VendorName   string `json:"vendor_name"`
	VendorModel  string `json:"vendor_model"`
}

func NewApplyCommand(client ApplyClient, output Output) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply <path_to_file>",
		Short: "Apply an Agent Definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("apply client is required")
			}

			path := args[0]
			body, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read agent definition %s: %w", path, err)
			}
			response, err := client.Apply(cmd.Context(), ApplyRequest{
				SourcePath: path,
				Markdown:   string(body),
			})
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(response)
			}

			return output.Write(fmt.Sprintf("%s %s", response.Outcome, response.Agent.Name))
		},
	}

	return cmd
}
