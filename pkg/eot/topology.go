package eot

import "fmt"

// Topology decides who sees whose Thoughts per round.
type Topology interface {
	Name() string
	// TurnOrder returns the order in which agents speak within a round.
	TurnOrder(agentIDs []string, round int) []string
	// VisibleThoughts returns the peer thoughts a given agent can read
	// BEFORE it produces its own thought in this round.
	VisibleThoughts(agentIDs []string, all []*Thought, current string, round int) []*Thought
}

// -----------------------------------------------------------------------------
// Memory: shared blackboard — everyone sees everything prior.
// -----------------------------------------------------------------------------

type MemoryTopology struct{}

func (MemoryTopology) Name() string                                 { return "memory" }
func (MemoryTopology) TurnOrder(ids []string, _ int) []string       { return ids }
func (MemoryTopology) VisibleThoughts(_ []string, all []*Thought, current string, round int) []*Thought {
	out := make([]*Thought, 0, len(all))
	for _, t := range all {
		if t.Round < round || (t.Round == round && t.AgentID != current) {
			out = append(out, t)
		}
	}
	return out
}

// -----------------------------------------------------------------------------
// Report: peripheral agents -> central; central aggregates.
// -----------------------------------------------------------------------------

type ReportTopology struct{ Central string }

func (ReportTopology) Name() string { return "report" }

func (r ReportTopology) TurnOrder(ids []string, _ int) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != r.Central {
			out = append(out, id)
		}
	}
	out = append(out, r.Central)
	return out
}

func (r ReportTopology) VisibleThoughts(_ []string, all []*Thought, current string, round int) []*Thought {
	prev := round - 1
	out := make([]*Thought, 0)
	if current == r.Central {
		for _, t := range all {
			if (t.Round == round && t.AgentID != r.Central) || t.Round == prev {
				out = append(out, t)
			}
		}
		return out
	}
	for _, t := range all {
		if t.AgentID == r.Central && t.Round == prev {
			out = append(out, t)
		}
	}
	return out
}

// -----------------------------------------------------------------------------
// Relay: chain — agent i sees only agent i-1.
// -----------------------------------------------------------------------------

type RelayTopology struct{}

func (RelayTopology) Name() string                           { return "relay" }
func (RelayTopology) TurnOrder(ids []string, _ int) []string { return ids }

func (RelayTopology) VisibleThoughts(ids []string, all []*Thought, current string, round int) []*Thought {
	idx := indexOf(ids, current)
	if idx < 0 {
		return nil
	}
	var prevAgent string
	if idx == 0 {
		if round == 0 {
			return nil
		}
		prevAgent = ids[len(ids)-1]
		// wrap-around: last agent's previous-round thought
		for i := len(all) - 1; i >= 0; i-- {
			t := all[i]
			if t.AgentID == prevAgent && t.Round == round-1 {
				return []*Thought{t}
			}
		}
		return nil
	}
	prevAgent = ids[idx-1]
	// take most recent thought from the predecessor
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].AgentID == prevAgent {
			return []*Thought{all[i]}
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Debate: each agent sees all peers' previous-round thoughts.
// -----------------------------------------------------------------------------

type DebateTopology struct{}

func (DebateTopology) Name() string                           { return "debate" }
func (DebateTopology) TurnOrder(ids []string, _ int) []string { return ids }

func (DebateTopology) VisibleThoughts(_ []string, all []*Thought, current string, round int) []*Thought {
	prev := round - 1
	if prev < 0 {
		return nil
	}
	out := make([]*Thought, 0)
	for _, t := range all {
		if t.Round == prev && t.AgentID != current {
			out = append(out, t)
		}
	}
	return out
}

// -----------------------------------------------------------------------------
// Factory
// -----------------------------------------------------------------------------

// TopologySpec is used by Run() and CLI to describe a topology by string.
type TopologySpec struct {
	Name    string `json:"name"              yaml:"name"`
	Central string `json:"central,omitempty" yaml:"central,omitempty"` // for "report"
}

// BuildTopology returns the concrete Topology for a spec.
func BuildTopology(spec TopologySpec) (Topology, error) {
	switch spec.Name {
	case "memory", "":
		return MemoryTopology{}, nil
	case "debate":
		return DebateTopology{}, nil
	case "relay":
		return RelayTopology{}, nil
	case "report":
		if spec.Central == "" {
			return nil, fmt.Errorf("report topology requires 'central'")
		}
		return ReportTopology{Central: spec.Central}, nil
	default:
		return nil, fmt.Errorf("unknown topology %q (available: %v)", spec.Name, AvailableTopologies())
	}
}

// AvailableTopologies lists registered topology names.
func AvailableTopologies() []string {
	return []string{"memory", "report", "relay", "debate"}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
