package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/denkhaus/action-golang/magefiles/helpers"
	"github.com/magefile/mage/sh"
)

// PRManager handles pull request creation and Git operations
type PRManager struct {
	client     *Client
	print      *helpers.Printer
	config     *PRConfig
	workingDir string
}

// PRConfig holds configuration for PR creation
type PRConfig struct {
	RepoOwner     string
	RepoName      string
	DefaultBranch string
	AuthorName    string
	AuthorEmail   string
}

// NewPRManager creates a new PR manager
func NewPRManager(workingDir string) (*PRManager, error) {
	client := NewClient()

	// Get repo info
	owner, repo, err := client.GetRepoInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo info: %w", err)
	}

	// Get default branch
	defaultBranch, err := client.GetDefaultBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get default branch: %w", err)
	}

	return &PRManager{
		client:     client,
		print:      print,
		workingDir: workingDir,
		config: &PRConfig{
			RepoOwner:     owner,
			RepoName:      repo,
			DefaultBranch: defaultBranch,
			AuthorName:    "Claude Assistant",
			AuthorEmail:   "claude-assistant@anthropic.com",
		},
	}, nil
}

// HasChanges checks if there are any git changes
func (m *PRManager) HasChanges() (bool, error) {
	m.print.Info("Checking for changes...")

	output, err := sh.Output("git", "-C", m.workingDir, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}

	hasChanges := strings.TrimSpace(output) != ""
	if hasChanges {
		m.print.Success("Changes detected")
	} else {
		m.print.Plain("No changes detected")
	}

	return hasChanges, nil
}

// ConfigureGit sets up git configuration
func (m *PRManager) ConfigureGit() error {
	m.print.Info("Configuring git...")

	// Configure user
	if err := sh.Run("git", "-C", m.workingDir, "config", "--global", "user.name", m.config.AuthorName); err != nil {
		return fmt.Errorf("git config user.name failed: %w", err)
	}

	if err := sh.Run("git", "-C", m.workingDir, "config", "--global", "user.email", m.config.AuthorEmail); err != nil {
		return fmt.Errorf("git config user.email failed: %w", err)
	}

	// Add safe.directory for GitHub Actions workspace
	safeDir := m.workingDir
	if safeDir == "" {
		safeDir = "." // default to current directory
	}
	if err := sh.Run("git", "-C", m.workingDir, "config", "--global", "--add", "safe.directory", safeDir); err != nil {
		m.print.Warning("Failed to add safe.directory: %v", err)
	}

	m.print.Success("Git configured")
	return nil
}

// GenerateBranchName generates a unique branch name with timestamp
func (m *PRManager) GenerateBranchName() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("claude/auto-update-%s", timestamp)
}

// CreateBranch creates and checks out a new branch
func (m *PRManager) CreateBranch(branchName string) error {
	m.print.Info("Creating branch: %s", branchName)

	if err := sh.Run("git", "-C", m.workingDir, "checkout", "-b", branchName); err != nil {
		return fmt.Errorf("git checkout -b failed: %w", err)
	}

	m.print.Success("Branch created: %s", branchName)
	return nil
}

// CommitChanges stages and commits all changes
func (m *PRManager) CommitChanges(commitMessage string) error {
	m.print.Info("Staging and committing changes...")

	// Stage all changes
	if err := sh.Run("git", "-C", m.workingDir, "add", "."); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Commit changes
	if err := sh.Run("git", "-C", m.workingDir, "commit", "-m", commitMessage); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	m.print.Success("Changes committed")
	return nil
}

// PushBranch pushes the branch to remote
func (m *PRManager) PushBranch(branchName string) error {
	m.print.Info("Pushing branch to remote: %s", branchName)

	// Set up remote URL if needed
	remoteURL := fmt.Sprintf("git@github.com:%s/%s.git", m.config.RepoOwner, m.config.RepoName)
	if err := sh.Run("git", "-C", m.workingDir, "remote", "set-url", "origin", remoteURL); err != nil {
		m.print.Warning("Failed to set remote URL: %v", err)
	}

	// Push branch
	if err := sh.RunV("git", "-C", m.workingDir, "push", "-u", "origin", branchName); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	m.print.Success("Branch pushed to remote")
	return nil
}

