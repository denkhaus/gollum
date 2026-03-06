// Package builtin provides production-ready built-in hooks for common use cases.
// These hooks can be registered out-of-the-box and configured via environment variables.
package builtin

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// SecurityHook provides security validation for operations.
// It validates file paths, checks for command injection, and can block dangerous operations.
type SecurityHook struct {
	log    logger.LoggerService
	config *config.HooksConfig
}

// NewSecurityHook creates a new SecurityHook instance.
func NewSecurityHook(injector do.Injector) (*SecurityHook, error) {
	log := do.MustInvoke[logger.LoggerService](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	return &SecurityHook{
		log:    log,
		config: cfg.GetHooksConfig(),
	}, nil
}

// NewSecurityHookProvider creates a SecurityHook provider for DI registration.
func NewSecurityHookProvider(injector do.Injector) (hooks.HookFunc, error) {
	_, err := NewSecurityHook(injector)
	if err != nil {
		return nil, err
	}

	// Return a no-op HookFunc - actual hooks registered via RegisterSecurityHooks
	return func(ctx context.Context, hookCtx *hooks.HookContext, next func() error) error {
		return next()
	}, nil
}

// RegisterSecurityHooks registers all security hooks with the HookManager.
func RegisterSecurityHooks(hm hooks.HookManager, hook *SecurityHook) error {
	if !hook.isStrictMode() {
		return nil // Skip registration if not in strict mode
	}

	priority := 0      // Low priority to run validation first
	fatalError := true // Security violations should stop execution

	// Helper to create TypedHookMetadata
	meta := func(name string, point hooks.HookPoint) hooks.TypedHookMetadata {
		return hooks.TypedHookMetadata{
			Name:       name,
			Point:      point,
			Priority:   priority,
			FatalError: fatalError,
		}
	}

	// Register each hook and return immediately on error
	if err := hm.RegisterToolHook(hook.beforeToolExecutionHook, meta("security-before-tool", hooks.BeforeToolExecution)); err != nil {
		return err
	}

	// File operation hooks - validate file paths
	if err := hm.RegisterFileHook(hook.beforeFileReadHook, meta("security-before-file-read", hooks.BeforeFileRead)); err != nil {
		return err
	}
	if err := hm.RegisterFileHook(hook.beforeFileWriteHook, meta("security-before-file-write", hooks.BeforeFileWrite)); err != nil {
		return err
	}
	if err := hm.RegisterFileHook(hook.beforeFileDeleteHook, meta("security-before-file-delete", hooks.BeforeFileDelete)); err != nil {
		return err
	}
	if err := hm.RegisterFileHook(hook.beforeFileModifyHook, meta("security-before-file-modify", hooks.BeforeFileModify)); err != nil {
		return err
	}

	return nil
}

// beforeToolExecutionHook validates tool operations before execution.
func (h *SecurityHook) beforeToolExecutionHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ToolPayload], next func() error) error {
	// Validate tool operations
	if hookCtx.Payload.Name != "" {
		if err := h.validateToolOperation(hookCtx.Payload.Name.String(), hookCtx.Payload.Args); err != nil {
			h.log.Warn("Security check failed for tool operation",
				zap.String("tool", hookCtx.Payload.Name.String()),
				zap.Error(err))
			return err // Block the operation in strict mode
		}
	}

	return next()
}

// beforeFileReadHook validates file read operations.
func (h *SecurityHook) beforeFileReadHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	if err := h.validateFilePath(hookCtx.Payload.Path); err != nil {
		h.log.Warn("Security check failed for file path",
			zap.String("file_path", hookCtx.Payload.Path),
			zap.Error(err))
		return err
	}
	return next()
}

// beforeFileWriteHook validates file write operations.
func (h *SecurityHook) beforeFileWriteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	if err := h.validateFilePath(hookCtx.Payload.Path); err != nil {
		h.log.Warn("Security check failed for file path",
			zap.String("file_path", hookCtx.Payload.Path),
			zap.Error(err))
		return err
	}
	return next()
}

// beforeFileDeleteHook validates file delete operations.
func (h *SecurityHook) beforeFileDeleteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	if err := h.validateFilePath(hookCtx.Payload.Path); err != nil {
		h.log.Warn("Security check failed for file path",
			zap.String("file_path", hookCtx.Payload.Path),
			zap.Error(err))
		return err
	}
	return next()
}

