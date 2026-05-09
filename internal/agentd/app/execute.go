package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type ExecuteClient interface {
	Execute(context.Context, string, map[string]string) (RunResponse, error)
}

type RunResponse struct {
	RunID         string `json:"run_id"`
	AgentName     string `json:"agent_name"`
	AgentRevision string `json:"agent_revision,omitempty"`
	Status        string `json:"status"`
}

func NewExecuteCommand(client ExecuteClient, output Output) *cobra.Command {
	return newExecuteCommand("execute <agent_name>", "Execute an Agent immediately", client, output)
}

func NewRunCommand(client ExecuteClient, output Output) *cobra.Command {
	return newExecuteCommand("run <agent_name[:revision]>", "Run an Agent revision immediately", client, output)
}

func newExecuteCommand(use string, short string, client ExecuteClient, output Output) *cobra.Command {
	var inputPairs []string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("execute client is required")
			}
			inputs, err := parseInputPairs(inputPairs)
			if err != nil {
				return err
			}
			response, err := client.Execute(cmd.Context(), args[0], inputs)
			if err != nil {
				return err
			}
			if output.format == "json" {
				return output.Write(response)
			}

			return output.Write(fmt.Sprintf("%s %s %s", response.Status, response.AgentName, response.RunID))
		},
	}
	cmd.Flags().StringArrayVar(&inputPairs, "input", nil, "Run input as key=value")

	return cmd
}

func parseInputPairs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	inputs := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("input must be key=value: %s", pair)
		}
		inputs[key] = value
	}

	return inputs, nil
}
