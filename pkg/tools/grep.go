// Package tools provides tool implementations for the Gollum agent system.
// This includes file operations (ReadFile, WriteFile, Edit), search tools (Grep, Glob),
// agent management (SpawnAgent, RemoveAgent, ResumeAgent), and utility tools (CurrentTime).
package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

const (
	outputModeContent          = "content"
	outputModeFilesWithMatches = "files_with_matches"
	outputModeCount            = "count"
)

type (
	// grepToolImpl searches file contents with regex patterns
	grepToolImpl struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
	}

	// GrepToolProvider creates GrepTool instances via DI
	GrepToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
	}

	grepToolProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
	}
)

// NewGrepToolProvider creates a provider for Grep tools
func NewGrepToolProvider(injector do.Injector) (GrepToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	return &grepToolProvider{
		logService:  logService,
		hookManager: hookManager,
	}, nil
}

// CreateTool creates a new GrepTool with agent reference
func (p *grepToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &grepToolImpl{
		logService:  p.logService,
		hookManager: p.hookManager,
		agent:       agent,
	}
}

// Spec returns the tool specification for the Grep tool
func (t *grepToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameGrep.String(),
		Description: "A powerful search tool built on ripgrep. Use Grep for searching file content. ALWAYS use Grep for search tasks - NEVER invoke grep or rg as Bash command.",
		Parameters: map[string]*gollem.Parameter{
			"pattern": {
				Type:        gollem.TypeString,
				Description: "The regular expression pattern to search for",
			},
			"path": {
				Type:        gollem.TypeString,
				Description: "The directory or file to search in (default: current working directory)",
			},
			"output_mode": {
				Type:        gollem.TypeString,
				Description: "Output mode: 'content' for matches with context, 'files_with_matches' for matching files only, 'count' for match counts (default: content)",
			},
			"-C": {
				Type:        gollem.TypeNumber,
				Description: "Number of context lines to show before and after matches",
			},
			"-A": {
				Type:        gollem.TypeNumber,
				Description: "Number of context lines to show after matches",
			},
			"-B": {
				Type:        gollem.TypeNumber,
				Description: "Number of context lines to show before matches",
			},
			"-i": {
				Type:        gollem.TypeBoolean,
				Description: "Case-insensitive search (default: false)",
			},
			"glob": {
				Type:        gollem.TypeString,
				Description: "File filter pattern (e.g., '*.go') to limit search to specific file types",
			},
			"head_limit": {
				Type:        gollem.TypeNumber,
				Description: "Maximum number of results to return (default: 0 = unlimited)",
			},
			"multiline": {
				Type:        gollem.TypeBoolean,
				Description: "Enable multiline regex mode where . matches newlines (default: false)",
			},
			"-n": {
				Type:        gollem.TypeBoolean,
				Description: "Show line numbers (default: true)",
			},
		},
	}
}

// Run executes the Grep tool to search file contents
func (t *grepToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameGrep, args,
		func() (map[string]any, error) {
			return t.runGrep(ctx, args)
		})
}

