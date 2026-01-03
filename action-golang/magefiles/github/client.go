package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/denkhaus/action-golang/magefiles/helpers"
	"github.com/magefile/mage/sh"
)

// Client handles GitHub API operations using gh CLI
type Client struct {
	print *helpers.Printer
}

// NewClient creates a new GitHub client
func NewClient() *Client {
	return &Client{
		print: print,
	}
}

// PRCreateRequest represents a pull request creation request
type PRCreateRequest struct {
	Title string
	Body  string
	Head  string
	Base  string
}

// PR represents a created pull request
type PR struct {
	Number    int    `json:"number"`
	HTMLURL   string `json:"html_url"`
	Title     string `json:"title"`
	BranchURL string `json:"url"`
}

// CreatePR creates a pull request using gh CLI
func (c *Client) CreatePR(req PRCreateRequest) (*PR, error) {
	c.print.Info("Creating pull request: %s", req.Title)

	// Build gh CLI command
	args := []string{
		"pr", "create",
		"--title", req.Title,
		"--body", req.Body,
		"--head", req.Head,
		"--base", req.Base,
		"--json", "number,htmlUrl,title,url",
	}

	// Run gh CLI
	output, err := sh.Output("gh", args...)
	if err != nil {
		return nil, fmt.Errorf("gh pr create failed: %w", err)
	}

	var pr PR
	if err := json.Unmarshal([]byte(output), &pr); err != nil {
		return nil, fmt.Errorf("failed to parse gh response: %w", err)
	}

	c.print.Success("Pull request created: #%d", pr.Number)
	return &pr, nil
}

// CreateIssueComment creates a comment on an issue or PR
func (c *Client) CreateIssueComment(issueNumber int, body string) error {
	c.print.Info("Posting comment to issue #%d", issueNumber)

	args := []string{
		"issue", "comment",
		fmt.Sprint(issueNumber),
		"--body", body,
	}

	if err := sh.Run("gh", args...); err != nil {
		return fmt.Errorf("gh issue comment failed: %w", err)
	}

	c.print.Success("Comment posted to issue #%d", issueNumber)
	return nil
}

// AddReaction adds a reaction to an issue comment or PR
func (c *Client) AddReaction(targetType string, targetID int, content string) error {
	args := []string{
		"reaction", "--create", content,
	}

	switch targetType {
	case "issue_comment":
		args = append(args, fmt.Sprintf("--comment %d", targetID))
	case "issue":
		args = append(args, fmt.Sprintf("--issue %d", targetID))
	}

	if err := sh.Run("gh", args...); err != nil {
		// Non-fatal, just log
		c.print.Plain("Failed to add reaction: %v", err)
	}

	return nil
}

// GetRepoInfo returns repository owner and name
func (c *Client) GetRepoInfo() (owner, repo string, err error) {
	output, err := sh.Output("gh", "repo", "view", "--json", "owner,name", "-q", ".owner.login + \" \" + .name")
	if err != nil {
		return "", "", fmt.Errorf("failed to get repo info: %w", err)
	}

	parts := strings.Fields(output)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected repo info format: %s", output)
	}

	return parts[0], parts[1], nil
}

// GetDefaultBranch returns the default branch of the repository
func (c *Client) GetDefaultBranch() (string, error) {
	output, err := sh.Output("gh", "repo", "view", "--json", "defaultBranchRef", "-q", ".defaultBranchRef.name")
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// GetGitHubToken returns the GitHub token from environment
func GetGitHubToken() string {
	// Check GITHUB_TOKEN first (GitHub Actions)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	// Check GH_TOKEN (gh CLI)
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}
	return ""
}
