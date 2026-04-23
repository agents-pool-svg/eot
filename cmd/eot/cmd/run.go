package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agents-pool-svg/eot/pkg/eot"
)

type runFlags struct {
	question    string
	questionFile string

	topology string
	central  string

	agentsFile   string   // path to JSON file with []AgentSpec
	agents       []string // inline: "ID:system prompt"
	defaultModel string

	apiBase string
	apiKey  string
	model   string

	maxRounds int
	threshold float64
	verbose   bool
	output    string // "text" | "json"
}

func newRunCmd() *cobra.Command {
	var f runFlags
	c := &cobra.Command{
		Use:   "run",
		Short: "Run an Exchange-of-Thought session",
		Long: `Run an EoT session with the configured agents and topology.

Agents can be supplied in two ways (choose one):
  --agents-file <path>   A JSON file: [{"id":"A","system":"...","model":"...","temperature":0.2}, ...]
  --agent ID:SYSTEM      Repeatable; inline shortcut (uses default model).

Examples:

  # Minimal: debate between two agents
  eot run --question "2+2*3=?" \
    --topo debate \
    --agent "Planner:You are a careful planner." \
    --agent "Checker:You are a rigorous checker."

  # Report topology with explicit central aggregator and JSON agent spec
  eot run --question-file problem.txt \
    --topo report --central Reviewer \
    --agents-file agents.json \
    --api-base https://api.openai.com --api-key $OPENAI_API_KEY
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEoT(cmd, &f)
		},
	}

	c.Flags().StringVarP(&f.question, "question", "q", "", "Question to reason about")
	c.Flags().StringVar(&f.questionFile, "question-file", "", "Read question from file (use '-' for stdin)")

	c.Flags().StringVar(&f.topology, "topo", "memory", "Topology: memory|report|relay|debate")
	c.Flags().StringVar(&f.central, "central", "", "Central agent ID (required for --topo report)")

	c.Flags().StringVar(&f.agentsFile, "agents-file", "", "Path to JSON file defining agents")
	c.Flags().StringArrayVar(&f.agents, "agent", nil, "Inline agent 'ID:system prompt' (repeatable)")
	c.Flags().StringVar(&f.defaultModel, "default-model", "", "Default model for inline agents (overrides env)")

	c.Flags().StringVar(&f.apiBase, "api-base", "", "LLM API base URL (overrides env/.env)")
	c.Flags().StringVar(&f.apiKey, "api-key", "", "LLM API key (overrides env/.env)")
	c.Flags().StringVar(&f.model, "model", "", "Model name (overrides env/.env)")

	c.Flags().IntVar(&f.maxRounds, "rounds", 3, "Maximum rounds")
	c.Flags().Float64Var(&f.threshold, "threshold", 1.0, "Convergence threshold (1.0=unanimous, 0.5=majority)")
	c.Flags().BoolVarP(&f.verbose, "verbose", "v", false, "Print each thought as it streams")
	c.Flags().StringVarP(&f.output, "output", "o", "text", "Output format: text|json")

	return c
}

func runEoT(cmd *cobra.Command, f *runFlags) error {
	// Resolve question
	question, err := resolveQuestion(f)
	if err != nil {
		return err
	}

	// Build agent specs
	specs, err := resolveAgents(f)
	if err != nil {
		return err
	}
	if len(specs) < 2 {
		return fmt.Errorf("EoT requires at least 2 agents; got %d", len(specs))
	}

	// Build request
	req := eot.RunRequest{
		Question: question,
		Agents:   specs,
		Topology: eot.TopologySpec{
			Name:    f.topology,
			Central: f.central,
		},
		MaxRounds:            f.maxRounds,
		ConvergenceThreshold: f.threshold,
		Verbose:              f.verbose,
		ConfigOpts: []eot.ConfigOption{
			eot.WithAPIBase(f.apiBase),
			eot.WithAPIKey(f.apiKey),
			eot.WithModel(f.model),
		},
	}

	res, err := eot.Run(context.Background(), req)
	if err != nil {
		return err
	}

	return printResult(cmd.OutOrStdout(), res, f)
}

func resolveQuestion(f *runFlags) (string, error) {
	if f.question != "" && f.questionFile != "" {
		return "", fmt.Errorf("--question and --question-file are mutually exclusive")
	}
	if f.question != "" {
		return f.question, nil
	}
	if f.questionFile == "" {
		return "", fmt.Errorf("must provide --question or --question-file")
	}
	var data []byte
	var err error
	if f.questionFile == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(f.questionFile)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func resolveAgents(f *runFlags) ([]eot.AgentSpec, error) {
	if f.agentsFile != "" && len(f.agents) > 0 {
		return nil, fmt.Errorf("--agents-file and --agent are mutually exclusive")
	}
	if f.agentsFile != "" {
		data, err := os.ReadFile(f.agentsFile)
		if err != nil {
			return nil, fmt.Errorf("read agents-file: %w", err)
		}
		var specs []eot.AgentSpec
		if err := json.Unmarshal(data, &specs); err != nil {
			return nil, fmt.Errorf("parse agents-file: %w", err)
		}
		return specs, nil
	}
	if len(f.agents) == 0 {
		return nil, fmt.Errorf("must provide --agent (repeatable) or --agents-file")
	}
	specs := make([]eot.AgentSpec, 0, len(f.agents))
	for _, raw := range f.agents {
		idx := strings.Index(raw, ":")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid --agent %q; expected 'ID:system prompt'", raw)
		}
		specs = append(specs, eot.AgentSpec{
			ID:     strings.TrimSpace(raw[:idx]),
			System: strings.TrimSpace(raw[idx+1:]),
			Model:  f.defaultModel,
		})
	}
	return specs, nil
}

type jsonOut struct {
	Topology    string           `json:"topology"`
	Rounds      int              `json:"rounds"`
	Converged   bool             `json:"converged"`
	FinalAnswer string           `json:"final_answer"`
	Thoughts    []jsonOutThought `json:"thoughts"`
}

type jsonOutThought struct {
	Agent   string `json:"agent"`
	Round   int    `json:"round"`
	Content string `json:"content"`
	Answer  string `json:"answer,omitempty"`
}

func printResult(w io.Writer, res *eot.Result, f *runFlags) error {
	if f.output == "json" {
		out := jsonOut{
			Topology:    f.topology,
			Rounds:      res.Rounds,
			Converged:   res.Converged,
			FinalAnswer: res.FinalAnswer,
		}
		for _, t := range res.Thoughts {
			out.Thoughts = append(out.Thoughts, jsonOutThought{
				Agent: t.AgentID, Round: t.Round, Content: t.Content, Answer: t.Answer,
			})
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Fprintln(w, strings.Repeat("=", 60))
	fmt.Fprintf(w, "Topology      : %s\n", f.topology)
	fmt.Fprintf(w, "Rounds used   : %d\n", res.Rounds)
	fmt.Fprintf(w, "Converged     : %v\n", res.Converged)
	fmt.Fprintf(w, "Final answer  : %s\n", res.FinalAnswer)
	fmt.Fprintln(w, strings.Repeat("=", 60))
	return nil
}
