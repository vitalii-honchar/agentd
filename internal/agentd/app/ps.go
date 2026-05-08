package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewPSCommand(_ QueryClient, _ Output) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List Agent Runs",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("ps command is not implemented")
		},
	}
	cmd.Flags().BoolP("all", "a", false, "show all runs, including finished runs")

	return cmd
}
