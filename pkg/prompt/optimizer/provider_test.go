// Package optimizer provides tests for the DI provider functions
package optimizer

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/stretchr/testify/assert"
)

// TestMapStrategy tests the string to OptimizerStrategy mapping
func TestMapStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected OptimizerStrategy
	}{
		{"gradient", StrategyGradient},
		{"Gradient", StrategyGradient},
		{"GRADIENT", StrategyGradient},
		{"metaprompt", StrategyMetaPrompt},
		{"MetaPrompt", StrategyMetaPrompt},
		{"METAPROMPT", StrategyMetaPrompt},
		{"meta-prompt", StrategyMetaPrompt},
		{"meta_prompt", StrategyMetaPrompt},
		{"promptmemory", StrategyPromptMemory},
		{"PromptMemory", StrategyPromptMemory},
		{"PROMPTMEMORY", StrategyPromptMemory},
		{"prompt-memory", StrategyPromptMemory},
		{"prompt_memory", StrategyPromptMemory},
		{"invalid", StrategyUnknown},
		{"", StrategyUnknown},
		{"foo", StrategyUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := MapStrategy(tt.input)
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
			result := MapProvider(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
