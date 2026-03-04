package tools

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInvokeSkillTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := &InvokeSkillTool{
		senderID: uuid.New(),
	}

	spec := tool.Spec()

	assert.Equal(t, shared.ToolNameInvokeSkill, spec.Name)
	assert.Contains(t, spec.Description, "skill")
	assert.Contains(t, spec.Parameters, "name")
	assert.Contains(t, spec.Parameters, "input")
	assert.Contains(t, spec.Parameters, "context_mode")
	assert.Contains(t, spec.Parameters, "model")

	// Verify required parameters
	assert.Equal(t, gollem.TypeString, spec.Parameters["name"].Type)
	assert.Equal(t, gollem.TypeString, spec.Parameters["input"].Type)
}

func TestInvokeSkillTool_MissingRequiredParameters(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		errContains string
	}{
		{
			name:        "missing name",
			args:        map[string]any{"input": "test task"},
			errContains: "name is required",
		},
		{
			name:        "empty name",
			args:        map[string]any{"name": "", "input": "test task"},
			errContains: "name is required",
		},
		{
			name:        "missing input",
			args:        map[string]any{"name": "test-skill"},
			errContains: "input is required",
		},
		{
			name:        "empty input",
			args:        map[string]any{"name": "test-skill", "input": ""},
			errContains: "input is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			senderID := uuid.New()
			mockLogger := mocks.NewMockLoggerService(ctrl)
			mockHookManager := mocks.NewMockHookManager(ctrl)

			// Allow any logger calls
			mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			// Set up hook manager to pass through
			mockHookManager.EXPECT().WithToolHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _ uuid.UUID, _ shared.ToolName, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
					return work()
				})

			tool := &InvokeSkillTool{
				logService:  mockLogger,
				senderID:    senderID,
				hookManager: mockHookManager,
				executionHelper: &testExecutionHelper{},
			}

			result, err := tool.Run(context.Background(), tt.args)
			require.NoError(t, err)
			assert.Contains(t, result["error"], tt.errContains)
		})
	}
}

func TestInvokeSkillTool_SkillNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	senderID := uuid.New()
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	mockSkillService := mocks.NewMockSkillService(ctrl)

	// Set up hook manager to pass through
	mockHookManager.EXPECT().WithToolHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ uuid.UUID, _ shared.ToolName, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
			return work()
		})

	mockSkillService.EXPECT().Get("nonexistent-skill").Return(nil, skills.ErrSkillNotFound("nonexistent-skill"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	tool := &InvokeSkillTool{
		logService:      mockLogger,
		skillService:    mockSkillService,
		senderID:        senderID,
		hookManager:     mockHookManager,
		executionHelper: &testExecutionHelper{},
	}

	args := map[string]any{
		"name":  "nonexistent-skill",
		"input": "test task",
	}

	result, err := tool.Run(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "not found")
}

func TestInvokeSkillTool_InvalidContextMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	senderID := uuid.New()
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	mockSkillService := mocks.NewMockSkillService(ctrl)

	// Set up hook manager to pass through
	mockHookManager.EXPECT().WithToolHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ uuid.UUID, _ shared.ToolName, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
			return work()
		})

	skill := &skills.Skill{
		Name:    "test-skill",
		Type:    skills.SkillTypeAgent,
		Content: "Test skill content",
	}
	mockSkillService.EXPECT().Get("test-skill").Return(skill, nil)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	tool := &InvokeSkillTool{
		logService:      mockLogger,
		skillService:    mockSkillService,
		senderID:        senderID,
		hookManager:     mockHookManager,
		executionHelper: &testExecutionHelper{},
	}

	args := map[string]any{
		"name":         "test-skill",
		"input":        "test task",
		"context_mode": "invalid-mode",
	}

	result, err := tool.Run(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "invalid context_mode")
}

func TestInvokeSkillTool_InvalidModel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	senderID := uuid.New()
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	mockSkillService := mocks.NewMockSkillService(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)

	// Set up hook manager to pass through
	mockHookManager.EXPECT().WithToolHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ uuid.UUID, _ shared.ToolName, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
			return work()
		})

	skill := &skills.Skill{
		Name:    "test-skill",
		Type:    skills.SkillTypeAgent,
		Content: "Test skill content",
	}
	mockSkillService.EXPECT().Get("test-skill").Return(skill, nil)
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	tool := &InvokeSkillTool{
		logService:      mockLogger,
		skillService:    mockSkillService,
		senderID:        senderID,
		hookManager:     mockHookManager,
		executionHelper: &testExecutionHelper{},
		registry:        mockRegistry,
	}

	args := map[string]any{
		"name":  "test-skill",
		"input": "test task",
		"model": "invalid-model",
	}

	result, err := tool.Run(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "invalid model")
}

