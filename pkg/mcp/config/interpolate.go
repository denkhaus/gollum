package config

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// interpolateConfig expands shell commands and environment variables in config values
func interpolateConfig(cfg MCPServerConfig) MCPServerConfig {
	result := cfg
	if result.Env != nil {
		result.Env = interpolateEnvMap(result.Env)
	}
	if result.Headers != nil {
		result.Headers = interpolateEnvMap(result.Headers)
	}
	return result
}

// interpolateEnvMap expands all values in a map
func interpolateEnvMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = interpolateValue(v)
	}
	return result
}

// interpolateValue expands $(command) and $VAR patterns in a string
func interpolateValue(value string) string {
	// First, handle shell command interpolation: $(command)
	value = expandShellCommands(value)

	// Then, handle environment variable expansion: $VAR or ${VAR}
	value = expandEnvVars(value)

	return value
}

// expandShellCommands executes $(command) patterns and replaces with output
func expandShellCommands(value string) string {
	// Match $(command) patterns - but avoid $$() which should be literal
	re := regexp.MustCompile(`\$\(([^)]+)\)`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		// Skip if it's escaped with $$
		if strings.HasPrefix(match, "$$(") {
			return match[1:] // Remove one $, keep the rest
		}

		// Extract the command inside $(...)
		cmdStr := re.FindStringSubmatch(match)[1]

		// Execute the command
		cmd := exec.Command("sh", "-c", cmdStr)
		output, err := cmd.Output()
		if err != nil {
			// On error, return the original string
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
