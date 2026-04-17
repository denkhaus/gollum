package testutil

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestInjector_BasicSetup(t *testing.T) {
	injector := NewTestInjector(t)

	require.NotNil(t, injector)
	assert.IsType(t, do.New(), injector)
}

func TestNewTestInjector_AllServicesProvided(t *testing.T) {
	injector := NewTestInjector(t)

	// Verify all common services are available
	configService := do.MustInvoke[config.ConfigService](injector)
	assert.NotNil(t, configService)

	loggerService := do.MustInvoke[logger.LoggerService](injector)
	assert.NotNil(t, loggerService)

	eventBus := do.MustInvoke[events.Bus](injector)
	assert.NotNil(t, eventBus)

	promptManager := do.MustInvoke[manager.PromptManager](injector)
	assert.NotNil(t, promptManager)

	workspaceService := do.MustInvoke[workspace.Service](injector)
	assert.NotNil(t, workspaceService)

	skillService := do.MustInvoke[skills.SkillService](injector)
	assert.NotNil(t, skillService)
}

func TestNewTestInjector_WithCustomConfig(t *testing.T) {
	customLimits := &config.AgentLimitsConfig{
		MaxSubAgentsPerParent: 42,
		MaxTotalAgents:        100,
	}

	injector := NewTestInjector(t, func(cfg *TestInjectorConfig) {
		cfg.AgentLimits = customLimits
	})

	configService := do.MustInvoke[config.ConfigService](injector)
	limits := configService.GetAgentLimits()

	assert.Equal(t, 42, limits.MaxSubAgentsPerParent)
	assert.Equal(t, 100, limits.MaxTotalAgents)
}

func TestNewTestInjector_DefaultConfig(t *testing.T) {
	injector := NewTestInjector(t)

	configService := do.MustInvoke[config.ConfigService](injector)
	limits := configService.GetAgentLimits()

	// Verify default values from DefaultTestInjectorConfig
	assert.Equal(t, 3, limits.MaxSubAgentsPerParent)
	assert.Equal(t, 50, limits.MaxTotalAgents)
}
