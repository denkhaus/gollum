package extensions

import "errors"

// Common extension system errors
var (
	ErrInvalidInjector   = errors.New("invalid injector")
	ErrFuncNotFound      = errors.New("function not found")
	ErrFuncLoadFailed    = errors.New("function load failed")
	ErrExecFailed        = errors.New("execution failed")
	ErrNilHookFunction   = errors.New("hook function cannot be nil")
	ErrExtensionNotFound = errors.New("extension not found")
)
