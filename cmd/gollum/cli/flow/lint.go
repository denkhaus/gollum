package flow

import (
	"context"
	"fmt"
	"io"

	"github.com/denkhaus/gollum/pkg/flows/linter"
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

	result := linter.LintModule(path)
	return WriteLintResult(cmd.Root().Writer, result)
}

// WriteLintResult outputs the lint result and returns appropriate exit code
func WriteLintResult(w io.Writer, result *linter.ModuleLinterResult) error {
	fmt.Fprint(w, result.String())

	if !result.Valid {
		return cli.Exit("", 1)
	}
	return nil
}
