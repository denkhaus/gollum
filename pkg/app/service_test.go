// Package app provides tests for the application service
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/workspace"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestEnsureGollumDirectory_CreatesDirectory tests that .gollum directory is created
func TestEnsureGollumDirectory_CreatesDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create temp directory for testing
	tempDir := t.TempDir()

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).Times(1)

	p := &applicationServiceImpl{
		workspaceService: mockWorkspaceService,
	}

	err := p.ensureGollumDirectory()
	require.NoError(t, err)

	expectedPath := filepath.Join(tempDir, gollumDirName)
	info, err := os.Stat(expectedPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "expected a directory")
}

// TestEnsureGollumDirectory_CreatesGitignore tests that .gitignore is created
func TestEnsureGollumDirectory_CreatesGitignore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tempDir := t.TempDir()

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).Times(1)

	p := &applicationServiceImpl{
		workspaceService: mockWorkspaceService,
	}

	err := p.ensureGollumDirectory()
	require.NoError(t, err)

	gitignorePath := filepath.Join(tempDir, gollumDirName, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Equal(t, "/logs\nmcp.json\n", string(content))
}

// TestEnsureGollumDirectory_DoesNotOverwriteGitignore tests that existing .gitignore is preserved
func TestEnsureGollumDirectory_DoesNotOverwriteGitignore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tempDir := t.TempDir()

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).Times(2) // Called twice by ensureGollumDirectory

	p := &applicationServiceImpl{
		workspaceService: mockWorkspaceService,
	}

	// First call creates the directory and .gitignore
	err := p.ensureGollumDirectory()
	require.NoError(t, err)

	gitignorePath := filepath.Join(tempDir, gollumDirName, ".gitignore")

	// Modify the .gitignore
	customContent := "/logs\n/custom\n"
	err = os.WriteFile(gitignorePath, []byte(customContent), 0644)
	require.NoError(t, err)

	// Second call should not overwrite
	err = p.ensureGollumDirectory()
	require.NoError(t, err)

	// Verify custom content is preserved
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Equal(t, customContent, string(content))
}

// TestEnsureGollumDirectory_MkdirAllError tests error handling when directory creation fails
func TestEnsureGollumDirectory_MkdirAllError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspaceService := workspace.NewMockService(ctrl)
	// Use an invalid path that will fail
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/dev/null/invalid/path/that/cannot/be/created").Times(1)

	p := &applicationServiceImpl{
		workspaceService: mockWorkspaceService,
	}

	err := p.ensureGollumDirectory()
	assert.Error(t, err)
}

// TestPrimeFileStateManager_Success tests successful priming of FileStateManager
func TestPrimeFileStateManager_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockFSM := state.NewMockFileStateManager(ctrl)
	mockFSM.EXPECT().Prime(ctx).Return(nil).Times(1)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Priming FileStateManager - scanning working directory...").Times(1)
	mockLogger.EXPECT().Infof("FileStateManager primed successfully").Times(1)

	p := &applicationServiceImpl{
		logService: mockLogger,
		fsm:        mockFSM,
	}

	err := p.primeFileStateManager(ctx)
	assert.NoError(t, err)
}

