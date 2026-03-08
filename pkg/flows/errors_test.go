package flows

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinterError_FormattedCorrectly(t *testing.T) {
	err := LinterError{
		Line:    23,
		Column:  12,
		Code:    ErrRelativePath,
		Message: "field reference must use absolute path",
		Context: `when="GT(complexity, 10)"`,
	}

	assert.Contains(t, err.String(), "E004")
	assert.Equal(t, 23, err.Line)
}
