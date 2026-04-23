package eot

import "fmt"

// Thought is the atomic unit exchanged between agents in a round.
type Thought struct {
	AgentID  string
	Round    int
	Content  string // full reasoning text (CoT)
	Answer   string // optional, extracted from "#### <answer>"
	Metadata map[string]any
}

// Render formats a thought for inclusion in another agent's prompt.
func (t *Thought) Render(withHeader bool) string {
	if withHeader {
		return fmt.Sprintf("[%s @ round %d]\n%s", t.AgentID, t.Round, t.Content)
	}
	return t.Content
}
