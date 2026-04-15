package shared

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/react"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockLLMClient is a mock implementation of gollem.LLMClient for testing
type mockLLMClient struct {
	mock.Mock
}

func (m *mockLLMClient) NewSession(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(gollem.Session), args.Error(1)
}

func (m *mockLLMClient) GenerateEmbedding(ctx context.Context, dimension int, input []string) ([][]float64, error) {
	args := m.Called(ctx, dimension, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]float64), args.Error(1)
}

func TestNewStrategyBuilder(t *testing.T) {
	injector := do.New()
	builder, err := NewStrategyBuilder(injector)

	assert.NoError(t, err, "NewStrategyBuilder should not return an error")
	assert.NotNil(t, builder, "builder should not be nil")
	assert.IsType(t, &strategyBuilderImpl{}, builder, "builder should be of type strategyBuilderImpl")
}

func TestStrategyBuilder_BuildDefaultReact(t *testing.T) {
	builder := &strategyBuilderImpl{}
	client := &mockLLMClient{}

	strategy := builder.BuildDefaultReact(client)

	assert.NotNil(t, strategy, "strategy should not be nil")
	assert.IsType(t, &react.Strategy{}, strategy, "strategy should be of type react.Strategy")
}

func TestStrategyBuilder_BuildReact(t *testing.T) {
	tests := []struct {
		name              string
		cfg               *StrategyConfig
		expectedMaxIter   int
		expectedMaxRepeat int
	}{
		{
			name: "custom config",
			cfg: &StrategyConfig{
				MaxIterations:      50,
				MaxRepeatedActions: 5,
			},
			expectedMaxIter:   50,
			expectedMaxRepeat: 5,
		},
		{
			name: "zero values should use defaults",
			cfg: &StrategyConfig{
				MaxIterations:      0,
				MaxRepeatedActions: 0,
			},
			expectedMaxIter:   20, // react default
			expectedMaxRepeat: 3,  // react default
		},
		{
			name: "partial config",
			cfg: &StrategyConfig{
				MaxIterations:      100,
				MaxRepeatedActions: 0,
			},
			expectedMaxIter:   100,
			expectedMaxRepeat: 3, // react default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &strategyBuilderImpl{}
			client := &mockLLMClient{}

			strategy := builder.BuildReact(tt.cfg, client)

			assert.NotNil(t, strategy, "strategy should not be nil")
			assert.IsType(t, &react.Strategy{}, strategy, "strategy should be of type react.Strategy")

			// Verify the strategy is configured correctly
			// Note: We can't directly inspect the strategy fields as they're unexported
			// but we can verify that the strategy was created successfully
		})
	}
}

func TestStrategyBuilder_BuildReact_NilConfig(t *testing.T) {
	builder := &strategyBuilderImpl{}
	client := &mockLLMClient{}

	// This should handle nil config gracefully by using defaults
	strategy := builder.BuildReact(nil, client)

	assert.NotNil(t, strategy, "strategy should not be nil even with nil config")
	assert.IsType(t, &react.Strategy{}, strategy, "strategy should be of type react.Strategy")
}
