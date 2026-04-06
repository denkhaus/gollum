// Package channel provides unit tests for the command manager service.
package channel

import (
	"context"
	"errors"

	"testing"

	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestNewCommandManager tests that NewCommandManager creates a valid instance
func TestNewCommandManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))

	service, err := NewCommandManager(injector)

	require.NoError(t, err)
	assert.NotNil(t, service)

	// Verify it implements CommandManager interface
	_, ok := service.(CommandManager)
	assert.True(t, ok, "NewCommandManager should return a CommandManager implementation")
}

// TestCommandManager_Register_Success tests successful command registration
func TestCommandManager_Register_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	cmd := Command{
		Name:        "/test",
		Description: "A test command",
		Handler: func(ctx context.Context, session *shared.Session, args string) (string, error) {
			return "test response", nil
		},
	}

	err = service.Register(cmd)

	assert.NoError(t, err)

	// Verify command is registered
	commands := service.List()
	assert.Len(t, commands, 1)
	assert.Equal(t, "/test", commands[0].Name)
	assert.Equal(t, "A test command", commands[0].Description)
}

// TestCommandManager_Register_Duplicate tests that registering a duplicate command returns an error
func TestCommandManager_Register_Duplicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	cmd := Command{
		Name:        "/test",
		Description: "A test command",
		Handler: func(ctx context.Context, session *shared.Session, args string) (string, error) {
			return "test response", nil
		},
	}

	// Register first time - should succeed
	err = service.Register(cmd)
	require.NoError(t, err)

	// Register second time - should fail
	err = service.Register(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	assert.Contains(t, err.Error(), "/test")
}

// TestCommandManager_Unregister_Success tests successful command unregistration
func TestCommandManager_Unregister_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	cmd := Command{
		Name:        "/test",
		Description: "A test command",
		Handler: func(ctx context.Context, session *shared.Session, args string) (string, error) {
			return "test response", nil
		},
	}

	// Register command
	err = service.Register(cmd)
	require.NoError(t, err)

	// Verify it's registered
	commands := service.List()
	assert.Len(t, commands, 1)

	// Unregister command
	err = service.Unregister("/test")
	assert.NoError(t, err)

	// Verify it's gone
	commands = service.List()
	assert.Len(t, commands, 0)
}

// TestCommandManager_Unregister_NonExistent tests that unregistering a non-existent command doesn't error
func TestCommandManager_Unregister_NonExistent(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	// Unregister non-existent command - should not error
	err = service.Unregister("/nonexistent")
	assert.NoError(t, err)
}

// TestCommandManager_Execute_EmptyInput tests that empty input returns (false, "", nil)
func TestCommandManager_Execute_EmptyInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	ctx := context.Background()
	handled, response, err := service.Execute(ctx, "test-session", "")

	assert.False(t, handled)
	assert.Empty(t, response)
	assert.NoError(t, err)
}

// TestCommandManager_Execute_NonCommandInput tests that non-command input (no "/") returns (false, "", nil)
func TestCommandManager_Execute_NonCommandInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	ctx := context.Background()

	testCases := []struct {
		name  string
		input string
	}{
		{"plain text", "hello world"},
		{"text with spaces", "this is not a command"},
		{"text without slash prefix", "test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handled, response, err := service.Execute(ctx, "test-session", tc.input)

			assert.False(t, handled)
			assert.Empty(t, response)
			assert.NoError(t, err)
		})
	}
}

// TestCommandManager_Execute_UnknownCommand tests that unknown command returns (false, "", nil)
func TestCommandManager_Execute_UnknownCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	ctx := context.Background()
	handled, response, err := service.Execute(ctx, "test-session", "/unknown")

	assert.False(t, handled)
	assert.Empty(t, response)
	assert.NoError(t, err)
}

// TestCommandManager_Execute_ParseCommandNameAndArgs tests command parsing with various inputs
func TestCommandManager_Execute_ParseCommandNameAndArgs(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	mockSM := session.NewMockSessionManager(ctrl)
	do.ProvideValue[session.SessionManager](injector, mockSM)
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	// Create a test session
	_, _ = mockSM.CreateSession("test-session", uuid.New())

	// Track what args were passed to the handler
	var capturedArgs string
	cmd := Command{
		Name:        "/test",
		Description: "A test command",
		Handler: func(ctx context.Context, session *shared.Session, args string) (string, error) {
			capturedArgs = args
			return "response", nil
		},
	}

	err = service.Register(cmd)
	require.NoError(t, err)

	ctx := context.Background()

	testCases := []struct {
		name         string
		input        string
		expectedArgs string
	}{
		{"command without args", "/test", ""},
		{"command with args", "/test arg1 arg2", "arg1 arg2"},
		{"command with single arg", "/test single", "single"},
		{"command with multiple spaces", "/test  arg1   arg2", " arg1   arg2"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capturedArgs = ""
			handled, response, err := service.Execute(ctx, "test-session", tc.input)

			assert.True(t, handled)
			assert.Equal(t, "response", response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedArgs, capturedArgs)
		})
	}
}

