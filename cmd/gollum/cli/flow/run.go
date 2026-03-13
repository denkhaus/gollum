package flow

import (
	"context"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/urfave/cli/v3"
)

// RunCommand returns the flow run command
func RunCommand() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "Execute a flow",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			shared.VerboseFlag,
		},
		Action: RunAction,
	}
}

// RunAction executes the flow run command
func RunAction(ctx context.Context, cmd *cli.Command) error {
	path, err := shared.GetPathArg(cmd)
	if err != nil {
		return err
	}

	// TODO: Implement actual run logic
	return WriteRunResult(cmd, path)
}

// WriteRunResult outputs the run result
func WriteRunResult(cmd *cli.Command, path string) error {
	// Placeholder implementation
	cmd.Root().Writer.Write([]byte("Running: " + path + "\n"))
	return nil
}
