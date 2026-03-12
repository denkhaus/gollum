package main

import (
	"fmt"
	"os"

	"github.com/denkhaus/gollum/pkg/flows/linter"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: flows-lint <flow.xml | module-directory>\n")
		os.Exit(1)
	}

	path := os.Args[1]

	// Lint module (handles both files and directories)
	result := linter.LintModule(path)

	// Output results
	fmt.Print(result.String())

	if !result.Valid {
		os.Exit(1)
	}
}
