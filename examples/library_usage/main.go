// Package main demonstrates embedding the EoT framework as a library.
//
// Run with:
//
//	go run ./examples/library_usage
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/agents-pool-svg/eot/pkg/eot"
)

func main() {
	ctx := context.Background()

	res, err := eot.Run(ctx, eot.RunRequest{
		Question: "A bakery sold 135 muffins on Monday. On Tuesday it sold twice " +
			"as many as Monday minus 20. On Wednesday it sold the average of " +
			"Monday and Tuesday, rounded down. How many muffins were sold in total?",
		Agents: []eot.AgentSpec{
			{ID: "Planner", System: "You are a careful math planner. Break problems into numbered steps."},
			{ID: "Calculator", System: "You are a precise calculator. Double-check arithmetic.", Temperature: 0.2},
			{ID: "Reviewer", System: "You are a skeptical reviewer. Verify assumptions against the problem."},
		},
		Topology:  eot.TopologySpec{Name: "debate"},
		MaxRounds: 2,
		Verbose:   true,
		// Credentials auto-resolved from env / .env / ReadMe.md, OR pass explicitly:
		// ConfigOpts: []eot.ConfigOption{
		//     eot.WithAPIBase("https://api.openai.com"),
		//     eot.WithAPIKey("sk-..."),
		//     eot.WithModel("gpt-4o-mini"),
		// },
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n===== Library demo result =====")
	fmt.Printf("converged=%v  rounds=%d  answer=%s\n",
		res.Converged, res.Rounds, res.FinalAnswer)
}
