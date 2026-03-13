package flow

import (
	"context"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/urfave/cli/v3"
)

// LintCommand returns the flow lint command
func LintCommand() *cli.Command {
	return &cli.Command{
		Name:      "lint",
		Usage:     "Validate a flow file or module directory",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			shared.VerboseFlag,
			shared.OutputFormatFlag,
		},
		Action: LintAction,
	}
}

// LintAction executes the flow lint command
func LintAction(ctx context.Context, cmd *cli.Command) error {
	path, err := shared.GetPathArg(cmd)
	if err != nil {
		return err
	}

	// TODO: Implement actual linting logic
	return WriteLintResult(cmd, path)
}

// WriteLintResult outputs the lint result and returns appropriate exit code
func WriteLintResult(cmd *cli.Command, path string) error {
	// Placeholder implementation
	cmd.Root().Writer.Write([]byte("Linting: " + path + "\n"))
	return nil
}
