// Package startup tests for startup context service.
package startup

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStartupContextService(t *testing.T) {
	service := NewStartupContextService()
	require.NotNil(t, service)
}

func TestStartupContextService_HasContent_Empty(t *testing.T) {
	service := NewStartupContextService()

	assert.False(t, service.HasContent())
	assert.Empty(t, service.GetContextText())
}

func TestStartupContextService_SetAndGet_Context(t *testing.T) {
	service := NewStartupContextService()

	outputs := map[string]any{
		"context": "Test context text",
	}
	service.SetContext(outputs)

	assert.True(t, service.HasContent())
	assert.Equal(t, "Test context text", service.GetContextText())
}

func TestStartupContextService_SetAndGet_Text(t *testing.T) {
	service := NewStartupContextService()

	outputs := map[string]any{
		"text": "Alternative text field",
	}
	service.SetContext(outputs)

	assert.True(t, service.HasContent())
	assert.Equal(t, "Alternative text field", service.GetContextText())
}

func TestStartupContextService_SetAndGet_MultipleFields(t *testing.T) {
	service := NewStartupContextService()

	outputs := map[string]any{
		"project": "E-commerce",
		"focus":   "Security",
		"stage":   "production",
	}
	service.SetContext(outputs)

	assert.True(t, service.HasContent())
	context := service.GetContextText()
	assert.Contains(t, context, "project: E-commerce")
	assert.Contains(t, context, "focus: Security")
	assert.Contains(t, context, "stage: production")
}

func TestStartupContextService_Clear(t *testing.T) {
	service := NewStartupContextService()

	service.SetContext(map[string]any{"test": "value"})
	assert.True(t, service.HasContent())

	service.Clear()
	assert.False(t, service.HasContent())
	assert.Empty(t, service.GetContextText())
}

func TestStartupContextService_ReplaceContext(t *testing.T) {
	service := NewStartupContextService()

	// Set initial context
	service.SetContext(map[string]any{"old": "value"})
	assert.Contains(t, service.GetContextText(), "old: value")

	// Replace with new context
	service.SetContext(map[string]any{"new": "value2"})
	assert.Contains(t, service.GetContextText(), "new: value2")
	assert.NotContains(t, service.GetContextText(), "old")
}

func TestStartupContextService_HasContent_EmptyString(t *testing.T) {
	service := NewStartupContextService()

	// Empty string should not count as content
	service.SetContext(map[string]any{"context": ""})
	assert.False(t, service.HasContent())
}

func TestStartupContextService_HasContent_WithNilValues(t *testing.T) {
	service := NewStartupContextService()

	// Map with nil values still has content
	service.SetContext(map[string]any{"key": nil})
	assert.True(t, service.HasContent())
}

func TestStartupContextService_ConcurrentAccess(t *testing.T) {
	service := NewStartupContextService()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			service.SetContext(map[string]any{"index": idx})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.GetContextText()
			_ = service.HasContent()
		}()
	}

	wg.Wait()
	// If we got here without deadlock/race, test passes
}
