// Package app integration tests for startup context.
package app

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/startup"
	do "github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

func TestStartupContext_FlowExecutionStoresContext(t *testing.T) {
	// This test requires a full DI setup and is optional
	// For now, we'll test the integration manually

	// TODO: Create a test startup flow and verify:
	// 1. Flow executes on startup
	// 2. Context is stored in StartupContextService
	// 3. PromptManager includes context in system prompt

	t.Skip("Integration test - requires full DI container setup")
}

func TestStartupContext_NoFlow_NoContextInjection(t *testing.T) {
	// Verify that when no startup flow exists,
	// HasContent() returns false

	injector := do.New()

	// Register startup service
	do.Provide(injector, startup.NewStartupContextService)

	startupSvc := do.MustInvoke[startup.StartupContextService](injector)

	// Initially, no context should be set
	assert.False(t, startupSvc.HasContent())
	assert.Empty(t, startupSvc.GetContextText())
}
