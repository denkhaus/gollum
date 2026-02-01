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
}

// HooksConfig defines configuration for built-in hooks system
type HooksConfig struct {
	// Logging enables the LoggingHook that logs all hook events
	LoggingEnabled bool `envconfig:"LOGGING_ENABLED" default:"true"`
	// LoggingLevel controls the log level for LoggingHook (debug, info, warn, error)
	LoggingLevel string `envconfig:"LOGGING_LEVEL" default:"info"`
	// SecurityMode controls SecurityHook strictness (strict, lenient)
	// strict: blocks potentially dangerous operations, lenient: only logs warnings
	SecurityMode string `envconfig:"SECURITY_MODE" default:"lenient"`
	// MetricsEnabled enables the MetricsHook for Prometheus-compatible metrics
	MetricsEnabled bool `envconfig:"METRICS_ENABLED" default:"false"`
	// AuditEnabled enables the AuditHook for append-only audit trail
	AuditEnabled bool `envconfig:"AUDIT_ENABLED" default:"false"`
}

// GetSecurityMode returns the security mode with validation
func (c *HooksConfig) GetSecurityMode() string {
	switch c.SecurityMode {
	case "strict", "lenient":
		return c.SecurityMode
	default:
		return "lenient"
	}
}

type PromptStoreType string

const (
	PromptStoreTypeMemory   PromptStoreType = "memory"
	PromptStoreTypeFile     PromptStoreType = "file"
	PromptStoreTypeLangfuse PromptStoreType = "langfuse"
)

// PromptStoreConfig contains configuration for prompt store implementations.
type PromptStoreConfig struct {
	// Type of store backend ("memory", "file", "langfuse")
	Type PromptStoreType `envconfig:"default" env:"PROMPT_STORE_TYPE"`

	// FilePath for file-based storage
	FilePath string `envconfig:"default:\"./data/prompts\"" env:"PROMPT_STORE_FILE_PATH"`

	// Langfuse configuration for Langfuse backend
	LangfusePublicKey string `envconfig:"" env:"LANGFUSE_PUBLIC_KEY"`
	LangfuseSecretKey string `envconfig:"" env:"LANGFUSE_SECRET_KEY"`
	LangfuseHost      string `envconfig:"default:\"https://cloud.langfuse.com\"" env:"LANGFUSE_HOST"`

	// CacheEnabled enables caching for store operations
	CacheEnabled bool `envconfig:"default" env:"PROMPT_STORE_CACHE_ENABLED"`
}

// ConfigService defines the configuration service interface
//
//revive:disable-next-line:exported
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
	GetHooksConfig() *HooksConfig
	GetPromptStoreConfig() *PromptStoreConfig
}

// serviceImpl implements the ConfigService interface
type serviceImpl struct {
	Anthropic   AnthropicConfig   `envconfig:"ANTHROPIC"`
	Gemini      GeminiConfig      `envconfig:"GEMINI"`
	OpenAI      OpenAIConfig      `envconfig:"OPENAI"`
	AgentLimits AgentLimitsConfig `envconfig:"AGENT_LIMITS"`
	Files       FilesConfig       `envconfig:"FILES"`
	Logging     LoggingConfig     `envconfig:"LOGGING"`
	Bash        BashConfig        `envconfig:"BASH"`
	Hooks       HooksConfig       `envconfig:"HOOKS"`
	LogLevel    string            `envconfig:"LOG_LEVEL" default:"info"`
	Development bool              `envconfig:"DEVELOPMENT" default:"false"`
	PromptStore PromptStoreConfig `envconfig:"PROMPT_STORE"`
}

// NewService creates a new configuration service
func NewService(_ do.Injector) (ConfigService, error) {
	var s serviceImpl
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
func (s *serviceImpl) GetLogLevel() string {
	return s.LogLevel
}

func (s *serviceImpl) IsDevMode() bool {
	return s.Development
}

func (s *serviceImpl) GetGeminiConfig() *GeminiConfig {
	return &s.Gemini
}

func (s *serviceImpl) GetAnthropicConfig() *AnthropicConfig {
	return &s.Anthropic
}

func (s *serviceImpl) GetOpenAIConfig() *OpenAIConfig {
	return &s.OpenAI
}

func (s *serviceImpl) GetAgentLimits() *AgentLimitsConfig {
	return &s.AgentLimits
}

func (s *serviceImpl) GetFilesConfig() *FilesConfig {
	return &s.Files
}

func (s *serviceImpl) GetLoggingConfig() *LoggingConfig {
	return &s.Logging
}

func (s *serviceImpl) GetBashConfig() *BashConfig {
	return &s.Bash
}

func (s *serviceImpl) GetHooksConfig() *HooksConfig {
	return &s.Hooks
}

func (s *serviceImpl) GetPromptStoreConfig() *PromptStoreConfig {
	return &s.PromptStore
}
