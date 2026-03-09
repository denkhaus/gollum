package config

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/logger"
	"go.uber.org/zap"
)

// interpolateConfigWithTimeout expands shell commands and environment variables
// in config values using the specified timeout for shell command execution.
func interpolateConfigWithTimeout(cfg MCPServerConfig, timeout time.Duration, log logger.LoggerService) MCPServerConfig {
	result := cfg
	if result.Env != nil {
		result.Env = interpolateEnvMapWithTimeout(result.Env, timeout, log)
	}
	if result.Headers != nil {
		// Headers may reference variables from the env map
		result.Headers = interpolateHeadersWithTimeout(result.Headers, result.Env, timeout, log)
	}
	return result
}

// interpolateEnvMapWithTimeout expands all values in a map with timeout.

// interpolateHeadersWithTimeout expands header values with access to env map.
// Headers can reference variables from the env map (e.g., $API_TOKEN).
// Also supports OS environment variables for backward compatibility.
func interpolateHeadersWithTimeout(headers map[string]string, env map[string]string, timeout time.Duration, log logger.LoggerService) map[string]string {
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		// First interpolate shell commands
		interpolated := expandShellCommands(v, timeout, log)
		// Then interpolate from the config's env map first
		interpolated = expandFromMap(interpolated, env)
		// Finally, expand any remaining OS env variables (for backward compatibility)
		interpolated = expandEnvVars(interpolated)
		result[k] = interpolated
	}
	return result
}

// interpolateEnvMapWithTimeout expands all values in a map with timeout.
// Supports cross-references between keys in the same map (e.g., $PORT in another value).
func interpolateEnvMapWithTimeout(m map[string]string, timeout time.Duration, log logger.LoggerService) map[string]string {
	// First pass: interpolate only shell commands (not OS env vars yet)
	shellPass := make(map[string]string, len(m))
	for k, v := range m {
		shellPass[k] = expandShellCommands(v, timeout, log)
	}

	// Second pass: interpolate cross-references within the map
	// This allows values to reference other keys in the same map
	crossRefPass := make(map[string]string, len(m))
	for k, v := range shellPass {
		crossRefPass[k] = expandFromMap(v, shellPass)
	}

	// Third pass: interpolate OS environment variables
	// This allows values to reference OS env vars as well
	result := make(map[string]string, len(m))
	for k, v := range crossRefPass {
		result[k] = expandEnvVars(v)
	}

	return result
}

// interpolateValueWithTimeout expands $(command) and $VAR patterns in a string.
func interpolateValueWithTimeout(value string, timeout time.Duration, log logger.LoggerService) string {
	// First, handle shell command interpolation: $(command)
	value = expandShellCommands(value, timeout, log)

	// Then, handle environment variable expansion: $VAR or ${VAR}
	value = expandEnvVars(value)

	return value
}

// expandShellCommands executes $(command) patterns and replaces with output
// Commands use the provided timeout to prevent blocking indefinitely.
// Logs errors using the provided logger.
func expandShellCommands(value string, timeout time.Duration, log logger.LoggerService) string {
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
			// Log error but return original string to allow config to load
			log.Warn("shell command interpolation failed",
				zap.String("command", cmdStr),
				zap.Error(err),
				zap.String("original_value", match),
			)
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

// expandFromMap expands $VAR and ${VAR} patterns from a provided map
// This allows cross-referencing values within the same config map
func expandFromMap(value string, vars map[string]string) string {
	// Debug: print input
	// fmt.Printf("[DEBUG] expandFromMap called: value=%q, vars=%v\n", value, vars)

	// Use a custom replacer that looks up values in the provided map
	re := regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*)|\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		// Debug: print what we're trying to match
		// fmt.Printf("[DEBUG] expandFromMap: match=%q, value=%q, vars=%v\n", match, value, vars)

		// Check if it's ${VAR} format
		if strings.HasPrefix(match, "${") {
			// Extract var name between ${ and }
			varName := match[2 : len(match)-1]
			// fmt.Printf("[DEBUG] Braced format: varName=%q, looking in vars...\n", varName)
			if val, ok := vars[varName]; ok {
				// fmt.Printf("[DEBUG] Found: %q\n", val)
				return val
			}
			// fmt.Printf("[DEBUG] Not found in vars\n")
			return match // Return original if not found
		}

		// Otherwise it's $VAR format - extract var name after $
		varName := match[1:]
		// fmt.Printf("[DEBUG] Unbraced format: varName=%q, looking in vars...\n", varName)
		if val, ok := vars[varName]; ok {
			// fmt.Printf("[DEBUG] Found: %q\n", val)
			return val
		}
		// fmt.Printf("[DEBUG] Not found in vars\n")
		return match // Return original if not found
	})
}