// runGrep implements the core Grep logic
func (t *grepToolImpl) runGrep(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	pattern, errResp := args.MustGetString(shared.ParamPattern)
	if errResp != nil {
		return errResp, nil
	}

	// Get optional path (default: current directory)
	searchPath := args.GetString(shared.ParamPath, ".")
	if searchPath == "" {
		searchPath = "."
	}

	// Convert relative path to absolute
	searchPath, err := filepath.Abs(searchPath)
	if err != nil {
		t.logService.ErrorWithContext("Grep operation failed: failed to resolve absolute path",
			t.agent.ToSessionContext(),
			zap.String("path", searchPath),
			zap.Error(err))
		return ErrorResponse("failed to resolve absolute path: %v", err), nil
	}

	// Get output mode (default: content)
	outputMode := args.GetString(shared.ParamOutputMode, outputModeContent)
	if outputMode == "" {
		outputMode = outputModeContent
	}

	// Validate output mode
	if outputMode != outputModeContent && outputMode != outputModeFilesWithMatches && outputMode != outputModeCount {
		t.logService.ErrorWithContext("Grep operation failed: invalid output_mode",
			t.agent.ToSessionContext(),
			zap.String("output_mode", outputMode))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("invalid output_mode: %s (must be '%s', '%s', or '%s')", outputMode, outputModeContent, outputModeFilesWithMatches, outputModeCount),
		}, nil
	}

	// Get case-insensitive flag
	caseInsensitive := false
	if flagVal, exists := args["-i"].(bool); exists {
		caseInsensitive = flagVal
	}

	// Get multiline flag
	multiline := false
	if flagVal, exists := args["multiline"].(bool); exists {
		multiline = flagVal
	}

	// Build regex pattern with flags
	regexPattern := pattern
	if caseInsensitive {
		regexPattern = "(?i)" + regexPattern
	}
	if multiline {
		regexPattern = "(?s)" + regexPattern
	}

	// Compile regex
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		t.logService.ErrorWithContext("Grep operation failed: invalid regex pattern",
			t.agent.ToSessionContext(),
			zap.String("pattern", pattern),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("invalid regex pattern: %v", err),
		}, nil
	}

	// Get context settings
	contextBefore := 0
	if cVal, exists := args["-B"].(float64); exists {
		contextBefore = int(cVal)
	}
	contextAfter := 0
	if cVal, exists := args["-A"].(float64); exists {
		contextAfter = int(cVal)
	}
	if cVal, exists := args["-C"].(float64); exists {
		contextBefore = int(cVal)
		contextAfter = int(cVal)
	}

	// Get head limit
	headLimit := 0
	if limitVal, exists := args["head_limit"].(float64); exists {
		headLimit = int(limitVal)
	}

	// Get show line numbers flag (default: true)
	showLineNumbers := true
	if nVal, exists := args["-n"].(bool); exists {
		showLineNumbers = nVal
	}

	// Get glob pattern for file filtering
	globPattern := ""
	if globVal, exists := args["glob"].(string); exists && globVal != "" {
		globPattern = globVal
	}

	t.logService.InfoWithContext("Grep search operation started",
		t.agent.ToSessionContext(),
		zap.String("path", searchPath),
		zap.String("pattern", pattern),
		zap.String("output_mode", outputMode),
		zap.Bool("case_insensitive", caseInsensitive),
		zap.Bool("multiline", multiline),
		zap.Int("context_before", contextBefore),
		zap.Int("context_after", contextAfter),
		zap.Int("head_limit", headLimit),
		zap.String("glob_pattern", globPattern))

	// Perform search
	var result map[string]any
	switch outputMode {
	case "content":
		result, err = t.grepContent(ctx, searchPath, regex, globPattern, contextBefore, contextAfter, headLimit, showLineNumbers)
	case "files_with_matches":
		result, err = t.grepFiles(ctx, searchPath, regex, globPattern, headLimit)
	case "count":
		result, err = t.grepCount(ctx, searchPath, regex, globPattern, headLimit)
	default:
		err = nil
		result = map[string]any{
			"success": false,
			"error":   fmt.Sprintf("unsupported output_mode: %s", outputMode),
		}
	}

	if err != nil {
		t.logService.ErrorWithContext("Grep search operation failed",
			t.agent.ToSessionContext(),
			zap.String("path", searchPath),
			zap.String("pattern", pattern),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	// Log success with results summary
	if result["success"].(bool) {
		t.logService.InfoWithContext("Grep search operation completed successfully",
			t.agent.ToSessionContext(),
			zap.String("path", searchPath),
			zap.String("output_mode", outputMode),
			zap.Any("total_count", result["total_count"]))
	} else {
		t.logService.InfoWithContext("Grep search operation completed with no matches",
			t.agent.ToSessionContext(),
			zap.String("path", searchPath),
			zap.String("pattern", pattern))
	}

	return result, nil
}

// grepContent returns matching lines with context
func (t *grepToolImpl) grepContent(ctx context.Context, searchPath string, regex *regexp.Regexp, globPattern string, contextBefore, contextAfter, headLimit int, showLineNumbers bool) (map[string]any, error) {
	results := []map[string]any{}
	totalMatches := 0
	filesScanned := 0

	walkErr := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.logService.DebugWithContext("Skipping file due to access error",
				t.agent.ToSessionContext(),
				zap.String("path", path),
				zap.Error(err))
			return nil // Skip files we can't access
		}

		if info.IsDir() {
			return nil
		}

		// Apply glob filter
		if globPattern != "" {
			matched, err := filepath.Match(globPattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}

		filesScanned++

		// Search file
		matches, err := t.searchFile(ctx, path, regex, contextBefore, contextAfter, showLineNumbers)
		if err != nil {
			t.logService.DebugWithContext("Error searching file",
				t.agent.ToSessionContext(),
				zap.String("path", path),
				zap.Error(err))
			return nil // Skip files with errors
		}

		if len(matches) > 0 {
			t.logService.DebugWithContext("Found matches in file",
				t.agent.ToSessionContext(),
				zap.String("path", path),
				zap.Int("match_count", len(matches)))

			for _, match := range matches {
				match["path"] = path
				results = append(results, match)
				totalMatches++

				if headLimit > 0 && totalMatches >= headLimit {
					return errs.Conflict("limit reached")
				}
			}
		}

		return nil
	})

	// Check if we hit the limit (not an error)
	if walkErr != nil && !errs.IsType(walkErr, errs.TypeConflict) {
		return nil, walkErr
	}

	if walkErr != nil && errs.IsType(walkErr, errs.TypeConflict) {
		t.logService.InfoWithContext("Grep reached head limit",
			t.agent.ToSessionContext(),
			zap.Int("limit", headLimit),
			zap.Int("matches_found", totalMatches))
	}

	t.logService.DebugWithContext("Grep content search completed",
		t.agent.ToSessionContext(),
		zap.String("search_path", searchPath),
		zap.Int("files_scanned", filesScanned),
		zap.Int("total_matches", totalMatches))

	return map[string]any{
		"success":     len(results) > 0,
		"matches":     results,
		"total_count": totalMatches,
		"output_mode": "content",
	}, nil
}

