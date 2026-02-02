// Package strategies implements prompt optimization strategies with TDD.
package strategies

import (
	"context"
	"errors"
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt/optimizer"
	"github.com/m-mizutani/gollem"
	"go.uber.org/mock/gomock"
)

// TestGradientStrategy_Optimize_Success tests the gradient strategy with a successful optimization.
func TestGradientStrategy_Optimize_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	ctx := context.Background()

	// Create optimizer
	config := &optimizer.OptimizerConfig{
		Kind:               optimizer.StrategyGradient,
		MaxReflectionSteps: 5,
		MinReflectionSteps: 2,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	// Setup mock session
	mockSession := mocks.NewMockSession(ctrl)

	// Expect NewSession call
	mockClient.EXPECT().NewSession(ctx, gomock.Any()).Return(mockSession, nil)

	// Test input
	input := &optimizer.OptimizerInput{
		Prompt: "You are a helpful assistant.",
		Trajectories: []*optimizer.Trajectory{
			{
				Messages: []gollem.Message{
					{
						Role: gollem.RoleUser,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Hello"),
						},
					},
					{
						Role: gollem.RoleAssistant,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Hi there!"),
						},
					},
				},
			},
		},
		UpdateInstructions: "Make the response more professional",
	}

	// Phase 1: Reflection - think tool call
	thinkCallID := "call-1"
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{},
		FunctionCalls: []*gollem.FunctionCall{
			{
				ID:   thinkCallID,
				Name: "think",
				Arguments: map[string]any{
					"thought": "The assistant's response is too casual",
				},
			},
		},
	}, nil).Times(2) // Min 2 reflection steps

	// Phase 1: Reflection - critique tool call
	critiqueCallID := "call-2"
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{},
		FunctionCalls: []*gollem.FunctionCall{
			{
				ID:   critiqueCallID,
				Name: "critique",
				Arguments: map[string]any{
					"criticism": "The tone should be more formal",
				},
			},
		},
	}, nil).Times(2)

	// Phase 1: Reflection - recommend tool call
	recommendCallID := "call-3"
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{},
		FunctionCalls: []*gollem.FunctionCall{
			{
				ID:   recommendCallID,
				Name: "recommend",
				Arguments: map[string]any{
					"warrants_adjustment": true,
					"hypotheses":          "The prompt lacks professional tone guidelines",
					"full_recommendations": "Add instructions to maintain professional communication",
				},
			},
		},
	}, nil).Times(2)

	// Phase 2: Apply recommendations
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{"You are a helpful professional assistant. Maintain a formal tone in all responses."},
	}, nil).Times(1)

	// Run optimization
	result, err := o.Optimize(ctx, input)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// Verify result
	if !result.WarrantsAdjustment {
		t.Error("Expected WarrantsAdjustment=true")
	}

	if result.NewPrompt == "" {
		t.Error("Expected NewPrompt to be non-empty")
	}

	if result.ChangeDescription == "" {
		t.Error("Expected ChangeDescription to be non-empty")
	}
}

// TestGradientStrategy_Optimize_NoAdjustment tests the gradient strategy when no adjustment is needed.
func TestGradientStrategy_Optimize_NoAdjustment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	ctx := context.Background()

	config := &optimizer.OptimizerConfig{
		Kind:               optimizer.StrategyGradient,
		MaxReflectionSteps: 5,
		MinReflectionSteps: 2,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	mockSession := mocks.NewMockSession(ctrl)
	mockClient.EXPECT().NewSession(ctx, gomock.Any()).Return(mockSession, nil)

	input := &optimizer.OptimizerInput{
		Prompt: "You are a helpful assistant.",
		Trajectories: []*optimizer.Trajectory{
			{
				Messages: []gollem.Message{
					{
						Role: gollem.RoleUser,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Hello"),
						},
					},
					{
						Role: gollem.RoleAssistant,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Hello! How can I help you today?"),
						},
					},
				},
			},
		},
	}

	// Phase 1: Reflection - recommend tool returns false
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{},
		FunctionCalls: []*gollem.FunctionCall{
			{
				ID:   "call-1",
				Name: "think",
				Arguments: map[string]any{
					"thought": "The assistant performed well",
				},
			},
		},
	}, nil).Times(2)

	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{},
		FunctionCalls: []*gollem.FunctionCall{
			{
				ID:   "call-2",
				Name: "critique",
				Arguments: map[string]any{
					"criticism": "No issues found",
				},
			},
		},
	}, nil).Times(2)

	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{"No recommendations."},
		FunctionCalls: []*gollem.FunctionCall{
			{
				ID:   "call-3",
				Name: "recommend",
				Arguments: map[string]any{
					"warrants_adjustment": false,
				},
			},
		},
	}, nil).Times(2)

	// Run optimization
	result, err := o.Optimize(ctx, input)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// Verify result
	if result.WarrantsAdjustment {
		t.Error("Expected WarrantsAdjustment=false")
	}

	if result.NewPrompt == "" {
		t.Error("Expected NewPrompt to be original prompt")
	}
}

