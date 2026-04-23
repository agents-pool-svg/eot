package eot

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Result is the outcome of an EoT run.
type Result struct {
	FinalAnswer string
	Rounds      int
	Converged   bool
	Thoughts    []*Thought
}

// Transcript concatenates every thought for logging/debugging.
func (r *Result) Transcript() string {
	var b strings.Builder
	for i, t := range r.Thoughts {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(t.Render(true))
	}
	return b.String()
}

// Runner orchestrates multi-round thought exchange among agents.
type Runner struct {
	Agents               []*Agent
	Topology             Topology
	MaxRounds            int
	ConvergenceThreshold float64 // 1.0 = unanimous
	Verbose              bool
	OnThought            func(*Thought) // optional callback for streaming UIs
}

// Run executes the EoT loop until convergence or MaxRounds.
func (r *Runner) Run(ctx context.Context, question string) (*Result, error) {
	if len(r.Agents) < 2 {
		return nil, fmt.Errorf("EoT requires at least 2 agents, got %d", len(r.Agents))
	}
	if r.MaxRounds <= 0 {
		r.MaxRounds = 3
	}
	if r.ConvergenceThreshold == 0 {
		r.ConvergenceThreshold = 1.0
	}

	ids := make([]string, 0, len(r.Agents))
	byID := map[string]*Agent{}
	for _, a := range r.Agents {
		ids = append(ids, a.ID)
		byID[a.ID] = a
	}

	var all []*Thought
	for round := 0; round < r.MaxRounds; round++ {
		order := r.Topology.TurnOrder(ids, round)
		for _, id := range order {
			ag := byID[id]
			peers := r.Topology.VisibleThoughts(ids, all, id, round)
			th, err := ag.Think(ctx, question, peers, round)
			if err != nil {
				return nil, err
			}
			all = append(all, th)
			if r.Verbose {
				fmt.Printf("\n========== round %d · %s (topology=%s) ==========\n",
					round, id, r.Topology.Name())
				fmt.Println(th.Content)
				if th.Answer != "" {
					fmt.Printf("[answer] %s\n", th.Answer)
				}
			}
			if r.OnThought != nil {
				r.OnThought(th)
			}
		}

		roundAnswers := collectAnswers(all, round)
		if len(roundAnswers) > 0 && converged(roundAnswers, r.ConvergenceThreshold) {
			return &Result{
				FinalAnswer: majority(roundAnswers),
				Rounds:      round + 1,
				Converged:   true,
				Thoughts:    all,
			}, nil
		}
	}

	finals := collectAnswers(all, r.MaxRounds-1)
	return &Result{
		FinalAnswer: majority(finals),
		Rounds:      r.MaxRounds,
		Converged:   false,
		Thoughts:    all,
	}, nil
}

func collectAnswers(all []*Thought, round int) []string {
	out := make([]string, 0)
	for _, t := range all {
		if t.Round == round && t.Answer != "" {
			out = append(out, t.Answer)
		}
	}
	return out
}

func converged(answers []string, threshold float64) bool {
	if len(answers) == 0 {
		return false
	}
	counts := map[string]int{}
	for _, a := range answers {
		counts[strings.ToLower(strings.TrimSpace(a))]++
	}
	best := 0
	for _, c := range counts {
		if c > best {
			best = c
		}
	}
	return float64(best)/float64(len(answers)) >= threshold
}

func majority(answers []string) string {
	if len(answers) == 0 {
		return ""
	}
	counts := map[string]int{}
	for _, a := range answers {
		counts[strings.TrimSpace(a)]++
	}
	// deterministic tie-break: alphabetical
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best, bestN := "", -1
	for _, k := range keys {
		if counts[k] > bestN {
			best, bestN = k, counts[k]
		}
	}
	return best
}
