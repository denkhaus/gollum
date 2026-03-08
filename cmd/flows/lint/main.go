package main

import (
	"fmt"
	"os"

	"github.com/denkhaus/gollum/pkg/flows/linter"
	"github.com/denkhaus/gollum/pkg/flows/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: flows-lint <flow.xml>\n")
		os.Exit(1)
	}

	path := os.Args[1]

	// Parse
	flow, err := parser.Parse(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// Lint (with path for module resolution)
	result := linter.LintPath(path, flow)

	// Output results
	if result.Valid {
		fmt.Printf("✓ %s: valid\n", flow.Name)
	} else {
		fmt.Printf("✗ %s: invalid\n", flow.Name)
		for _, e := range result.Errors {
			fmt.Printf("  %s\n", e.String())
		}
	}

	if len(result.Warnings) > 0 {
		for _, w := range result.Warnings {
			fmt.Printf("  Warning: %s\n", w.String())
		}
	}

	if !result.Valid {
		os.Exit(1)
	}
}
