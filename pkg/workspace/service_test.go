package workspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test service implementation directly without full DI setup

func TestGetCurrentWorkspace_Empty(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "",
		history:        []string{},
		maxHistorySize: 10,
	}

	result := service.GetCurrentWorkspace()

	assert.Equal(t, "", result)
}

func TestGetCurrentWorkspace_WithValue(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "/test/path",
		history:        []string{},
		maxHistorySize: 10,
	}

	result := service.GetCurrentWorkspace()

	assert.Equal(t, "/test/path", result)
}

func TestGetWorkspaceHistory_Empty(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "/test/path",
		history:        []string{},
		maxHistorySize: 10,
	}

	result := service.GetWorkspaceHistory()

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestSetCurrentWorkspace_AddsToHistory(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "",
		history:        []string{},
		maxHistorySize: 10,
	}

	_ = service.setCurrentWorkspace("/path1")
	_ = service.setCurrentWorkspace("/path2")
	_ = service.setCurrentWorkspace("/path3")

	history := service.GetWorkspaceHistory()

	assert.Len(t, history, 3)
	assert.Equal(t, "/path3", history[0])  // Most recent first
	assert.Equal(t, "/path2", history[1])
	assert.Equal(t, "/path1", history[2])
}

func TestSetCurrentWorkspace_DuplicatePath(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "",
		history:        []string{},
		maxHistorySize: 10,
	}

	_ = service.setCurrentWorkspace("/path1")
	_ = service.setCurrentWorkspace("/path2")
	_ = service.setCurrentWorkspace("/path1")  // Duplicate

	history := service.GetWorkspaceHistory()

	assert.Len(t, history, 2)
	assert.Equal(t, "/path1", history[0])  // Moved to front
	assert.Equal(t, "/path2", history[1])
}

func TestSetCurrentWorkspace_TrimsHistory(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "",
		history:        []string{},
		maxHistorySize: 3,  // Small max for testing
	}

	// Add more items than maxHistorySize
	_ = service.setCurrentWorkspace("/path1")
	_ = service.setCurrentWorkspace("/path2")
	_ = service.setCurrentWorkspace("/path3")
	_ = service.setCurrentWorkspace("/path4")

	history := service.GetWorkspaceHistory()

	assert.Len(t, history, 3)  // Should be trimmed
	assert.Equal(t, "/path4", history[0])
	assert.Equal(t, "/path3", history[1])
	assert.Equal(t, "/path2", history[2])
}

func TestSetCurrentWorkspace_EmptyString(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "",
		history:        []string{},
		maxHistorySize: 10,
	}

	_ = service.setCurrentWorkspace("/path1")
	_ = service.setCurrentWorkspace("")  // Empty string should be ignored

	history := service.GetWorkspaceHistory()

	assert.Len(t, history, 1)
	assert.Equal(t, "/path1", history[0])
}

func TestGetWorkspaceHistory_ReturnsCopy(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "",
		history:        []string{},
		maxHistorySize: 10,
	}

	_ = service.setCurrentWorkspace("/path1")

	history1 := service.GetWorkspaceHistory()
	history2 := service.GetWorkspaceHistory()

	// Should return copies, not same slice
	assert.Equal(t, history1, history2)
	assert.NotSame(t, &history1[0], &history2[0])
}

func TestAddToHistoryLocked_MaintainsOrder(t *testing.T) {
	service := &serviceImpl{
		currentPath:    "",
		history:        []string{},
		maxHistorySize: 10,
	}

	// Add paths in order
	service.addToHistoryLocked("/path3")
	service.addToHistoryLocked("/path2")
	service.addToHistoryLocked("/path1")

	history := service.GetWorkspaceHistory()

	assert.Equal(t, "/path1", history[0])
	assert.Equal(t, "/path2", history[1])
	assert.Equal(t, "/path3", history[2])
}

func TestService_Constants(t *testing.T) {
	// Verify source name constant is set
	assert.Equal(t, "workspace_service", SourceName)
}
