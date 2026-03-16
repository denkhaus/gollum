package tools

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
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

	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	tool := &editToolImpl{
		fsm:         mockFSM,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
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

	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	tool := &editToolImpl{
		fsm:         mockFSM,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
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
	mockFSM := mocks.NewMockFileStateManager(ctrl)

	provider := &editToolProvider{
		logService: logService,
		fsm:        mockFSM,
	}

	agentID := uuid.New()
	tool := provider.CreateTool(agentID)
	toolImpl := tool.(*editToolImpl)

	require.NotNil(t, tool)
	assert.Equal(t, agentID, toolImpl.agentID)
	assert.Equal(t, logService, toolImpl.logService)
	assert.Equal(t, mockFSM, toolImpl.fsm)
}
