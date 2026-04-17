package shared_test

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/testutil"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewTestInjector_BasicSetup(t *testing.T) {
	injector := testutil.NewTestInjector(t)

	assert.NotNil(t, injector)
	assert.IsType(t, do.New(), injector)
}

func TestNewTestInjector_WithConfigService(t *testing.T) {
	injector := testutil.NewTestInjector(t)

	configService := do.MustInvoke[config.ConfigService](injector)
	assert.NotNil(t, configService)
}

func TestNewTestInjector_WithCustomConfig(t *testing.T) {
	customLimits := &config.AgentLimitsConfig{
		MaxSubAgentsPerParent: 999,
		MaxTotalAgents:        999,
	}

	injector := testutil.NewTestInjector(t, func(cfg *testutil.TestInjectorConfig) {
		cfg.AgentLimits = customLimits
	})

	configService := do.MustInvoke[config.ConfigService](injector)
	limits := configService.GetAgentLimits()

	assert.Equal(t, 999, limits.MaxSubAgentsPerParent)
	assert.Equal(t, 999, limits.MaxTotalAgents)
}
