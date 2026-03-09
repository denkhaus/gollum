// Package config provides configuration management for the Gollum application,
// including LLM provider settings (Anthropic, OpenAI, Gemini) and agent limits.
package config

import (
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/kelseyhightower/envconfig"
	"github.com/samber/do/v2"
)

// AnthropicConfig holds configuration for Anthropic's Claude API.
type AnthropicConfig struct {
	APIKey  string `envconfig:"API_KEY"`
	BaseURL string `envconfig:"BASE_URL"`
	Model   string `envconfig:"MODEL"`
}

// OpenAIConfig holds configuration for OpenAI's GPT API.
type OpenAIConfig struct {
	APIKey  string `envconfig:"API_KEY"`
	BaseURL string `envconfig:"BASE_URL"`
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
	// MaxSessionLogFiles is the maximum number of session log files to retain.
	// Min: 0 (unlimited), Max: 100, Default: 10
	// When set to 0, no automatic cleanup is performed.
	MaxSessionLogFiles int `envconfig:"MAX_SESSION_LOG_FILES" default:"10"`
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

	// Langfuse tracing configuration
	// LangfuseEnabled enables Langfuse tracing for LLM, tool, and agent operations
	LangfuseEnabled bool `envconfig:"LANGFUSE_ENABLED" default:"false"`
	// LangfuseHost is the Langfuse server host URL
	LangfuseHost string `envconfig:"LANGFUSE_HOST" default:"https://cloud.langfuse.com"`
	// LangfusePublicKey is the Langfuse public key for authentication
	LangfusePublicKey string `envconfig:"LANGFUSE_PUBLIC_KEY"`
	// LangfuseSecretKey is the Langfuse secret key for authentication
	LangfuseSecretKey string `envconfig:"LANGFUSE_SECRET_KEY"`
	// LangfuseFlushInterval is the flush interval in milliseconds
	LangfuseFlushInterval int `envconfig:"LANGFUSE_FLUSH_INTERVAL" default:"1000"`
	// LangfuseMaxQueueSize is the maximum queue size for buffered traces
	LangfuseMaxQueueSize int `envconfig:"LANGFUSE_MAX_QUEUE_SIZE" default:"100"`
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

// LangfuseConfig is an alias for HooksConfig to provide clearer API for tracing config
type LangfuseConfig = HooksConfig

// PromptStoreType represents the type of prompt store backend.
type PromptStoreType string

const (
	// PromptStoreTypeMemory stores prompts in memory (ephemeral).
	PromptStoreTypeMemory PromptStoreType = "memory"
	// PromptStoreTypeFile stores prompts in JSON files on disk.
	PromptStoreTypeFile PromptStoreType = "file"
	// PromptStoreTypeLangfuse stores prompts in Langfuse service (deferred to v2).
	PromptStoreTypeLangfuse PromptStoreType = "langfuse"
)

// PromptStoreConfig contains configuration for prompt store implementations.
type PromptStoreConfig struct {
	// Type of store backend ("memory", "file", "langfuse")
	Type PromptStoreType `envconfig:"TYPE" default:"memory"`

	// FilePath for file-based storage
	FilePath string `envconfig:"FILE_PATH" default:"./data/prompts"`

	// NOTE: Langfuse configuration fields (LangfusePublicKey, LangfuseSecretKey, LangfuseHost)
	// were previously here but have been migrated to HooksConfig for tracing purposes.
	// Langfuse prompt store integration is deferred to v2 scope.

	// CacheEnabled enables caching for store operations
	CacheEnabled bool `envconfig:"CACHE_ENABLED" default:"false"`
}

// PromptOptimizerConfig holds configuration for the prompt optimizer
type PromptOptimizerConfig struct {
	// DefaultStrategy is the optimization strategy to use (gradient, metaprompt, prompt_memory)
	DefaultStrategy shared.OptimizerStrategy `envconfig:"STRATEGY" default:"gradient"`

	// DefaultProvider is the LLM provider to use (anthropic, openai, gemini)
	DefaultProvider shared.LLMProvider `envconfig:"PROVIDER" default:"anthropic"`

	// MaxReflectionSteps is the maximum number of reflection iterations
	MaxReflectionSteps int `envconfig:"MAX_REFLECTION" default:"5"`

	// MinReflectionSteps is the minimum number of reflection iterations
	MinReflectionSteps int `envconfig:"MIN_REFLECTION" default:"2"`
}

// EventsConfig holds configuration for the event bus.
type EventsConfig struct {
	// MaxRetries is the maximum number of retry attempts for failed handlers.
	MaxRetries int `envconfig:"MAX_RETRIES" default:"3"`

	// RetryDelayMs is the initial delay between retries in milliseconds.
	RetryDelayMs int `envconfig:"RETRY_DELAY_MS" default:"100"`

	// RetryBackoff is the multiplier for exponential backoff.
	RetryBackoff int `envconfig:"RETRY_BACKOFF" default:"2"`
}

// MCPConfig holds configuration for MCP (Model Context Protocol) discovery service.
type MCPConfig struct {
	// CommandTimeoutSeconds is the timeout in seconds for shell command execution
	// during interpolation of $(command) patterns in mcp.json config files.
	// Default: 5 seconds. Min: 1, Max: 60.
	CommandTimeoutSeconds int `envconfig:"COMMAND_TIMEOUT_SECONDS" default:"5"`
}

// GetCommandTimeout returns the command timeout as time.Duration with validation.
// Ensures timeout is between 1 and 60 seconds.
func (c *MCPConfig) GetCommandTimeout() time.Duration {
	timeout := c.CommandTimeoutSeconds
	if timeout < 1 {
		timeout = 1
	}
	if timeout > 60 {
		timeout = 60
	}
	return time.Duration(timeout) * time.Second
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
	GetPromptOptimizerConfig() *PromptOptimizerConfig
	GetLangfuseConfig() *LangfuseConfig
	GetEventsConfig() *EventsConfig
	GetMCPConfig() *MCPConfig
}

// serviceImpl implements the ConfigService interface
type serviceImpl struct {
	Anthropic       AnthropicConfig       `envconfig:"ANTHROPIC"`
	Gemini          GeminiConfig          `envconfig:"GEMINI"`
	OpenAI          OpenAIConfig          `envconfig:"OPENAI"`
	AgentLimits     AgentLimitsConfig     `envconfig:"AGENT_LIMITS"`
	Files           FilesConfig           `envconfig:"FILES"`
	Logging         LoggingConfig         `envconfig:"LOGGING"`
	Bash            BashConfig            `envconfig:"BASH"`
	Hooks           HooksConfig           `envconfig:"HOOKS"`
	LogLevel        string                `envconfig:"LOG_LEVEL" default:"info"`
	Development     bool                  `envconfig:"DEVELOPMENT" default:"false"`
	PromptStore     PromptStoreConfig     `envconfig:"PROMPT_STORE"`
	PromptOptimizer PromptOptimizerConfig `envconfig:"OPTIMIZER"`
	Events          EventsConfig          `envconfig:"EVENTS"`
	MCP             MCPConfig             `envconfig:"MCP"`
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
	if s.Logging.MaxSessionLogFiles < 0 {
		s.Logging.MaxSessionLogFiles = 0
	}
	if s.Logging.MaxSessionLogFiles > 100 {
		s.Logging.MaxSessionLogFiles = 100
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

func (s *serviceImpl) GetPromptOptimizerConfig() *PromptOptimizerConfig {
	return &s.PromptOptimizer
}

func (s *serviceImpl) GetLangfuseConfig() *LangfuseConfig {
	return (*LangfuseConfig)(&s.Hooks)
}

func (s *serviceImpl) GetEventsConfig() *EventsConfig {
	return &s.Events
}

func (s *serviceImpl) GetMCPConfig() *MCPConfig {
	return &s.MCP
}
