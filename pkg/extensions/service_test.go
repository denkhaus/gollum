package extensions

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/google/uuid"
	"github.com/open2b/scriggo"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExtensionService_NewExtensionServiceWithWorkspace(t *testing.T) {
	injector := do.New()

	// Mock required dependencies
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) {
		return &mockLogger{}, nil
	})
	do.Provide(injector, func(i do.Injector) (DIGateway, error) {
		return NewGatewayService(i)
	})
	do.Provide(injector, func(i do.Injector) (YaegiLoader, error) {
		return NewYaegiLoader(i)
	})
	do.Provide(injector, func(i do.Injector) (ScriggoRunner, error) {
		return NewScriggoRunner(i)
	})
	do.Provide(injector, func(i do.Injector) (workspace.Service, error) {
		return &mockWorkspace{}, nil
	})

	service, err := NewExtensionServiceWithWorkspace(injector)

	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestExtensionService_GetFuncRunner(t *testing.T) {
	service := &extensionServiceImpl{
		scriggoRunner: &scriggoRunnerImpl{
			funcs: make(map[string]*scriggo.Program),
		},
	}

	runner := service.GetFuncRunner()
	assert.NotNil(t, runner)
}

// Mocks
type mockLogger struct{}

func (m *mockLogger) Info(msg string, fields ...zap.Field) {}
func (m *mockLogger) Infof(template string, args ...interface{}) {}
func (m *mockLogger) Error(msg string, fields ...zap.Field) {}
func (m *mockLogger) Errorf(template string, args ...interface{}) {}
func (m *mockLogger) Debug(msg string, fields ...zap.Field) {}
func (m *mockLogger) Debugf(template string, args ...interface{}) {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field) {}
func (m *mockLogger) Warnf(template string, args ...interface{}) {}
func (m *mockLogger) GetLogger() *zap.Logger { return nil }
func (m *mockLogger) GetLogs(filter logger.LogFilter) []logger.LogEntry { return nil }
func (m *mockLogger) GetLogStats() map[string]interface{} { return nil }
func (m *mockLogger) SetTUIMode(enabled bool) {}
func (m *mockLogger) IsTUIMode() bool { return false }
func (m *mockLogger) EnableFileLogging(gollumDir string, sessionID uuid.UUID) error { return nil }
func (m *mockLogger) CloseFileLogging() error { return nil }
func (m *mockLogger) Flush() error { return nil }

type mockWorkspace struct{}

func (m *mockWorkspace) GetCurrentWorkspace() string { return "/test/workspace" }
func (m *mockWorkspace) GetWorkspaceHistory() []string { return nil }
