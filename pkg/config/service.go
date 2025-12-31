// Package config provides configuration management for the Gollum application,
// including LLM provider settings (Anthropic, OpenAI, Gemini) and agent limits.
package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/samber/do/v2"
)

// AnthropicConfig holds configuration for Anthropic's Claude API.
type AnthropicConfig struct {
	APIKey  string `envconfig:"API_KEY"`
	BaseURL string `envconfig:"BASE_URL"` // Fixed var-naming: was BaseUrl
	Model   string `envconfig:"MODEL"`
}

// OpenAIConfig holds configuration for OpenAI's GPT API.
type OpenAIConfig struct {
	APIKey  string `envconfig:"API_KEY"`
	BaseURL string `envconfig:"BASE_URL"` // Fixed var-naming: was BaseUrl
	Model   string `envconfig:"MODEL"`
}

// GeminiConfig holds configuration for Google's Gemini API.
type GeminiConfig struct {
	ProjectID string `envconfig:"PROJECT_ID"`
	Location  string `envconfig:"LOCATION"`
}

// AgentLimitsConfig defines limits for agent creation and management.
type AgentLimitsConfig struct {
	MaxSubAgentsPerParent int `envconfig:"MAX_SUB_AGENTS_PER_PARENT" default:"3"`
	MaxTotalAgents        int `envconfig:"MAX_TOTAL_AGENTS" default:"50"`
}

// FilesConfig combines file watcher and file state configuration
type FilesConfig struct {
	// Watcher settings
	WatcherEnabled bool     `envconfig:"WATCHER_ENABLED" default:"true"`
	DebounceMs     int      `envconfig:"WATCHER_DEBOUNCE_MS" default:"100"` // milliseconds
	IgnoredDirs    []string `envconfig:"WATCHER_IGNORE_DIRS" default:".git,node_modules"`

	// Scan settings
	RootDir       string `envconfig:"ROOT_DIR" default:""` // leer = cwd
	MaxDepth      int    `envconfig:"SCAN_MAX_DEPTH" default:"10"`
	IncludeHidden bool   `envconfig:"SCAN_INCLUDE_HIDDEN" default:"false"`
	FilePattern   string `envconfig:"SCAN_FILE_PATTERN" default:"*"`
	Concurrency   int    `envconfig:"SCAN_CONCURRENCY" default:"4"`

	// Cache for quick lookup
	ignoredDirsSet map[string]bool
}

// GetIgnoredDirsSet returns a set of ignored directories for quick lookup
func (c *FilesConfig) GetIgnoredDirsSet() map[string]bool {
	if c.ignoredDirsSet == nil {
		c.ignoredDirsSet = make(map[string]bool)
		for _, dir := range c.IgnoredDirs {
			c.ignoredDirsSet[dir] = true
		}
	}
	return c.ignoredDirsSet
}

// GetDebounceDuration returns the debounce duration as time.Duration
func (c *FilesConfig) GetDebounceDuration() time.Duration {
	return time.Duration(c.DebounceMs) * time.Millisecond
}

// LoggingConfig defines session log buffer configuration.
type LoggingConfig struct {
	// SessionLogBufferSize is the maximum number of log entries to keep in memory.
	// Min: 100, Max: 100000, Default: 1000
	SessionLogBufferSize int `envconfig:"SESSION_LOG_BUFFER_SIZE" default:"1000"`
	// SessionLogEnabled enables or disables in-memory log buffering.
	SessionLogEnabled bool `envconfig:"SESSION_LOG_ENABLED" default:"true"`
}

// BashConfig defines configuration for the Bash tool's file change tracking
type BashConfig struct {
	// TrackChanges enables file change detection for bash commands
	// When enabled, detected changes are logged and returned in the tool response
	// This provides transparency about what files were modified by bash commands
	TrackChanges bool `envconfig:"TRACK_CHANGES" default:"true"`
	// DebounceMs is the time to wait after command execution for the file watcher to process events
	// This allows the background watcher to update file stats before we detect changes
	DebounceMs int `envconfig:"DEBOUNCE_MS" default:"50"` // milliseconds
}

// ConfigService defines the configuration service interface
type ConfigService interface {
	GetLogLevel() string
	IsDevMode() bool
	GetAnthropicConfig() *AnthropicConfig
	GetGeminiConfig() *GeminiConfig
	GetOpenAIConfig() *OpenAIConfig
	GetAgentLimits() *AgentLimitsConfig
	GetFilesConfig() *FilesConfig
	GetLoggingConfig() *LoggingConfig
	GetBashConfig() *BashConfig
}

// service implements the Service interface
type service struct {
	Anthropic   AnthropicConfig   `envconfig:"ANTHROPIC"`
	Gemini      GeminiConfig      `envconfig:"GEMINI"`
	OpenAI      OpenAIConfig      `envconfig:"OPENAI"`
	AgentLimits AgentLimitsConfig `envconfig:"AGENT_LIMITS"`
	Files       FilesConfig       `envconfig:"FILES"`
	Logging     LoggingConfig     `envconfig:"LOGGING"`
	Bash        BashConfig        `envconfig:"BASH"`
	LogLevel    string            `envconfig:"LOG_LEVEL" default:"info"`
	Development bool              `envconfig:"DEVELOPMENT" default:"false"`
}

// NewService creates a new configuration service
func NewService(_ do.Injector) (ConfigService, error) {
	var s service
	err := envconfig.Process("GOLLUM", &s)
	if err != nil {
		return nil, err
	}

	// Validate LoggingConfig
	if s.Logging.SessionLogBufferSize < 100 {
		s.Logging.SessionLogBufferSize = 100
	}
	if s.Logging.SessionLogBufferSize > 100000 {
		s.Logging.SessionLogBufferSize = 100000
	}

	return &s, nil
}

// Implement Service interface methods
func (s *service) GetLogLevel() string {
	return s.LogLevel
}

func (s *service) IsDevMode() bool {
	return s.Development
}

func (s *service) GetGeminiConfig() *GeminiConfig {
	return &s.Gemini
}

func (s *service) GetAnthropicConfig() *AnthropicConfig {
	return &s.Anthropic
}

func (s *service) GetOpenAIConfig() *OpenAIConfig {
	return &s.OpenAI
}

func (s *service) GetAgentLimits() *AgentLimitsConfig {
	return &s.AgentLimits
}

func (s *service) GetFilesConfig() *FilesConfig {
	return &s.Files
}

func (s *service) GetLoggingConfig() *LoggingConfig {
	return &s.Logging
}

func (s *service) GetBashConfig() *BashConfig {
	return &s.Bash
}
