package tools

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

// setupTestInjector creates an injector with all required services for testing
func setupTestInjector() do.Injector {
	injector := do.New()

	// Register config service
	do.Provide(injector, config.NewService)

	// Register logger service
	do.Provide(injector, logger.NewService)

	// Register agent registry (needed for execution helper)
	do.Provide(injector, registry.NewAgentRegistry)

	// Register agent execution helper
	do.Provide(injector, NewAgentExecutionHelper)

	// Register tool providers for testing (only those without complex dependencies)
	do.Provide(injector, NewBashToolProvider)
	do.Provide(injector, NewCurrentTimeToolProvider)
	do.Provide(injector, NewGlobToolProvider)
	do.Provide(injector, NewGrepToolProvider)

	return injector
}

// setupMockExecutionHelperWithDefaults configures a mock AgentExecutionHelper with default behavior
// for response helper methods ( ErrorResponse, SuccessResponseAsync, etc.)
// This should be called after creating a mock to set up the utility method expectations.
func setupMockExecutionHelperWithDefaults(mockExecHelper *mocks.MockAgentExecutionHelper) {
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
