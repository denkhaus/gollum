package flow

import "github.com/urfave/cli/v3"

// FlowCommandGroup returns the flow command group
func FlowCommandGroup() *cli.Command {
	return &cli.Command{
		Name:     "flow",
		Usage:    "Flow management commands",
		Commands: []*cli.Command{
			LintCommand(),
			RunCommand(),
			MigrateCommand(),
		},
	}
}
