package eot

import "context"

// RunRequest is the declarative form of an EoT run — everything a SDK user
// or the CLI needs to describe a complete experiment.
type RunRequest struct {
	// Question is the problem all agents reason about.
	Question string

	// Agents defines the participating agents. At least 2 required.
	Agents []AgentSpec

	// Topology selects the communication pattern.
	Topology TopologySpec

	// MaxRounds caps the number of exchange rounds (default 3).
	MaxRounds int

	// ConvergenceThreshold: fraction of agents that must agree to stop early.
	// 1.0 = unanimous (default), 0.5 = majority.
	ConvergenceThreshold float64

	// Verbose prints each thought as it is produced.
	Verbose bool

	// Optional: pre-built LLM config; otherwise resolved via LoadConfig + ConfigOpts.
	LLMConfig  *LLMConfig
	ConfigOpts []ConfigOption

	// Optional streaming callback.
	OnThought func(*Thought)
}

// Run is the one-shot convenience entry point suitable for library integration.
//
// Minimal usage:
//
//	res, err := eot.Run(ctx, eot.RunRequest{
//	    Question: "2+2*3=?",
//	    Agents: []eot.AgentSpec{
//	        {ID: "A", System: "You are agent A."},
//	        {ID: "B", System: "You are agent B."},
//	    },
//	    Topology: eot.TopologySpec{Name: "debate"},
//	    ConfigOpts: []eot.ConfigOption{
//	        eot.WithAPIBase("https://api.openai.com"),
//	        eot.WithAPIKey("sk-..."),
//	        eot.WithModel("gpt-4o-mini"),
//	    },
//	})
func Run(ctx context.Context, req RunRequest) (*Result, error) {
	cfg := req.LLMConfig
	if cfg == nil {
		c, err := LoadConfig(req.ConfigOpts...)
		if err != nil {
			return nil, err
		}
		cfg = c
	}
	llm, err := NewLLMClient(cfg)
	if err != nil {
		return nil, err
	}

	topo, err := BuildTopology(req.Topology)
	if err != nil {
		return nil, err
	}

	agents := make([]*Agent, 0, len(req.Agents))
	for _, spec := range req.Agents {
		agents = append(agents, NewAgent(llm, spec))
	}

	runner := &Runner{
		Agents:               agents,
		Topology:             topo,
		MaxRounds:            req.MaxRounds,
		ConvergenceThreshold: req.ConvergenceThreshold,
		Verbose:              req.Verbose,
		OnThought:            req.OnThought,
	}
	return runner.Run(ctx, req.Question)
}
