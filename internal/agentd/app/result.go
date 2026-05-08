package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewResultCommand(_ QueryClient, _ Output) *cobra.Command {
	return &cobra.Command{
		Use:   "result <agent-name|run-id>",
		Short: "Read Agent Run results",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("result command is not implemented")
		},
	}
}
