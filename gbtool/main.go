package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/denkhaus/gbtool/internal/mapping"
)

// HookInput represents the input JSON from Claude Code
type HookInput struct {
	SessionID      string          `json:"session_id"`
	HookEventName string          `json:"hook_event_name"` // PreToolUse, PostToolUse, Stop
	ToolName      string          `json:"tool_name"`
	FilePath      string          `json:"file_path"`
	ToolInput     json.RawMessage `json:"tool_input"`
}

// HookOutput represents the output back to Claude Code
type HookOutput struct {
	Continue    bool   `json:"continue"`
	StopReason  string `json:"stopReason,omitempty"`
	SuppressOutput bool `json:"suppressOutput,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fail("Usage: gbtool <hook-command> [args...]")
	}

	hookCommand := os.Args[1]
	args := os.Args[2:]

	// Read all input from stdin
	inputBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail("Failed to read stdin: " + err.Error())
	}

	// Parse JSON input
	var input HookInput
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		// If JSON parsing fails, just forward to GitButler
		forwardToGitButler(hookCommand, args, inputBytes)
		return
	}

	// Validate session_id
	if input.SessionID == "" {
		forwardToGitButler(hookCommand, args, inputBytes)
		return
	}

	// Get mapping
	homeDir, _ := os.UserHomeDir()
	mappingFile := filepath.Join(homeDir, ".gitbutler", "session-map.json")
	mgr := mapping.NewManager(mappingFile)

	// Find branch for this session
	branchName, err := mgr.GetBranchForSession(input.SessionID)
	if err != nil {
		// No mapping found - forward to GitButler, it will create default
		forwardToGitButler(hookCommand, args, inputBytes)
		return
	}

	// Ensure virtual branch exists
	ensureVirtualBranch(branchName)

	// Forward to GitButler with the same input
	forwardToGitButler(hookCommand, args, inputBytes)
}

func ensureVirtualBranch(branchName string) {
	// Check if branch exists
	cmd := exec.Command("but", "branch", "list")
	output, _ := cmd.Output()

	// Simple check if branch name exists in output
	// In production, you'd want proper JSON parsing
	// For now, we'll just try to create it and ignore errors if it exists
	if !contains(string(output), branchName) {
		exec.Command("but", "branch", "new", branchName).Run()
	}
}

func forwardToGitButler(command string, args []string, inputBytes []byte) {
	// Build GitButler command
	cmdArgs := append([]string{"claude", command}, args...)
	cmd := exec.Command("but", cmdArgs...)

	// Pipe input bytes to stdin
	cmd.Stdin = bytes.NewReader(inputBytes)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func fail(msg string) {
	fmt.Fprintf(os.Stderr, "gbtool: %s\n", msg)
	os.Exit(1)
}