func TestInvokeSkillTool_SkillWithNoContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	senderID := uuid.New()
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	mockSkillService := mocks.NewMockSkillService(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)

	// Set up hook manager to pass through
	mockHookManager.EXPECT().WithToolHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ uuid.UUID, _ shared.ToolName, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
			return work()
		})

	skill := &skills.Skill{
		Name:    "empty-skill",
		Type:    skills.SkillTypeAgent,
		Content: "", // Empty content
	}
	mockSkillService.EXPECT().Get("empty-skill").Return(skill, nil)
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)

	// BeforeSkillInvoked hook is triggered before content check
	mockHookManager.EXPECT().TriggerHooks(gomock.Any(), gomock.Eq(hooks.BeforeSkillInvoked), gomock.Any()).Return(hooks.HookResult{})

	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	tool := &InvokeSkillTool{
		logService:      mockLogger,
		skillService:    mockSkillService,
		senderID:        senderID,
		hookManager:     mockHookManager,
		executionHelper: &testExecutionHelper{},
		registry:        mockRegistry,
	}

	args := map[string]any{
		"name":  "empty-skill",
		"input": "test task",
	}

	result, err := tool.Run(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "no content defined")
}

func TestContextMode_IsValid(t *testing.T) {
	tests := []struct {
		mode     ContextMode
		expected bool
	}{
		{ContextModeInherited, true},
		{ContextModeIsolated, true},
		{ContextMode("invalid"), false},
		{ContextMode(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mode.IsValid())
		})
	}
}

// testExecutionHelper is a minimal implementation of AgentExecutionHelper for testing
type testExecutionHelper struct{}

func (m *testExecutionHelper) ExecuteSynchronously(ctx context.Context, agent shared.Agent, prompt string) (map[string]any, error) {
	return map[string]any{"status": "success", "output": "test output"}, nil
}

func (m *testExecutionHelper) ExecuteInBackground(ctx context.Context, agent shared.Agent, prompt string) {}

func (m *testExecutionHelper) SuccessResponseSync(agentID uuid.UUID, response string) map[string]any {
	return map[string]any{"success": true, "agent_id": agentID.String(), "response": response}
}

func (m *testExecutionHelper) SuccessResponseAsync(agentID uuid.UUID, role, description string) map[string]any {
	return map[string]any{"agent_id": agentID.String(), "status": "running", "role": role, "description": description}
}

func (m *testExecutionHelper) SuccessResponseResumeAsync(agentID uuid.UUID, role, description string) map[string]any {
	return map[string]any{"agent_id": agentID.String(), "status": "running", "role": role, "description": description}
}

func (m *testExecutionHelper) ErrorResponse(msg string) map[string]any {
	return map[string]any{"error": msg}
}

// Test that InvokeSkillTool implements gollem.Tool interface
func TestInvokeSkillTool_ImplementsGollemTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := &InvokeSkillTool{
		senderID: uuid.New(),
	}

	// This will fail to compile if InvokeSkillTool doesn't implement gollem.Tool
	var _ gollem.Tool = tool
}

// Test skill invocation with skill hooks
func TestInvokeSkillTool_TriggersSkillHooks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	senderID := uuid.New()
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	mockSkillService := mocks.NewMockSkillService(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockAgent := mocks.NewMockAgent(ctrl)
	mockExecutionHelper := mocks.NewMockAgentExecutionHelper(ctrl)

	skill := &skills.Skill{
		Name:    "test-skill",
		Type:    skills.SkillTypeAgent,
		Content: "Test skill content",
	}

	// Set up expectations
	mockSkillService.EXPECT().Get("test-skill").Return(skill, nil)

	// GetAgent is called twice: once for LLM provider, once for context inheritance
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false).Times(2)

	// Tool hooks pass through
	mockHookManager.EXPECT().WithToolHooks(gomock.Any(), gomock.Any(), gomock.Any(), shared.ToolNameInvokeSkill, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ uuid.UUID, _ shared.ToolName, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
			return work()
		})

	// BeforeSkillInvoked hook
	mockHookManager.EXPECT().TriggerHooks(gomock.Any(), gomock.Eq(hooks.BeforeSkillInvoked), gomock.Any()).Return(hooks.HookResult{})

	// AfterSkillInvoked hook
	mockHookManager.EXPECT().TriggerHooks(gomock.Any(), gomock.Eq(hooks.AfterSkillInvoked), gomock.Any()).Return(hooks.HookResult{})

	// Logger calls - use AnyTimes() for flexible argument matching
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()

	// Agent factory creates agent
	mockFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).Return(mockAgent, nil)
	mockAgent.EXPECT().GetID().Return(uuid.New())

	// Registry registers agent
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil)

	// Execution helper executes synchronously
	mockExecutionHelper.EXPECT().ExecuteSynchronously(gomock.Any(), gomock.Any(), "test task").
		Return(map[string]any{"status": "success"}, nil)

	tool := &InvokeSkillTool{
		logService:      mockLogger,
		agentFactory:    mockFactory,
		registry:        mockRegistry,
		skillService:    mockSkillService,
		senderID:        senderID,
		hookManager:     mockHookManager,
		executionHelper: mockExecutionHelper,
	}

	args := map[string]any{
		"name":  "test-skill",
		"input": "test task",
	}

	result, err := tool.Run(context.Background(), args)
	require.NoError(t, err)
	assert.Equal(t, "success", result["status"])
	assert.Equal(t, "test-skill", result["skill_name"])
}
