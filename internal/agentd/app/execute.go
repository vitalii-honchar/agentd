package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type ExecuteClient interface {
	Execute(context.Context, string, map[string]string) (RunResponse, error)
}

type StructuredExecuteClient interface {
	ExecuteWithInput(context.Context, string, RunInput) (RunResponse, error)
}

type RunInput struct {
	Input        json.RawMessage
	LegacyInputs map[string]string
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
	var inputJSON string
	var inputFile string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if client == nil {
				return fmt.Errorf("execute client is required")
			}
			runInput, hasStructuredInput, err := parseRunInput(inputPairs, inputJSON, inputFile)
			if err != nil {
				return err
			}
			if hasStructuredInput {
				structured, ok := client.(StructuredExecuteClient)
				if !ok {
					return fmt.Errorf("execute client does not support structured input")
				}
				response, err := structured.ExecuteWithInput(cmd.Context(), args[0], runInput)
				if err != nil {
					return err
				}
				if output.format == "json" {
					return output.Write(response)
				}

				return output.Write(fmt.Sprintf("%s %s %s", response.Status, response.AgentName, response.RunID))
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
	cmd.Flags().StringVar(&inputJSON, "input-json", "", "Run input as a JSON object")
	cmd.Flags().StringVar(&inputFile, "input-file", "", "Read run input JSON object from a file")

	return cmd
}

func parseRunInput(inputPairs []string, inputJSON string, inputFile string) (RunInput, bool, error) {
	if strings.TrimSpace(inputJSON) != "" && strings.TrimSpace(inputFile) != "" {
		return RunInput{}, false, fmt.Errorf("input-json and input-file cannot be used together")
	}
	if strings.TrimSpace(inputJSON) != "" {
		raw := json.RawMessage(inputJSON)
		if !json.Valid(raw) {
			return RunInput{}, false, fmt.Errorf("input-json must be valid JSON")
		}

		return RunInput{Input: raw}, true, nil
	}
	if strings.TrimSpace(inputFile) != "" {
		body, err := os.ReadFile(inputFile)
		if err != nil {
			return RunInput{}, false, fmt.Errorf("read input file: %w", err)
		}
		raw := json.RawMessage(body)
		if !json.Valid(raw) {
			return RunInput{}, false, fmt.Errorf("input-file must contain valid JSON")
		}

		return RunInput{Input: raw}, true, nil
	}
	inputs, err := parseInputPairs(inputPairs)
	if err != nil {
		return RunInput{}, false, err
	}
	if len(inputs) > 0 {
		return RunInput{LegacyInputs: inputs}, false, nil
	}

	return RunInput{}, false, nil
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
