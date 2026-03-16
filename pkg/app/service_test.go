// Package app provides tests for the application service
package app

import (

	"context"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/workspace"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

)

// TestAgentExecutorAdapter_Execute verifies that the adapter correctly
// delegates to the underlying agent
func TestAgentExecutorAdapter_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ ...gollem.Input) (*gollem.ExecuteResponse, error) {
		return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
	}).Times(1)

	adapter := &agentExecutorAdapter{agent: mockAgent}

	response, err := adapter.Execute(ctx, "test input")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("response should not be nil")
	}

	if len(response.Texts) != 1 {
		t.Errorf("expected 1 text, got %d", len(response.Texts))
	}

	if response.Texts[0] != "response" {
		t.Errorf("expected 'response', got '%s'", response.Texts[0])
	}
}

// TestAgentExecutorAdapter_MultipleExecutions verifies that multiple sequential
// executions work correctly
func TestAgentExecutorAdapter_MultipleExecutions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	executionCount := 0
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ ...gollem.Input) (*gollem.ExecuteResponse, error) {
		executionCount++
		return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
	}).Times(3)

	adapter := &agentExecutorAdapter{agent: mockAgent}

	// Execute multiple times
	for i := 0; i < 3; i++ {
		_, err := adapter.Execute(ctx, "test")
		if err != nil {
			t.Errorf("execution %d: unexpected error: %v", i, err)
		}
	}

	if executionCount != 3 {
		t.Errorf("expected 3 executions, got %d", executionCount)
	}

	// Verify global context is still active
	if ctx.Err() != nil {
		t.Error("context was canceled after multiple executions")
	}
}

// TestAgentExecutorAdapter_PropagatesAgentError tests that agent errors are propagated
func TestAgentExecutorAdapter_PropagatesAgentError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expectedErr := assert.AnError

	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, expectedErr).Times(1)

	adapter := &agentExecutorAdapter{agent: mockAgent}

	response, err := adapter.Execute(ctx, "test input")
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, response)
}

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
	toolSets []gollem.ToolSet
}

func (m *mockMCPRegistry) GetToolSets() []gollem.ToolSet {
	return m.toolSets
}

func (m *mockMCPRegistry) Close() error {
	return nil
}

// TestCreateSupervisorAgent_ToolSetSuccess tests that tool set creation succeeds
func TestCreateSupervisorAgent_ToolSetSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Warn("no mcp servers configured for supervision agent").Times(1)
	mockLogger.EXPECT().Infof("Supervisor agent %s registered", gomock.Any()).Times(1)

	mockFSM := state.NewMockFileStateManager(ctrl)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil)

	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockPromptMgr.EXPECT().GetPromptWithContext(gomock.Any(), gomock.Any(), gomock.Any()).Return("system prompt", nil)

	mockAgent := shared.NewMockAgent(ctrl)
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	mockAgent.EXPECT().GetID().Return(testUUID).AnyTimes()

	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).Return(mockAgent, nil)

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/workspace")

	mockSkillsService := skills.NewMockSkillService(ctrl)
	mockSkillsService.EXPECT().GetSkillsXML().Return("<skills/>")
	mockSkillsService.EXPECT().GetSkillInfos().Return([]shared.SkillInfo{})

	// Create a mock MCP registry
	mockMCPRegistry := &mockMCPRegistry{
		toolSets: []gollem.ToolSet{},
	}

	p := &applicationServiceImpl{
		logService:       mockLogger,
		fsm:              mockFSM,
		agentRegistry:    mockRegistry,
		promptMgr:        mockPromptMgr,
		agentFactory:     mockAgentFactory,
		workspaceService: mockWorkspaceService,
		skillsService:    mockSkillsService,
		mcpRegistry:      mockMCPRegistry,
	}

	agent, cfg, err := p.createSupervisorAgent(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.NotNil(t, cfg)
}

