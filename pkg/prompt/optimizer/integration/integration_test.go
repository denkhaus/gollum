// Package integration provides integration tests for the optimizer package.
package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt/optimizer"
	"github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestOptimizeAndSave_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Setup mock optimizer
	mockOptimizer := mocks.NewMockPromptOptimizer(ctrl)

	// Setup memory store with existing prompt
	st := store.NewMemoryStore()
	existing, err := st.SaveNewVersion(ctx, "test-prompt", "original content", "Test Prompt")
	require.NoError(t, err)
	require.NotNil(t, existing)

	// Setup input
	input := &optimizer.OptimizerInput{}

	// Expect optimizer to be called and return improved prompt
	expectedResult := &optimizer.OptimizerResult{
		NewPrompt:          "improved content",
		WarrantsAdjustment: true,
		ChangeDescription:  "Improved clarity",
	}

	mockOptimizer.EXPECT().Optimize(ctx, gomock.Any()).Return(expectedResult, nil)

	// Run OptimizeAndSave
	result, err := optimizer.OptimizeAndSave(ctx, mockOptimizer, st, "test-prompt", input)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "improved content", result.Content)
	assert.NotEqual(t, existing.ID, result.ID) // Should be new version
	assert.Contains(t, result.ID, "@")        // Should have version

	// Verify new version was saved with incremented patch version
	versions, err := st.ListVersions(ctx, "test-prompt")
	require.NoError(t, err)
	assert.Len(t, versions, 2) // original + new version
}

func TestOptimizeAndSave_NoAdjustment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Setup mock optimizer
	mockOptimizer := mocks.NewMockPromptOptimizer(ctrl)

	// Setup memory store with existing prompt
	st := store.NewMemoryStore()
	existing, err := st.SaveNewVersion(ctx, "no-adjust", "original content", "No Adjust")
	require.NoError(t, err)

	// Setup input
	input := &optimizer.OptimizerInput{}

	// Expect optimizer to return no adjustment needed
	expectedResult := &optimizer.OptimizerResult{
		NewPrompt:          "original content",
		WarrantsAdjustment: false,
		ChangeDescription:  "No adjustment needed",
	}

	mockOptimizer.EXPECT().Optimize(ctx, gomock.Any()).Return(expectedResult, nil)

	// Run OptimizeAndSave
	result, err := optimizer.OptimizeAndSave(ctx, mockOptimizer, st, "no-adjust", input)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, existing.ID, result.ID) // Should be same prompt
	assert.Equal(t, "original content", result.Content)

	// Verify no new version was saved
	versions, err := st.ListVersions(ctx, "no-adjust")
	require.NoError(t, err)
	assert.Len(t, versions, 1) // Only original version
}

func TestOptimizeAndSave_PromptNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Setup mock optimizer
	mockOptimizer := mocks.NewMockPromptOptimizer(ctrl)

	// Setup memory store (empty)
	st := store.NewMemoryStore()

	// Setup input
	input := &optimizer.OptimizerInput{}

	// Run OptimizeAndSave with non-existent prompt
	result, err := optimizer.OptimizeAndSave(ctx, mockOptimizer, st, "nonexistent", input)

	// Verify
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "prompt not found")

	// Optimizer should not have been called
	mockOptimizer.EXPECT().Optimize(gomock.Any(), gomock.Any()).MaxTimes(0)
}

func TestOptimizeAndSave_OptimizerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Setup mock optimizer
	mockOptimizer := mocks.NewMockPromptOptimizer(ctrl)

	// Setup memory store with existing prompt
	st := store.NewMemoryStore()
	_, err := st.SaveNewVersion(ctx, "error-test", "content", "Error Test")
	require.NoError(t, err)

	// Setup input
	input := &optimizer.OptimizerInput{}

	// Expect optimizer to return error
	mockOptimizer.EXPECT().Optimize(ctx, gomock.Any()).Return(nil, errors.New("LLM unavailable"))

	// Run OptimizeAndSave
	result, err := optimizer.OptimizeAndSave(ctx, mockOptimizer, st, "error-test", input)

	// Verify
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "optimization failed")
}

func TestExtractBaseID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "versioned ID",
			input:    "supervisor@1.0.0",
			expected: "supervisor",
		},
		{
			name:     "non-versioned ID",
			input:    "supervisor",
			expected: "supervisor",
		},
		{
			name:     "complex versioned ID",
			input:    "my-prompt@2.1.3",
			expected: "my-prompt",
		},
		{
			name:     "ID with multiple @ symbols",
			input:    "my@prompt@1.0.0",
			expected: "my@prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizer.ExtractBaseID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedVer string
		expectError bool
	}{
		{
			name:        "valid versioned ID",
			input:       "supervisor@1.0.0",
			expectedVer: "1.0.0",
			expectError: false,
		},
		{
			name:        "valid version with pre-release",
			input:       "supervisor@1.0.0-alpha",
			expectedVer: "1.0.0-alpha",
			expectError: false,
		},
		{
			name:        "non-versioned ID returns error",
			input:       "supervisor",
			expectedVer: "",
			expectError: true,
		},
		{
			name:        "invalid version returns error",
			input:       "supervisor@invalid",
			expectedVer: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := optimizer.ParseVersion(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedVer, result.Original())
			}
		})
	}
}
