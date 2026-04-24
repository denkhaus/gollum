package di

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/startup"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainer_RegisterServices_Success(t *testing.T) {
	ctx := context.Background()
	container := NewContainer()
	injector := container.RegisterServices(ctx)

	// Should be able to invoke basic services without circular dependency
	cfg := do.MustInvoke[config.ConfigService](injector)
	assert.NotNil(t, cfg)

	log := do.MustInvoke[logger.LoggerService](injector)
	assert.NotNil(t, log)
}

func TestContainer_ProvideStartupContextService(t *testing.T) {
	ctx := context.Background()
	container := NewContainer()
	injector := container.RegisterServices(ctx)
	defer container.Shutdown()

	service, err := do.Invoke[startup.StartupContextService](injector)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Verify it implements the interface by calling a method
	assert.False(t, service.HasContent(), "Initial service should have no content")

	// Test SetContext works
	service.SetContext(map[string]any{"test": "value"})
	assert.True(t, service.HasContent(), "Service should have content after SetContext")

	// Test GetContextText works
	text := service.GetContextText()
	assert.Contains(t, text, "test", "Context text should contain our test key")

	// Test Clear works
	service.Clear()
	assert.False(t, service.HasContent(), "Service should have no content after Clear")
}
