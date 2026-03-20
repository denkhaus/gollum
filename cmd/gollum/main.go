// Package main provides the CLI entry point for the Gollum agent system.
package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/denkhaus/gollum/pkg/cli"
	"github.com/denkhaus/gollum/pkg/flows/linter"
)

func main() {
	ctx := context.Background()

	// Initialize XSD validator for XML schema validation
	if err := linter.InitXSD(); err != nil {
		log.Fatalf("failed to initialize XSD validator: %v", err)
	}
	defer linter.CleanupXSD()

	// Build the CLI application
	rootCmd := cli.RootCommand()

	// Run the CLI
	if err := rootCmd.Run(ctx, os.Args); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("application error: %v", err)
	}
}
