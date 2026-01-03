package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/denkhaus/action-golang/magefiles/helpers"
)

var (
	print = helpers.NewPrinter()
)

// GitHubEvent represents the GitHub webhook event structure
type GitHubEvent struct {
	Comment     *Comment     `json:"comment,omitempty"`
	Issue       *Issue       `json:"issue,omitempty"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`
}

type Comment struct {
	Body string `json:"body"`
}

type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// EventResult contains the extracted information from a GitHub event
type EventResult struct {
	Type    string // "issue", "issue_comment", "pull_request"
	Title   string
	Body    string
	Comment string // For issue_comment events
	Number  int
}

// ReadGitHubEvent reads the GitHub event from the available sources
// Priority: GITHUB_EVENT_PATH > /tmp/event.json
// Returns the raw JSON data and the source path
func ReadGitHubEvent() ([]byte, string, error) {
	// Try GITHUB_EVENT_PATH first (GitHub Actions provides this)
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath != "" {
		print.Info("Reading event from: %s", eventPath)
		bytes, err := os.ReadFile(eventPath)
		if err != nil {
			return nil, "", fmt.Errorf("could not read event from %s: %w", eventPath, err)
		}
		// Also write to /tmp/event.json for backward compatibility
		_ = os.WriteFile("/tmp/event.json", bytes, 0644)
		return bytes, eventPath, nil
	}

	// Fallback: Try reading from /tmp/event.json for local testing
	print.Info("Reading event from: /tmp/event.json (local testing)")
	bytes, err := os.ReadFile("/tmp/event.json")
	if err != nil {
		return nil, "", fmt.Errorf("could not read event from /tmp/event.json: %w", err)
	}
	return bytes, "/tmp/event.json", nil
}

// ParseGitHubEvent parses the raw JSON event data into a GitHubEvent struct
func ParseGitHubEvent(data []byte) (*GitHubEvent, error) {
	var event GitHubEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event JSON: %w", err)
	}
	return &event, nil
}

// GetEventType returns the type of the GitHub event
// Reads from GITHUB_EVENT_NAME env var or defaults based on event structure
func GetEventType(event *GitHubEvent) string {
	if eventType := os.Getenv("GITHUB_EVENT_NAME"); eventType != "" {
		return eventType
	}

	// Detect from event structure
	if event.Comment != nil {
		return "issue_comment"
	}
	if event.Issue != nil {
		return "issues"
	}
	if event.PullRequest != nil {
		return "pull_request"
	}

	return "unknown"
}

// ExtractEventInfo extracts relevant information from the GitHub event
func ExtractEventInfo(event *GitHubEvent, eventType string) *EventResult {
	result := &EventResult{
		Type: eventType,
	}

	switch eventType {
	case "issue_comment":
		if event.Issue != nil {
			result.Number = event.Issue.Number
			result.Title = event.Issue.Title
			result.Body = event.Issue.Body
		}
		if event.Comment != nil {
			result.Comment = event.Comment.Body
		}

	case "issues":
		if event.Issue != nil {
			result.Number = event.Issue.Number
			result.Title = event.Issue.Title
			result.Body = event.Issue.Body
		}

	case "pull_request":
		if event.PullRequest != nil {
			result.Number = event.PullRequest.Number
			result.Title = event.PullRequest.Title
			result.Body = event.PullRequest.Body
		}
	}

	return result
}

// BuildPrompt constructs the prompt text from the event information
func BuildPrompt(info *EventResult) string {
	var prompt string

	switch info.Type {
	case "issue_comment":
		prompt = fmt.Sprintf("## Issue: %s\n\n### Issue Description\n%s\n\n### Latest Comment\n%s",
			info.Title, info.Body, info.Comment)

	case "issues":
		prompt = fmt.Sprintf("## Issue: %s\n\n### Issue Description\n%s",
			info.Title, info.Body)

	case "pull_request":
		prompt = fmt.Sprintf("## Pull Request: %s\n\n### PR Description\n%s",
			info.Title, info.Body)
	}

	return prompt
}

// CleanPrompt removes @claude mentions and normalizes whitespace
func CleanPrompt(prompt string) string {
	cleaned := strings.ReplaceAll(prompt, "@claude", "")
	cleaned = strings.TrimSpace(cleaned)

	// Set default for empty prompts
	if cleaned == "" || cleaned == "## Issue:" || cleaned == "## Pull Request:" {
		cleaned = "Hello! How can I help you? Please analyze this project and let me know your suggestions."
	}

	return cleaned
}

