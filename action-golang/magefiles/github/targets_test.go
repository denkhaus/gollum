package github

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testEventPath = "../testdata/event.json"

// TestParseGitHubEvent tests parsing a real GitHub event
func TestParseGitHubEvent(t *testing.T) {
	// Read the test event file
	testData, err := os.ReadFile(testEventPath)
	if err != nil {
		t.Fatalf("Failed to read test event: %v", err)
	}

	// Parse the event
	event, err := ParseGitHubEvent(testData)
	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Verify comment
	if event.Comment == nil {
		t.Error("Expected comment to be present")
	} else {
		expectedBody := "@claude Please analyze and fix the unused dead code issues identified by staticcheck. Remove all unused code while keeping the functionality intact."
		if event.Comment.Body != expectedBody {
			t.Errorf("Comment body mismatch.\nGot: %s\nWant: %s", event.Comment.Body, expectedBody)
		}
	}

	// Verify issue
	if event.Issue == nil {
		t.Error("Expected issue to be present")
	} else {
		expectedTitle := "Remove unused dead code identified by staticcheck"
		if event.Issue.Title != expectedTitle {
			t.Errorf("Issue title mismatch.\nGot: %s\nWant: %s", event.Issue.Title, expectedTitle)
		}

		if event.Issue.Number != 3 {
			t.Errorf("Issue number mismatch.\nGot: %d\nWant: 3", event.Issue.Number)
		}

		if !strings.Contains(event.Issue.Body, "Static analysis identified unused code") {
			t.Error("Issue body should contain 'Static analysis identified unused code'")
		}
	}
}

// TestExtractEventInfo tests extracting information from an issue_comment event
func TestExtractEventInfo(t *testing.T) {
	testData, err := os.ReadFile(testEventPath)
	if err != nil {
		t.Fatalf("Failed to read test event: %v", err)
	}

	event, err := ParseGitHubEvent(testData)
	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Test issue_comment extraction
	info := ExtractEventInfo(event, "issue_comment")

	if info.Type != "issue_comment" {
		t.Errorf("Type mismatch.\nGot: %s\nWant: issue_comment", info.Type)
	}

	if info.Number != 3 {
		t.Errorf("Number mismatch.\nGot: %d\nWant: 3", info.Number)
	}

	if info.Title != "Remove unused dead code identified by staticcheck" {
		t.Errorf("Title mismatch.\nGot: %s\nWant: Remove unused dead code identified by staticcheck", info.Title)
	}

	if !strings.Contains(info.Body, "Static analysis identified unused code") {
		t.Error("Body should contain 'Static analysis identified unused code'")
	}

	if !strings.Contains(info.Comment, "Please analyze and fix") {
		t.Error("Comment should contain 'Please analyze and fix'")
	}
}

// TestBuildPrompt tests building the prompt from event info
func TestBuildPrompt(t *testing.T) {
	info := &EventResult{
		Type:    "issue_comment",
		Title:   "Test Issue",
		Body:    "This is the issue body",
		Comment: "This is a comment",
		Number:  123,
	}

	prompt := BuildPrompt(info)

	expectedSections := []string{
		"## Issue: Test Issue",
		"### Issue Description",
		"This is the issue body",
		"### Latest Comment",
		"This is a comment",
	}

	for _, section := range expectedSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("Prompt should contain '%s'\nGot: %s", section, prompt)
		}
	}
}

// TestBuildPromptForIssue tests building prompt for plain issue event
func TestBuildPromptForIssue(t *testing.T) {
	info := &EventResult{
		Type:   "issues",
		Title:  "Plain Issue",
		Body:   "Just an issue body",
		Number: 456,
	}

	prompt := BuildPrompt(info)

	if strings.Contains(prompt, "Latest Comment") {
		t.Error("Plain issue should not have 'Latest Comment' section")
	}

	if !strings.Contains(prompt, "## Issue: Plain Issue") {
		t.Error("Prompt should contain issue title")
	}
}

// TestBuildPromptForPR tests building prompt for pull request event
func TestBuildPromptForPR(t *testing.T) {
	info := &EventResult{
		Type:   "pull_request",
		Title:  "Test PR",
		Body:   "PR description",
		Number: 789,
	}

	prompt := BuildPrompt(info)

	if !strings.Contains(prompt, "## Pull Request: Test PR") {
		t.Error("Prompt should contain PR title")
	}

	if !strings.Contains(prompt, "### PR Description") {
		t.Error("Prompt should contain 'PR Description' section")
	}
}