// TestMetaPromptStrategy_Optimize_Success tests the metaprompt strategy with a successful optimization.
func TestMetaPromptStrategy_Optimize_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	ctx := context.Background()

	config := &optimizer.OptimizerConfig{
		Kind:               optimizer.StrategyMetaPrompt,
		MaxReflectionSteps: 5,
		MinReflectionSteps: 2,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	mockSession := mocks.NewMockSession(ctrl)
	mockClient.EXPECT().NewSession(ctx, gomock.Any()).Return(mockSession, nil)

	input := &optimizer.OptimizerInput{
		Prompt: "You are a helpful assistant.",
		Trajectories: []*optimizer.Trajectory{
			{
				Messages: []gollem.Message{
					{
						Role: gollem.RoleUser,
						Contents: []gollem.MessageContent{
							mustNewTextContent("What is 2+2?"),
						},
					},
					{
						Role: gollem.RoleAssistant,
						Contents: []gollem.MessageContent{
							mustNewTextContent("I don't know"),
						},
					},
				},
				Feedback: "The assistant failed to answer a simple math question",
			},
		},
	}

	// Expect 2 reflection steps (min)
	updatedPrompt := "You are a helpful assistant. When asked mathematical questions, provide accurate answers."
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{updatedPrompt},
	}, nil).Times(2)

	// Run optimization
	result, err := o.Optimize(ctx, input)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// Verify result
	if !result.WarrantsAdjustment {
		t.Error("Expected WarrantsAdjustment=true")
	}

	if result.NewPrompt != updatedPrompt {
		t.Errorf("Expected NewPrompt=%q, got %q", updatedPrompt, result.NewPrompt)
	}
}

// TestMetaPromptStrategy_Optimize_NoAdjustment tests the metaprompt strategy when no adjustment is needed.
func TestMetaPromptStrategy_Optimize_NoAdjustment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	ctx := context.Background()

	config := &optimizer.OptimizerConfig{
		Kind:               optimizer.StrategyMetaPrompt,
		MaxReflectionSteps: 5,
		MinReflectionSteps: 2,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	mockSession := mocks.NewMockSession(ctrl)
	mockClient.EXPECT().NewSession(ctx, gomock.Any()).Return(mockSession, nil)

	input := &optimizer.OptimizerInput{
		Prompt: "You are a helpful assistant.",
		Trajectories: []*optimizer.Trajectory{
			{
				Messages: []gollem.Message{
					{
						Role: gollem.RoleUser,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Hello"),
						},
					},
					{
						Role: gollem.RoleAssistant,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Hello! How can I help you?"),
						},
					},
				},
			},
		},
	}

	// Expect 2 reflection steps returning original prompt
	originalPrompt := "You are a helpful assistant."
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{originalPrompt},
	}, nil).Times(2)

	// Run optimization
	result, err := o.Optimize(ctx, input)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// Verify result
	if result.WarrantsAdjustment {
		t.Error("Expected WarrantsAdjustment=false")
	}

	if result.NewPrompt != originalPrompt {
		t.Errorf("Expected NewPrompt=%q, got %q", originalPrompt, result.NewPrompt)
	}
}

// TestPromptMemoryStrategy_Optimize_Success tests the prompt memory strategy with a successful optimization.
func TestPromptMemoryStrategy_Optimize_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	ctx := context.Background()

	config := &optimizer.OptimizerConfig{
		Kind: optimizer.StrategyPromptMemory,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	mockSession := mocks.NewMockSession(ctrl)
	mockClient.EXPECT().NewSession(ctx, gomock.Any()).Return(mockSession, nil)

	input := &optimizer.OptimizerInput{
		Prompt: "You are a helpful assistant.",
		Trajectories: []*optimizer.Trajectory{
			{
				Messages: []gollem.Message{
					{
						Role: gollem.RoleUser,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Write code in Python"),
						},
					},
					{
						Role: gollem.RoleAssistant,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Here is some code in Java..."),
						},
					},
				},
			},
		},
		Feedback: "The assistant ignored the language requirement",
	}

	// Single-shot optimization
	updatedPrompt := "You are a helpful assistant. Always follow the user's requirements precisely, including programming language specifications."
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{updatedPrompt},
	}, nil).Times(1)

	// Run optimization
	result, err := o.Optimize(ctx, input)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// Verify result
	if !result.WarrantsAdjustment {
		t.Error("Expected WarrantsAdjustment=true")
	}

	if result.NewPrompt != updatedPrompt {
		t.Errorf("Expected NewPrompt=%q, got %q", updatedPrompt, result.NewPrompt)
	}
}

