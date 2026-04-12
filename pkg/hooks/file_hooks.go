package hooks

import (
	"context"
	"path/filepath"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/shared"
	"go.uber.org/zap"
)

// WithFileReadHooks wraps a file read operation with hooks.
// Returns the file content, potentially modified by AfterFileRead hooks.
//
// Hook execution flow:
// 1. BeforeFileRead hooks run - can validate and block reads
// 2. File read work executes - returns content
// 3. AfterFileRead hooks run - can transform content via Payload.Content
//
// The returned content is the final content after any modifications by AfterFileRead hooks.
func (p *hookManagerImpl) WithFileReadHooks(
	ctx context.Context,
	loggingContext shared.LoggingContext,
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

	// BeforeFileRead hook with typed context
	hookCtx := NewTypedHookContext(loggingContext,
		FilePayload{Path: filePath, Operation: FileOperationRead},
	)

	result := p.TriggerFileHooks(ctx, BeforeFileRead, hookCtx)
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
	hookCtx.Payload.Content = content

	// AfterFileRead hook runs even when work fails
	// Hooks can modify Content to transform what's returned
	result = p.TriggerFileHooks(ctx, AfterFileRead, hookCtx)

	// If a fatal 'after' hook failed, its error takes precedence
	if result.Error != nil {
		if workErr != nil {
			p.log.ErrorWithContext("The original work function also returned an error, which is being superseded by the AfterFileRead hook error",
				loggingContext,
				zap.String("file_path", filePath),
				zap.Error(workErr))
		}
		return "", result.Error
	}

	// Always return hookCtx.Payload.Content (hooks may have modified it, possibly to empty string)
	return hookCtx.Payload.Content, workErr
}

// WithFileWriteHooks wraps a file write operation with hooks.
// BeforeFileWrite hooks can transform the content before writing.
// AfterFileWrite hooks can log/audit the write operation.
//
// Hook execution flow:
// 1. BeforeFileWrite hooks run - can transform content via Payload.Content
// 2. File write work executes - receives the final content after hook transformations
// 3. AfterFileWrite hooks run - can log/audit
func (p *hookManagerImpl) WithFileWriteHooks(
	ctx context.Context,
	loggingContext shared.LoggingContext,
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

	// BeforeFileWrite hook with typed context
	hookCtx := NewTypedHookContext(
		loggingContext,
		FilePayload{Path: filePath, Content: content, Operation: FileOperationWrite},
	)

	result := p.TriggerFileHooks(ctx, BeforeFileWrite, hookCtx)
	if result.Stopped {
		// Hook blocked execution
		if result.Error != nil {
			return result.Error
		}
		// Hook stopped without error - blocked successfully
		return nil
	}

	// Execute work with potentially modified content
	finalContent := hookCtx.Payload.Content
	workErr := work(finalContent)

	// AfterFileWrite hook (always runs, even if work failed)
	// Note: Hooks can access Content via hookCtx.Payload
	result = p.TriggerFileHooks(ctx, AfterFileWrite, hookCtx)

	// If a fatal 'after' hook failed, its error takes precedence
	if result.Error != nil {
		if workErr != nil {
			p.log.ErrorWithContext("The original work function also returned an error, which is being superseded by the AfterFileWrite hook error",
				loggingContext,
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
// 1. BeforeFileWrite hooks run with Content in TypedHookContext[FilePayload]
//   - Hooks can modify Payload.Content to change what gets written
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
// 1. BeforeFileModify hooks run with OldContent and NewContent in Payload
//   - Hooks can modify Payload.NewContent to change the replacement
//   - Hooks can block execution by not calling next()
//
// 2. File modify work executes
// 3. AfterFileModify hooks run - can log/audit the modification
func (p *hookManagerImpl) WithFileHooks(
	ctx context.Context,
	loggingContext shared.LoggingContext,
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

	// Determine the "before" and "after" hook points and operation type
	var beforePoint, afterPoint HookPoint
	var operation FileOperation

	switch point {
	case BeforeFileRead, AfterFileRead:
		beforePoint = BeforeFileRead
		afterPoint = AfterFileRead
		operation = FileOperationRead
	case BeforeFileWrite, AfterFileWrite:
		beforePoint = BeforeFileWrite
		afterPoint = AfterFileWrite
		operation = FileOperationWrite
	case BeforeFileDelete, AfterFileDelete:
		beforePoint = BeforeFileDelete
		afterPoint = AfterFileDelete
		operation = FileOperationDelete
	case BeforeFileModify, AfterFileModify:
		beforePoint = BeforeFileModify
		afterPoint = AfterFileModify
		operation = FileOperationModify
	default:
		return errs.Validationf("invalid file hook point: %s", point)
	}

	// Before hook with typed context
	hookCtx := NewTypedHookContext(loggingContext,
		FilePayload{Path: filePath, Operation: operation},
	)

	result := p.TriggerFileHooks(ctx, beforePoint, hookCtx)
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
	// Note: Hooks can access Content, OldContent, NewContent via hookCtx.Payload
	result = p.TriggerFileHooks(ctx, afterPoint, hookCtx)

	// If a fatal 'after' hook failed, its error takes precedence
	if result.Error != nil {
		if workErr != nil {
			p.log.ErrorWithContext("The original work function also returned an error, which is being superseded by the after hook error",
				loggingContext,
				zap.String("file_path", filePath),
				zap.Error(workErr))
		}
		return result.Error
	}

	// Otherwise, return the error from the original work function (if any)
	return workErr
}
