package hooks

import (
	"context"
	"path/filepath"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WithFileReadHooks wraps a file read operation with hooks.
// Returns the file content, potentially modified by AfterFileRead hooks.
//
// Hook execution flow:
// 1. BeforeFileRead hooks run - can validate and block reads
// 2. File read work executes - returns content
// 3. AfterFileRead hooks run - can transform content via HookContext.FileContent
//
// The returned content is the final content after any modifications by AfterFileRead hooks.
func (p *hookManagerImpl) WithFileReadHooks(
	ctx context.Context,
	sessionID, agentID uuid.UUID,
	filePath string,
	work func() (string, error),
) (string, error) {
	// Validate inputs immediately (fail fast)
	if filePath == "" {
		return "", errs.Validation("file path cannot be empty")
	}
	// Basic path sanitization check
	cleanPath := filepath.Clean(filePath)
	if cleanPath != filePath {
		return "", errs.Validationf("file path contains suspicious elements: %s", filePath)
	}
	if work == nil {
		return "", errs.Validation("work function cannot be nil")
	}

	// BeforeFileRead hook
	hookCtx := &HookContext{
		SessionID: sessionID,
		AgentID:   agentID,
		FilePath:  filePath,
		Data:      make(map[string]any),
	}

	result := p.TriggerHooks(ctx, BeforeFileRead, hookCtx)
	if result.Stopped {
		// Hook blocked execution
		if result.Error != nil {
			return "", result.Error
		}
		// Hook stopped without error - blocked successfully
		return "", nil
	}

	// Execute the file read work - returns content
	content, workErr := work()

	// Store content in context for after hooks
	hookCtx.FileContent = content

	// AfterFileRead hook runs even when work fails
	// Hooks can modify FileContent to transform what's returned
	result = p.TriggerHooks(ctx, AfterFileRead, hookCtx)

	// If a fatal 'after' hook failed, its error takes precedence
	if result.Error != nil {
		if workErr != nil {
			p.log.Error("The original work function also returned an error, which is being superseded by the AfterFileRead hook error",
				zap.String("file_path", filePath),
				zap.Error(workErr))
		}
		return "", result.Error
	}

	// Always return hookCtx.FileContent (hooks may have modified it, possibly to empty string)
	return hookCtx.FileContent, workErr
}

// WithFileWriteHooks wraps a file write operation with hooks.
// BeforeFileWrite hooks can transform the content before writing.
// AfterFileWrite hooks can log/audit the write operation.
//
// Hook execution flow:
// 1. BeforeFileWrite hooks run - can transform content via HookContext.FileContent
// 2. File write work executes - receives the final content after hook transformations
// 3. AfterFileWrite hooks run - can log/audit
func (p *hookManagerImpl) WithFileWriteHooks(
	ctx context.Context,
	sessionID, agentID uuid.UUID,
	filePath string,
	content string,
	work func(string) error,
) error {
	// Validate inputs immediately (fail fast)
	if filePath == "" {
		return errs.Validation("file path cannot be empty")
	}
	// Basic path sanitization check
	cleanPath := filepath.Clean(filePath)
	if cleanPath != filePath {
		return errs.Validationf("file path contains suspicious elements: %s", filePath)
	}
	if work == nil {
		return errs.Validation("work function cannot be nil")
	}

	// BeforeFileWrite hook
	hookCtx := &HookContext{
		SessionID:   sessionID,
		AgentID:     agentID,
		FilePath:    filePath,
		FileContent: content, // Initial content
		Data:        make(map[string]any),
	}

	result := p.TriggerHooks(ctx, BeforeFileWrite, hookCtx)
	if result.Stopped {
		// Hook blocked execution
		if result.Error != nil {
			return result.Error
		}
		// Hook stopped without error - blocked successfully
		return nil
	}

	// Execute work with potentially modified content
	finalContent := hookCtx.FileContent
	workErr := work(finalContent)

	// AfterFileWrite hook (always runs, even if work failed)
	// Note: Hooks can access FileContent via hookCtx
	result = p.TriggerHooks(ctx, AfterFileWrite, hookCtx)

	// If a fatal 'after' hook failed, its error takes precedence
	if result.Error != nil {
		if workErr != nil {
			p.log.Error("The original work function also returned an error, which is being superseded by the AfterFileWrite hook error",
				zap.String("file_path", filePath),
				zap.Error(workErr))
		}
		return result.Error
	}

	// Otherwise, return the error from the original work function (if any)
	return workErr
}

// WithFileHooks wraps a function with file operation hooks.
//
// The workflow depends on the hook point:
//
// For Read operations (BeforeFileRead/AfterFileRead):
// 1. BeforeFileRead hooks run - can block by not calling next()
// 2. File read work executes
// 3. AfterFileRead hooks run - can log/audit (cannot return modified content)
//
// For Write operations (BeforeFileWrite/AfterFileWrite):
// 1. BeforeFileWrite hooks run with FileContent in HookContext
//   - Hooks can modify FileContent to change what gets written
//   - Hooks can block execution by not calling next()
//
// 2. File write work executes (potentially with modified content)
// 3. AfterFileWrite hooks run - can log/audit the write
//
// For Delete operations (BeforeFileDelete/AfterFileDelete):
// 1. BeforeFileDelete hooks run - can block by not calling next()
// 2. File delete work executes
// 3. AfterFileDelete hooks run - can log/audit the delete
//
// For Modify operations (BeforeFileModify/AfterFileModify):
// 1. BeforeFileModify hooks run with OldContent and NewContent in HookContext
//   - Hooks can modify NewContent to change the replacement
//   - Hooks can block execution by not calling next()
//
// 2. File modify work executes
// 3. AfterFileModify hooks run - can log/audit the modification
func (p *hookManagerImpl) WithFileHooks(
	ctx context.Context,
	sessionID, agentID uuid.UUID,
	point HookPoint,
	filePath string,
	work func() error,
) error {
	// Validate inputs immediately (fail fast)
	if filePath == "" {
		return errs.Validation("file path cannot be empty")
	}
	// Basic path sanitization check
	cleanPath := filepath.Clean(filePath)
	if cleanPath != filePath {
		return errs.Validationf("file path contains suspicious elements: %s", filePath)
	}
	if work == nil {
		return errs.Validation("work function cannot be nil")
	}

	// Validate hook point
	switch point {
	case BeforeFileRead, AfterFileRead,
		BeforeFileWrite, AfterFileWrite,
		BeforeFileDelete, AfterFileDelete,
		BeforeFileModify, AfterFileModify:
		// valid file hook points
	default:
		return errs.Validationf("invalid file hook point: %s", point)
	}

	// Determine the "before" and "after" hook points
	var beforePoint, afterPoint HookPoint

	switch point {
	case BeforeFileRead, AfterFileRead:
		beforePoint = BeforeFileRead
		afterPoint = AfterFileRead
	case BeforeFileWrite, AfterFileWrite:
		beforePoint = BeforeFileWrite
		afterPoint = AfterFileWrite
	case BeforeFileDelete, AfterFileDelete:
		beforePoint = BeforeFileDelete
		afterPoint = AfterFileDelete
	case BeforeFileModify, AfterFileModify:
		beforePoint = BeforeFileModify
		afterPoint = AfterFileModify
	}

	// Before hook
	hookCtx := &HookContext{
		SessionID: sessionID,
		AgentID:   agentID,
		FilePath:  filePath,
		Data:      make(map[string]any),
	}

	result := p.TriggerHooks(ctx, beforePoint, hookCtx)
	if result.Stopped {
		// Hook blocked execution
		if result.Error != nil {
			return result.Error
		}
		// Hook stopped without error - blocked successfully
		return nil
	}

	// Execute the file operation work
	workErr := work()

	// After hook (always runs, even if work failed)
	// Note: Hooks can access FileContent, OldContent, NewContent via hookCtx
	result = p.TriggerHooks(ctx, afterPoint, hookCtx)

	// If a fatal 'after' hook failed, its error takes precedence
	if result.Error != nil {
		if workErr != nil {
			p.log.Error("The original work function also returned an error, which is being superseded by the after hook error",
				zap.String("file_path", filePath),
				zap.Error(workErr))
		}
		return result.Error
	}

	// Otherwise, return the error from the original work function (if any)
	return workErr
}
