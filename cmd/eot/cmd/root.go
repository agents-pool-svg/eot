package cmd

import (
	"github.com/spf13/cobra"

	"github.com/agents-pool-svg/eot/pkg/eot"
)

// NewRootCmd builds the root `eot` command with all sub-commands registered.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "eot",
		Short: "Exchange-of-Thought — multi-agent LLM reasoning framework",
		Long: `eot is a CLI for the Exchange-of-Thought (EoT) framework.

EoT lets multiple LLM agents exchange their intermediate reasoning
(thoughts) according to a configurable communication topology
(memory / report / relay / debate), in the style of the EMNLP'23 paper.

Configuration resolution (high to low priority):
  1. CLI flags (--api-base / --api-key / --model)
  2. Environment variables: EOT_API_BASE / EOT_API_KEY / EOT_MODEL
     (legacy aliases also accepted: CODEMATRIX_LLM_API_*, OPENAI_*)
  3. .env / ReadMe.md / README.md at the working directory
`,
		SilenceUsage: true,
		Version:      eot.Version,
	}

	root.AddCommand(newRunCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newVersionCmd())
	return root
}