// grepFiles returns list of files with matches
func (t *grepToolImpl) grepFiles(_ context.Context, searchPath string, regex *regexp.Regexp, globPattern string, headLimit int) (map[string]any, error) {
	matchingFiles := []string{}
	filesScanned := 0

	walkErr := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if info.IsDir() {
			return nil
		}

		if globPattern != "" {
			matched, err := filepath.Match(globPattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}

		filesScanned++

		file, err := os.Open(path)
		if err != nil {
			t.logService.DebugWithContext("Could not open file for reading",
				t.agent.ToSessionContext(),
				zap.String("path", path),
				zap.Error(err))
			return nil
		}
		defer func() {
			if err := file.Close(); err != nil {
				t.logService.DebugWithContext("Error closing file",
					t.agent.ToSessionContext(),
					zap.String("path", path),
					zap.Error(err))
			}
		}()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if regex.MatchString(scanner.Text()) {
				matchingFiles = append(matchingFiles, path)
				break
			}
		}

		if headLimit > 0 && len(matchingFiles) >= headLimit {
			return fmt.Errorf("limit reached")
		}

		return nil
	})

	// Check if we hit the limit (not an error)
	if walkErr != nil && walkErr.Error() != "limit reached" {
		return nil, walkErr
	}

	t.logService.DebugWithContext("Grep files search completed",
		t.agent.ToSessionContext(),
		zap.String("search_path", searchPath),
		zap.Int("files_scanned", filesScanned),
		zap.Int("files_with_matches", len(matchingFiles)))

	return map[string]any{
		"success":     len(matchingFiles) > 0,
		"matches":     matchingFiles,
		"total_count": len(matchingFiles),
		"output_mode": "files_with_matches",
	}, nil
}