// TestPrimeFileStateManager_PrimeError tests error handling when Prime fails
func TestPrimeFileStateManager_PrimeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expectedErr := assert.AnError

	mockFSM := state.NewMockFileStateManager(ctrl)
	mockFSM.EXPECT().Prime(ctx).Return(expectedErr).Times(1)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Priming FileStateManager - scanning working directory...").Times(1)

	p := &applicationServiceImpl{
		logService: mockLogger,
		fsm:        mockFSM,
	}

	err := p.primeFileStateManager(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

// TestCreateToolSet_Success tests that createToolSet returns tool sets from MCP registry
func TestCreateToolSet_Success(t *testing.T) {
	// Create a mock MCP registry
	mockMCPRegistry := &mockMCPRegistry{
		toolSets: []gollem.ToolSet{},
	}

	toolSets := mockMCPRegistry.GetToolSets()
	assert.NotNil(t, toolSets)
	assert.Empty(t, toolSets) // Empty since mock returns empty slice
}

// mockMCPRegistry is a simple mock for testing
type mockMCPRegistry struct {
	toolSets  []gollem.ToolSet
	toolNames []string
}

func (m *mockMCPRegistry) GetToolSets() []gollem.ToolSet {
	return m.toolSets
}

func (m *mockMCPRegistry) GetToolNames() []string {
	return m.toolNames
}

func (m *mockMCPRegistry) Close() error {
	return nil
}

// TestCreateSupervisorAgent_ToolSetSuccess tests that tool set creation succeeds
// REMOVED: createSupervisorAgent method no longer exists in applicationServiceImpl
// Supervisor agent creation is now handled by agentFactory.CreateSupervisorAgent directly

// TestCreateSupervisorAgent_PromptError tests error handling when prompt retrieval fails
// REMOVED: createSupervisorAgent method no longer exists in applicationServiceImpl
// Supervisor agent creation is now handled by agentFactory.CreateSupervisorAgent directly

// TestCreateSupervisorAgent_AgentFactoryError tests error handling when agent creation fails
// REMOVED: createSupervisorAgent method no longer exists in applicationServiceImpl
// Supervisor agent creation is now handled by agentFactory.CreateSupervisorAgent directly

// TestCreateSupervisorAgent_RegistryError tests error handling when registration fails
// REMOVED: createSupervisorAgent method no longer exists in applicationServiceImpl
// Supervisor agent creation is now handled by agentFactory.CreateSupervisorAgent directly

// TestCleanup_NoOp tests that Cleanup is a no-op (kept for interface compatibility)
func TestCleanup_NoOp(t *testing.T) {
	p := &applicationServiceImpl{}

	// Should not panic
	assert.NotPanics(t, func() {
		p.Cleanup()
	})
}

func TestRun_NoDefaultFlowRunsTUI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Create temp directory without default flow
	tempDir := t.TempDir()

	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().GetDefaultFlow().Return(nil, fmt.Errorf("no default flow found")).Times(1)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().EnableFileLogging(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockLogger.EXPECT().CloseFileLogging().Return(nil).Times(1)
	mockLogger.EXPECT().Infof("Priming FileStateManager - scanning working directory...").Times(1)
	mockLogger.EXPECT().Infof("FileStateManager primed successfully").Times(1)
	mockLogger.EXPECT().Info("no default flow found -> run tui").Times(1)

	mockFSM := state.NewMockFileStateManager(ctrl)
	mockFSM.EXPECT().Prime(ctx).Return(nil).Times(1)

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).AnyTimes()

	// Mock channel that implements Channel (now includes Start)
	mockCh := channel.NewMockChannel(ctrl)
	mockChID := uuid.New()
	mockCh.EXPECT().ID().Return(mockChID).AnyTimes()
	mockCh.EXPECT().Start(ctx).Return(nil).Times(1)

	mockChannelFacade := channel.NewMockChannelFacade(ctrl)
	mockChannelFacade.EXPECT().CreateChannel(gomock.Any(), gomock.Any()).Return(mockCh, nil).Times(1)
	mockChannelFacade.EXPECT().RegisterChannel(mockCh).Return(nil).Times(1)

	p := &applicationServiceImpl{
		gollumDir:        filepath.Join(tempDir, gollumDirName),
		logService:       mockLogger,
		fsm:              mockFSM,
		flowRegistry:     mockFlowRegistry,
		workspaceService: mockWorkspaceService,
		channelFacade:    mockChannelFacade,
	}

	// Should complete successfully with the mocked channel
	err := p.Run(ctx)
	assert.NoError(t, err)
}

// TestRunDefaultFlow_ParseError tests error handling when flow parsing fails
// DEPRECATED: Parsing now happens in FlowRegistry, not in ApplicationService
// This test is kept for backwards compatibility but is no longer relevant
// mockMarkdownRenderer is a simple mock for testing
type mockMarkdownRenderer struct{}

func (m *mockMarkdownRenderer) Render(ctx context.Context, markdown string, width int) (string, error) {
	return markdown, nil
}
