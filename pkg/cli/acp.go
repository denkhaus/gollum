package cli

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	conn := acp.NewConnection(injector, os.Stdin, os.Stdout)

	// Handle shutdown gracefully
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("ACP server shutting down...")
		cancel()
	}()

	// Start connection
	if err := conn.Start(ctx); err != nil {
		return err
	}

	// Wait for completion
	<-conn.Done()
	return nil
}
