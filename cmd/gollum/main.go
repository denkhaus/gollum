// Package main provides the CLI entry point for the Gollum agent system.
package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/denkhaus/gollum/pkg/cli"
)

func main() {
	ctx := context.Background()

	// Build the CLI application
	rootCmd := cli.RootCommand()

	// Run the CLI
	if err := rootCmd.Run(ctx, os.Args); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("application error: %v", err)
	}
}