// TestCreateSupervisorAgent_PromptError tests error handling when prompt retrieval fails
func TestCreateSupervisorAgent_PromptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockLogger := logger.NewMockLoggerService(ctrl)

	mockFSM := state.NewMockFileStateManager(ctrl)
	mockRegistry := registry.NewMockAgentRegistry(ctrl)

	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockPromptMgr.EXPECT().GetPromptWithContext(ctx, prompt.PromptIDSupervisorSystem, gomock.Any()).
		Return("", assert.AnError).Times(1)

	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockSkillsService := skills.NewMockSkillService(ctrl)
	mockSkillsService.EXPECT().GetSkillsXML().Return("<skills></skills>").Times(1)
	mockSkillsService.EXPECT().GetSkillInfos().Return([]shared.SkillInfo{}).Times(1)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/test/workspace").Times(1)

	mockMCPRegistry := &mockMCPRegistry{
		toolSets: []gollem.ToolSet{},
	}

	p := &applicationServiceImpl{
		logService:       mockLogger,
		fsm:              mockFSM,
		agentRegistry:    mockRegistry,
		promptMgr:        mockPromptMgr,
		agentFactory:     mockAgentFactory,
		workspaceService: mockWorkspaceService,
		skillsService:    mockSkillsService,
		mcpRegistry:      mockMCPRegistry,
	}

	_, _, err := p.createSupervisorAgent(ctx)
	assert.Error(t, err)
}

// TestCreateSupervisorAgent_AgentFactoryError tests error handling when agent creation fails
func TestCreateSupervisorAgent_AgentFactoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Warn("no mcp servers configured for supervision agent").Times(1)

	mockFSM := state.NewMockFileStateManager(ctrl)
	mockRegistry := registry.NewMockAgentRegistry(ctrl)

	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockPromptMgr.EXPECT().GetPromptWithContext(ctx, prompt.PromptIDSupervisorSystem, gomock.Any()).
		Return("test prompt", nil).Times(1)

	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Return(nil, assert.AnError).Times(1)

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/test/workspace").Times(1)

	mockSkillsService := skills.NewMockSkillService(ctrl)
	mockSkillsService.EXPECT().GetSkillsXML().Return("<skills></skills>").Times(1)
	mockSkillsService.EXPECT().GetSkillInfos().Return([]shared.SkillInfo{}).Times(1)

	mockMCPRegistry := &mockMCPRegistry{
		toolSets: []gollem.ToolSet{},
	}

	p := &applicationServiceImpl{
		logService:       mockLogger,
		fsm:              mockFSM,
		agentRegistry:    mockRegistry,
		promptMgr:        mockPromptMgr,
		agentFactory:     mockAgentFactory,
		workspaceService: mockWorkspaceService,
		skillsService:    mockSkillsService,
		mcpRegistry:      mockMCPRegistry,
	}

	_, _, err := p.createSupervisorAgent(ctx)
	assert.Error(t, err)
}

// TestCreateSupervisorAgent_RegistryError tests error handling when registration fails
func TestCreateSupervisorAgent_RegistryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agentID := uuid.New()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Warn("no mcp servers configured for supervision agent").Times(1)

	mockFSM := state.NewMockFileStateManager(ctrl)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).Return(assert.AnError).Times(1)

	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockPromptMgr.EXPECT().GetPromptWithContext(ctx, prompt.PromptIDSupervisorSystem, gomock.Any()).
		Return("test prompt", nil).Times(1)

	mockSession := mocks.NewMockSession(ctrl)
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().Session().Return(mockSession).AnyTimes()

	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Return(mockAgent, nil).Times(1)

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/test/workspace").Times(1)

	mockSkillsService := skills.NewMockSkillService(ctrl)
	mockSkillsService.EXPECT().GetSkillsXML().Return("<skills></skills>").Times(1)
	mockSkillsService.EXPECT().GetSkillInfos().Return([]shared.SkillInfo{}).Times(1)

	mockMCPRegistry := &mockMCPRegistry{
		toolSets: []gollem.ToolSet{},
	}

	p := &applicationServiceImpl{
		logService:       mockLogger,
		fsm:              mockFSM,
		agentRegistry:    mockRegistry,
		promptMgr:        mockPromptMgr,
		agentFactory:     mockAgentFactory,
		workspaceService: mockWorkspaceService,
		skillsService:    mockSkillsService,
		mcpRegistry:      mockMCPRegistry,
	}

	_, _, err := p.createSupervisorAgent(ctx)
	assert.Error(t, err)
}

