package tools

import (
	"context"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

// setupTestInjector creates an injector with all required services for testing
func setupTestInjector() do.Injector {
	ctrl := gomock.NewController(nil)

	injector := do.New()

	// Register config service
	do.Provide(injector, config.NewService)

	// Register logger service
	do.Provide(injector, logger.NewService)

	// Register mock event bus (required by registry)
	mockEventBus := events.NewMockBus(ctrl)
	mockEventBus.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("test-subscription-id", nil).AnyTimes()
	do.ProvideValue[events.Bus](injector, mockEventBus)

	// Register mock prompt manager (required by registry)
	mockPromptManager := manager.NewMockPromptManager(ctrl)
	do.ProvideValue[manager.PromptManager](injector, mockPromptManager)

	// Register mock workspace service (required by registry)
	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/test/workspace").AnyTimes()
	do.ProvideValue[workspace.Service](injector, mockWorkspaceService)

	// Register mock skill service (required by registry)
	mockSkillService := skills.NewMockSkillService(ctrl)
	mockSkillService.EXPECT().GetSkillsXML().Return("").AnyTimes()
	mockSkillService.EXPECT().GetSkillInfos().Return([]shared.SkillInfo{}).AnyTimes()
	do.ProvideValue[skills.SkillService](injector, mockSkillService)

	// Register HookManager (needed by tool providers)
	do.Provide(injector, hooks.NewHookManager)

	// Register ToolRegistry (needed by SpawnAgentTool)
	do.Provide(injector, NewToolRegistry)

	// Register file state manager (needed by BashTool)
	do.Provide(injector, state.NewFileStateManager)

	// Register agent registry (needed for execution helper)
	do.Provide(injector, registry.NewAgentRegistry)

	// Register agent execution helper
	do.Provide(injector, NewAgentExecutionHelper)

	// Note: Tool providers (BashToolProvider, EditToolProvider, etc.) are NOT registered here
	// because tests create tool instances directly with their own mock dependencies.
	// This allows tests to have full control over tool behavior.

	return injector
}

// setupMockHookManagerPassThrough configures a mock HookManager to pass through all calls
// This is useful for tests that don't need to verify hook behavior
func setupMockHookManagerPassThrough(mockHookManager *hooks.MockHookManager) {
	// WithToolHooks - pass through to work function
	mockHookManager.EXPECT().WithToolHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ shared.SessionContext, _ shared.ToolName, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
			return work()
		}).AnyTimes()

	// WithFileReadHooks - pass through to work function
	mockHookManager.EXPECT().WithFileReadHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ shared.SessionContext, _ string, work func() (string, error)) (string, error) {
			return work()
		}).AnyTimes()

	// WithFileWriteHooks - pass through to work function
	mockHookManager.EXPECT().WithFileWriteHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ shared.SessionContext, _ string, content string, work func(string) error) error {
			return work(content)
		}).AnyTimes()

	// WithAgentHooks - pass through to work function
	mockHookManager.EXPECT().WithAgentHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ shared.SessionContext, _ hooks.HookPoint, work func() error) error {
			if work != nil {
				return work()
			}
			return nil
		}).AnyTimes()

	// WithSessionHooks - pass through to work function
	mockHookManager.EXPECT().WithSessionHooks(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ shared.SessionContext, work func() error) error {
			return work()
		}).AnyTimes()

	// WithFileHooks - pass through to work function
	mockHookManager.EXPECT().WithFileHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ shared.SessionContext, _ hooks.HookPoint, _ string, work func() error) error {
			return work()
		}).AnyTimes()

	// WithLLMHooks - pass through to work function
	mockHookManager.EXPECT().WithLLMHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ shared.SessionContext, _ string, input string, work func(string) (string, error)) (string, error) {
			return work(input)
		}).AnyTimes()

	// UnregisterHook - return false (hook not found in mock)
	mockHookManager.EXPECT().UnregisterHook(gomock.Any()).Return(false).AnyTimes()
}

// setupMockExecutionHelperWithDefaults configures a mock AgentExecutionHelper with default behavior
// for response helper methods ( ErrorResponse, SuccessResponseAsync, etc.)
// This should be called after creating a mock to set up the utility method expectations.
func setupMockExecutionHelperWithDefaults(mockExecHelper *MockAgentExecutionHelper) {
	// ErrorResponse - simple utility function
	mockExecHelper.EXPECT().ErrorResponse(gomock.Any()).DoAndReturn(func(errMsg string) map[string]any {
		return map[string]any{
			"success": false,
			"error":   errMsg,
		}
	}).AnyTimes()

	// SuccessResponseAsync - simple utility function
	mockExecHelper.EXPECT().SuccessResponseAsync(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(agentID uuid.UUID, role, description string) map[string]any {
			return map[string]any{
				"success":     true,
				"agent_id":    agentID.String(),
				"role":        role,
				"description": description,
				"status":      "running",
				"message":     "Agent started in background. Use AgentOutputTool to retrieve results.",
			}
		}).AnyTimes()

	// SuccessResponseResumeAsync - simple utility function
	mockExecHelper.EXPECT().SuccessResponseResumeAsync(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(agentID uuid.UUID, role, description string) map[string]any {
			return map[string]any{
				"success":     true,
				"agent_id":    agentID.String(),
				"role":        role,
				"description": description,
				"status":      "running",
				"message":     "Agent resumed in background. Use AgentOutputTool to retrieve results.",
			}
		}).AnyTimes()

	// SuccessResponseSync - simple utility function
	mockExecHelper.EXPECT().SuccessResponseSync(gomock.Any(), gomock.Any()).DoAndReturn(
		func(agentID uuid.UUID, response string) map[string]any {
			return map[string]any{
				"success":  true,
				"agent_id": agentID.String(),
				"response": response,
				"status":   "completed",
				"message":  "Agent completed successfully",
			}
		}).AnyTimes()
}
