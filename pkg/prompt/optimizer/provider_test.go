// Package provider provides tests for the DI provider functions
package provider_test

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/prompt/optimizer"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"github.com/samber/do/v2"
)

// TestNewOptimizerProvider_ValidConfiguration tests successful optimizer creation
func TestNewOptimizerProvider_ValidConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		strategy       string
		provider       string
		expectedStrat  optimizer.OptimizerStrategy
		wantErr        bool
	}{
		{
			name:          "gradient strategy with anthropic provider",
			strategy:      "gradient",
			provider:      "anthropic",
			expectedStrat: optimizer.StrategyGradient,
			wantErr:       false,
		},
		{
			name:          "metaprompt strategy with openai provider",
			strategy:      "metaprompt",
			provider:      "openai",
			expectedStrat: optimizer.StrategyMetaPrompt,
			wantErr:       false,
		},
		{
			name:          "meta-prompt with hyphen maps to metaprompt",
			strategy:      "meta-prompt",
			provider:      "gemini",
			expectedStrat: optimizer.StrategyMetaPrompt,
			wantErr:       false,
		},
		{
			name:          "prompt_memory strategy with gemini provider",
			strategy:      "prompt_memory",
			provider:      "gemini",
			expectedStrat: optimizer.StrategyPromptMemory,
			wantErr:       false,
		},
		{
			name:          "prompt-memory with hyphen maps to prompt_memory",
			strategy:      "prompt-memory",
			provider:      "anthropic",
			expectedStrat: optimizer.StrategyPromptMemory,
			wantErr:       false,
		},
		{
			name:          "GRADIENT uppercase works",
			strategy:      "GRADIENT",
			provider:      "anthropic",
			expectedStrat: optimizer.StrategyGradient,
			wantErr:       false,
		},
		{
			name:          "MixedCase MetaPrompt works",
			strategy:      "MetaPrompt",
			provider:      "openai",
			expectedStrat: optimizer.StrategyMetaPrompt,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock dependencies
			mockConfig := mocks.NewMockConfigService(ctrl)
			mockClientProvider := mocks.NewMockClientProvider(ctrl)
			mockPromptManager := mocks.NewMockPromptManager(ctrl)
			mockLLMClient := mocks.NewMockLLMClient(ctrl)

			// Setup expectations
			mockConfig.EXPECT().GetPromptOptimizerConfig().Return(&config.PromptOptimizerConfig{
				DefaultStrategy:     tt.strategy,
				DefaultProvider:     tt.provider,
				MaxReflectionSteps:  5,
				MinReflectionSteps:  2,
			})

			mockClientProvider.EXPECT().GetClient(gomock.Any(), gomock.Any()).Return(mockLLMClient, nil)

			// Create real injector and register mocks
			injector := do.New()
			do.ProvideValue[injector](config.ConfigService(mockConfig))
			do.ProvideValue[injector](llm.ClientProvider(mockClientProvider))
			do.ProvideValue[injector](manager.PromptManager(mockPromptManager))

			// Create optimizer
			optim, err := optimizer.NewOptimizerProvider(injector)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, optim)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, optim)
				assert.Implements(t, (*optimizer.PromptOptimizer)(nil), optim)
			}
		})
	}
}

// TestNewOptimizerProvider_InvalidStrategy tests error handling for invalid strategy
func TestNewOptimizerProvider_InvalidStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfigService(ctrl)
	mockClientProvider := mocks.NewMockClientProvider(ctrl)
	mockPromptManager := mocks.NewMockPromptManager(ctrl)

	// Setup expectations
	mockConfig.EXPECT().GetPromptOptimizerConfig().Return(&config.PromptOptimizerConfig{
		DefaultStrategy:     "invalid_strategy",
		DefaultProvider:     "anthropic",
		MaxReflectionSteps:  5,
		MinReflectionSteps:  2,
	})

	// Create real injector and register mocks
	injector := do.New()
	do.ProvideValue[injector](config.ConfigService(mockConfig))
	do.ProvideValue[injector](llm.ClientProvider(mockClientProvider))
	do.ProvideValue[injector](manager.PromptManager(mockPromptManager))

	optim, err := optimizer.NewOptimizerProvider(injector)

	assert.Error(t, err)
	assert.Nil(t, optim)
	assert.Contains(t, err.Error(), "unknown optimizer strategy")
}

// TestNewOptimizerProvider_InvalidProvider tests error handling for invalid provider
func TestNewOptimizerProvider_InvalidProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfigService(ctrl)
	mockClientProvider := mocks.NewMockClientProvider(ctrl)
	mockPromptManager := mocks.NewMockPromptManager(ctrl)

	// Setup expectations
	mockConfig.EXPECT().GetPromptOptimizerConfig().Return(&config.PromptOptimizerConfig{
		DefaultStrategy:     "gradient",
		DefaultProvider:     "invalid_provider",
		MaxReflectionSteps:  5,
		MinReflectionSteps:  2,
	})

	// Create real injector and register mocks
	injector := do.New()
	do.ProvideValue[injector](config.ConfigService(mockConfig))
	do.ProvideValue[injector](llm.ClientProvider(mockClientProvider))
	do.ProvideValue[injector](manager.PromptManager(mockPromptManager))

	optim, err := optimizer.NewOptimizerProvider(injector)

	assert.Error(t, err)
	assert.Nil(t, optim)
	assert.Contains(t, err.Error(), "unknown LLM provider")
}

