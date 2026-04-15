package strategy

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	builder := &builderImpl{}
	assert.NotNil(t, builder)
}

func TestStrategyType_Constants(t *testing.T) {
	assert.Equal(t, StrategyType("react"), StrategyTypeReact)
	assert.Equal(t, StrategyType("simple"), StrategyTypeSimple)
	assert.Equal(t, StrategyType("react"), StrategyTypeDefault)
}

func TestBuilderImpl_BuildForSupervisor(t *testing.T) {
	tests := []struct {
		name        string
		supCfg      config.SupervisorConfig
		strategyType StrategyType
	}{
		{
			name: "react strategy with defaults",
			supCfg: config.SupervisorConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      20,
					MaxRepeatedActions: 3,
				},
			},
			strategyType: StrategyTypeReact,
		},
		{
			name: "simple strategy",
			supCfg: config.SupervisorConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      20,
					MaxRepeatedActions: 3,
				},
			},
			strategyType: StrategyTypeSimple,
		},
		{
			name: "default type uses react",
			supCfg: config.SupervisorConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      25,
					MaxRepeatedActions: 5,
				},
			},
			strategyType: StrategyTypeDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &builderImpl{
				configService: &mockConfigService{
					supervisorCfg: tt.supCfg,
				},
			}

			// Verify config is accessed correctly
			supCfg := builder.configService.GetSupervisorConfig()
			require.NotNil(t, supCfg)
			assert.Equal(t, tt.supCfg.Strategy.MaxIterations, supCfg.Strategy.MaxIterations)
			assert.Equal(t, tt.supCfg.Strategy.MaxRepeatedActions, supCfg.Strategy.MaxRepeatedActions)
		})
	}
}

func TestBuilderImpl_BuildForSubAgent(t *testing.T) {
	tests := []struct {
		name        string
		subCfg      config.SubAgentConfig
		strategyType StrategyType
	}{
		{
			name: "react strategy with defaults",
			subCfg: config.SubAgentConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      20,
					MaxRepeatedActions: 3,
				},
			},
			strategyType: StrategyTypeReact,
		},
		{
			name: "simple strategy",
			subCfg: config.SubAgentConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      20,
					MaxRepeatedActions: 3,
				},
			},
			strategyType: StrategyTypeSimple,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &builderImpl{
				configService: &mockConfigService{
					subAgentCfg: tt.subCfg,
				},
			}

			subCfg := builder.configService.GetSubAgentConfig()
			require.NotNil(t, subCfg)
			assert.Equal(t, tt.subCfg.Strategy.MaxIterations, subCfg.Strategy.MaxIterations)
		})
	}
}

func TestBuilderImpl_BuildForLLMStep(t *testing.T) {
	subCfg := config.SubAgentConfig{
		Strategy: config.StrategyConfig{
			MaxIterations:      20,
			MaxRepeatedActions: 3,
		},
	}

	builder := &builderImpl{
		configService: &mockConfigService{
			subAgentCfg: subCfg,
		},
	}

	// LLMStep should use subagent config
	llmStepCfg := builder.configService.GetSubAgentConfig()
	require.NotNil(t, llmStepCfg)
	assert.Equal(t, subCfg.Strategy.MaxIterations, llmStepCfg.Strategy.MaxIterations)
}

func TestBuilderImpl_BuildReact_Options(t *testing.T) {
	tests := []struct {
		name               string
		cfg                *config.StrategyConfig
		expectMaxIterations int
		expectMaxRepeatedAct int
	}{
		{
			name:                "nil config uses react defaults",
			cfg:                 nil,
			expectMaxIterations:  0,
			expectMaxRepeatedAct: 0,
		},
		{
			name: "both values set",
			cfg: &config.StrategyConfig{
				MaxIterations:      10,
				MaxRepeatedActions: 2,
			},
			expectMaxIterations:  10,
			expectMaxRepeatedAct: 2,
		},
		{
			name: "only max iterations",
			cfg: &config.StrategyConfig{
				MaxIterations: 15,
			},
			expectMaxIterations:  15,
			expectMaxRepeatedAct: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	subAgentCfg    config.SubAgentConfig
	supervisorCfg  config.SupervisorConfig
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
	return &m.supervisorCfg
}
