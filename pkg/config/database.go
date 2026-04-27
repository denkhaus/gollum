// Package config provides configuration management for the Gollum application,
// including LLM provider settings (Anthropic, OpenAI, Gemini) and agent limits.
package config

import "path/filepath"

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Driver string // "sqlite3", "postgres", "mysql"
	DSN    string // Database connection string
}

// GetDatabaseConfig returns the database configuration with defaults.
// Defaults to SQLite with a file in the .gollum workspace directory if not configured.
func (c *serviceImpl) GetDatabaseConfig() DatabaseConfig {
	driver := c.Database.Driver
	dsn := c.Database.DSN

	// Apply defaults if not configured
	if driver == "" {
		driver = "sqlite3" // Ent uses "sqlite3" not "sqlite"
	}
	if dsn == "" {
		// Default to SQLite in .gollum workspace directory
		dsn = filepath.Join(".gollum", "gollum.db")
	}

	return DatabaseConfig{
		Driver: driver,
		DSN:    dsn,
	}
}