// TestCommandManager_Execute_CallsHandler tests that Execute calls the handler and returns results
func TestCommandManager_Execute_CallsHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	mockSM := session.NewMockSessionManager(ctrl)
	do.ProvideValue[session.SessionManager](injector, mockSM)
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	ctx := context.Background()
	// Create a test session
	_, _ = mockSM.CreateSession("test-session", uuid.New())

	testCases := []struct {
		name            string
		handlerResponse string
		handlerError    error
		expectedHandled bool
		expectedResp    string
		expectedErr     error
	}{
		{
			name:            "handler returns success",
			handlerResponse: "success response",
			handlerError:    nil,
			expectedHandled: true,
			expectedResp:    "success response",
			expectedErr:     nil,
		},
		{
			name:            "handler returns error",
			handlerResponse: "",
			handlerError:    errors.New("handler error"),
			expectedHandled: true,
			expectedResp:    "",
			expectedErr:     errors.New("handler error"),
		},
		{
			name:            "handler returns empty response",
			handlerResponse: "",
			handlerError:    nil,
			expectedHandled: true,
			expectedResp:    "",
			expectedErr:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := Command{
				Name:        "/test",
				Description: "A test command",
				Handler: func(ctx context.Context, session *shared.Session, args string) (string, error) {
					return tc.handlerResponse, tc.handlerError
				},
			}

			err = service.Register(cmd)
			require.NoError(t, err)

			handled, response, err := service.Execute(ctx, "test-session", "/test args")

			assert.Equal(t, tc.expectedHandled, handled)
			assert.Equal(t, tc.expectedResp, response)
			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			// Clean up for next test
			_ = service.Unregister("/test")
		})
	}
}

// TestCommandManager_List_ReturnsAllCommands tests that List returns all registered commands
func TestCommandManager_List_ReturnsAllCommands(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	// Initially empty
	commands := service.List()
	assert.Len(t, commands, 0)

	// Add multiple commands
	cmds := []Command{
		{Name: "/cmd1", Description: "Command 1", Handler: func(ctx context.Context, session *shared.Session, args string) (string, error) { return "", nil }},
		{Name: "/cmd2", Description: "Command 2", Handler: func(ctx context.Context, session *shared.Session, args string) (string, error) { return "", nil }},
		{Name: "/cmd3", Description: "Command 3", Handler: func(ctx context.Context, session *shared.Session, args string) (string, error) { return "", nil }},
	}

	for _, cmd := range cmds {
		err = service.Register(cmd)
		require.NoError(t, err)
	}

	commands = service.List()
	assert.Len(t, commands, 3)

	// Verify all commands are present (order doesn't matter)
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name] = true
	}

	assert.True(t, commandNames["/cmd1"])
	assert.True(t, commandNames["/cmd2"])
	assert.True(t, commandNames["/cmd3"])
}

// TestCommandManager_IsCommand_RegisteredCommand tests that IsCommand returns true for registered commands
func TestCommandManager_IsCommand_RegisteredCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	cmd := Command{
		Name:        "/test",
		Description: "A test command",
		Handler:     func(ctx context.Context, session *shared.Session, args string) (string, error) { return "", nil },
	}

	err = service.Register(cmd)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"exact command", "/test", true},
		{"command with args", "/test args", true},
		{"command with multiple args", "/test arg1 arg2 arg3", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isCmd := service.IsCommand(tc.input)
			assert.Equal(t, tc.expected, isCmd)
		})
	}
}

// TestCommandManager_IsCommand_UnregisteredCommand tests that IsCommand returns false for unregistered commands
func TestCommandManager_IsCommand_UnregisteredCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	// Register one command
	cmd := Command{
		Name:        "/test",
		Description: "A test command",
		Handler:     func(ctx context.Context, session *shared.Session, args string) (string, error) { return "", nil },
	}

	err = service.Register(cmd)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"different command", "/other", false},
		{"different command with args", "/other args", false},
		{"non-command input", "hello world", false},
		{"empty string", "", false},
		{"just slash", "/", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isCmd := service.IsCommand(tc.input)
			assert.Equal(t, tc.expected, isCmd)
		})
	}
}

// TestCommandManager_Concurrency tests that concurrent access is safe
func TestCommandManager_Concurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	mockSM := session.NewMockSessionManager(ctrl)
	do.ProvideValue[session.SessionManager](injector, mockSM)
	service, err := NewCommandManager(injector)
	require.NoError(t, err)

	ctx := context.Background()
	// Create a test session for concurrent execution
	_, _ = mockSM.CreateSession("test-session", uuid.New())
	done := make(chan bool)

	// Register commands concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			cmdName := "/cmd" + string(rune('0'+idx))
			cmd := Command{
				Name:        cmdName,
				Description: "Concurrent test command",
				Handler:     func(ctx context.Context, session *shared.Session, args string) (string, error) { return "", nil },
			}
			_ = service.Register(cmd)
			done <- true
		}(i)
	}

	// Execute commands concurrently
	for i := 0; i < 10; i++ {
		go func() {
			_, _, _ = service.Execute(ctx, "test-session", "/cmd0 test")
			_ = service.List()
			_ = service.IsCommand("/cmd0")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify final state
	commands := service.List()
	assert.Greater(t, len(commands), 0)
}

// setupTestCommandManager creates a command manager with a mock session manager
func setupTestCommandManager(t *testing.T) (CommandManager, *session.MockSessionManager) {
	ctrl := gomock.NewController(t)
	injector := do.New()
	sm := session.NewMockSessionManager(ctrl)

	// Setup default mock behavior
	testSession := &shared.Session{
		ID:        "test-session",
		ChannelID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	sm.EXPECT().CreateSession("test-session", uuid.MustParse("00000000-0000-0000-0000-000000000001")).
		Return(testSession, nil).AnyTimes()
	sm.EXPECT().GetSession("test-session").Return(testSession, true).AnyTimes()

	do.ProvideValue[session.SessionManager](injector, sm)
	service, err := NewCommandManager(injector)
	require.NoError(t, err)
	return service, sm
}
