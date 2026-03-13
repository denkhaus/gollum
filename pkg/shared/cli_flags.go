package shared

import "github.com/urfave/cli/v3"

// Common reusable flags for CLI commands
var (
	// VerboseFlag enables verbose output
	VerboseFlag = &cli.BoolFlag{
		Name:    "verbose",
		Aliases: []string{"v"},
		Usage:   "enable verbose output",
	}

	// OutputFormatFlag specifies output format (text, json)
	OutputFormatFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "output format (text, json)",
		Value:   "text",
	}
)
