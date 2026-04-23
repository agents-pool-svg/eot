package main

import (
	"fmt"
	"os"

	"github.com/agents-pool-svg/eot/cmd/eot/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
