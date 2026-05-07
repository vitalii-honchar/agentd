package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"agentd/internal/agentd/config"

	"github.com/spf13/cobra"
)

type RootOptions struct {
	Config        *config.Config
	Client        ApplyClient
	ExecuteClient ExecuteClient
	StopClient    StopClient
	QueryClient   QueryClient
	Out           io.Writer
	Err           io.Writer
}

type Output struct {
	format string
	writer io.Writer
}

func NewRootCommand(opts RootOptions) *cobra.Command {
	cfg := opts.Config
	if cfg == nil {
		cfg = &config.Config{
			ServerURL:      config.DefaultServerURL,
			OutputFormat:   config.DefaultOutputFormat,
			RequestTimeout: config.DefaultRequestTimeout,
		}
	}
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Err
	if errOut == nil {
		errOut = os.Stderr
	}

	cmd := &cobra.Command{
		Use:           "agentd",
		Short:         "Control the local agentd daemon",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return cfg.Validate()
		},
	}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.PersistentFlags().StringVar(&cfg.ServerURL, "server", cfg.ServerURL, "agentdserver URL")
	cmd.PersistentFlags().StringVar(&cfg.OutputFormat, "output", cfg.OutputFormat, "output format: text or json")
	if opts.Client != nil {
		cmd.AddCommand(NewApplyCommand(opts.Client, NewOutput(cfg.OutputFormat, out)))
	}
	if opts.ExecuteClient != nil {
		cmd.AddCommand(NewExecuteCommand(opts.ExecuteClient, NewOutput(cfg.OutputFormat, out)))
	}
	if opts.StopClient != nil {
		cmd.AddCommand(NewStopCommand(opts.StopClient, NewOutput(cfg.OutputFormat, out)))
	}
	if opts.QueryClient != nil {
		queryOutput := NewOutput(cfg.OutputFormat, out)
		cmd.AddCommand(NewListCommand(opts.QueryClient, queryOutput))
		cmd.AddCommand(NewInspectCommand(opts.QueryClient, queryOutput))
		cmd.AddCommand(NewLogsCommand(opts.QueryClient, queryOutput))
	}

	return cmd
}

func NewOutput(format string, writer io.Writer) Output {
	if writer == nil {
		writer = os.Stdout
	}

	return Output{format: format, writer: writer}
}

func (o Output) Write(value any) error {
	if o.format == config.OutputJSON {
		encoder := json.NewEncoder(o.writer)
		encoder.SetIndent("", "  ")

		return encoder.Encode(value)
	}
	if text, ok := value.(string); ok {
		_, err := fmt.Fprintln(o.writer, text)

		return err
	}
	_, err := fmt.Fprintln(o.writer, value)

	return err
}
