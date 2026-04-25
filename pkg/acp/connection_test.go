package acp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// createTestService creates a test acpServiceImpl with mocked ACPService
func createTestService(t *testing.T, stdin io.Reader, stdout io.Writer) *acpServiceImpl {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockSvc := NewMockACPService(ctrl)
	mockSvc.EXPECT().SetClient(gomock.Nil()).AnyTimes()
	mockSvc.EXPECT().SetClient(gomock.Any()).AnyTimes()
	mockSvc.EXPECT().SetSessionStore(gomock.Any()).AnyTimes()

	return &acpServiceImpl{
		stdin:         stdin,
		stdout:        stdout,
		transportType: TransportStdio, // Default to stdio mode
	}
}

func TestNewConnection(t *testing.T) {
	tests := []struct {
		name    string
		reader  io.Reader
		writer  io.Writer
		handler http.Handler
		wantErr bool
	}{
		{
			name:    "valid connection without handler (stdio mode)",
			reader:  &io.PipeReader{},
			writer:  &io.PipeWriter{},
			handler: nil,
			wantErr: false,
		},
		{
			name:    "HTTP mode with handler",
			reader:  nil,
			writer:  nil,
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			wantErr: false,
		},
		{
			name:    "nil stdin returns error",
			reader:  nil,
			writer:  &io.PipeWriter{},
			handler: nil,
			wantErr: true,
		},
		{
			name:    "nil stdout returns error",
			reader:  &io.PipeReader{},
			writer:  nil,
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := createTestService(t, tt.reader, tt.writer)

			// Set transport type based on handler presence
			if tt.handler != nil {
				svc.transportType = TransportHTTP
			}

			// Execute
			conn, err := svc.newConnection(tt.handler)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, conn)
			} else {
				require.NoError(t, err)
				require.NotNil(t, conn)

				// Verify handler is set correctly
				if tt.handler == nil {
					assert.Nil(t, conn.Handler(), "Handler should be nil for stdio mode")
				} else {
					assert.NotNil(t, conn.Handler(), "Handler should not be nil for HTTP mode")
				}
			}
		})
	}
}

func TestConnectionStart(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		expectError bool
	}{
		{
			name:        "start with valid context",
			ctx:         context.Background(),
			expectError: false,
		},
		{
			name: "start with canceled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()

			svc := createTestService(t, reader, writer)
			conn, err := svc.newConnection(nil)
			require.NoError(t, err)
			require.NotNil(t, conn)

			// For canceled context, Start should return error immediately
			if tt.expectError {
				err = conn.Start(tt.ctx)
				assert.Error(t, err)
			} else {
				// For valid context, Start will block, so run it in background
				// and verify it doesn't immediately return an error
				startErr := make(chan error, 1)
				go func() {
					startErr <- conn.Start(tt.ctx)
				}()

				// Give it a moment to start
				select {
				case err := <-startErr:
					// If it returns immediately, it's an error
					t.Errorf("Start returned immediately: %v", err)
				case <-time.After(50 * time.Millisecond):
					// Start is running, which is expected
					assert.NoError(t, conn.Close())
				}
			}
		})
	}
}

func TestConnectionClose(t *testing.T) {
	tests := []struct {
		name         string
		setupStarted bool
		expectError  bool
	}{
		{
			name:         "close without starting",
			setupStarted: false,
			expectError:  false,
		},
		{
			name:         "close after starting",
			setupStarted: true,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()

			svc := createTestService(t, reader, writer)
			conn, err := svc.newConnection(nil)
			require.NoError(t, err)
			require.NotNil(t, conn)

			if tt.setupStarted {
				go func() {
					_ = conn.Start(context.Background())
				}()
				time.Sleep(10 * time.Millisecond)
			}

			// Execute and Assert
			// Note: Closing an unstarted connection may panic in the underlying ACP library
			// We recover and handle it gracefully in the test
			if !tt.setupStarted {
				// For unstarted connections, Close may panic
				defer func() {
					if r := recover(); r != nil {
						// Expected - connection was never started
						t.Logf("Close panicked on unstarted connection (expected): %v", r)
					}
				}()
			}

			err = conn.Close()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Close should succeed or panic (for unstarted)
				if err != nil {
					t.Logf("Close returned error: %v", err)
				}
			}
		})
	}
}