// CreatePullRequest creates a pull request for the branch
func (m *PRManager) CreatePullRequest(branchName, title, body string) (*PR, error) {
	m.print.Info("Creating pull request...")

	pr, err := m.client.CreatePR(PRCreateRequest{
		Title: title,
		Body:  body,
		Head:  branchName,
		Base:  m.config.DefaultBranch,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	m.print.Success("Pull request created: #%d", pr.Number)
	return pr, nil
}

// PostResultComment posts a comment with the result to an issue
func (m *PRManager) PostResultComment(issueNumber int, result string) error {
	return m.client.CreateIssueComment(issueNumber, result)
}

// CreatePRFromChanges creates a PR from current changes
func (m *PRManager) CreatePRFromChanges(issueNumber int, result string) error {
	// Check for changes
	hasChanges, err := m.HasChanges()
	if err != nil {
		return err
	}

	if !hasChanges {
		// No changes - just post result comment
		body := fmt.Sprintf("## Response from Claude Bot\n\n%s\n\n---\n*Claude Assistant | Execution time: %s*",
			result,
			time.Now().Format("2006-01-02 15:04:05 MST"),
		)
		return m.PostResultComment(issueNumber, body)
	}

	// Configure git
	if err := m.ConfigureGit(); err != nil {
		return fmt.Errorf("git configuration failed: %w", err)
	}

	// Get changes for description
	changesOutput, err := sh.Output("git", "-C", m.workingDir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to get changes: %w", err)
	}

	// Generate branch name
	branchName := m.GenerateBranchName()

	// Create and checkout branch
	if err := m.CreateBranch(branchName); err != nil {
		return err
	}

	// Commit changes
	commitMessage := fmt.Sprintf("Claude Assistant: %s", result)
	if err := m.CommitChanges(commitMessage); err != nil {
		return err
	}

	// Push branch
	if err := m.PushBranch(branchName); err != nil {
		return err
	}

	// Create PR
	prTitle := "Automatic Update by Claude Assistant"
	prBody := fmt.Sprintf("## Automatic Update by Claude Assistant\n\n%s\n\n### Changes\n```\n%s\n```\n\n---\n*This pull request was automatically generated by Claude Assistant*",
		result,
		strings.TrimSpace(changesOutput),
	)

	pr, err := m.CreatePullRequest(branchName, prTitle, prBody)
	if err != nil {
		// Post error comment
		errorBody := fmt.Sprintf("## Response from Claude Bot\n\n%s\n\nFailed to automatically create pull request: %v\n\n---\n*Claude Assistant | Execution time: %s*",
			result,
			err,
			time.Now().Format("2006-01-02 15:04:05 MST"),
		)
		return m.PostResultComment(issueNumber, errorBody)
	}

	// Post success comment with PR link
	successBody := fmt.Sprintf("**Changes Complete!**\n\n%s\n\n**Pull request created:** #%d\n\n[View Pull Request](%s)\n\n---\n*Claude Assistant | Execution time: %s*",
		result,
		pr.Number,
		pr.HTMLURL,
		time.Now().Format("2006-01-02 15:04:05 MST"),
	)

	return m.PostResultComment(issueNumber, successBody)
}

// GetClaudeResult reads the Claude execution result from the output file
func GetClaudeResult() (string, error) {
	filePath := "/tmp/claude-execution-output.json"

	data, err := os.ReadFile(filePath)
	if err != nil {
		// File doesn't exist, use default
		return "Files updated", nil
	}

	// Try to parse as JSON array
	var resultItems []map[string]interface{}
	if err := json.Unmarshal(data, &resultItems); err == nil {
		for _, item := range resultItems {
			if item["type"] == "result" {
				if result, ok := item["result"].(string); ok {
					return result, nil
				}
			}
		}
	}

	// Fallback to default
	return "Files updated", nil
}
