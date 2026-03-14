// Package main provides the CLI entry point for the Gollum agent system.
package main

import (
	"context"
	"errors"
	"log"
	"os"

	gollumcli "github.com/denkhaus/gollum/cmd/gollum/cli"
	"github.com/denkhaus/gollum/pkg/profiling"
)

func main() {
	ctx := context.Background()

	// Define profiling flags before CLI parsing
	profiling.DefineFlags()

	// Build the CLI application
	rootCmd := gollumcli.RootCommand()

	// Run the CLI
	if err := rootCmd.Run(ctx, os.Args); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("application error: %v", err)
	}
}
