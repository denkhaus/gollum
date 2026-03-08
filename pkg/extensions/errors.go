package extensions

import "fmt"

// Common extension system errors
var (
	ErrInvalidInjector = fmt.Errorf("invalid injector")
	ErrFuncNotFound    = fmt.Errorf("function not found")
	ErrFuncLoadFailed  = fmt.Errorf("function load failed")
	ErrExecFailed      = fmt.Errorf("execution failed")
)
