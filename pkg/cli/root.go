package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/denkhaus/gollum/pkg/acp"
	"github.com/denkhaus/gollum/pkg/app"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/cli/flow"
	"github.com/denkhaus/gollum/pkg/di"
	"github.com/denkhaus/gollum/pkg/flows/linter"
	"github.com/denkhaus/gollum/pkg/profiling"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/tui"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
)

type rootHandler struct {
}

// registerChannels registers all available channels with the DI container.
func registerChannels(injector do.Injector) error {
	tui.RegisterChannels(injector)
	acp.RegisterChannels(injector)
	return nil
}

// RootCommand returns the root CLI command
func RootCommand() *cli.Command {
	r := &rootHandler{}

	return &cli.Command{
		Name:   "gollum",
		Usage:  "AI agent workflow system",
		Before: r.before,
		After:  r.after,
		Action: r.run,
		Commands: []*cli.Command{
			flow.FlowCommandGroup(),
			ACPCommand(),
		},
	}
}

func (p *rootHandler) before(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	// Initialize XSD validator for XML schema validation
	if err := linter.InitXSD(); err != nil {
		return nil, fmt.Errorf("XSD validator initialization failed: libxml2 required but not available. Install with: sudo apt-get install -y libxml2-dev\n(original error: %w)", err)
	}

	// Define profiling flags before CLI parsing
	profiling.DefineFlags()
	// Create cancellable context for shutdown
	shutdownCtx, cancel := context.WithCancel(ctx)

	// Setup container and services
	container := di.NewContainer()
	injector := container.RegisterServices(shutdownCtx)

	// Register all channels
	if err := registerChannels(injector); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register channels: %w", err)
	}

	// ChannelFacade will automatically discover providers during construction
	_ = do.MustInvoke[channel.ChannelFacade](injector)

	// Store injector in command metadata for subcommands to access
	shared.SetInjector(cmd, injector)

	// Handle graceful shutdown for SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	// Parse profiling configuration and enable if configured
	profilingConfig := profiling.ParseFlags()
	if profilingConfig.Enable {
		setupProfiling(profilingConfig, injector)
	}

	return shutdownCtx, nil
}

func (p *rootHandler) after(ctx context.Context, cmd *cli.Command) error {
	// Cleanup XSD validator resources
	linter.CleanupXSD()

	// Get injector from metadata
	injector, err := shared.GetInjector(cmd)
	if err != nil {
		return nil // No injector was set, nothing to clean up
	}

	// Get container from injector and shutdown
	// The DI container manages its own shutdown
	_ = injector // Container shutdown is handled by the injector lifecycle

	return nil
}

// run executes the default application behavior.
// Delegates to ApplicationService which handles:
// - Default flow detection and execution
// - Fallback to TUI if no default flow
func (p *rootHandler) run(ctx context.Context, cmd *cli.Command) error {
	injector := shared.MustGetInjector(cmd)
	appSvc := do.MustInvoke[app.ApplicationService](injector)
	return appSvc.Run(ctx)
}
