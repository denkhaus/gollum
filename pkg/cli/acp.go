package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
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
				Name:    "transport",
				Aliases: []string{"t"},
				Usage:   "Transport type: stdio or http (default: stdio)",
				Value:   "stdio",
				Sources: cli.EnvVars("GOLLUM_ACP_TRANSPORT"),
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
			&cli.DurationFlag{
				Name:    "shutdown-timeout",
				Usage:   "HTTP server graceful shutdown timeout (default: 10s)",
				Value:   10 * time.Second,
				Sources: cli.EnvVars("GOLLUM_ACP_SHUTDOWN_TIMEOUT"),
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

		// Validate host and port
		if err := validateHostPort(host, port); err != nil {
			return fmt.Errorf("invalid host/port configuration: %w", err)
		}

		options = append(options,
			acp.WithHost(host),
			acp.WithPort(port),
		)
	}

	ch, err := facade.CreateAndRegister(acp.Identifier, options...)
	if err != nil {
		return err
	}

	// Ensure cleanup on exit
	defer func() {
		_ = facade.UnregisterChannel(ch.ID())
	}()

	// Start channel
	// For HTTP mode: creates connection, returns immediately
	// For stdio mode: blocks until client disconnects
	if err := ch.Start(ctx); err != nil {
		return err
	}

	// For HTTP transport, start HTTP server in background
	// This must be called AFTER ch.Start() so the connection is created
	if transportType == acp.TransportHTTP {
		host := cmd.String("host")
		port := cmd.Int("port")
		shutdownTimeout := cmd.Duration("shutdown-timeout")

		if err := startHTTPServer(ctx, cmd, ch, host, port, shutdownTimeout); err != nil {
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}

	return nil
}

// validateHostPort validates host and port configuration.
// Port validation is already done by WithPort, but we validate host here.
func validateHostPort(host string, port int) error {
	// Validate host is not empty
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("host cannot be empty")
	}

	// Validate port is in valid range (redundant with WithPort but defensive)
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	return nil
}

// startHTTPServer starts an HTTP server for the ACP channel with proper resource management.
//
// The server is configured with timeouts to prevent resource exhaustion:
// - ReadTimeout: 15s - maximum time to read the request
// - WriteTimeout: 15s - maximum time to write the response
// - IdleTimeout: 60s - maximum time to wait for next request
//
// The server runs in a goroutine and supports graceful shutdown with configurable timeout.
// It ensures the server goroutine can be stopped even if ch.Start() returns an error.
//
// Parameters:
//   - ctx: Context for server lifecycle (cancellation triggers shutdown)
//   - cmd: CLI command for accessing injector
//   - ch: Channel implementing ACPService interface
//   - host: Server bind address
//   - port: Server port number
//   - shutdownTimeout: Graceful shutdown timeout duration
//
// Returns error if server fails to start or shutdown encounters issues.
func startHTTPServer(ctx context.Context, cmd *cli.Command, ch channel.Channel, host string, port int, shutdownTimeout time.Duration) error {
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

	// Debug: Check if handler is nil
	if handler == nil {
		return fmt.Errorf("ACP handler is nil - connection may not have been created properly")
	}

	// Wrap handler with logging middleware for debugging
	rawHandler := handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("HTTP request received",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("raw_handler_type", fmt.Sprintf("%T", rawHandler)),
		)
		rawHandler.ServeHTTP(w, r)
	})

	// Build address
	addr := fmt.Sprintf("%s:%d", host, port)

	// Create HTTP server with timeouts to prevent resource exhaustion
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to signal server is ready to accept connections
	readyChan := make(chan struct{})

	// Channel for server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		// Use a custom listener to detect when server is ready
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			serverErr <- fmt.Errorf("failed to create listener: %w", err)
			return
		}
		defer listener.Close()

		// Signal that server is ready
		close(readyChan)

		logger.Info("HTTP server started",
			zap.String("address", addr),
			zap.String("channel_id", ch.ID().String()),
		)

		// Serve accepts connections until server.Shutdown() is called
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			select {
			case serverErr <- err:
			default: // Avoid blocking if error already sent
			}
		}
	}()

	// Wait for server to be ready or fail
	select {
	case <-readyChan:
		// Server is ready, print clear user-facing message first
		fmt.Printf("\n✅ ACP HTTP server started on http://%s\n\n", addr)
		fmt.Println("Ready to accept connections. Press Ctrl+C to stop.")
		os.Stdout.Sync() // Force flush output buffer

	case err := <-serverErr:
		// Server failed to start
		return fmt.Errorf("HTTP server failed to start: %w", err)
	case <-ctx.Done():
		// Context cancelled before server was ready
		return fmt.Errorf("context cancelled before server started")
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	logger.Info("shutting down HTTP server",
		zap.String("channel_id", ch.ID().String()),
		zap.Duration("timeout", shutdownTimeout),
	)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("HTTP server shutdown error: %w", err)
	}

	logger.Info("HTTP server stopped",
		zap.String("channel_id", ch.ID().String()),
	)

	return nil
}
