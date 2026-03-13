package cli

import (
	"context"

	"github.com/denkhaus/gollum/cmd/gollum/cli/flow"
	"github.com/denkhaus/gollum/pkg/app"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
)

// RootCommand returns the root CLI command
func RootCommand() *cli.Command {
	return &cli.Command{
		Name:  "gollum",
		Usage: "AI agent workflow system",
		Action: RunDefault,
		Commands: []*cli.Command{
			flow.FlowCommandGroup(),
		},
	}
}

// RunDefault executes the default application behavior.
// Delegates to ApplicationService which handles:
// - Default flow detection and execution
// - Fallback to TUI if no default flow
func RunDefault(ctx context.Context, cmd *cli.Command) error {
	injector := shared.MustGetInjector(cmd)
	appSvc := do.MustInvoke[app.ApplicationService](injector)
	return appSvc.Run(ctx)
}
