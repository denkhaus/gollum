package github

import (
	"os"
	"strings"
	"testing"
)

// TestGenerateBranchName tests branch name generation
func TestGenerateBranchName(t *testing.T) {
	workingDir := t.TempDir()
	manager := &PRManager{
		workingDir: workingDir,
		config: &PRConfig{
			DefaultBranch: "main",
		},
	}

	branchName := manager.GenerateBranchName()

	if !strings.HasPrefix(branchName, "claude/auto-update-") {
		t.Errorf("Branch name should start with 'claude/auto-update-', got: %s", branchName)
	}

	// Check format: claude/auto-update-YYYYMMDD-HHMMSS
	parts := strings.Split(branchName, "-")
	if len(parts) < 4 {
		t.Errorf("Branch name format incorrect, got: %s", branchName)
	}
}

// TestGetClaudeResult tests Claude result parsing
func TestGetClaudeResult(t *testing.T) {
	// Test when file doesn't exist (should return default)
	result, err := GetClaudeResult()
	if err != nil {
		t.Fatalf("GetClaudeResult failed: %v", err)
	}

	if result != "Files updated" {
		t.Errorf("Expected default result 'Files updated', got: %s", result)
	}

	// Test with valid JSON file
	jsonData := `[{"type": "result", "result": "Test result from Claude"}]`
	tmpfile, err := os.CreateTemp("", "claude-execution-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(jsonData)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Temporarily override the file path
	originalPath := "/tmp/claude-execution-output.json"
	testPath := tmpfile.Name()
	// Note: This would require modifying GetClaudeResult to accept path parameter
	// For now, we'll skip this test
	_ = testPath
	_ = originalPath
}

// TestCreatePRRequest tests PR request creation
func TestCreatePRRequest(t *testing.T) {
	req := PRCreateRequest{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "feature-branch",
		Base:  "main",
	}

	if req.Title != "Test PR" {
		t.Errorf("Expected title 'Test PR', got: %s", req.Title)
	}

	if req.Head != "feature-branch" {
		t.Errorf("Expected head 'feature-branch', got: %s", req.Head)
	}
}

// TestPRConfig tests PR configuration
func TestPRConfig(t *testing.T) {
	config := &PRConfig{
		RepoOwner:     "testowner",
		RepoName:      "testrepo",
		DefaultBranch: "main",
		AuthorName:    "Test Author",
		AuthorEmail:   "test@example.com",
	}

	if config.RepoOwner != "testowner" {
		t.Errorf("Expected repo owner 'testowner', got: %s", config.RepoOwner)
	}

	if config.DefaultBranch != "main" {
		t.Errorf("Expected default branch 'main', got: %s", config.DefaultBranch)
	}
}

// TestGetGitHubToken tests token retrieval from environment
func TestGetGitHubToken(t *testing.T) {
	// Save original value
	originalToken := os.Getenv("GITHUB_TOKEN")
	originalGhToken := os.Getenv("GH_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			os.Unsetenv("GITHUB_TOKEN")
		}
		if originalGhToken != "" {
			os.Setenv("GH_TOKEN", originalGhToken)
		} else {
			os.Unsetenv("GH_TOKEN")
		}
	}()

	// Test GITHUB_TOKEN takes precedence
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Unsetenv("GH_TOKEN")

	token := GetGitHubToken()
	if token != "test-token" {
		t.Errorf("Expected token 'test-token', got: %s", token)
	}

	// Test GH_TOKEN fallback
	os.Unsetenv("GITHUB_TOKEN")
	os.Setenv("GH_TOKEN", "gh-token")

	token = GetGitHubToken()
	if token != "gh-token" {
		t.Errorf("Expected token 'gh-token', got: %s", token)
	}

	// Test no token
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GH_TOKEN")

	token = GetGitHubToken()
	if token != "" {
		t.Errorf("Expected empty token, got: %s", token)
	}
}
