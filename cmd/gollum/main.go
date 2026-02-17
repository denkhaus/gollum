// Package main provides the CLI entry point for the Gollum agent system.
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/denkhaus/gollum/pkg/app"
	"github.com/denkhaus/gollum/pkg/di"
	"github.com/denkhaus/gollum/pkg/profiling"
	"github.com/samber/do/v2"
)

func startup(startupCtx context.Context) error {
	// Define and parse profiling flags
	profiling.DefineFlags()

	// Create cancellable context for shutdown
	ctx, cancel := context.WithCancel(startupCtx)
	defer cancel()

	// Setup container and services
	container := di.NewContainer()
	injector := container.RegisterServices(ctx)
	defer func() {
		container.Shutdown()
	}()

	// Handle graceful shutdown for SIGTERM (Ctrl-C is handled in raw mode by stdinReader)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	// Invoke application service from DI container
	applicationService := do.MustInvoke[app.ApplicationService](injector)

	// Parse profiling configuration
	profilingConfig := profiling.ParseFlags()

	// Enable profiling if configured
	if profilingConfig.Enable {
		cleanup := setupProfiling(profilingConfig, injector)
		defer cleanup()
	}

	// Ensure terminal is restored on exit
	defer applicationService.Cleanup()

	// Run the application in simple CLI mode
	return applicationService.Run(ctx)
}

func main() {
	ctx := context.Background()

	if err := startup(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("application error: %v", err)
	}
}
