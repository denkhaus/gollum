package flows

import (
	"fmt"
)

// ErrorCode represents a unique error code
type ErrorCode string

const (
	// Parser errors (Pxxx)
	ErrXMLParse      ErrorCode = "P001"
	ErrInvalidSyntax ErrorCode = "P002"

	// Schema errors (Sxxx)
	ErrMissingInput     ErrorCode = "S001"
	ErrMissingOutput    ErrorCode = "S002"
	ErrFuncNoOutput     ErrorCode = "S003"
	ErrInputHasComputed ErrorCode = "S004"
	ErrAssignToInput    ErrorCode = "S005"
	ErrOutputAsParam    ErrorCode = "S006"

	// Expression errors (Exxx)
	ErrInvalidExpr      ErrorCode = "E001"
	ErrCircularDeps     ErrorCode = "E002"
	ErrFieldNotFound    ErrorCode = "E003"
	ErrRelativePath     ErrorCode = "E004"

	// Graph errors (Gxxx)
	ErrNoInitialState    ErrorCode = "G001"
	ErrUnreachableState  ErrorCode = "G002"
	ErrInvalidTransition ErrorCode = "G003"
)

// LinterError represents a validation error
type LinterError struct {
	Line    int
	Column  int
	Code    ErrorCode
	Message string
	Context string
}

// String returns a formatted error string
func (e LinterError) String() string {
	if e.Context != "" {
		return fmt.Sprintf("%d:%d: %s - %s\n   |\n   | %s\n   | %s^",
			e.Line, e.Column, e.Code, e.Message, e.Context, caret(e.Column))
	}
	return fmt.Sprintf("%d:%d: %s - %s", e.Line, e.Column, e.Code, e.Message)
}

// caret returns a caret string positioned at the given column
func caret(col int) string {
	if col <= 1 {
		return "^"
	}
	return string(make([]byte, col-1)) + "^"
}

// LinterResult contains validation results
type LinterResult struct {
	Valid    bool
	Errors   []LinterError
	Warnings []LinterError
	Hints    []LinterError
}
