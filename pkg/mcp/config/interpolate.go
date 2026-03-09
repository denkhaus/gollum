package config

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// interpolateConfigWithTimeout expands shell commands and environment variables
// in config values using the specified timeout for shell command execution.
func interpolateConfigWithTimeout(cfg MCPServerConfig, timeout time.Duration) MCPServerConfig {
	result := cfg
	if result.Env != nil {
		result.Env = interpolateEnvMapWithTimeout(result.Env, timeout)
	}
	if result.Headers != nil {
		result.Headers = interpolateEnvMapWithTimeout(result.Headers, timeout)
	}
	return result
}

// interpolateEnvMapWithTimeout expands all values in a map with timeout.
func interpolateEnvMapWithTimeout(m map[string]string, timeout time.Duration) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = interpolateValueWithTimeout(v, timeout)
	}
	return result
}

// interpolateValueWithTimeout expands $(command) and $VAR patterns in a string.
func interpolateValueWithTimeout(value string, timeout time.Duration) string {
	// First, handle shell command interpolation: $(command)
	value = expandShellCommands(value, timeout)

	// Then, handle environment variable expansion: $VAR or ${VAR}
	value = expandEnvVars(value)

	return value
}

// expandShellCommands executes $(command) patterns and replaces with output
// Commands use the provided timeout to prevent blocking indefinitely.
func expandShellCommands(value string, timeout time.Duration) string {
	// Match $(command) patterns - but avoid $$() which should be literal
	re := regexp.MustCompile(`\$\(([^)]+)\)`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		// Skip if it's escaped with $$
		if strings.HasPrefix(match, "$$(") {
			return match[1:] // Remove one $, keep the rest
		}

		// Extract the command inside $(...)
		cmdStr := re.FindStringSubmatch(match)[1]

		// Create context with timeout for command execution
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Execute the command with timeout
		cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
		output, err := cmd.Output()
		if err != nil {
			// On error or timeout, return the original string
			// This allows the config to load even if a command fails
			return match
		}

		// Trim trailing newlines and whitespace
		return strings.TrimRight(strings.TrimSpace(string(output)), "\n\r")
	})
}

// expandEnvVars expands $VAR and ${VAR} patterns from environment
func expandEnvVars(value string) string {
	// Use os.ExpandEnv which handles $VAR and ${VAR} formats
	// But we need to be careful not to expand $$ which is a literal $
	expanded := os.ExpandEnv(value)
	return expanded
}

// interpolateConfig expands shell commands and environment variables in config values
// Uses a default 5-second timeout for shell command execution.
// Deprecated: Use interpolateConfigWithTimeout for production code.
func interpolateConfig(cfg MCPServerConfig) MCPServerConfig {
	return interpolateConfigWithTimeout(cfg, 5*time.Second)
}

// interpolateEnvMap expands all values in a map with 5-second timeout.
// Deprecated: Use interpolateEnvMapWithTimeout for production code.
func interpolateEnvMap(m map[string]string) map[string]string {
	return interpolateEnvMapWithTimeout(m, 5*time.Second)
}

// interpolateValue expands $(command) and $VAR patterns in a string.
// Uses a default 5-second timeout for shell command execution.
// Deprecated: Use interpolateValueWithTimeout for production code.
func interpolateValue(value string) string {
	return interpolateValueWithTimeout(value, 5*time.Second)
}

// Interpolate applies interpolation to the MCPServerConfig
func (c *MCPServerConfig) Interpolate() {
	c.Env = interpolateEnvMap(c.Env)
	if c.Headers != nil {
		c.Headers = interpolateEnvMap(c.Headers)
	}
}

// GetEnvWithExpansion returns the interpolated env map
func (c *MCPServerConfig) GetEnvWithExpansion() map[string]string {
	return interpolateEnvMap(c.Env)
}

// GetHeadersWithExpansion returns the interpolated headers map
func (c *MCPServerConfig) GetHeadersWithExpansion() map[string]string {
	if c.Headers == nil {
		return nil
	}
	return interpolateEnvMap(c.Headers)
}
