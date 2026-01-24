package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/denkhaus/gbtool/internal/mapping"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: register <issue-number>")
		fmt.Fprintln(os.Stderr, "       register status")
		os.Exit(1)
	}

	command := os.Args[1]

	homeDir, _ := os.UserHomeDir()
	mappingFile := filepath.Join(homeDir, ".gitbutler", "session-map.json")
	mgr := mapping.NewManager(mappingFile)

	switch command {
	case "status":
		showStatus(mgr)
	default:
		// Assume it's an issue number
		issueNumber := command
		registerIssue(mgr, issueNumber)
	}
}

func registerIssue(mgr *mapping.Manager, issueNumber string) {
	sessionID := os.Getenv("CLAUDE_SESSION_ID")
	if sessionID == "" {
		// Fallback: use PID as session identifier
		sessionID = fmt.Sprintf("session-%d", os.Getpid())
	}

	branchName := fmt.Sprintf("issue-%s", issueNumber)

	// Register the session in the mapping
	if err := mgr.RegisterSession(sessionID, branchName); err != nil {
		fmt.Fprintf(os.Stderr, "Error registering session: %v\n", err)
		os.Exit(1)
	}

	// Create the virtual branch in GitButler
	if err := createVirtualBranch(branchName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create virtual branch: %v\n", err)
		fmt.Fprintf(os.Stderr, "The branch will be created automatically when needed.\n")
	}

	fmt.Printf("✓ Registered session %s → %s\n", sessionID, branchName)
	fmt.Printf("✓ Virtual branch: %s\n", branchName)
}

func createVirtualBranch(branchName string) error {
	// Check if branch already exists
	cmd := exec.Command("but", "branch", "list")
	output, err := cmd.Output()
	if err != nil {
		// GitButler might not be available, try creating anyway
		return createBranch(branchName)
	}

	// Simple check if branch name exists in output
	if contains(output, []byte(branchName)) {
		return nil // Already exists
	}

	// Create the branch
	return createBranch(branchName)
}

func createBranch(branchName string) error {
	cmd := exec.Command("but", "branch", "new", branchName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func contains(haystack, needle []byte) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return true
		}
	}
	return false
}

func showStatus(mgr *mapping.Manager) {
	mapping, err := mgr.GetMapping()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading mapping: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Current GitButler Session Mapping:")
	fmt.Println("==================================")
	fmt.Println()

	fmt.Println("Sessions:")
	for sessionID, branchName := range mapping.Sessions {
		fmt.Printf("  %s → %s\n", sessionID, branchName)
	}
	fmt.Println()

	fmt.Println("Issues:")
	for issueName, info := range mapping.Issues {
		fmt.Printf("  %s\n", issueName)
		fmt.Printf("    Status: %s\n", info.Status)
		fmt.Printf("    Active Session: %s\n", info.ActiveSession)
		fmt.Printf("    Branch: %s\n", info.Branch)
	}
}
