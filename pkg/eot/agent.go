package eot

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// AgentSpec is a lightweight description of an agent (used by Run() and CLI).
type AgentSpec struct {
	ID          string  `json:"id"           yaml:"id"`
	System      string  `json:"system"       yaml:"system"`
	Model       string  `json:"model,omitempty"       yaml:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
}

// Agent is a single LLM reasoning entity with a persistent history.
type Agent struct {
	ID          string
	System      string
	Model       string
	Temperature float64
	LLM         *LLMClient
	History     []*Thought
}

// NewAgent is a convenience constructor.
func NewAgent(llm *LLMClient, spec AgentSpec) *Agent {
	t := spec.Temperature
	if t == 0 {
		t = 0.7
	}
	return &Agent{
		ID:          spec.ID,
		System:      spec.System,
		Model:       spec.Model,
		Temperature: t,
		LLM:         llm,
	}
}

var answerRE = regexp.MustCompile(`(?m)^####\s*(.+?)\s*$`)

// Think produces a new Thought given the question and peers' visible thoughts.
func (a *Agent) Think(ctx context.Context, question string, peers []*Thought, round int) (*Thought, error) {
	user := buildUserPrompt(question, peers, round)
	msgs := []ChatMessage{
		{Role: "system", Content: a.System},
		{Role: "user", Content: user},
	}
	raw, err := a.LLM.Chat(ctx, msgs, ChatOptions{
		Model:       a.Model,
		Temperature: a.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("agent %s: %w", a.ID, err)
	}
	th := &Thought{
		AgentID: a.ID,
		Round:   round,
		Content: raw,
		Answer:  extractAnswer(raw),
	}
	a.History = append(a.History, th)
	return th, nil
}

func buildUserPrompt(question string, peers []*Thought, round int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Question:\n%s\n\n", question)
	if len(peers) > 0 {
		b.WriteString("Peer reasoning so far (consider, critique & improve):\n")
		for _, p := range peers {
			fmt.Fprintf(&b, "--- %s (round %d) ---\n%s\n\n", p.AgentID, p.Round, p.Content)
		}
	}
	fmt.Fprintf(&b,
		"This is round %d. Reason step-by-step, consider peers' views (agree, disagree, or refine), "+
			"then put your final answer on the last line prefixed by '#### '.", round)
	return b.String()
}

func extractAnswer(text string) string {
	matches := answerRE.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return ""
	}
	return strings.TrimSpace(matches[len(matches)-1][1])
}