// beforeFileModifyHook validates file modify operations.
func (h *SecurityHook) beforeFileModifyHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	if err := h.validateFilePath(hookCtx.Payload.Path); err != nil {
		h.log.Warn("Security check failed for file path",
			zap.String("file_path", hookCtx.Payload.Path),
			zap.Error(err))
		return err
	}
	return next()
}

// isStrictMode returns true if security mode is set to strict.
func (h *SecurityHook) isStrictMode() bool {
	return h.config.GetSecurityMode() == "strict"
}

// validateFilePath validates that a file path is safe and doesn't contain path traversal or suspicious patterns.
func (h *SecurityHook) validateFilePath(path string) error {
	// Check for path traversal
	if strings.Contains(path, "..") {
		return errs.Validationf("path traversal detected: %s", path)
	}

	// Clean the path and check if it still contains suspicious elements
	cleaned := filepath.Clean(path)
	// Check if the cleaned path contains .. (which would indicate path traversal)
	if strings.Contains(cleaned, "..") {
		return errs.Validationf("path traversal detected in cleaned path: %s", cleaned)
	}

	// Check for absolute paths to system directories
	dangerousPrefixes := []string{
		"/etc/", "/sys/", "/proc/", "/dev/", "/boot/",
		"\\Windows\\", "\\Program Files\\", "\\System32\\",
	}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(strings.ToLower(cleaned), strings.ToLower(prefix)) {
			return errs.Validationf("access to system directory not allowed: %s", path)
		}
	}

	return nil
}

// validateToolOperation validates that a tool operation is safe.
func (h *SecurityHook) validateToolOperation(toolName string, args map[string]any) error {
	// Check for dangerous bash commands
	if toolName == "bash" {
		if cmd, ok := args["command"].(string); ok {
			if err := h.validateBashCommand(cmd); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateBashCommand validates that a bash command is safe.
func (h *SecurityHook) validateBashCommand(cmd string) error {
	// List of dangerous command patterns
	dangerousPatterns := []struct {
		pattern string
		name    string
	}{
		{"rm -rf /", "recursive root delete"},
		{"rm -rf /*", "recursive root delete wildcard"},
		{"rm -rf ~", "recursive home delete"},
		{"mkfs.", "format filesystem"},
		{"dd if=/dev/zero", "disk wipe"},
		{":(){:|:&};:", "fork bomb"},
		{"chmod 000", "remove all permissions"},
		{"chown -R", "recursive ownership change"},
	}

	// Check for dangerous patterns (case-insensitive)
	cmdLower := strings.ToLower(cmd)
	for _, dp := range dangerousPatterns {
		if strings.Contains(cmdLower, dp.pattern) {
			return errs.Validationf("dangerous command detected: %s (%s)", dp.name, truncateString(cmd, 50))
		}
	}

	// Check for command injection patterns
	injectionPatterns := []string{"&&", "|", ";", "`", "$(", "curl", "wget", "nc -l", "netcat"}
	for _, pattern := range injectionPatterns {
		if strings.Contains(cmd, pattern) {
			// Allow some safe patterns
			if pattern == "&&" || pattern == ";" || pattern == "|" {
				// These could be legitimate in scripts, but warn
				h.log.Warn("Potentially dangerous command pattern",
					zap.String("pattern", pattern),
					zap.String("command", truncateString(cmd, 100)))
			} else {
				return errs.Validationf("command injection pattern detected: %s", pattern)
			}
		}
	}

	return nil
}

// validateLLMInput validates LLM input for prompt injection attempts.
func (h *SecurityHook) validateLLMInput(input string) error {
	// Check for prompt injection patterns (basic implementation)
	injectionPatterns := []string{
		"ignore previous instructions",
		"disregard all instructions",
		"forget everything",
		"<system>",
		"</system>",
		"<instruction>",
		"</instruction>",
		"jailbreak",
		"developer mode",
		"override protocol",
	}

	inputLower := strings.ToLower(input)
	for _, pattern := range injectionPatterns {
		if strings.Contains(inputLower, pattern) {
			return errs.Validationf("potential prompt injection detected: %s", pattern)
		}
	}

	return nil
}