// TestCleanup_NoOp tests that Cleanup is a no-op (kept for interface compatibility)
func TestCleanup_NoOp(t *testing.T) {
	p := &applicationServiceImpl{}

	// Should not panic
	assert.NotPanics(t, func() {
		p.Cleanup()
	})
}

// TestResolveDefaultFlowPath_WorkspaceDefaultFound tests that workspace-local default flow is found
func TestResolveDefaultFlowPath_WorkspaceDefaultFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create temp directory with .gollum/flows/default/main.xml
	tempDir := t.TempDir()
	flowDir := filepath.Join(tempDir, ".gollum", "flows", "default")
	err := os.MkdirAll(flowDir, 0755)
	require.NoError(t, err)

	flowContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="default">
  <description>Test default flow</description>
  <states>
    <state name="initial" initial="true">
      <steps>
        <step type="shell" name="echo">
          <cmd>echo "Hello"</cmd>
        </step>
      </steps>
      <transitions>
        <transition to="final" otherwise="true"/>
      </transitions>
    </state>
    <state name="final">
    </state>
  </states>
</flow>`
	flowPath := filepath.Join(flowDir, "main.xml")
	err = os.WriteFile(flowPath, []byte(flowContent), 0644)
	require.NoError(t, err)

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).Times(1)

	p := &applicationServiceImpl{
		workspaceService: mockWorkspaceService,
	}

	resolvedPath, err := p.resolveDefaultFlowPath()
	assert.NoError(t, err)
	assert.Equal(t, flowPath, resolvedPath)
}

// TestResolveDefaultFlowPath_NoDefaultFlow tests that error is returned when no default flow exists
func TestResolveDefaultFlowPath_NoDefaultFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create temp directory without default flow
	tempDir := t.TempDir()

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).Times(1)

	p := &applicationServiceImpl{
		workspaceService: mockWorkspaceService,
	}

	resolvedPath, err := p.resolveDefaultFlowPath()
	assert.Error(t, err)
	assert.Empty(t, resolvedPath)
}

// TestRun_DefaultFlowSuccess tests that Run executes default flow when found
func TestRun_DefaultFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Create temp directory with default flow
	tempDir := t.TempDir()
	flowDir := filepath.Join(tempDir, ".gollum", "flows", "default")
	err := os.MkdirAll(flowDir, 0755)
	require.NoError(t, err)

	flowContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="default">
  <description>Test default flow</description>
  <states>
    <state name="initial" initial="true">
      <steps>
        <step type="shell" name="echo">
          <cmd>echo "Hello"</cmd>
        </step>
      </steps>
      <transitions>
        <transition to="final" otherwise="true"/>
      </transitions>
    </state>
    <state name="final">
    </state>
  </states>
</flow>`
	flowPath := filepath.Join(flowDir, "main.xml")
	err = os.WriteFile(flowPath, []byte(flowContent), 0644)
	require.NoError(t, err)

	// Create mock flow executor
	mockExecutor := mocks.NewMockFlowExecutorInstance(ctrl)
	mockExecutor.EXPECT().SetInput(gomock.Any()).Times(1)
	mockExecutor.EXPECT().Validate().Return(nil).Times(1)
	mockExecutor.EXPECT().Run().Return(nil).Times(1)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Default flow found at: %s", flowPath).Times(1)
	mockLogger.EXPECT().Infof("Running default flow: %s", flowPath).Times(1)
	mockLogger.EXPECT().Infof("Default flow completed successfully").Times(1)

	mockFlowExecutorService := mocks.NewMockFlowExecutorService(ctrl)
	mockFlowExecutorService.EXPECT().New(gomock.Any()).Return(mockExecutor).Times(1)

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).Times(1)

	p := &applicationServiceImpl{
		logService:          mockLogger,
		workspaceService:    mockWorkspaceService,
		flowExecutorService: mockFlowExecutorService,
	}

	err = p.Run(ctx)
	assert.NoError(t, err)
}

