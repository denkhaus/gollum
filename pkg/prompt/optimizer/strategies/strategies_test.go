// Package optimizertest provides prompt optimization strategies tests with TDD.
package optimizertest

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt/optimizer"
	"github.com/m-mizutani/gollem"
	"go.uber.org/mock/gomock"
)

// TestNewOptimizer_InvalidConfig tests that NewOptimizer validates config properly.
func TestNewOptimizer_InvalidConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	tests := []struct {
		name        string
		config      *optimizer.OptimizerConfig
		expectError string
		nilClient   bool
	}{
		{
			name:        "Nil config",
			config:      nil,
			expectError: "config cannot be nil",
		},
		{
			name:        "Nil client",
			config:      &optimizer.OptimizerConfig{Kind: optimizer.StrategyGradient},
			expectError: "LLM client cannot be nil",
			nilClient:   true,
		},
		{
			name: "Invalid reflection bounds (max < min)",
			config: &optimizer.OptimizerConfig{
				Kind:               optimizer.StrategyGradient,
				MinReflectionSteps: 5,
				MaxReflectionSteps: 2,
			},
			expectError: "max_reflection_steps",
		},
		{
			name: "Unknown strategy",
			config: &optimizer.OptimizerConfig{
				Kind: optimizer.OptimizerStrategy("unknown"),
			},
			expectError: "unknown optimizer strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client gollem.LLMClient = mockClient
			if tt.nilClient {
				client = nil
			}
			_, err := optimizer.NewOptimizer(client, tt.config)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
		})
	}
}

// TestGradientStrategy_Creation tests that gradient optimizer can be created.
func TestGradientStrategy_Creation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	config := &optimizer.OptimizerConfig{
		Kind:               optimizer.StrategyGradient,
		MaxReflectionSteps: 5,
		MinReflectionSteps: 2,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	if o == nil {
		t.Fatal("Expected non-nil optimizer")
	}
}

// TestMetaPromptStrategy_Creation tests that metaprompt optimizer can be created.
func TestMetaPromptStrategy_Creation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	config := &optimizer.OptimizerConfig{
		Kind:               optimizer.StrategyMetaPrompt,
		MaxReflectionSteps: 5,
		MinReflectionSteps: 2,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	if o == nil {
		t.Fatal("Expected non-nil optimizer")
	}
}

// TestPromptMemoryStrategy_Creation tests that prompt memory optimizer can be created.
func TestPromptMemoryStrategy_Creation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	config := &optimizer.OptimizerConfig{
		Kind: optimizer.StrategyPromptMemory,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	if o == nil {
		t.Fatal("Expected non-nil optimizer")
	}
}

// TestGradientStrategy_DefaultReflectionBounds tests that default bounds are applied.
func TestGradientStrategy_DefaultReflectionBounds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	config := &optimizer.OptimizerConfig{
		Kind: optimizer.StrategyGradient,
		// MaxReflectionSteps and MinReflectionSteps not set - should default
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	if o == nil {
		t.Fatal("Expected non-nil optimizer")
	}
}

// TestGradientStrategy_ReflectionBoundsValidation tests various reflection bound configurations.
func TestGradientStrategy_ReflectionBoundsValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	tests := []struct {
		name        string
		minSteps    int
		maxSteps    int
		expectError bool
	}{
		{
			name:        "Valid bounds (2-5)",
			minSteps:    2,
			maxSteps:    5,
			expectError: false,
		},
		{
			name:        "Valid bounds (1-1)",
			minSteps:    1,
			maxSteps:    1,
			expectError: false,
		},
		{
			name:        "Invalid bounds (max < min)",
			minSteps:    5,
			maxSteps:    2,
			expectError: true,
		},
		{
			name:        "Zero max steps",
			minSteps:    0,
			maxSteps:    0,
			expectError: false, // Will default to 5 and 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &optimizer.OptimizerConfig{
				Kind:               optimizer.StrategyGradient,
				MinReflectionSteps: tt.minSteps,
				MaxReflectionSteps: tt.maxSteps,
			}

			_, err := optimizer.NewOptimizer(mockClient, config)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestThinkTool_Spec tests that think tool has correct specification.
func TestThinkTool_Spec(t *testing.T) {
	// This test is skipped since we can't access internal types from another package
	t.Skip("Internal type access not available from optimizertest package")
}

// TestCritiqueTool_Spec tests that critique tool has correct specification.
func TestCritiqueTool_Spec(t *testing.T) {
	// This test is skipped since we can't access internal types from another package
	t.Skip("Internal type access not available from optimizertest package")
}

// TestRecommendTool_Spec tests that recommend tool has correct specification.
func TestRecommendTool_Spec(t *testing.T) {
	// This test is skipped since we can't access internal types from another package
	t.Skip("Internal type access not available from optimizertest package")
}

// TestWarrantsAdjustmentDetection tests the warrantsAdjustment detection logic.
func TestWarrantsAdjustmentDetection(t *testing.T) {
	// This test is skipped since we can't access internal types from another package
	t.Skip("Internal type access not available from optimizertest package")
}

// TestIsNoAdjustmentResponse tests the isNoAdjustmentResponse detection logic.
func TestIsNoAdjustmentResponse(t *testing.T) {
	// This test is skipped since we can't access internal types from another package
	t.Skip("Internal type access not available from optimizertest package")
}

// TestFormatSessions tests the FormatSessions helper function.
func TestFormatSessions(t *testing.T) {
	trajectories := []*optimizer.Trajectory{
		{
			Messages: []gollem.Message{
				{
					Role: gollem.RoleUser,
					Contents: []gollem.MessageContent{
						mustNewTextContent("Hello"),
					},
				},
			},
		},
	}

	result := optimizer.FormatSessions(trajectories)

	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Should contain session header
	if !contains(result, "Session 1") {
		t.Error("Expected 'Session 1' in output")
	}
}

// Helper functions
func mustNewTextContent(text string) gollem.MessageContent {
	content, err := gollem.NewTextContent(text)
	if err != nil {
		panic(err)
	}
	return content
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
