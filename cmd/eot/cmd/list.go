package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agents-pool-svg/eot/pkg/eot"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-topologies",
		Short: "List available communication topologies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, t := range eot.AvailableTopologies() {
				fmt.Fprintln(cmd.OutOrStdout(), "-", t, describeTopology(t))
			}
			return nil
		},
	}
}

func describeTopology(name string) string {
	switch name {
	case "memory":
		return "— shared blackboard; every agent sees every prior thought"
	case "report":
		return "— peripheral agents report to a central aggregator"
	case "relay":
		return "— chain; each agent sees only its predecessor"
	case "debate":
		return "— fully-connected; each agent sees all peers' last round"
	default:
		return strings.Repeat(" ", 0)
	}
}
