package llm

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// mockConfigService is a minimal implementation of ConfigService for testing.
//
// NOTE: This inline mock is required instead of using pkg/mocks.MockConfigService
// because of an import cycle:
//
//	pkg/llm → pkg/mocks → pkg/prompt/optimizer → pkg/llm
//
// Using generated mocks from pkg/mocks would create this cycle.
// This mock only implements the methods needed for provider tests.
type mockConfigService struct {
	anthropicConfig *config.AnthropicConfig
	geminiConfig    *config.GeminiConfig
	openaiConfig    *config.OpenAIConfig
}

func (m *mockConfigService) GetLogLevel() string                         { return "info" }
func (m *mockConfigService) IsDevMode() bool                             { return false }
func (m *mockConfigService) GetAnthropicConfig() *config.AnthropicConfig { return m.anthropicConfig }
func (m *mockConfigService) GetGeminiConfig() *config.GeminiConfig       { return m.geminiConfig }
func (m *mockConfigService) GetOpenAIConfig() *config.OpenAIConfig       { return m.openaiConfig }
func (m *mockConfigService) GetAgentLimits() *config.AgentLimitsConfig {
	return &config.AgentLimitsConfig{}
}
func (m *mockConfigService) GetFilesConfig() *config.FilesConfig     { return &config.FilesConfig{} }
func (m *mockConfigService) GetLoggingConfig() *config.LoggingConfig { return &config.LoggingConfig{} }
func (m *mockConfigService) GetBashConfig() *config.BashConfig       { return &config.BashConfig{} }
func (m *mockConfigService) GetHooksConfig() *config.HooksConfig     { return &config.HooksConfig{} }
func (m *mockConfigService) GetPromptStoreConfig() *config.PromptStoreConfig {
	return &config.PromptStoreConfig{}
}
func (m *mockConfigService) GetPromptOptimizerConfig() *config.PromptOptimizerConfig {
	return &config.PromptOptimizerConfig{}
}
func (m *mockConfigService) GetLangfuseConfig() *config.LangfuseConfig {
	return &config.LangfuseConfig{}
}
func (m *mockConfigService) GetEventsConfig() *config.EventsConfig { return &config.EventsConfig{} }
func (m *mockConfigService) GetMCPConfig() *config.MCPConfig       { return &config.MCPConfig{} }
func (m *mockConfigService) GetACPConfig() *config.ACPConfig       { return &config.ACPConfig{} }
func (m *mockConfigService) GetDatabaseConfig() config.DatabaseConfig { return config.DatabaseConfig{} }
func (m *mockConfigService) GetSubAgentConfig() *config.SubAgentConfig {
	return &config.SubAgentConfig{}
}
func (m *mockConfigService) GetSupervisorConfig() *config.SupervisorConfig {
	return &config.SupervisorConfig{}
}

// mockLoggerService is a minimal implementation of LoggerService for testing.
//
// NOTE: This inline mock is required instead of using pkg/mocks.MockLoggerService
// because of an import cycle:
//
//	pkg/llm → pkg/mocks → pkg/prompt/optimizer → pkg/llm
//
// Using generated mocks from pkg/mocks would create this cycle.
// This mock only implements the methods needed for provider tests.
type mockLoggerService struct{}

func (m *mockLoggerService) Info(msg string, fields ...zap.Field)              {}
func (m *mockLoggerService) Infof(template string, args ...any)                {}
func (m *mockLoggerService) Error(msg string, fields ...zap.Field)             {}
func (m *mockLoggerService) Errorf(template string, args ...any)               {}
func (m *mockLoggerService) Debug(msg string, fields ...zap.Field)             {}
func (m *mockLoggerService) Debugf(template string, args ...any)               {}
func (m *mockLoggerService) Warn(msg string, fields ...zap.Field)              {}
func (m *mockLoggerService) Warnf(template string, args ...any)                {}
func (m *mockLoggerService) GetLogger() *zap.Logger                            { return zap.NewNop() }
func (m *mockLoggerService) GetLogs(filter logger.LogFilter) []logger.LogEntry { return nil }
func (m *mockLoggerService) GetLogStats() map[string]any                       { return nil }
func (m *mockLoggerService) SetTUIMode(enabled bool)                           {}
func (m *mockLoggerService) IsTUIMode() bool                                   { return false }
func (m *mockLoggerService) EnableFileLogging(gollumDir string, sessionID uuid.UUID) error {
	return nil
}
func (m *mockLoggerService) CloseFileLogging() error                                           { return nil }
func (m *mockLoggerService) Flush() error                                                      { return nil }
func (m *mockLoggerService) InfoWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)  {}
func (m *mockLoggerService) ErrorWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {}
func (m *mockLoggerService) DebugWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field) {}
func (m *mockLoggerService) WarnWithAgent(msg string, agentID uuid.UUID, fields ...zap.Field)  {}
func (m *mockLoggerService) InfoWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field) {
}
func (m *mockLoggerService) DebugWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field) {
}
func (m *mockLoggerService) ErrorWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field) {
}
func (m *mockLoggerService) WarnWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field) {
}
func (m *mockLoggerService) InfoWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field) {
}
func (m *mockLoggerService) ErrorWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field) {
}
func (m *mockLoggerService) DebugWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field) {
}
func (m *mockLoggerService) WarnWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field) {
}
func (m *mockLoggerService) SetLogForwarder(forwarder shared.LogForwarder) {}