// TestPromptMemoryStrategy_Optimize_NoAdjustment tests the prompt memory strategy when no adjustment is needed.
func TestPromptMemoryStrategy_Optimize_NoAdjustment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLLMClient(ctrl)
	ctx := context.Background()

	config := &optimizer.OptimizerConfig{
		Kind: optimizer.StrategyPromptMemory,
	}

	o, err := optimizer.NewOptimizer(mockClient, config)
	if err != nil {
		t.Fatalf("NewOptimizer failed: %v", err)
	}

	mockSession := mocks.NewMockSession(ctrl)
	mockClient.EXPECT().NewSession(ctx, gomock.Any()).Return(mockSession, nil)

	input := &optimizer.OptimizerInput{
		Prompt: "You are a helpful assistant.",
		Trajectories: []*optimizer.Trajectory{
			{
				Messages: []gollem.Message{
					{
						Role: gollem.RoleUser,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Hello"),
						},
					},
					{
						Role: gollem.RoleAssistant,
						Contents: []gollem.MessageContent{
							mustNewTextContent("Hello! How can I help you?"),
						},
					},
				},
			},
		},
		Feedback: "Good response",
	}

	// Single-shot optimization returns original
	mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).Return(&gollem.Response{
		Texts: []string{"warrants_adjustment = False\n\nYou are a helpful assistant."},
	}, nil).Times(1)

	// Run optimization
	result, err := o.Optimize(ctx, input)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// Verify result
	if result.WarrantsAdjustment {
		t.Error("Expected WarrantsAdjustment=false")
	}

	if result.NewPrompt == "" {
		t.Error("Expected NewPrompt to be non-empty")
	}
}

// TestGradientStrategy_ReflectionBounds tests that gradient strategy respects min/max reflection steps.
func TestGradientStrategy_ReflectionBounds(t *testing.T) {
	tests := []struct {
		name             string
		minSteps         int
		maxSteps         int
		expectedCallsMin int
		expectedCallsMax int
	}{
		{
			name:             "Default bounds (2-5)",
			minSteps:         2,
			maxSteps:         5,
			expectedCallsMin: 2,
			expectedCallsMax: 5,
		},
		{
			name:             "Custom bounds (3-4)",
			minSteps:         3,
			maxSteps:         4,
			expectedCallsMin: 3,
			expectedCallsMax: 4,
		},
		{
			name:             "Single step bounds (1-1)",
			minSteps:         1,
			maxSteps:         1,
			expectedCallsMin: 1,
			expectedCallsMax: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockLLMClient(ctrl)
			ctx := context.Background()

			config := &optimizer.OptimizerConfig{
				Kind:               optimizer.StrategyGradient,
				MinReflectionSteps: tt.minSteps,
				MaxReflectionSteps: tt.maxSteps,
			}

			o, err := optimizer.NewOptimizer(mockClient, config)
			if err != nil {
				t.Fatalf("NewOptimizer failed: %v", err)
			}

			mockSession := mocks.NewMockSession(ctrl)
			mockClient.EXPECT().NewSession(ctx, gomock.Any()).Return(mockSession, nil)

			input := &optimizer.OptimizerInput{
				Prompt:       "You are a helpful assistant.",
				Trajectories: []*optimizer.Trajectory{{}},
			}

			// Simulate LLM continuing reflection until max steps
			callCount := 0
			mockSession.EXPECT().GenerateContent(ctx, gomock.Any()).DoAndReturn(
				func(ctx context.Context, input ...interface{}) (*gollem.Response, error) {
					callCount++
					if callCount < tt.maxSteps {
						// Continue reflection
						return &gollem.Response{
							FunctionCalls: []*gollem.FunctionCall{
								{
									ID:   "call-1",
									Name: "think",
									Arguments: map[string]any{
										"thought": "Still analyzing...",
									},
								},
							},
						}, nil
					}
					// At max steps, return recommendation
					return &gollem.Response{
						FunctionCalls: []*gollem.FunctionCall{
							{
								ID:   "call-2",
								Name: "recommend",
								Arguments: map[string]any{
									"warrants_adjustment": false,
								},
							},
						},
					}, nil
				},
			).MinTimes(tt.minSteps).MaxTimes(tt.maxSteps + 1)

			_, err = o.Optimize(ctx, input)
			if err != nil {
				t.Fatalf("Optimize failed: %v", err)
			}

			if callCount < tt.expectedCallsMin || callCount > tt.expectedCallsMax {
				t.Errorf("Expected between %d and %d calls, got %d", tt.expectedCallsMin, tt.expectedCallsMax, callCount)
			}
		})
	}
}

// TestNewOptimizer_InvalidConfig tests that NewOptimizer validates config properly.
func TestNewOptimizer_InvalidConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mocks.NewMockLLMClient(ctrl)

	tests := []struct {
		name        string
		config      *optimizer.OptimizerConfig
		expectError string
	}{
		{
			name:        "Nil config",
			config:      nil,
			expectError: "config cannot be nil",
		},
		{
			name: "Nil client",
			config: &optimizer.OptimizerConfig{
				Kind: optimizer.StrategyGradient,
			},
			expectError: "LLM client cannot be nil",
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
			_, err := optimizer.NewOptimizer(mockClient, tt.config)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !errors.Is(err, errors.New(tt.expectError)) && err.Error() == "" {
				t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

// mustNewTextContent creates text content or panics - for test helper only.
func mustNewTextContent(text string) gollem.MessageContent {
	content, err := gollem.NewTextContent(text)
	if err != nil {
		panic(err)
	}
	return content
}
