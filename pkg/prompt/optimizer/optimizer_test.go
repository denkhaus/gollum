// Package optimizer_test provides tests for prompt optimization strategies.
package optimizer_test

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt/optimizer"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestNewOptimizer_ValidConfig tests creating optimizers with valid config.
func TestNewOptimizer_ValidConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	mockPM := mocks.NewMockPromptManager(ctrl)

	tests := []struct {
		name     string
		strategy shared.OptimizerStrategy
	}{
		{"gradient strategy", shared.StrategyGradient},
		{"metaprompt strategy", shared.StrategyMetaPrompt},
		{"prompt memory strategy", shared.StrategyPromptMemory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &optimizer.OptimizerConfig{
				Kind: tt.strategy,
			}

			opt, err := optimizer.NewOptimizer(mockClient, mockPM, config)
			require.NoError(t, err)
			assert.NotNil(t, opt)
			assert.Implements(t, (*optimizer.PromptOptimizer)(nil), opt)
		})
	}
}

// TestNewOptimizer_NilConfig tests error handling for nil config.
func TestNewOptimizer_NilConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	mockPM := mocks.NewMockPromptManager(ctrl)

	opt, err := optimizer.NewOptimizer(mockClient, mockPM, nil)
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

// TestNewOptimizer_NilClient tests error handling for nil client.
func TestNewOptimizer_NilClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPM := mocks.NewMockPromptManager(ctrl)
	config := &optimizer.OptimizerConfig{
		Kind: shared.StrategyGradient,
	}

	opt, err := optimizer.NewOptimizer(nil, mockPM, config)
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "LLM client cannot be nil")
}

// TestNewOptimizer_NilPromptManager tests error handling for nil PromptManager.
func TestNewOptimizer_NilPromptManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	config := &optimizer.OptimizerConfig{
		Kind: shared.StrategyGradient,
	}

	opt, err := optimizer.NewOptimizer(mockClient, nil, config)
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "promptManager cannot be nil")
}

// TestNewOptimizer_InvalidReflectionBounds tests error handling for invalid bounds.
func TestNewOptimizer_InvalidReflectionBounds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	mockPM := mocks.NewMockPromptManager(ctrl)
	config := &optimizer.OptimizerConfig{
		Kind:               shared.StrategyGradient,
		MinReflectionSteps: 5,
		MaxReflectionSteps: 2, // max < min
	}

	opt, err := optimizer.NewOptimizer(mockClient, mockPM, config)
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "max_reflection_steps")
}

// TestNewOptimizer_DefaultReflectionBounds tests that default bounds are set.
func TestNewOptimizer_DefaultReflectionBounds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	mockPM := mocks.NewMockPromptManager(ctrl)

	config := &optimizer.OptimizerConfig{
		Kind: shared.StrategyGradient,
		// Bounds left as zero - should default to 1 min, 5 max
	}

	opt, err := optimizer.NewOptimizer(mockClient, mockPM, config)
	require.NoError(t, err)
	require.NotNil(t, opt)
	assert.Implements(t, (*optimizer.PromptOptimizer)(nil), opt)
}

// TestNewOptimizer_UnknownStrategy tests error handling for unknown strategy.
func TestNewOptimizer_UnknownStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	mockPM := mocks.NewMockPromptManager(ctrl)
	config := &optimizer.OptimizerConfig{
		Kind: shared.OptimizerStrategy("unknown"),
	}

	opt, err := optimizer.NewOptimizer(mockClient, mockPM, config)
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "unknown optimizer strategy")
}

// TestExtractBaseID tests the ExtractBaseID helper function.
func TestExtractBaseID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"versioned ID", "supervisor@1.0.0", "supervisor"},
		{"unversioned ID", "supervisor", "supervisor"},
		{"ID with multiple @", "user@host@1.0.0", "user@host"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizer.ExtractBaseID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFormatSessions tests the FormatSessions helper function.
func TestFormatSessions(t *testing.T) {
	textContent, err := gollem.NewTextContent("Hello")
	require.NoError(t, err)

	trajectory := &optimizer.Trajectory{
		Messages: []gollem.Message{
			{Role: gollem.RoleUser, Contents: []gollem.MessageContent{textContent}},
		},
	}

	result := optimizer.FormatSessions([]*optimizer.Trajectory{trajectory})
	assert.Contains(t, result, "Session 1")
	assert.Contains(t, result, "Hello")
}
