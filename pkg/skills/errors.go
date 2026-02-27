package skills

import (
	"github.com/denkhaus/gollum/pkg/errs"
)

// Skill-specific error constructors using the errs package

// ErrParseFailed creates a parse error for skill YAML frontmatter
func ErrParseFailed(skillPath, message string) *errs.Error {
	return errs.Validationf("failed to parse skill %s: %s", skillPath, message)
}

// ErrParseFailedWithCause creates a parse error with underlying cause
func ErrParseFailedWithCause(skillPath string, cause error) *errs.Error {
	return errs.Wrap(cause, errs.TypeValidation, "failed to parse skill "+skillPath)
}

// ErrMissingRequired creates a validation error for missing required field
func ErrMissingRequired(skillPath, fieldName string) *errs.Error {
	return errs.Validationf("skill %s: missing required field '%s'", skillPath, fieldName)
}

// ErrInvalidValue creates a validation error for invalid field value
func ErrInvalidValue(skillPath, fieldName, value string) *errs.Error {
	return errs.Validationf("skill %s: invalid value '%s' for field '%s'", skillPath, value, fieldName)
}

// ErrEmptyField creates a validation error for empty required field
func ErrEmptyField(skillPath, fieldName string) *errs.Error {
	return errs.Validationf("skill %s: field '%s' cannot be empty", skillPath, fieldName)
}

// ErrSkillNotFound creates a not found error for skill
func ErrSkillNotFound(skillName string) *errs.Error {
	return errs.NotFoundf("skill '%s' not found", skillName)
}

// ErrDiscoveryFailed creates an internal error for discovery failures
func ErrDiscoveryFailed(rootDir string, cause error) *errs.Error {
	return errs.Wrap(cause, errs.TypeInternal, "skill discovery failed in "+rootDir)
}

// ErrSkillLoadFailed creates an internal error for skill loading failures
func ErrSkillLoadFailed(skillPath string, cause error) *errs.Error {
	return errs.Wrap(cause, errs.TypeInternal, "failed to load skill from "+skillPath)
}

// ErrInvalidSkillPath creates a validation error for invalid skill path
func ErrInvalidSkillPath(path string) *errs.Error {
	return errs.Validationf("invalid skill path: %s", path)
}

// ErrDuplicateSkill creates a conflict error for duplicate skill names
func ErrDuplicateSkill(skillName, existingPath, newPath string) *errs.Error {
	return errs.Conflictf("duplicate skill '%s': already exists at %s, found again at %s",
		skillName, existingPath, newPath)
}
