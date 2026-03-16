package tools

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

// TestGrepTool_Run_MissingPattern tests validation when pattern is missing
func TestGrepTool_Run_MissingPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &grepToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"path": "/some/path",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false when pattern is missing")
	}

	if result["error"] != "pattern is required and must be a non-empty string" {
		t.Errorf("Expected specific error message, got: %v", result["error"])
	}
}

// TestGrepTool_Run_EmptyPattern tests validation when pattern is empty
func TestGrepTool_Run_EmptyPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &grepToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"pattern": "",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false when pattern is empty")
	}
}

// TestGrepTool_Run_InvalidOutputMode tests validation of output_mode parameter
func TestGrepTool_Run_InvalidOutputMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	tool := &grepToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"pattern":     "test",
		"output_mode": "invalid_mode",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false for invalid output_mode")
	}

	if result["error"] == nil {
		t.Error("Expected error message for invalid output_mode")
	}
}

// TestGrepTool_Run_InvalidRegex tests validation of regex pattern
func TestGrepTool_Run_InvalidRegex(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	tool := &grepToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"pattern": "[invalid",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false for invalid regex")
	}

	if result["error"] == nil {
		t.Error("Expected error message for invalid regex")
	}
}

// TestGrepTool_Spec tests the tool specification
func TestGrepTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &grepToolImpl{hookManager: mockHookManager}

	spec := tool.Spec()

	if spec.Name != shared.ToolNameGrep.String() {
		t.Errorf("Expected tool name '%s', got '%s'", shared.ToolNameGrep, spec.Name)
	}

	if spec.Description == "" {
		t.Error("Expected non-empty description")
	}

	// Check pattern parameter
	if patternParam, exists := spec.Parameters["pattern"]; !exists {
		t.Error("Missing 'pattern' parameter in spec")
	} else {
		if patternParam.Type != gollem.TypeString {
			t.Errorf("Expected 'pattern' parameter type to be String, got %v", patternParam.Type)
		}
	}

	// Check output_mode parameter
	if modeParam, exists := spec.Parameters["output_mode"]; !exists {
		t.Error("Missing 'output_mode' parameter in spec")
	} else {
		if modeParam.Type != gollem.TypeString {
			t.Errorf("Expected 'output_mode' parameter type to be String, got %v", modeParam.Type)
		}
	}

	// Check -i parameter (case insensitive)
	if iParam, exists := spec.Parameters["-i"]; !exists {
		t.Error("Missing '-i' parameter in spec")
	} else {
		if iParam.Type != gollem.TypeBoolean {
			t.Errorf("Expected '-i' parameter type to be Boolean, got %v", iParam.Type)
		}
	}

	// Check glob parameter
	if globParam, exists := spec.Parameters["glob"]; !exists {
		t.Error("Missing 'glob' parameter in spec")
	} else {
		if globParam.Type != gollem.TypeString {
			t.Errorf("Expected 'glob' parameter type to be String, got %v", globParam.Type)
		}
	}
}

// TestGrepToolProvider_CreateTool tests the provider's CreateTool method
func TestGrepToolProvider_CreateTool(t *testing.T) {
	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	provider := &grepToolProvider{logService: logService}
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tool := provider.CreateTool(testUUID)
	toolImpl := tool.(*grepToolImpl)

	if tool == nil {
		t.Fatal("Expected non-nil tool")
	}

	if toolImpl.logService == nil {
		t.Error("Expected tool to have logService")
	}

	if toolImpl.agentID != testUUID {
		t.Errorf("Expected agentID %v, got %v", testUUID, toolImpl.agentID)
	}
}

// TestNewGrepToolProvider tests the provider constructor
func TestNewGrepToolProvider(t *testing.T) {
	injector := setupTestInjector()

	provider, err := NewGrepToolProvider(injector)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}

	// Verify provider can create tool
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tool := provider.CreateTool(testUUID)

	if tool == nil {
		t.Error("Expected provider to create non-nil tool")
	}
}
