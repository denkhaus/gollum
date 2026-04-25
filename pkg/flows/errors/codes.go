package errors

const (
	ErrCodeUnknownField = "UNKNOWN_FIELD"
	ErrCodeTypeMismatch = "TYPE_MISMATCH"
	ErrCodeCircularDep  = "CIRCULAR_DEPENDENCY"
	ErrCodeExpression   = "EXPRESSION_ERROR"
	ErrCodeImmutable    = "IMMUTABLE_FIELD"
	ErrCodeValidation   = "VALIDATION_FAILED"

	// Output binding errors
	ErrCodeOutputFieldReadOnly = "OUTPUT_FIELD_READONLY"
	ErrCodeOutputBinding       = "OUTPUT_BINDING_ERROR"

	// Environment variable errors
	ErrCodeEnvVarNotFound = "ENV_VAR_NOT_FOUND"
	ErrCodeEnvVarEmpty    = "ENV_VAR_EMPTY"
)