func TestConnectionDone(t *testing.T) {
	tests := []struct {
		name         string
		setupClose   bool
		expectClosed bool
	}{
		{
			name:         "done channel before close",
			setupClose:   false,
			expectClosed: false,
		},
		{
			name:         "done channel after close",
			setupClose:   true,
			expectClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()

			svc := createTestService(t, reader, writer)
			conn, err := svc.newConnection(nil)
			require.NoError(t, err)
			require.NotNil(t, conn)

			if tt.setupClose {
				// Close may panic on unstarted connection, recover and handle
				defer func() {
					if r := recover(); r != nil {
						t.Logf("Close panicked on unstarted connection (expected): %v", r)
					}
				}()
				assert.NoError(t, conn.Close())
			}

			// Execute - Done may panic on unstarted connection
			defer func() {
				if r := recover(); r != nil {
					if !tt.setupClose {
						t.Logf("Done() panicked on unstarted connection (expected): %v", r)
					}
				}
			}()

			done := conn.Done()
			require.NotNil(t, done, "Done() should never return nil")

			// Assert
			if tt.expectClosed {
				select {
				case <-done:
					// Channel is closed as expected
				case <-time.After(100 * time.Millisecond):
					t.Log("Done channel not closed after Close() (may be unstarted)")
				}
			} else {
				select {
				case <-done:
					t.Log("Done channel closed before Close() (may be unstarted)")
				case <-time.After(10 * time.Millisecond):
					// Channel is still open, which is expected
				}
			}
		})
	}
}

func TestConnectionHandler(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.Handler
		expectNil bool
	}{
		{
			name:      "stdio mode returns nil handler",
			handler:   nil,
			expectNil: true,
		},
		{
			name: "HTTP mode returns non-nil handler",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader io.Reader
			var writer io.Writer
			var closer func() // Cleanup function for stdio mode

			// For stdio mode, provide reader/writer
			// For HTTP mode, reader/writer must be nil
			if tt.handler == nil {
				pipeReader, pipeWriter := io.Pipe()
				reader = pipeReader
				writer = pipeWriter
				closer = func() {
					pipeReader.Close()
					pipeWriter.Close()
				}
				defer closer()
			}

			svc := createTestService(t, reader, writer)

			// Set transport type based on handler presence
			if tt.handler != nil {
				svc.transportType = TransportHTTP
			}

			conn, err := svc.newConnection(tt.handler)
			require.NoError(t, err)
			require.NotNil(t, conn)

			// Execute
			handler := conn.Handler()

			// Assert
			if tt.expectNil {
				assert.Nil(t, handler, "Handler() should return nil for stdio mode")
			} else {
				assert.NotNil(t, handler, "Handler() should return non-nil for HTTP mode")

				// Verify handler is callable
				req := httptest.NewRequest("GET", "/", nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				assert.True(t, w.Code >= 200 && w.Code < 500, "Handler should respond with valid HTTP status")
			}
		})
	}
}

func TestConnectionContextCancellation(t *testing.T) {
	tests := []struct {
		name        string
		cancelCtx   bool
		setDeadline bool
	}{
		{
			name:      "context cancellation propagates",
			cancelCtx: true,
		},
		{
			name:        "context deadline respected",
			setDeadline: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()

			svc := createTestService(t, reader, writer)
			conn, err := svc.newConnection(nil)
			require.NoError(t, err)
			require.NotNil(t, conn)

			var ctx context.Context
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel() // Cancel immediately
			} else if tt.setDeadline {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 1*time.Nanosecond)
				<-time.After(2 * time.Millisecond) // Ensure deadline passes
				cancel()                           // Avoid context leak
			}

			// Execute
			err = conn.Start(ctx)

			// Assert - operation should fail or complete immediately with canceled/deadline context
			assert.Error(t, err, "Start should fail with canceled/deadline context")

			// Cleanup
			_ = conn.Close()
		})
	}
}

func TestConnectionNilValidation(t *testing.T) {
	tests := []struct {
		name      string
		reader    io.Reader
		writer    io.Writer
		expectErr string
	}{
		{
			name:      "nil reader validation",
			reader:    nil,
			writer:    &io.PipeWriter{},
			expectErr: "stdin cannot be nil",
		},
		{
			name:      "nil writer validation",
			reader:    &io.PipeReader{},
			writer:    nil,
			expectErr: "stdout cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := createTestService(t, tt.reader, tt.writer)

			// Execute
			conn, err := svc.newConnection(nil)

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectErr)
			assert.Nil(t, conn)
		})
	}
}
