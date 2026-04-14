package extensions

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"go.uber.org/mock/gomock"
)

func TestService_AuditLogging(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)

	// Expect logging calls during LoadAll
	mockLogger.EXPECT().Info("Extension service: starting load", gomock.Any()).Times(1)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info("Extension service: load complete", gomock.Any()).Times(1)
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes() // For any errors that might occur

	// Create a minimal service implementation for testing
	service := &extensionServiceImpl{
		logService:  mockLogger,
		yaegiLoader: &mockYaegiLoader{},
		yaegiFuncRunner: &mockYaegiFuncRunner{},
		loadedFuncs: make(map[string]string),
	}

	// Run LoadAll - it should fail since we don't have a real workspace,
	// but we're testing that logging happens
	err := service.LoadAll(context.Background())

	// The test verifies that logging was called via the mock expectations above
	// We don't care if LoadAll succeeds or fails for this test
	_ = err
}

// Mock implementations for testing

type mockYaegiLoader struct{}

func (m *mockYaegiLoader) LoadExtension(path string) (*Extension, error) {
	return nil, nil
}

func (m *mockYaegiLoader) InitExtension(ext *Extension) error {
	return nil
}

func (m *mockYaegiLoader) UnloadExtension(ext *Extension) error {
	return nil
}

func (m *mockYaegiLoader) GetExtension(name string) (*Extension, error) {
	return nil, nil
}

func (m *mockYaegiLoader) ListExtensions() []string {
	return []string{}
}

func (m *mockYaegiLoader) LoadExtensions(workspaceDir string) error {
	return nil
}

type mockYaegiFuncRunner struct{}

func (m *mockYaegiFuncRunner) LoadFunc(name, source string) error {
	return nil
}

func (m *mockYaegiFuncRunner) ExecuteFunc(name string, args map[string]any) (any, error) {
	return nil, nil
}

func (m *mockYaegiFuncRunner) ListFuncs() []string {
	return []string{}
}
