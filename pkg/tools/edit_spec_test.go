package tools

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestEditToolSpec verifies the tool specification
func TestEditToolSpec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFSM := state.NewMockFileStateManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)

	// Create mock agent
	agentID := uuid.New()
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &editToolImpl{
		fsm:         mockFSM,
		hookManager: mockHookManager,
		agent:       mockAgent,
	}

	spec := tool.Spec()

	assert.Equal(t, "edit", spec.Name)
	assert.Contains(t, spec.Description, "exact string replacements")
	assert.Contains(t, spec.Description, "read first")

	// Check required parameters

	// Check all parameters exist
	require.Contains(t, spec.Parameters, "file_path")
	require.Contains(t, spec.Parameters, "old_string")
	require.Contains(t, spec.Parameters, "new_string")
	require.Contains(t, spec.Parameters, "replace_all")

	// replace_all should not be required
	param := spec.Parameters["replace_all"]
	assert.NotNil(t, param)
}

// TestEditToolSpecIsConstant verifies that calling Spec() multiple times returns consistent results
func TestEditToolSpecIsConstant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFSM := state.NewMockFileStateManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)

	// Create mock agent
	agentID := uuid.New()
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &editToolImpl{
		fsm:         mockFSM,
		hookManager: mockHookManager,
		agent:       mockAgent,
	}

	spec1 := tool.Spec()
	spec2 := tool.Spec()

	assert.Equal(t, spec1.Name, spec2.Name)
	assert.Equal(t, spec1.Description, spec2.Description)
}

// TestEditToolProvider tests the provider
func TestEditToolProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := state.NewMockFileStateManager(ctrl)

	provider := &editToolProvider{
		logService: logService,
		fsm:        mockFSM,
	}

	// Create mock agent
	agentID := uuid.New()
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := provider.CreateTool(mockAgent)
	toolImpl := tool.(*editToolImpl)

	require.NotNil(t, tool)
	assert.Equal(t, agentID, toolImpl.agent.GetID())
	assert.Equal(t, logService, toolImpl.logService)
	assert.Equal(t, mockFSM, toolImpl.fsm)
}