// TestGetClient_AnthropicWithConfig tests GetClient with Anthropic provider and full config
func TestGetClient_AnthropicWithConfig(t *testing.T) {
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (logger.LoggerService, error) {
		return &mockLoggerService{}, nil
	})
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{
			anthropicConfig: &config.AnthropicConfig{
				APIKey:      "test-key",
				BaseURL:     "https://api.anthropic.com",
				Model:       "claude-3-5-sonnet-20241022",
				Temperature: 0.7,
				MaxTokens:   8192,
				TopP:        1.0,
			},
		}, nil
	})

	provider, err := NewClientProvider(injector)
	if err != nil {
		t.Fatalf("NewClientProvider() error = %v", err)
	}

	ctx := context.Background()
	temp := 0.5
	maxTokens := 4096
	topP := 0.9

	cnf := &shared.LLMClientConfig{
		Model:       "anthropic/claude-3-5-sonnet-20241022",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		TopP:        &topP,
	}

	client, err := provider.GetClient(ctx, cnf)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if client == nil {
		t.Error("GetClient() returned nil client")
	}
}

// TestGetClient_AnthropicWithDefaults tests GetClient with Anthropic provider using defaults
func TestGetClient_AnthropicWithDefaults(t *testing.T) {
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (logger.LoggerService, error) {
		return &mockLoggerService{}, nil
	})
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{
			anthropicConfig: &config.AnthropicConfig{
				APIKey:      "test-key",
				BaseURL:     "https://api.anthropic.com",
				Model:       "claude-3-5-sonnet-20241022",
				Temperature: 0.7,
				MaxTokens:   8192,
				TopP:        1.0,
			},
		}, nil
	})

	provider, err := NewClientProvider(injector)
	if err != nil {
		t.Fatalf("NewClientProvider() error = %v", err)
	}

	ctx := context.Background()
	cnf := &shared.LLMClientConfig{
		Model: "anthropic/claude-3-opus-20250219",
	}

	client, err := provider.GetClient(ctx, cnf)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if client == nil {
		t.Error("GetClient() returned nil client")
	}
}

// TestGetClient_InvalidModelFormat tests GetClient with invalid model format
func TestGetClient_InvalidModelFormat(t *testing.T) {
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (logger.LoggerService, error) {
		return &mockLoggerService{}, nil
	})
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{
			anthropicConfig: &config.AnthropicConfig{
				APIKey: "test-key",
			},
		}, nil
	})

	provider, err := NewClientProvider(injector)
	if err != nil {
		t.Fatalf("NewClientProvider() error = %v", err)
	}

	ctx := context.Background()
	cnf := &shared.LLMClientConfig{
		Model: "invalid-model-format",
	}

	_, err = provider.GetClient(ctx, cnf)
	if err != shared.ErrInvalidModelFormat {
		t.Errorf("GetClient() error = %v, want %v", err, shared.ErrInvalidModelFormat)
	}
}

// TestGetClient_MissingAPIKey tests GetClient when provider API key is not configured
func TestGetClient_MissingAPIKey(t *testing.T) {
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (logger.LoggerService, error) {
		return &mockLoggerService{}, nil
	})
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{
			anthropicConfig: &config.AnthropicConfig{
				APIKey: "",
			},
		}, nil
	})

	provider, err := NewClientProvider(injector)
	if err != nil {
		t.Fatalf("NewClientProvider() error = %v", err)
	}

	ctx := context.Background()
	cnf := &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}

	_, err = provider.GetClient(ctx, cnf)
	if err == nil {
		t.Error("GetClient() expected error for missing API key, got nil")
	}
}
