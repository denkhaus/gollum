package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// validateCommonXMLErrors checks for common XML syntax issues that
// encoding/xml reports with confusing line numbers
func validateCommonXMLErrors(data []byte) error {
	r := bufio.NewReader(bytes.NewReader(data))
	lineNum := 0
	inComment := false
	commentStartLine := 0

	for {
		line, err := r.ReadBytes('\n')
		lineNum++

		if err == io.EOF {
			if inComment {
				return &xmlParseError{
					Line:    commentStartLine,
					Message: "unclosed XML comment (missing '-->'): comment starts here but never closes",
				}
			}
			break
		}
		if err != nil {
			return err
		}

		lineStr := string(line)

		if !inComment {
			// Not in a comment - look for comment start
			openIdx := strings.Index(lineStr, "<!--")
			if openIdx != -1 {
				// Found comment start
				inComment = true
				commentStartLine = lineNum
				// Check if it closes on same line
				closeIdx := strings.Index(lineStr[openIdx+4:], "-->")
				if closeIdx != -1 {
					// Comment closes on same line - valid
					inComment = false
				}
			}
		} else {
			// Already in a comment - check for nested comment FIRST
			openIdx := strings.Index(lineStr, "<!--")
			closeIdx := strings.Index(lineStr, "-->")

			// If we find both markers, check which comes first
			if openIdx != -1 && closeIdx != -1 {
				if openIdx < closeIdx {
					// <!-- comes before --> : nested comment!
					// Report the line where the unclosed comment STARTED, not where we detected it
					return &xmlParseError{
						Line:    commentStartLine,
						Message: fmt.Sprintf("unclosed XML comment: comment on line %d was never closed before another comment starts on line %d", commentStartLine, lineNum),
						Hint:    fmt.Sprintf("add '-->' at the end of line %d to close the comment", commentStartLine),
					}
				}
				// --> comes before <!-- : normal close
				inComment = false
			} else if openIdx != -1 {
				// Found <!-- but no -->: nested comment!
				return &xmlParseError{
					Line:    commentStartLine,
					Message: fmt.Sprintf("unclosed XML comment: comment on line %d was never closed before another comment starts on line %d", commentStartLine, lineNum),
					Hint:    fmt.Sprintf("add '-->' at the end of line %d to close the comment", commentStartLine),
				}
			} else if closeIdx != -1 {
				// Found -->: comment closes normally
				inComment = false
			}
			// If neither found, stay in comment
		}
	}

	return nil
}

// xmlParseError represents a parsing error with context
type xmlParseError struct {
	Line    int
	Message string
	Hint    string
}

func (e *xmlParseError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("line %d: %s\n  Hint: %s", e.Line, e.Message, e.Hint)
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}
