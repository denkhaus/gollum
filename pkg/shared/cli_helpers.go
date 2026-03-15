package shared

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

// GetPathArg validates and returns a required path argument
func GetPathArg(cmd *cli.Command) (string, error) {
	if cmd.Args().Len() < 1 {
		return "", cli.Exit("missing path argument", 2)
	}
	return cmd.Args().Get(0), nil
}

// ExitCode extracts the exit code from an error
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(cli.ExitCoder); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// WriteOutput writes formatted output to the command's writer
func WriteOutput(cmd *cli.Command, format string, args ...any) error {
	_, err := fmt.Fprintf(cmd.Root().Writer, format, args...)
	return err
}

// HandleError writes error and returns exit code
func HandleError(cmd *cli.Command, err error, exitCode int) error {
	_, _ = fmt.Fprintf(cmd.Root().Writer, "Error: %v\n", err)
	return cli.Exit("", exitCode)
}
