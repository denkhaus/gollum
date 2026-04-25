// Package tools provides tool implementations for the Gollum agent system.
package tools

import (
	"fmt"
	"path/filepath"

	"github.com/denkhaus/gollum/pkg/shared"
)

// ToolRequestParams is a type alias for tool request arguments with convenience methods
// for type-safe parameter extraction. It wraps map[string]any to provide helpers
// for common patterns like getting strings, bools, ints, and file paths.
type ToolRequestParams map[string]any

// Has returns true if the key exists in the params.
func (p ToolRequestParams) Has(key shared.ToolParamKeys) bool {
	_, exists := p[string(key)]
	return exists
}

// GetString returns a string value for the given key, or the default if not present or not a string.
func (p ToolRequestParams) GetString(key shared.ToolParamKeys, def string) string {
	if val, ok := p[string(key)].(string); ok {
		return val
	}
	return def
}

// GetStringAllowEmpty returns a string value for the given key.
// Returns an error response if the key is missing or not a string type.
// Unlike MustGetString, this allows empty strings as valid values.
// This is useful for parameters that can be empty (like file content) but must be provided.
func (p ToolRequestParams) GetStringAllowEmpty(key shared.ToolParamKeys) (string, map[string]any) {
	val, exists := p[string(key)]
	if !exists {
		return "", ErrorResponse("%s is required and must be a string", key)
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	// Key exists but wrong type
	return "", ErrorResponse("%s is required and must be a string", key)
}

// GetBool returns a bool value for the given key, or the default if not present or not a bool.
func (p ToolRequestParams) GetBool(key shared.ToolParamKeys, def bool) bool {
	if val, ok := p[string(key)].(bool); ok {
		return val
	}
	return def
}

// MustGetString returns a string value for the given key.
// Returns an error response map if the key is missing or not a string.
func (p ToolRequestParams) MustGetString(key shared.ToolParamKeys) (string, map[string]any) {
	val, ok := p[string(key)].(string)
	if !ok || val == "" {
		return "", ErrorResponse("%s is required and must be a non-empty string", key)
	}
	return val, nil
}

// MustGetBool returns a bool value for the given key.
// Returns an error response map if the key is missing or not a bool.
func (p ToolRequestParams) MustGetBool(key shared.ToolParamKeys) (bool, map[string]any) {
	val, ok := p[string(key)].(bool)
	if !ok {
		return false, ErrorResponse("%s is required and must be a boolean", key)
	}
	return val, nil
}

// GetInt returns an int value for the given key (from float64 JSON representation or int),
// or the default if not present or not a number.
func (p ToolRequestParams) GetInt(key shared.ToolParamKeys, def int) int {
	if val, ok := p[string(key)].(float64); ok {
		return int(val)
	}
	if val, ok := p[string(key)].(int); ok {
		return val
	}
	return def
}

// MustGetInt returns an int value for the given key (from float64 JSON representation or int).
// Returns an error response map if the key is missing or not a number.
func (p ToolRequestParams) MustGetInt(key shared.ToolParamKeys) (int, map[string]any) {
	if val, ok := p[string(key)].(float64); ok {
		return int(val), nil
	}
	if val, ok := p[string(key)].(int); ok {
		return val, nil
	}
	return 0, ErrorResponse("%s is required and must be a number", key)
}

// GetInt64 returns an int64 value for the given key (from float64 JSON representation),
// or the default if not present or not a number.
func (p ToolRequestParams) GetInt64(key shared.ToolParamKeys, def int64) int64 {
	if val, ok := p[string(key)].(float64); ok {
		return int64(val)
	}
	return def
}

// MustGetInt64 returns an int64 value for the given key (from float64 JSON representation).
// Returns an error response map if the key is missing or not a number.
func (p ToolRequestParams) MustGetInt64(key shared.ToolParamKeys) (int64, map[string]any) {
	val, ok := p[string(key)].(float64)
	if !ok {
		return 0, ErrorResponse("%s is required and must be a number", key)
	}
	return int64(val), nil
}

// GetFloat returns a float64 value for the given key, or the default if not present.
func (p ToolRequestParams) GetFloat(key shared.ToolParamKeys, def float64) float64 {
	if val, ok := p[string(key)].(float64); ok {
		return val
	}
	return def
}

// MustGetFloat returns a float64 value for the given key.
// Returns an error response map if the key is missing or not a number.
func (p ToolRequestParams) MustGetFloat(key shared.ToolParamKeys) (float64, map[string]any) {
	val, ok := p[string(key)].(float64)
	if !ok {
		return 0, ErrorResponse("%s is required and must be a number", key)
	}
	return val, nil
}

// GetFilePath returns an absolute file path for the given key.
// Returns an error response if the key is missing, empty, or cannot be resolved.
func (p ToolRequestParams) GetFilePath(key shared.ToolParamKeys) (string, map[string]any) {
	path, errResp := p.MustGetString(key)
	if errResp != nil {
		return "", errResp
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", ErrorResponse("failed to resolve absolute path for %s: %v", key, err)
	}

	return absPath, nil
}

// GetFilePathOrDefault returns an absolute file path for the given key,
// or the default path resolved to absolute if the key is not present.
func (p ToolRequestParams) GetFilePathOrDefault(key shared.ToolParamKeys, def string) (string, error) {
	path := p.GetString(key, def)
	if path == "" {
		path = def
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absPath, nil
}

// GetStringSlice returns a string slice for the given key.
// Handles both []string and []any (JSON arrays).
func (p ToolRequestParams) GetStringSlice(key shared.ToolParamKeys) []string {
	if val, ok := p[string(key)].([]string); ok {
		return val
	}

	if val, ok := p[string(key)].([]any); ok {
		result := make([]string, 0, len(val))
		for _, v := range val {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	return nil
}

// MustGetStringSlice returns a string slice for the given key.
// Returns an error response map if the key is missing or not an array.
func (p ToolRequestParams) MustGetStringSlice(key shared.ToolParamKeys) ([]string, map[string]any) {
	if !p.Has(key) {
		return nil, ErrorResponse("%s is required and must be an array", key)
	}

	if val, ok := p[string(key)].([]string); ok {
		return val, nil
	}

	if val, ok := p[string(key)].([]any); ok {
		result := make([]string, 0, len(val))
		for i, v := range val {
			if s, ok := v.(string); ok {
				result = append(result, s)
			} else {
				return nil, ErrorResponse("%s[%d] must be a string", key, i)
			}
		}
		return result, nil
	}

	return nil, ErrorResponse("%s must be an array of strings", key)
}

// GetStringMap returns a map[string]string for the given key.
// Handles both map[string]string and map[string]any (JSON objects).
func (p ToolRequestParams) GetStringMap(key shared.ToolParamKeys) map[string]string {
	if val, ok := p[string(key)].(map[string]string); ok {
		return val
	}

	if val, ok := p[string(key)].(map[string]any); ok {
		result := make(map[string]string, len(val))
		for k, v := range val {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
		return result
	}

	return nil
}

// MustGetStringMap returns a map[string]string for the given key.
// Returns an error response map if the key is missing or not an object.
func (p ToolRequestParams) MustGetStringMap(key shared.ToolParamKeys) (map[string]string, map[string]any) {
	if !p.Has(key) {
		return nil, ErrorResponse("%s is required and must be an object", key)
	}

	if val, ok := p[string(key)].(map[string]string); ok {
		return val, nil
	}

	if val, ok := p[string(key)].(map[string]any); ok {
		result := make(map[string]string, len(val))
		for k, v := range val {
			if s, ok := v.(string); ok {
				result[k] = s
			} else {
				return nil, ErrorResponse("%s.%s must be a string", key, k)
			}
		}
		return result, nil
	}

	return nil, ErrorResponse("%s must be an object with string values", key)
}

// ErrorResponse creates a standard error response map.
func ErrorResponse(format string, args ...any) map[string]any {
	return map[string]any{
		string(shared.KeySuccess): false,
		string(shared.KeyError):   fmt.Sprintf(format, args...),
	}
}

// SuccessResponse creates a standard success response map with optional additional data.
func SuccessResponse(data map[string]any) map[string]any {
	if data == nil {
		data = make(map[string]any)
	}
	data[string(shared.KeySuccess)] = true
	return data
}

// ToolResponse creates a response map with success status and optional error.
// If err is provided, returns an error response; otherwise returns success with data.
func ToolResponse(data map[string]any, err error) map[string]any {
	if err != nil {
		return ErrorResponse("%s", err.Error())
	}
	return SuccessResponse(data)
}
