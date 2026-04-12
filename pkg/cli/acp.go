package cli

import (
	"context"
	"os"

	"github.com/denkhaus/gollum/pkg/acp"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
)

// ACPCommand returns the ACP server command
func ACPCommand() *cli.Command {
	return &cli.Command{
		Name:   "acp",
		Usage:  "Start Gollum ACP server (Agent Client Protocol)",
		Action: runACPServer,
	}
}

func runACPServer(ctx context.Context, cmd *cli.Command) error {
	// Get injector from root command
	injector := shared.MustGetInjectorFromRoot(cmd)

	// Get channel facade
	facade := do.MustInvoke[channel.ChannelFacade](injector)

	// Create ACP channel via factory with stdin/stdout options
	ch, err := facade.CreateChannel(acp.Identifier,
		acp.WithStdin(os.Stdin),
		acp.WithStdout(os.Stdout),
	)
	if err != nil {
		return err
	}

	// Register channel with facade
	if err := facade.RegisterChannel(ch); err != nil {
		return err
	}

	// Ensure cleanup on exit
	defer func() {
		_ = facade.UnregisterChannel(ch.ID())
	}()

	// Start channel (encapsulates connection creation and lifecycle)
	return ch.Start(ctx)
}