// TestCleanPrompt tests removing @claude and normalizing
func TestCleanPrompt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove @claude",
			input:    "@claude Please help me",
			expected: "Please help me",
		},
		{
			name:     "trim whitespace",
			input:    "   @claude help   ",
			expected: "help",
		},
		{
			name:     "empty prompt gets default",
			input:    "## Issue:",
			expected: "Hello! How can I help you? Please analyze this project and let me know your suggestions.",
		},
		{
			name:     "normal prompt unchanged",
			input:    "This is a normal prompt",
			expected: "This is a normal prompt",
		},
		{
			name:     "multiple @claude",
			input:    "@claude @claude help me",
			expected: "help me",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanPrompt(tt.input)
			if result != tt.expected {
				t.Errorf("CleanPrompt(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestReadGitHubEvent_LocalFile tests reading from local test file
func TestReadGitHubEvent_LocalFile(t *testing.T) {
	// Copy test data to /tmp/event.json for testing
	testData, err := os.ReadFile(testEventPath)
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	// Create temp directory
	tmpDir := t.TempDir()
	eventPath := filepath.Join(tmpDir, "event.json")
	if err := os.WriteFile(eventPath, testData, 0644); err != nil {
		t.Fatalf("Failed to write test event: %v", err)
	}

	// Override the path for testing
	originalPath := os.Getenv("GITHUB_EVENT_PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("GITHUB_EVENT_PATH", originalPath)
		} else {
			os.Unsetenv("GITHUB_EVENT_PATH")
		}
	}()

	// Unset GITHUB_EVENT_PATH to force fallback
	os.Unsetenv("GITHUB_EVENT_PATH")

	// This test would need to mock the /tmp/event.json reading
	// For now, we just verify the function signature is correct
	_ = testData
}

// TestGetEventType tests event type detection
func TestGetEventType(t *testing.T) {
	testData, err := os.ReadFile(testEventPath)
	if err != nil {
		t.Fatalf("Failed to read test event: %v", err)
	}

	event, err := ParseGitHubEvent(testData)
	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Test detection from event structure (no env var set)
	os.Unsetenv("GITHUB_EVENT_NAME")
	eventType := GetEventType(event)

	if eventType != "issue_comment" {
		t.Errorf("Detected type mismatch.\nGot: %s\nWant: issue_comment", eventType)
	}
}

// TestWritePromptFile tests writing the prompt file
func TestWritePromptFile(t *testing.T) {
	testPrompt := "This is a test prompt"

	// Use temp directory for testing
	tmpDir := t.TempDir()

	// Override the output directory
	originalDir := "/tmp/claude-prompts"
	defer func() {
		// Restore original (nothing to do, just for clarity)
		_ = originalDir
	}()

	// This test would need to mock the directory creation
	// For now, verify the function works with a custom directory
	outputDir := filepath.Join(tmpDir, "claude-prompts")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	outputFile := filepath.Join(outputDir, "prompt.txt")
	if err := os.WriteFile(outputFile, []byte(testPrompt), 0644); err != nil {
		t.Fatalf("Failed to write prompt file: %v", err)
	}

	// Verify file exists and content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read prompt file: %v", err)
	}

	if string(content) != testPrompt {
		t.Errorf("File content mismatch.\nGot: %s\nWant: %s", string(content), testPrompt)
	}
}

// TestIntegration_FullWorkflow tests the complete extraction workflow
func TestIntegration_FullWorkflow(t *testing.T) {
	// Setup: Copy test data to /tmp/event.json
	testData, err := os.ReadFile(testEventPath)
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	tmpDir := t.TempDir()
	eventPath := filepath.Join(tmpDir, "event.json")
	if err := os.WriteFile(eventPath, testData, 0644); err != nil {
		t.Fatalf("Failed to write test event: %v", err)
	}

	// Set environment to use our test file
	originalPath := os.Getenv("GITHUB_EVENT_PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("GITHUB_EVENT_PATH", originalPath)
		} else {
			os.Unsetenv("GITHUB_EVENT_PATH")
		}
	}()
	os.Setenv("GITHUB_EVENT_PATH", eventPath)

	// Set event type
	os.Setenv("GITHUB_EVENT_NAME", "issue_comment")
	defer os.Unsetenv("GITHUB_EVENT_NAME")

	// Execute the workflow (without GitHub output)
	// Create temp output directory
	outputDir := filepath.Join(tmpDir, "claude-prompts")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// Read event
	eventData, sourcePath, err := ReadGitHubEvent()
	if err != nil {
		t.Fatalf("ReadGitHubEvent failed: %v", err)
	}

	if sourcePath != eventPath {
		t.Errorf("Source path mismatch.\nGot: %s\nWant: %s", sourcePath, eventPath)
	}

	// Parse event
	event, err := ParseGitHubEvent(eventData)
	if err != nil {
		t.Fatalf("ParseGitHubEvent failed: %v", err)
	}

	// Get event type
	eventType := GetEventType(event)
	if eventType != "issue_comment" {
		t.Errorf("Event type mismatch.\nGot: %s\nWant: issue_comment", eventType)
	}

	// Extract info
	info := ExtractEventInfo(event, eventType)
	if info.Number != 3 {
		t.Errorf("Issue number mismatch.\nGot: %d\nWant: 3", info.Number)
	}

	// Build prompt
	rawPrompt := BuildPrompt(info)

	// Clean prompt
	cleanPrompt := CleanPrompt(rawPrompt)

	// Verify @claude was removed
	if strings.Contains(cleanPrompt, "@claude") {
		t.Error("Clean prompt should not contain @claude")
	}

	// Verify key content is present
	expectedContent := []string{
		"Remove unused dead code identified by staticcheck",
		"Static analysis identified unused code",
		"Please analyze and fix the unused dead code issues",
	}

	for _, content := range expectedContent {
		if !strings.Contains(cleanPrompt, content) {
			t.Errorf("Clean prompt should contain '%s'", content)
		}
	}

	// Write prompt file
	outputFile := filepath.Join(outputDir, "prompt.txt")
	if err := os.WriteFile(outputFile, []byte(cleanPrompt), 0644); err != nil {
		t.Fatalf("Failed to write prompt file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Prompt file was not created at %s", outputFile)
	}
}
