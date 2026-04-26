// Package config provides configuration management for the Gollum application,
// including LLM provider settings (Anthropic, OpenAI, Gemini) and agent limits.
package config

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Driver string // "sqlite", "postgres", "mysql"
	DSN    string // Database connection string
}

// GetDatabaseConfig returns the database configuration.
func (c *serviceImpl) GetDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Driver: c.Database.Driver,
		DSN:    c.Database.DSN,
	}
}
