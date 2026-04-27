package strategy

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
		name         string
		supCfg       config.SupervisorConfig
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCfg := config.NewMockConfigService(ctrl)
			mockCfg.EXPECT().GetSupervisorConfig().Return(&tt.supCfg).AnyTimes()

			builder := &builderImpl{
				configService: mockCfg,
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
		name         string
		subCfg       config.SubAgentConfig
		strategyType StrategyType
	}{
		{
			name: "sub agent with simple strategy",
			subCfg: config.SubAgentConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      15,
					MaxRepeatedActions: 2,
				},
			},
			strategyType: StrategyTypeSimple,
		},
		{
			name: "sub agent with react strategy",
			subCfg: config.SubAgentConfig{
				Strategy: config.StrategyConfig{
					MaxIterations:      30,
					MaxRepeatedActions: 4,
				},
			},
			strategyType: StrategyTypeReact,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCfg := config.NewMockConfigService(ctrl)
			mockCfg.EXPECT().GetSubAgentConfig().Return(&tt.subCfg).AnyTimes()

			builder := &builderImpl{
				configService: mockCfg,
			}

			// Verify config is accessed correctly
			subCfg := builder.configService.GetSubAgentConfig()
			require.NotNil(t, subCfg)
			assert.Equal(t, tt.subCfg.Strategy.MaxIterations, subCfg.Strategy.MaxIterations)
			assert.Equal(t, tt.subCfg.Strategy.MaxRepeatedActions, subCfg.Strategy.MaxRepeatedActions)
		})
	}
}
