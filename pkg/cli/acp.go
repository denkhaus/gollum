package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/denkhaus/gollum/pkg/acp"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
)

// ACPCommand returns the ACP server command
func ACPCommand() *cli.Command {
	return &cli.Command{
		Name:   "acp",
		Usage:  "Start Gollum ACP server (Agent Client Protocol)",
		Action: runACPServer,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "transport",
				Aliases:  []string{"t"},
				Usage:    "Transport type: stdio or http (default: stdio)",
				Value:    "stdio",
				Sources:  cli.EnvVars("GOLLUM_ACP_TRANSPORT"),
			},
			&cli.StringFlag{
				Name:    "host",
				Usage:   "HTTP server bind address (default: 0.0.0.0)",
				Value:   "0.0.0.0",
				Sources: cli.EnvVars("GOLLUM_ACP_HOST"),
			},
			&cli.IntFlag{
				Name:    "port",
				Usage:   "HTTP server port (default: 8080)",
				Value:   8080,
				Sources: cli.EnvVars("GOLLUM_ACP_PORT"),
			},
		},
	}
}

func runACPServer(ctx context.Context, cmd *cli.Command) error {
	// Parse transport type
	transportStr := cmd.String("transport")
	transportType, err := acp.ParseTransportType(transportStr)
	if err != nil {
		return fmt.Errorf("invalid transport type %q: %w", transportStr, err)
	}

	// Get injector from root command
	injector := shared.MustGetInjectorFromRoot(cmd)

	// Get channel facade
	facade := do.MustInvoke[channel.ChannelFacade](injector)

	// Build channel options based on transport type
	var options []channel.ChannelOption
	if transportType == acp.TransportStdio {
		options = append(options,
			acp.WithStdin(os.Stdin),
			acp.WithStdout(os.Stdout),
		)
	}
	options = append(options, acp.WithTransport(transportType))

	// Add HTTP-specific options
	if transportType == acp.TransportHTTP {
		host := cmd.String("host")
		port := cmd.Int("port")
		options = append(options,
			acp.WithHost(host),
			acp.WithPort(port),
		)
	}

	// Create ACP channel via factory
	ch, err := facade.CreateChannel(acp.Identifier, options...)
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

	// For HTTP transport, start HTTP server in background
	if transportType == acp.TransportHTTP {
		if err := startHTTPServer(ctx, cmd, ch); err != nil {
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}

	// Start channel (encapsulates connection creation and lifecycle)
	return ch.Start(ctx)
}

// startHTTPServer starts an HTTP server for the ACP channel
func startHTTPServer(ctx context.Context, cmd *cli.Command, ch channel.Channel) error {
	// Get injector from root command
	injector := shared.MustGetInjectorFromRoot(cmd)

	// Get logger
	logService := do.MustInvoke[logger.LoggerService](injector)
	logger := logService.GetLogger()

	// Type assert to get ACP service
	acpService, ok := ch.(shared.ACPService)
	if !ok {
		return fmt.Errorf("channel does not implement ACPService interface")
	}

	// Get HTTP handler
	handler := acpService.GetHandler()

	// Get host and port from flags
	host := cmd.String("host")
	port := cmd.Int("port")
	addr := fmt.Sprintf("%s:%d", host, port)

	// Create HTTP server
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting HTTP server",
			zap.String("address", addr),
			zap.String("channel_id", ch.ID().String()),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		logger.Info("shutting down HTTP server",
			zap.String("channel_id", ch.ID().String()),
		)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}

		logger.Info("HTTP server stopped",
			zap.String("channel_id", ch.ID().String()),
		)
		return nil

	case err := <-serverErr:
		// Server failed to start
		return fmt.Errorf("HTTP server error: %w", err)
	}
}