// TestNewOptimizerProvider_ClientError tests error handling when client provider fails
func TestNewOptimizerProvider_ClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mocks.NewMockConfigService(ctrl)
	mockClientProvider := mocks.NewMockClientProvider(ctrl)
	mockPromptManager := mocks.NewMockPromptManager(ctrl)

	// Setup expectations
	mockConfig.EXPECT().GetPromptOptimizerConfig().Return(&config.PromptOptimizerConfig{
		DefaultStrategy:     "gradient",
		DefaultProvider:     "anthropic",
		MaxReflectionSteps:  5,
		MinReflectionSteps:  2,
	})

	mockClientProvider.EXPECT().GetClient(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	// Create real injector and register mocks
	injector := do.New()
	do.ProvideValue[injector](config.ConfigService(mockConfig))
	do.ProvideValue[injector](llm.ClientProvider(mockClientProvider))
	do.ProvideValue[injector](manager.PromptManager(mockPromptManager))

	optim, err := optimizer.NewOptimizerProvider(injector)

	assert.Error(t, err)
	assert.Nil(t, optim)
	assert.Contains(t, err.Error(), "failed to get LLM client")
}

// TestNewOptimizerProvider_ReflectionStepsConfig tests that reflection steps are passed correctly
func TestNewOptimizerProvider_ReflectionStepsConfig(t *testing.T) {
	tests := []struct {
		name     string
		maxSteps int
		minSteps int
	}{
		{
			name:     "default reflection steps",
			maxSteps: 5,
			minSteps: 2,
		},
		{
			name:     "custom reflection steps",
			maxSteps: 10,
			minSteps: 3,
		},
		{
			name:     "single reflection step",
			maxSteps: 1,
			minSteps: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConfig := mocks.NewMockConfigService(ctrl)
			mockClientProvider := mocks.NewMockClientProvider(ctrl)
			mockPromptManager := mocks.NewMockPromptManager(ctrl)
			mockLLMClient := mocks.NewMockLLMClient(ctrl)

			// Setup expectations
			mockConfig.EXPECT().GetPromptOptimizerConfig().Return(&config.PromptOptimizerConfig{
				DefaultStrategy:     "gradient",
				DefaultProvider:     "anthropic",
				MaxReflectionSteps:  tt.maxSteps,
				MinReflectionSteps:  tt.minSteps,
			})

			mockClientProvider.EXPECT().GetClient(gomock.Any(), gomock.Any()).Return(mockLLMClient, nil)

			// Create real injector and register mocks
			injector := do.New()
			do.ProvideValue[injector](config.ConfigService(mockConfig))
			do.ProvideValue[injector](llm.ClientProvider(mockClientProvider))
			do.ProvideValue[injector](manager.PromptManager(mockPromptManager))

			optim, err := optimizer.NewOptimizerProvider(injector)

			require.NoError(t, err)
			assert.NotNil(t, optim)
		})
	}
}

// TestMapStrategy tests the string to OptimizerStrategy mapping
func TestMapStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected optimizer.OptimizerStrategy
	}{
		{"gradient", optimizer.StrategyGradient},
		{"Gradient", optimizer.StrategyGradient},
		{"GRADIENT", optimizer.StrategyGradient},
		{"metaprompt", optimizer.StrategyMetaPrompt},
		{"MetaPrompt", optimizer.StrategyMetaPrompt},
		{"METAPROMPT", optimizer.StrategyMetaPrompt},
		{"meta-prompt", optimizer.StrategyMetaPrompt},
		{"meta_prompt", optimizer.StrategyMetaPrompt},
		{"promptmemory", optimizer.StrategyPromptMemory},
		{"PromptMemory", optimizer.StrategyPromptMemory},
		{"PROMPTMEMORY", optimizer.StrategyPromptMemory},
		{"prompt-memory", optimizer.StrategyPromptMemory},
		{"prompt_memory", optimizer.StrategyPromptMemory},
		{"invalid", optimizer.StrategyUnknown},
		{"", optimizer.StrategyUnknown},
		{"foo", optimizer.StrategyUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := optimizer.MapStrategy(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMapProvider tests the string to LLMProvider mapping
func TestMapProvider(t *testing.T) {
	tests := []struct {
		input    string
		expected shared.LLMProvider
	}{
		{"anthropic", shared.LLMProviderAnthropic},
		{"Anthropic", shared.LLMProviderAnthropic},
		{"ANTHROPIC", shared.LLMProviderAnthropic},
		{"openai", shared.LLMProviderOpenAI},
		{"OpenAI", shared.LLMProviderOpenAI},
		{"OPENAI", shared.LLMProviderOpenAI},
		{"gemini", shared.LLMProviderGemini},
		{"Gemini", shared.LLMProviderGemini},
		{"GEMINI", shared.LLMProviderGemini},
		{"invalid", shared.LLMProvider("")},
		{"", shared.LLMProvider("")},
		{"foo", shared.LLMProvider("")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := optimizer.MapProvider(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