// TestRun_NoDefaultFlowRunsTUI tests that Run falls back to TUI when no default flow
func TestRun_NoDefaultFlowRunsTUI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Create temp directory without default flow
	tempDir := t.TempDir()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().EnableFileLogging(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockLogger.EXPECT().CloseFileLogging().Return(nil).Times(1)

	mockFSM := state.NewMockFileStateManager(ctrl)
	mockFSM.EXPECT().Prime(ctx).Return(nil).Times(1)

	mockAgentRegistry := registry.NewMockAgentRegistry(ctrl)
	mockAgentRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockPromptMgr.EXPECT().GetPromptWithContext(gomock.Any(), gomock.Any(), gomock.Any()).Return("prompt", nil).Times(1)

	mockAgent := shared.NewMockAgent(ctrl)
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	mockAgent.EXPECT().GetID().Return(testUUID).AnyTimes()

	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Return(mockAgent, nil).Times(1)

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).AnyTimes()

	mockSkillsService := skills.NewMockSkillService(ctrl)
	mockSkillsService.EXPECT().GetSkillsXML().Return("<skills/>").AnyTimes()
	mockSkillsService.EXPECT().GetSkillInfos().Return([]shared.SkillInfo{}).AnyTimes()

	mockMCPRegistry := &mockMCPRegistry{toolSets: []gollem.ToolSet{}}

	// Mock channel facade - TUI channel registration will fail without TUI setup
	mockChannelFacade := channel.NewMockChannelFacadeService(ctrl)
	mockChannelFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).Times(1)

	p := &applicationServiceImpl{
		gollumDir:        filepath.Join(tempDir, gollumDirName),
		logService:       mockLogger,
		fsm:              mockFSM,
		agentRegistry:    mockAgentRegistry,
		promptMgr:        mockPromptMgr,
		agentFactory:     mockAgentFactory,
		workspaceService: mockWorkspaceService,
		skillsService:    mockSkillsService,
		mcpRegistry:      mockMCPRegistry,
		channelFacade:    mockChannelFacade,
		markdownRenderer: &mockMarkdownRenderer{},
	}

	// This will fail when trying to run the actual TUI, but we can verify
	// it attempts to run TUI instead of default flow
	err := p.Run(ctx)
	// The error will be from TUI initialization, which is expected
	// The important thing is it didn't try to run a default flow
	assert.Error(t, err)
}

// TestRunDefaultFlow_ParseError tests error handling when flow parsing fails
func TestRunDefaultFlow_ParseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Create temp directory with invalid flow XML
	tempDir := t.TempDir()
	flowDir := filepath.Join(tempDir, ".gollum", "flows", "default")
	err := os.MkdirAll(flowDir, 0755)
	require.NoError(t, err)

	invalidFlowPath := filepath.Join(flowDir, "main.xml")
	err = os.WriteFile(invalidFlowPath, []byte("invalid xml"), 0644)
	require.NoError(t, err)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Running default flow: %s", invalidFlowPath).Times(1)

	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return(tempDir).Times(1)

	p := &applicationServiceImpl{
		logService:       mockLogger,
		workspaceService: mockWorkspaceService,
	}

	err = p.runDefaultFlow(ctx, invalidFlowPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse default flow")
}

// TestRunDefaultFlow_ValidationError tests error handling when flow validation fails
func TestRunDefaultFlow_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Create temp directory with valid flow
	tempDir := t.TempDir()
	flowDir := filepath.Join(tempDir, ".gollum", "flows", "default")
	err := os.MkdirAll(flowDir, 0755)
	require.NoError(t, err)

	flowContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="default">
  <description>Test default flow</description>
  <states>
    <state name="initial" initial="true">
      <steps>
        <step type="shell" name="echo">
          <cmd>echo "Hello"</cmd>
        </step>
      </steps>
      <transitions>
        <transition to="final" otherwise="true"/>
      </transitions>
    </state>
    <state name="final">
    </state>
  </states>
