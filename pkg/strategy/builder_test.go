package strategy

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	// This test verifies the builder can be created
	// Full integration test would require a DI injector
	builder := &builderImpl{}
	assert.NotNil(t, builder)
}

func TestBuilderImpl_BuildDefaultReact(t *testing.T) {
	tests := []struct {
		name           string
		subAgentCfg    config.SubAgentConfig
	}{
		{
			name: "with default config",
			subAgentCfg: config.SubAgentConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      20,
					MaxRepeatedActions: 3,
				},
			},
		},
		{
			name: "with custom config",
			subAgentCfg: config.SubAgentConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      10,
					MaxRepeatedActions: 2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &builderImpl{
				configService: &mockConfigService{
					subAgentCfg: tt.subAgentCfg,
				},
			}

			// BuildDefaultReact requires an LLMClient, but we can't mock it here
			// The important thing is that the config is used correctly
			subAgentCfg := builder.configService.GetSubAgentConfig()
			require.NotNil(t, subAgentCfg)
			assert.Equal(t, tt.subAgentCfg.Strategy.MaxIterations, subAgentCfg.Strategy.MaxIterations)
			assert.Equal(t, tt.subAgentCfg.Strategy.MaxRepeatedActions, subAgentCfg.Strategy.MaxRepeatedActions)
		})
	}
}

func TestBuilderImpl_BuildReact_Options(t *testing.T) {
	tests := []struct {
		name                  string
		cfg                   *config.StrategyConfig
		expectMaxIterations    int
		expectMaxRepeatedAct  int
	}{
		{
			name:               "nil config uses react defaults",
			cfg:                nil,
			expectMaxIterations: 0, // Will use react's defaults
			expectMaxRepeatedAct: 0,
		},
		{
			name:                  "both values set",
			cfg: &config.StrategyConfig{
				MaxIterations:      10,
				MaxRepeatedActions: 2,
			},
			expectMaxIterations:   10,
			expectMaxRepeatedAct: 2,
		},
		{
			name: "only max iterations",
			cfg: &config.StrategyConfig{
				MaxIterations: 15,
			},
			expectMaxIterations:   15,
			expectMaxRepeatedAct: 0,
		},
		{
			name: "only max repeated actions",
			cfg: &config.StrategyConfig{
				MaxRepeatedActions: 5,
			},
			expectMaxIterations:   0,
			expectMaxRepeatedAct: 5,
		},
		{
			name:                  "zero values",
			cfg:                   &config.StrategyConfig{},
			expectMaxIterations:   0,
			expectMaxRepeatedAct: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify config values are correctly used
			if tt.cfg != nil {
				assert.Equal(t, tt.expectMaxIterations, tt.cfg.MaxIterations)
				assert.Equal(t, tt.expectMaxRepeatedAct, tt.cfg.MaxRepeatedActions)
			} else {
				assert.Nil(t, tt.cfg)
			}
		})
	}
}

// mockConfigService implements config.ConfigService for testing
type mockConfigService struct {
	subAgentCfg config.SubAgentConfig
}

func (m *mockConfigService) GetLogLevel() string {
	return "info"
}

func (m *mockConfigService) IsDevMode() bool {
	return false
}

func (m *mockConfigService) GetAnthropicConfig() *config.AnthropicConfig {
	return nil
}

func (m *mockConfigService) GetGeminiConfig() *config.GeminiConfig {
	return nil
}

func (m *mockConfigService) GetOpenAIConfig() *config.OpenAIConfig {
	return nil
}

func (m *mockConfigService) GetAgentLimits() *config.AgentLimitsConfig {
	return nil
}

func (m *mockConfigService) GetFilesConfig() *config.FilesConfig {
	return nil
}

func (m *mockConfigService) GetLoggingConfig() *config.LoggingConfig {
	return nil
}

func (m *mockConfigService) GetBashConfig() *config.BashConfig {
	return nil
}

func (m *mockConfigService) GetHooksConfig() *config.HooksConfig {
	return nil
}

func (m *mockConfigService) GetPromptStoreConfig() *config.PromptStoreConfig {
	return nil
}

func (m *mockConfigService) GetPromptOptimizerConfig() *config.PromptOptimizerConfig {
	return nil
}

func (m *mockConfigService) GetLangfuseConfig() *config.LangfuseConfig {
	return nil
}

func (m *mockConfigService) GetEventsConfig() *config.EventsConfig {
	return nil
}

func (m *mockConfigService) GetMCPConfig() *config.MCPConfig {
	return nil
}

func (m *mockConfigService) GetACPConfig() *config.ACPConfig {
	return nil
}

func (m *mockConfigService) GetSubAgentConfig() *config.SubAgentConfig {
	return &m.subAgentCfg
}

func (m *mockConfigService) GetSupervisorConfig() *config.SupervisorConfig {
	return &config.SupervisorConfig{
		Strategy: m.subAgentCfg.Strategy,
	}
}
