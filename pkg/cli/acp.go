package cli

import (
	"context"
	"os"

	"github.com/denkhaus/gollum/pkg/acp"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/urfave/cli/v3"
)

// ACPCommand returns the ACP server command
func ACPCommand() *cli.Command {
	return &cli.Command{
		Name:  "acp",
	Usage: "Start Gollum ACP server (Agent Client Protocol)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runACPServer(ctx, cmd)
		},
	}
}

func runACPServer(ctx context.Context, cmd *cli.Command) error {
	// Get injector from root command
	injector := shared.MustGetInjectorFromRoot(cmd)

	// Create ACP connection via factory
	conn, err := acp.NewConnection(injector, os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	// Start connection
	if err := conn.Start(ctx); err != nil {
		return err
	}

	// Wait for completion
	<-conn.Done()
	return nil
}