// WritePromptFile writes the cleaned prompt to the output file
func WritePromptFile(prompt string) (string, error) {
	outputDir := "/tmp/claude-prompts"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFile := fmt.Sprintf("%s/prompt.txt", outputDir)
	if err := os.WriteFile(outputFile, []byte(prompt), 0644); err != nil {
		return "", fmt.Errorf("failed to write prompt file: %w", err)
	}

	return outputFile, nil
}

// WriteGitHubOutput writes outputs to the GitHub Actions environment file
func WriteGitHubOutput(promptFile string, issueNumber int) error {
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		return nil // Not in GitHub Actions, skip
	}

	f, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_OUTPUT file: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "prompt_file=%s\n", promptFile); err != nil {
		return err
	}

	if issueNumber > 0 {
		if _, err := fmt.Fprintf(f, "issue_number=%d\n", issueNumber); err != nil {
			return err
		}
	} else {
		print.Warning("Issue number is 0 or negative, skipping issue_number output. This may indicate a parsing problem.")
	}

	return nil
}

// ExtractPrompt extracts the prompt from GitHub event and writes to file
// This is the main entry point that orchestrates all the helper functions
func ExtractPrompt() error {
	// Step 1: Read the event
	eventData, _, err := ReadGitHubEvent()
	if err != nil {
		return err
	}

	// Step 2: Parse the event
	event, err := ParseGitHubEvent(eventData)
	if err != nil {
		return err
	}

	// Step 3: Get event type
	eventType := GetEventType(event)

	// Step 4: Extract information
	info := ExtractEventInfo(event, eventType)

	// Step 5: Build prompt
	rawPrompt := BuildPrompt(info)

	// Step 6: Clean prompt
	cleanPrompt := CleanPrompt(rawPrompt)

	// Step 7: Write to file
	promptFile, err := WritePromptFile(cleanPrompt)
	if err != nil {
		return err
	}

	// Step 8: Write GitHub outputs
	if err := WriteGitHubOutput(promptFile, info.Number); err != nil {
		return err

	}

	print.Success("Prompt extracted successfully to: %s", promptFile)

	print.Plain("Content:")
	print.Plain(cleanPrompt)

	return nil
}

// CreatePR creates a pull request from current changes
// This is the main entry point for automated PR creation
func CreatePR() error {
	// Get working directory (default to current)
	workingDir := os.Getenv("GITHUB_WORKSPACE")
	if workingDir == "" {
		workingDir = "."
	}

	// Get issue number from environment or prompt file
	issueNumberStr := os.Getenv("ISSUE_NUMBER")
	var issueNumber int
	if issueNumberStr != "" {
		fmt.Sscanf(issueNumberStr, "%d", &issueNumber)
	} else {
		// Try to read from prompt file
		promptFile := os.Getenv("PROMPT_FILE")
		if promptFile == "" {
			promptFile = "/tmp/claude-prompts/prompt.txt"
		}
		// Extract issue number from file path or content if needed
		// For now, default to 0 (no issue)
		issueNumber = 0
	}

	// Get Claude's result
	result, err := GetClaudeResult()
	if err != nil {
		return fmt.Errorf("failed to get Claude result: %w", err)
	}

	// Create PR manager
	manager, err := NewPRManager(workingDir)
	if err != nil {
		return fmt.Errorf("failed to create PR manager: %w", err)
	}

	// Create PR from changes
	if err := manager.CreatePRFromChanges(issueNumber, result); err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	return nil
}

// List lists all available github targets
func List() {
	print.Plain("")
	print.Printf(helpers.ColorBlue, "GitHub Targets - Mage Help")
	print.Plain("")
	print.Printf(helpers.ColorGreen, "  mage github:extractprompt - Extract prompt from GitHub event")
	print.Printf(helpers.ColorGreen, "  mage github:createpr      - Create pull request from changes")
	print.Printf(helpers.ColorGreen, "  mage github:list          - Show this help message")
	print.Plain("")
	print.Plain("Testing locally:")
	print.Plain("  1. Save event JSON to /tmp/event.json")
	print.Plain("  2. Run: mage github:extractprompt")
	print.Plain("")
}
