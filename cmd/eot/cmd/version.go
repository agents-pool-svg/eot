package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agents-pool-svg/eot/pkg/eot"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print eot version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "eot %s\n", eot.Version)
			return nil
		},
	}
}
