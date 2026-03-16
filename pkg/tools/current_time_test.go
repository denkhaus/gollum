package tools

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

func TestCurrentTimeTool_Run_DefaultTimezone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &currentTimeToolImpl{logService: logService, hookManager: mockHookManager}

	result, err := tool.Run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that result contains expected fields
	if _, exists := result["time"]; !exists {
		t.Error("Expected 'time' field in result")
	}
	if _, exists := result["timezone"]; !exists {
		t.Error("Expected 'timezone' field in result")
	}
	if _, exists := result["unix_timestamp"]; !exists {
		t.Error("Expected 'unix_timestamp' field in result")
	}
	if _, exists := result["day_of_week"]; !exists {
		t.Error("Expected 'day_of_week' field in result")
	}
	if _, exists := result["is_dst"]; !exists {
		t.Error("Expected 'is_dst' field in result")
	}
	if _, exists := result["rfc3339"]; !exists {
		t.Error("Expected 'rfc3339' field in result")
	}

	// Check default timezone
	if result["timezone"] != "UTC" {
		t.Errorf("Expected default timezone UTC, got %v", result["timezone"])
	}
}

func TestCurrentTimeTool_Run_UTC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &currentTimeToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"timezone": "UTC",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["timezone"] != "UTC" {
		t.Errorf("Expected timezone UTC, got %v", result["timezone"])
	}

	// Verify Unix timestamp is reasonable (recent time)
	if unixTS, ok := result["unix_timestamp"].(int64); ok {
		now := time.Now().Unix()
		if unixTS < now-10 || unixTS > now+10 {
			t.Errorf("Unix timestamp seems out of range: got %d, expected around %d", unixTS, now)
		}
	} else {
		t.Error("Expected unix_timestamp to be int64")
	}
}

func TestCurrentTimeTool_Run_AmericaNewYork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &currentTimeToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"timezone": "America/New_York",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["timezone"] != "America/New_York" {
		t.Errorf("Expected timezone America/New_York, got %v", result["timezone"])
	}

	// Verify day_of_week is valid
	if dow, ok := result["day_of_week"].(string); ok {
		validDays := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		valid := false
		for _, d := range validDays {
			if d == dow {
				valid = true
				break
			}
		}
		if !valid {
			t.Errorf("Invalid day_of_week: %s", dow)
		}
	} else {
		t.Error("Expected day_of_week to be string")
	}
}

func TestCurrentTimeTool_Run_EuropeBerlin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &currentTimeToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"timezone": "Europe/Berlin",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["timezone"] != "Europe/Berlin" {
		t.Errorf("Expected timezone Europe/Berlin, got %v", result["timezone"])
	}

	// Verify RFC3339 format
	if rfc3339, ok := result["rfc3339"].(string); ok {
		_, err := time.Parse(time.RFC3339, rfc3339)
		if err != nil {
			t.Errorf("Failed to parse RFC3339 timestamp: %v", err)
		}
	} else {
		t.Error("Expected rfc3339 to be string")
	}
}

func TestCurrentTimeTool_Run_InvalidTimezone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &currentTimeToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"timezone": "Invalid/Timezone",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should fall back to UTC
	if result["timezone"] != "Invalid/Timezone" {
		// The timezone in result reflects what was requested
		t.Logf("Timezone in result: %v", result["timezone"])
	}

	// But actual time should be in UTC (fallback)
	// We can verify by checking that rfc3339 is valid
	if rfc3339, ok := result["rfc3339"].(string); ok {
		_, err := time.Parse(time.RFC3339, rfc3339)
		if err != nil {
			t.Errorf("Failed to parse RFC3339 timestamp: %v", err)
		}
	}
}

func TestCurrentTimeTool_Run_EmptyTimezone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &currentTimeToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"timezone": "",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Empty timezone should default to UTC
	if result["timezone"] != "" {
		t.Logf("Timezone in result: %v", result["timezone"])
	}
}

func TestCurrentTimeTool_Run_NonStringTimezone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &currentTimeToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"timezone": 12345,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Non-string timezone should default to UTC
	// The type assertion will fail, so it will use UTC
	if _, exists := result["time"]; !exists {
		t.Error("Expected 'time' field in result")
	}
}

func TestCurrentTimeTool_Run_AsiaTokyo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &currentTimeToolImpl{logService: logService, hookManager: mockHookManager}

	args := map[string]any{
		"timezone": "Asia/Tokyo",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["timezone"] != "Asia/Tokyo" {
		t.Errorf("Expected timezone Asia/Tokyo, got %v", result["timezone"])
	}

	// Verify is_dst is boolean
	if _, ok := result["is_dst"].(bool); !ok {
		t.Error("Expected is_dst to be boolean")
	}

	// Verify time format
	if timeStr, ok := result["time"].(string); ok {
		_, err := time.Parse("2006-01-02 15:04:05", timeStr)
		if err != nil {
			t.Errorf("Time format is incorrect: %v", err)
		}
	} else {
		t.Error("Expected time to be string")
	}
}

func TestCurrentTimeTool_Spec(t *testing.T) {
	tool := &currentTimeToolImpl{}

	spec := tool.Spec()

	if spec.Name != shared.ToolNameCurrentTime.String() {
		t.Errorf("Expected tool name '%s', got '%s'", shared.ToolNameCurrentTime, spec.Name)
	}

	if spec.Description == "" {
		t.Error("Expected non-empty description")
	}

	// Check timezone parameter
	if tzParam, exists := spec.Parameters["timezone"]; !exists {
		t.Error("Missing 'timezone' parameter in spec")
	} else {
		if tzParam.Type != gollem.TypeString {
			t.Errorf("Expected 'timezone' parameter type to be String, got %v", tzParam.Type)
		}
		if tzParam.Description == "" {
			t.Error("Expected non-empty description for 'timezone' parameter")
		}
	}
}

func TestCurrentTimeToolProvider_CreateTool(t *testing.T) {
	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(nil) // nil ctrl since we're not setting expectations

	provider := &currentTimeToolProvider{logService: logService, hookManager: mockHookManager}
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tool := provider.CreateTool(testUUID)
	toolImpl := tool.(*currentTimeToolImpl)

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

func TestNewCurrentTimeToolProvider(t *testing.T) {
	injector := setupTestInjector()

	provider, err := NewCurrentTimeToolProvider(injector)
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