// grepCount returns match counts per file
func (t *grepToolImpl) grepCount(_ context.Context, searchPath string, regex *regexp.Regexp, globPattern string, headLimit int) (map[string]any, error) {
	counts := map[string]int{}
	filesScanned := 0

	walkErr := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if info.IsDir() {
			return nil
		}

		if globPattern != "" {
			matched, err := filepath.Match(globPattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}

		filesScanned++

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer func() {
			if err := file.Close(); err != nil {
				t.logService.DebugWithContext("Error closing file",
					t.agent.ToSessionContext(),
					zap.String("path", path),
					zap.Error(err))
			}
		}()

		count := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if regex.MatchString(scanner.Text()) {
				count++
			}
		}

		if count > 0 {
			counts[path] = count
		}

		if headLimit > 0 && len(counts) >= headLimit {
			return fmt.Errorf("limit reached")
		}

		return nil
	})

	// Check if we hit the limit (not an error)
	if walkErr != nil && walkErr.Error() != "limit reached" {
		return nil, walkErr
	}

	t.logService.DebugWithContext("Grep count search completed",
		t.agent.ToSessionContext(),
		zap.String("search_path", searchPath),
		zap.Int("files_scanned", filesScanned),
		zap.Int("files_with_matches", len(counts)))

	return map[string]any{
		"success":     len(counts) > 0,
		"matches":     counts,
		"total_count": len(counts),
		"output_mode": "count",
	}, nil
}

// searchFile searches a single file and returns matches with context
func (t *grepToolImpl) searchFile(ctx context.Context, path string, regex *regexp.Regexp, contextBefore, contextAfter int, showLineNumbers bool) ([]map[string]any, error) {
	// Wrap the file read operation with file read hooks
	content, err := t.hookManager.WithFileReadHooks(ctx, t.agent.ToSessionContext(), path,
		func() (string, error) {
			file, err := os.Open(path)
			if err != nil {
				return "", err
			}
			defer func() {
				if err := file.Close(); err != nil {
					t.logService.DebugWithContext("Error closing file",
						t.agent.ToSessionContext(),
						zap.String("path", path),
						zap.Error(err))
				}
			}()

			var lines []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				return "", err
			}

			// Return content as newline-separated string
			return strings.Join(lines, "\n"), nil
		})

	if err != nil {
		return nil, err
	}

	// Split content back into lines for processing
	lines := strings.Split(content, "\n")
	var matches []map[string]any

	for i, line := range lines {
		if regex.MatchString(line) {
			match := map[string]any{
				"line":      line,
				"line_text": line,
			}

			if showLineNumbers {
				match["line_number"] = i + 1
			}

			// Add context lines
			if contextBefore > 0 || contextAfter > 0 {
				contextLines := []string{}

				// Add before context
				startBefore := i - contextBefore
				if startBefore < 0 {
					startBefore = 0
				}
				for j := startBefore; j < i; j++ {
					prefix := ""
					if showLineNumbers {
						prefix = fmt.Sprintf("%d: ", j+1)
					}
					contextLines = append(contextLines, prefix+lines[j])
				}

				// Add matching line
				prefix := ""
				if showLineNumbers {
					prefix = fmt.Sprintf("%d: ", i+1)
				}
				contextLines = append(contextLines, prefix+line)

				// Add after context
				endAfter := i + contextAfter + 1
				if endAfter > len(lines) {
					endAfter = len(lines)
				}
				for j := i + 1; j < endAfter; j++ {
					prefix := ""
					if showLineNumbers {
						prefix = fmt.Sprintf("%d: ", j+1)
					}
					contextLines = append(contextLines, prefix+lines[j])
				}

				match["context"] = strings.Join(contextLines, "\n")
			}

			matches = append(matches, match)
		}
	}

	return matches, nil
}
