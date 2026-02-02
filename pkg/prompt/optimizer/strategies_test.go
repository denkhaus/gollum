// Package optimizer provides prompt optimization strategies with TDD.
package optimizer

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
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
		config      *OptimizerConfig
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
			config:      &OptimizerConfig{Kind: StrategyGradient},
			expectError: "LLM client cannot be nil",
			nilClient:   true,
		},
		{
			name: "Invalid reflection bounds (max < min)",
			config: &OptimizerConfig{
				Kind:               StrategyGradient,
				MinReflectionSteps: 5,
				MaxReflectionSteps: 2,
			},
			expectError: "max_reflection_steps",
		},
		{
			name: "Unknown strategy",
			config: &OptimizerConfig{
				Kind: OptimizerStrategy("unknown"),
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
			_, err := NewOptimizer(client, tt.config)
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

	config := &OptimizerConfig{
		Kind:               StrategyGradient,
		MaxReflectionSteps: 5,
		MinReflectionSteps: 2,
	}

	o, err := NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	if o == nil {
		t.Fatal("Expected non-nil optimizer")
	}

	// Verify it's the right type
	if _, ok := o.(*gradientOptimizer); !ok {
		t.Error("Expected gradientOptimizer type")
	}
}

// TestMetaPromptStrategy_Creation tests that metaprompt optimizer can be created.
func TestMetaPromptStrategy_Creation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	config := &OptimizerConfig{
		Kind:               StrategyMetaPrompt,
		MaxReflectionSteps: 5,
		MinReflectionSteps: 2,
	}

	o, err := NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	if o == nil {
		t.Fatal("Expected non-nil optimizer")
	}

	// Verify it's the right type
	if _, ok := o.(*metaPromptOptimizer); !ok {
		t.Error("Expected metaPromptOptimizer type")
	}
}

// TestPromptMemoryStrategy_Creation tests that prompt memory optimizer can be created.
func TestPromptMemoryStrategy_Creation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	config := &OptimizerConfig{
		Kind: StrategyPromptMemory,
	}

	o, err := NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	if o == nil {
		t.Fatal("Expected non-nil optimizer")
	}

	// Verify it's the right type
	if _, ok := o.(*promptMemoryOptimizer); !ok {
		t.Error("Expected promptMemoryOptimizer type")
	}
}

// TestGradientStrategy_DefaultReflectionBounds tests that default bounds are applied.
func TestGradientStrategy_DefaultReflectionBounds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	config := &OptimizerConfig{
		Kind: StrategyGradient,
		// MaxReflectionSteps and MinReflectionSteps not set - should default
	}

	o, err := NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	grad, ok := o.(*gradientOptimizer)
	if !ok {
		t.Fatal("Expected gradientOptimizer type")
	}

	// Check defaults were applied
	if grad.config.MaxReflectionSteps != 5 {
		t.Errorf("Expected MaxReflectionSteps=5, got %d", grad.config.MaxReflectionSteps)
	}
	if grad.config.MinReflectionSteps != 1 {
		t.Errorf("Expected MinReflectionSteps=1, got %d", grad.config.MinReflectionSteps)
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
			config := &OptimizerConfig{
				Kind:               StrategyGradient,
				MinReflectionSteps: tt.minSteps,
				MaxReflectionSteps: tt.maxSteps,
			}

			_, err := NewOptimizer(mockClient, config)
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
	tool := &thinkTool{}
	spec := tool.Spec()

	if spec.Name != "think" {
		t.Errorf("Expected name 'think', got %q", spec.Name)
	}

	if spec.Description == "" {
		t.Error("Expected non-empty description")
	}

	if _, ok := spec.Parameters["thought"]; !ok {
		t.Error("Expected 'thought' parameter")
	}

	if len(spec.Required) != 1 || spec.Required[0] != "thought" {
		t.Error("Expected 'thought' to be required")
	}
}

// TestCritiqueTool_Spec tests that critique tool has correct specification.
func TestCritiqueTool_Spec(t *testing.T) {
	tool := &critiqueTool{}
	spec := tool.Spec()

	if spec.Name != "critique" {
		t.Errorf("Expected name 'critique', got %q", spec.Name)
	}

	if spec.Description == "" {
		t.Error("Expected non-empty description")
	}

	if _, ok := spec.Parameters["criticism"]; !ok {
		t.Error("Expected 'criticism' parameter")
	}

	if len(spec.Required) != 1 || spec.Required[0] != "criticism" {
		t.Error("Expected 'criticism' to be required")
	}
}

// TestRecommendTool_Spec tests that recommend tool has correct specification.
func TestRecommendTool_Spec(t *testing.T) {
	tool := &recommendTool{}
	spec := tool.Spec()

	if spec.Name != "recommend" {
		t.Errorf("Expected name 'recommend', got %q", spec.Name)
	}

	if spec.Description == "" {
		t.Error("Expected non-empty description")
	}

	if _, ok := spec.Parameters["warrants_adjustment"]; !ok {
		t.Error("Expected 'warrants_adjustment' parameter")
	}

	if len(spec.Required) != 1 || spec.Required[0] != "warrants_adjustment" {
		t.Error("Expected 'warrants_adjustment' to be required")
	}
}

// TestWarrantsAdjustmentDetection tests the warrantsAdjustment detection logic.
func TestWarrantsAdjustmentDetection(t *testing.T) {
	o := &gradientOptimizer{}

	tests := []struct {
		name     string
		response string
		expected bool
	}{
		{
			name:     "No adjustment signal",
			response: "no adjustment needed",
			expected: false,
		},
		{
			name:     "Warrants adjustment false",
			response: "warrants_adjustment = false",
			expected: false,
		},
		{
			name:     "No recommendations",
			response: "no recommendations to make",
			expected: false,
		},
		{
			name:     "Valid recommendation",
			response: "Based on the analysis, I recommend adding more context",
			expected: true,
		},
		{
			name:     "Warrants adjustment true",
			response: "warrants_adjustment = true. The prompt should be updated.",
			expected: true,
		},
		{
			name:     "Empty response",
			response: "",
			expected: false,
		},
		{
			name:     "Short response",
			response: "OK",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := o.warrantsAdjustment(tt.response)
			if result != tt.expected {
				t.Errorf("warrantsAdjustment(%q) = %v, want %v", tt.response, result, tt.expected)
			}
		})
	}
}

// TestIsNoAdjustmentResponse tests the isNoAdjustmentResponse detection logic.
func TestIsNoAdjustmentResponse(t *testing.T) {
	o := &metaPromptOptimizer{}

	tests := []struct {
		name     string
		response string
		expected bool
	}{
		{
			name:     "No adjustment signal",
			response: "no adjustment needed",
			expected: true,
		},
		{
			name:     "Warrants adjustment false",
			response: "warrants_adjustment = false",
			expected: true,
		},
		{
			name:     "No recommendations",
			response: "no recommendations to make",
			expected: true,
		},
		{
			name:     "Valid adjustment",
			response: "The prompt should be updated to include more context",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := o.isNoAdjustmentResponse(tt.response)
			if result != tt.expected {
				t.Errorf("isNoAdjustmentResponse(%q) = %v, want %v", tt.response, result, tt.expected)
			}
		})
	}
}

// TestFormatSessions tests the FormatSessions helper function.
func TestFormatSessions(t *testing.T) {
	trajectories := []*Trajectory{
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

	result := FormatSessions(trajectories)

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
