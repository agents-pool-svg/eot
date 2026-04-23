// Package eot implements the Exchange-of-Thought framework.
//
// Reference:
//
//	Yao et al., "Exchange-of-Thought: Enhancing Large Language Model
//	Capabilities through Cross-Model Communication", EMNLP 2023.
//
// The core idea, as opposed to single-model Chain-of-Thought (CoT) or
// Tree-of-Thought (ToT), is to let multiple LLM agents exchange their
// *intermediate reasoning* (thoughts) according to a configurable
// communication topology: Memory, Report, Relay, or Debate.
package eot

// Version of the framework.
const Version = "0.1.0"
