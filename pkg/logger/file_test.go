package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestJSONLogFormat verifies that file logs are written in JSON format
func TestJSONLogFormat(t *testing.T) {
	// Create temp directory for logs
	tmpDir := t.TempDir()
	sessionID := uuid.New()

	// Create a minimal logger service
	logger, err := zap.NewProduction()
	require.NoError(t, err)

	svc := &service{
		logger:         logger,
		atomicLevel:    zap.NewAtomicLevelAt(zap.InfoLevel),
		config:        zap.NewProductionConfig(),
		logBuffer:      newLogBuffer(1000, false), // disabled for test
		configService: &mockConfigService{},
	}

	// Enable file logging
	logErr := svc.EnableFileLogging(tmpDir, sessionID)
	require.NoError(t, logErr)
	require.NoError(t, err)
	defer svc.CloseFileLogging()

	// Log with agent ID (using InfoWithAgent)
	agentID := uuid.New()
	svc.InfoWithAgent("Test message", agentID, zap.String("test_field", "test_value"))

	// Flush to ensure write
	err = svc.Flush()
	require.NoError(t, err)

	// Read the log file (it's in the logs/ subdirectory)
	logPath := filepath.Join(tmpDir, "logs", sessionID.String()+".log")
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	// Verify it's valid JSON
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Greater(t, len(lines), 1, "Log file should have at least two lines")

	// First line is "Session logging enabled", second line is our test message
	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(lines[1]), &logEntry)
	require.NoError(t, err, "Log line should be valid JSON")

	// Verify required fields
	assert.Contains(t, logEntry, "timestamp")
	assert.Contains(t, logEntry, "level")
	assert.Contains(t, logEntry, "message")
	assert.Contains(t, logEntry, "agent_id")
	assert.Contains(t, logEntry, "test_field")
	assert.Equal(t, "Test message", logEntry["message"])
	assert.Equal(t, agentID.String(), logEntry["agent_id"])
	assert.Equal(t, "test_value", logEntry["test_field"])
}

// mockConfigService for testing
type mockConfigService struct{}

func (m *mockConfigService) IsDevMode() bool                                      { return false }
func (m *mockConfigService) GetLogLevel() string                                   { return "info" }
func (m *mockConfigService) GetConfig() interface{}                                { return nil }
func (m *mockConfigService) GetLLMConfig(name string) (interface{}, bool)         { return nil, false }
func (m *mockConfigService) GetLLMConfigs() map[string]interface{}                { return nil }
func (m *mockConfigService) GetLoggingConfig() *config.LoggingConfig {
	return &config.LoggingConfig{
		SessionLogEnabled:    false,
		SessionLogBufferSize: 1000,
		MaxSessionLogFiles:   10,
	}
}
func (m *mockConfigService) GetBashConfig() *config.BashConfig { return &config.BashConfig{} }
func (m *mockConfigService) GetAgentLimits() *config.AgentLimitsConfig {
	return &config.AgentLimitsConfig{}
}
func (m *mockConfigService) GetFilesConfig() *config.FilesConfig {
	return &config.FilesConfig{}
}
func (m *mockConfigService) GetAnthropicConfig() *config.AnthropicConfig {
	return &config.AnthropicConfig{}
}
func (m *mockConfigService) GetGeminiConfig() *config.GeminiConfig {
	return &config.GeminiConfig{}
}
func (m *mockConfigService) GetOpenAIConfig() *config.OpenAIConfig {
	return &config.OpenAIConfig{}
}
func (m *mockConfigService) GetHooksConfig() *config.HooksConfig {
	return &config.HooksConfig{}
}
func (m *mockConfigService) GetPromptStoreConfig() *config.PromptStoreConfig {
	return &config.PromptStoreConfig{}
}
func (m *mockConfigService) GetPromptOptimizerConfig() *config.PromptOptimizerConfig {
	return &config.PromptOptimizerConfig{}
}
func (m *mockConfigService) GetLangfuseConfig() *config.LangfuseConfig {
	return &config.LangfuseConfig{}
}
func (m *mockConfigService) GetEventsConfig() *config.EventsConfig {
	return &config.EventsConfig{}
}
func (m *mockConfigService) GetMCPConfig() *config.MCPConfig {
	return &config.MCPConfig{}
}