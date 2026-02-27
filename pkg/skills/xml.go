package skills

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// XMLSkill represents a skill in OpenAI function-calling XML format
type XMLSkill struct {
	XMLName     xml.Name `xml:"skill"`
	Name        string   `xml:"name,attr"`
	Description string   `xml:"description,attr,omitempty"`
	Type        string   `xml:"type,attr,omitempty"`
	Arguments   string   `xml:"arguments,omitempty"`
	Content     string   `xml:",chardata"`
}

// XMLSkills represents a collection of skills in XML format
type XMLSkills struct {
	XMLName xml.Name   `xml:"skills"`
	Skills  []XMLSkill `xml:"skill"`
}

// ToPromptXML converts a skill to XML format for LLM prompts
func (s *Skill) ToPromptXML() string {
	// Build XML manually for better formatting
	var sb strings.Builder
	sb.WriteString("<skill")
	sb.WriteString(fmt.Sprintf(` name="%s"`, escapeXML(s.Name)))
	if s.Description != "" {
		sb.WriteString(fmt.Sprintf(` description="%s"`, escapeXML(s.Description)))
	}
	if s.Type != "" {
		sb.WriteString(fmt.Sprintf(` type="%s"`, string(s.Type)))
	}
	sb.WriteString(">")

	if s.Arguments != "" {
		sb.WriteString("\n  <arguments>")
		sb.WriteString(escapeXML(s.Arguments))
		sb.WriteString("</arguments>")
	}

	if s.Content != "" {
		sb.WriteString("\n  <content>")
		sb.WriteString(escapeXML(s.Content))
		sb.WriteString("</content>")
	}

	sb.WriteString("\n</skill>")
	return sb.String()
}

// ToOpenAIFunction converts a skill to OpenAI function-calling format
func (s *Skill) ToOpenAIFunction() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        s.Name,
			"description": s.DisplayName(),
			"parameters": map[string]any{
				"type":                 "object",
				"properties":           parseArgumentsToProperties(s.Arguments),
				"additionalProperties": false,
			},
		},
	}
}

// ToPromptXML converts a collection of skills to XML format
func (s Skills) ToPromptXML() string {
	if len(s) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<skills>")
	for _, skill := range s {
		sb.WriteString("\n")
		sb.WriteString(skill.ToPromptXML())
	}
	sb.WriteString("\n</skills>")
	return sb.String()
}

// ToOpenAIFunctions converts a collection of skills to OpenAI function-calling format
func (s Skills) ToOpenAIFunctions() []map[string]any {
	functions := make([]map[string]any, 0, len(s))
	for _, skill := range s {
		functions = append(functions, skill.ToOpenAIFunction())
	}
	return functions
}

// ToUserInvocableXML returns XML for all user-invocable skills
func (s Skills) ToUserInvocableXML() string {
	invocable := s.FilterUserInvocable()
	return invocable.ToPromptXML()
}

// jsonTypeString is the default JSON schema type for string values
const jsonTypeString = "string"

// escapeXML escapes special characters for XML content
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// parseArgumentsToProperties converts argument specification to JSON schema properties
// This is a simplified implementation - can be extended for full JSON schema support
func parseArgumentsToProperties(argsSpec string) map[string]any {
	if argsSpec == "" {
		return map[string]any{}
	}

	// Simple parsing: each line is "name: type description"
	properties := make(map[string]any)
	lines := strings.Split(argsSpec, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse "name: type description" format
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		typeDesc := strings.TrimSpace(parts[1])

		// Determine type
		jsonType := jsonTypeString // default
		desc := typeDesc

		typeParts := strings.SplitN(typeDesc, " ", 2)
		if len(typeParts) >= 1 {
			switch strings.ToLower(typeParts[0]) {
			case "string", "str":
				jsonType = jsonTypeString
				if len(typeParts) > 1 {
					desc = typeParts[1]
				}
			case "int", "integer", "number", "float":
				jsonType = "number"
				if len(typeParts) > 1 {
					desc = typeParts[1]
				}
			case "bool", "boolean":
				jsonType = "boolean"
				if len(typeParts) > 1 {
					desc = typeParts[1]
				}
			case "array", "list":
				jsonType = "array"
				if len(typeParts) > 1 {
					desc = typeParts[1]
				}
			case "object", "map":
				jsonType = "object"
				if len(typeParts) > 1 {
					desc = typeParts[1]
				}
			}
		}

		// Note: Required fields (marked with * suffix on name) could be tracked here
		// but are not currently included in the output for simplicity
		_ = strings.HasSuffix(name, "*") // Acknowledge required marker exists

		properties[name] = map[string]any{
			"type":        jsonType,
			"description": desc,
		}
	}

	return properties
}
