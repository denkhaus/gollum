package main

import (
	"fmt"
	"strings"
)

// ToUpper converts a string to uppercase
func ToUpper(input string) string {
	return strings.ToUpper(input)
}

// ToLower converts a string to lowercase
func ToLower(input string) string {
	return strings.ToLower(input)
}

// Replace replaces all occurrences of old with new
func Replace(input, old, new string) string {
	return strings.ReplaceAll(input, old, new)
}

// Format formats a string with arguments
// args is a map with keys: "template", "values"
func Format(args map[string]any) string {
	template, _ := args["template"].(string)
	values, _ := args["values"].([]string)

	return fmt.Sprintf(template, toAnySlice(values)...)
}

func toAnySlice(slice []string) []any {
	result := make([]any, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}