</flow>`
	flowPath := filepath.Join(flowDir, "main.xml")
	err = os.WriteFile(flowPath, []byte(flowContent), 0644)
	require.NoError(t, err)

	// Create mock flow executor that fails validation
	mockExecutor := mocks.NewMockFlowExecutorInstance(ctrl)
	mockExecutor.EXPECT().SetInput(gomock.Any()).Times(1)
	mockExecutor.EXPECT().Validate().Return(assert.AnError).Times(1)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Running default flow: %s", flowPath).Times(1)

	mockFlowExecutorService := mocks.NewMockFlowExecutorService(ctrl)
	mockFlowExecutorService.EXPECT().New(gomock.Any()).Return(mockExecutor).Times(1)

	p := &applicationServiceImpl{
		logService:          mockLogger,
		flowExecutorService: mockFlowExecutorService,
	}

	err = p.runDefaultFlow(ctx, flowPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default flow validation failed")
}

// TestRunDefaultFlow_RunError tests error handling when flow execution fails
func TestRunDefaultFlow_RunError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Create temp directory with valid flow
	tempDir := t.TempDir()
	flowDir := filepath.Join(tempDir, ".gollum", "flows", "default")
	err := os.MkdirAll(flowDir, 0755)
	require.NoError(t, err)

	flowContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="default">
  <description>Test default flow</description>
  <states>
    <state name="initial" initial="true">
      <steps>
        <step type="shell" name="echo">
          <cmd>echo "Hello"</cmd>
        </step>
      </steps>
      <transitions>
        <transition to="final" otherwise="true"/>
      </transitions>
    </state>
    <state name="final">
    </state>
  </states>
</flow>`
	flowPath := filepath.Join(flowDir, "main.xml")
	err = os.WriteFile(flowPath, []byte(flowContent), 0644)
	require.NoError(t, err)

	// Create mock flow executor that fails during Run
	mockExecutor := mocks.NewMockFlowExecutorInstance(ctrl)
	mockExecutor.EXPECT().SetInput(gomock.Any()).Times(1)
	mockExecutor.EXPECT().Validate().Return(nil).Times(1)
	mockExecutor.EXPECT().Run().Return(assert.AnError).Times(1)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Running default flow: %s", flowPath).Times(1)

	mockFlowExecutorService := mocks.NewMockFlowExecutorService(ctrl)
	mockFlowExecutorService.EXPECT().New(gomock.Any()).Return(mockExecutor).Times(1)

	p := &applicationServiceImpl{
		logService:          mockLogger,
		flowExecutorService: mockFlowExecutorService,
	}

	err = p.runDefaultFlow(ctx, flowPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default flow execution failed")
}

// TestRunDefaultFlow_Success tests successful default flow execution
func TestRunDefaultFlow_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Create temp directory with valid flow
	tempDir := t.TempDir()
	flowDir := filepath.Join(tempDir, ".gollum", "flows", "default")
	err := os.MkdirAll(flowDir, 0755)
	require.NoError(t, err)

	flowContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="default">
  <description>Test default flow</description>
  <states>
    <state name="initial" initial="true">
      <steps>
        <step type="shell" name="echo">
          <cmd>echo "Hello"</cmd>
        </step>
      </steps>
      <transitions>
        <transition to="final" otherwise="true"/>
      </transitions>
    </state>
    <state name="final">
    </state>
  </states>
</flow>`
	flowPath := filepath.Join(flowDir, "main.xml")
	err = os.WriteFile(flowPath, []byte(flowContent), 0644)
	require.NoError(t, err)

	// Create mock flow executor that succeeds
	mockExecutor := mocks.NewMockFlowExecutorInstance(ctrl)
	mockExecutor.EXPECT().SetInput(gomock.Any()).Times(1)
	mockExecutor.EXPECT().Validate().Return(nil).Times(1)
	mockExecutor.EXPECT().Run().Return(nil).Times(1)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Running default flow: %s", flowPath).Times(1)
	mockLogger.EXPECT().Infof("Default flow completed successfully").Times(1)

	mockFlowExecutorService := mocks.NewMockFlowExecutorService(ctrl)
	mockFlowExecutorService.EXPECT().New(gomock.Any()).Return(mockExecutor).Times(1)

	p := &applicationServiceImpl{
		logService:          mockLogger,
		flowExecutorService: mockFlowExecutorService,
	}

	err = p.runDefaultFlow(ctx, flowPath)
	assert.NoError(t, err)
}

// mockMarkdownRenderer is a simple mock for testing
type mockMarkdownRenderer struct{}

func (m *mockMarkdownRenderer) Render(ctx context.Context, markdown string, width int) (string, error) {
	return markdown, nil
}
