package extensions

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/testutil"
	"github.com/denkhaus/gollum/pkg/workspace"
	"go.uber.org/mock/gomock"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionService_NewExtensionServiceWithWorkspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	testutil.SetupMockLoggerPassThrough(mockLogger)

	injector := do.New()

	// Mock required dependencies
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) {
		return mockLogger, nil
	})
	do.Provide(injector, func(i do.Injector) (DIGateway, error) {
		return NewGatewayService(i)
	})
	do.Provide(injector, func(i do.Injector) (YaegiLoader, error) {
		return NewYaegiLoader(i)
	})
	do.Provide(injector, func(i do.Injector) (YaegiFuncRunner, error) {
		return NewYaegiFuncRunner(i)
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
		yaegiFuncRunner: &yaegiFuncRunnerImpl{
			funcs: make(map[string]*funcInfo),
		},
	}

	runner := service.GetFuncRunner()
	assert.NotNil(t, runner)
}

// Mocks
type mockWorkspace struct{}

func (m *mockWorkspace) GetCurrentWorkspace() string   { return "/test/workspace" }
func (m *mockWorkspace) GetWorkspaceHistory() []string { return nil }

func TestExtensionService_LoadExtensions_Integration(t *testing.T) {
	// Create temporary directory for extensions
	tempDir := t.TempDir()
	gollumDir := filepath.Join(tempDir, ".gollum")
	extDir := filepath.Join(gollumDir, "extensions")
	err := os.MkdirAll(extDir, 0755)
	require.NoError(t, err)

	// Create a test extension
	testExtDir := filepath.Join(extDir, "testext")
	err = os.Mkdir(testExtDir, 0755)
	require.NoError(t, err)

	mainGoContent := `
package main

import "fmt"

func Init() error {
	fmt.Println("Test extension initialized")
	return nil
}
`
	mainPath := filepath.Join(testExtDir, "main.go")
	err = os.WriteFile(mainPath, []byte(mainGoContent), 0644)
	require.NoError(t, err)

	// Setup DI container
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	testutil.SetupMockLoggerPassThrough(mockLogger)

	injector := do.New()

	// Mock workspace to return our temp directory
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) {
		return mockLogger, nil
	})
	do.Provide(injector, func(i do.Injector) (workspace.Service, error) {
		return &mockWorkspaceWithDir{dir: tempDir}, nil
	})

	// Register real services
	do.Provide(injector, func(i do.Injector) (DIGateway, error) {
		return NewGatewayService(i)
	})
	do.Provide(injector, func(i do.Injector) (YaegiLoader, error) {
		return NewYaegiLoader(i)
	})
	do.Provide(injector, func(i do.Injector) (YaegiFuncRunner, error) {
		return NewYaegiFuncRunner(i)
	})
	do.Provide(injector, func(i do.Injector) (ExtensionService, error) {
		return NewExtensionServiceWithWorkspace(i)
	})

	// Get service and load extensions
	service, err := do.Invoke[ExtensionService](injector)
	require.NoError(t, err)

	ctx := context.Background()

	// Check if directories exist
	funcsDir := filepath.Join(tempDir, ".gollum", "functions")
	extsDir := filepath.Join(tempDir, ".gollum", "extensions")

	t.Logf("Functions dir exists: %v (path: %s)", dirExists(funcsDir), funcsDir)
	t.Logf("Extensions dir exists: %v (path: %s)", dirExists(extsDir), extsDir)
	t.Logf("Test extension dir exists: %v (path: %s)", dirExists(testExtDir), testExtDir)
	t.Logf("main.go exists: %v (path: %s)", dirExists(mainPath), mainPath)

	err = service.LoadAll(ctx)
	require.NoError(t, err)

	// Verify extension was loaded
	exts := service.ListExtensions()
	assert.Contains(t, exts, "testext")

	// Verify extension state
	ext, err := service.GetExtension("testext")
	require.NoError(t, err)
	assert.Equal(t, StateReady, ext.State)
	assert.NotNil(t, ext.InitFunc)
	assert.NotNil(t, ext.Interpreter)
}

type mockWorkspaceWithDir struct {
	dir string
}

func (m *mockWorkspaceWithDir) GetCurrentWorkspace() string {
	return m.dir
}

func (m *mockWorkspaceWithDir) GetWorkspaceHistory() []string {
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
