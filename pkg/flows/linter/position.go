package linter

import (
	"bufio"
	"fmt"
	"strings"
)

// PositionTracker tracks line and column positions for XML elements
type PositionTracker struct {
	lines    []string
	offsets  []int // byte offset of each line
}

// NewPositionTracker creates a new position tracker from raw XML content
func NewPositionTracker(xmlContent string) *PositionTracker {
	scanner := bufio.NewScanner(strings.NewReader(xmlContent))
	var lines []string
	var offsets []int
	offset := 0

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		offsets = append(offsets, offset)
		offset += len(scanner.Bytes()) + 1 // +1 for newline
	}

	return &PositionTracker{
		lines:   lines,
		offsets: offsets,
	}
}

// FindElementPosition finds the line number of an XML element by searching for its pattern
// It searches for patterns like: <element name="value", <call ref="value", etc.
func (p *PositionTracker) FindElementPosition(elementType, attrName, attrValue string) (line, column int) {
	// Build search patterns - try multiple formats
	patterns := []string{
		fmt.Sprintf(`<%s %s="%s"`, elementType, attrName, attrValue),
		fmt.Sprintf(`<%s %s="%s"`, elementType, attrName, attrValue),
		fmt.Sprintf(`<%s %s='%s'`, elementType, attrName, attrValue),
		fmt.Sprintf(`<%s\n%s="%s"`, elementType, attrName, attrValue),
		fmt.Sprintf(`<%s\n%s='%s'`, elementType, attrName, attrValue),
	}

	// Also check for patterns with whitespace variations
	for i := 0; i < len(p.lines); i++ {
		lineContent := p.lines[i]

		// Check if any pattern matches
		for _, pattern := range patterns {
			idx := strings.Index(lineContent, pattern)
			if idx != -1 {
				return i + 1, idx + 1 // Convert to 1-indexed
			}
		}

		// Also check if the line contains the element type and attribute with our value
		// This handles multi-line definitions and various spacing
		if strings.Contains(lineContent, `<`+elementType) &&
		   strings.Contains(lineContent, attrValue) &&
		   (strings.Contains(lineContent, attrName+`=`) || strings.Contains(lineContent, attrName+` =`)) {
			// Try to find the column of the element start
			elemStart := strings.Index(lineContent, `<`+elementType)
			if elemStart != -1 {
				return i + 1, elemStart + 1
			}
		}
	}

	return 0, 0
}

// FindElementPositionByName finds the position of a named element (like computed fields)
func (p *PositionTracker) FindElementPositionByName(elementType, name string) (line, column int) {
	return p.FindElementPosition(elementType, "name", name)
}

// FindLine finds a line containing specific text and returns its number
func (p *PositionTracker) FindLine(searchText string) int {
	for i, line := range p.lines {
		if strings.Contains(line, searchText) {
			return i + 1 // 1-indexed
		}
	}
	return 0
}

// FindStatePosition finds the line number of a state definition
func (p *PositionTracker) FindStatePosition(stateName string) (line, column int) {
	return p.FindElementPosition("state", "name", stateName)
}

// FindCallPosition finds the line number of a call element
func (p *PositionTracker) FindCallPosition(callRef string) (line, column int) {
	return p.FindElementPosition("call", "ref", callRef)
}

// FindTransitionPosition finds the line number of a transition with a specific "to" value
func (p *PositionTracker) FindTransitionPosition(toState string) (line, column int) {
	return p.FindElementPosition("transition", "to", toState)
}

// FindComputedFieldPosition finds the line number of a computed field
func (p *PositionTracker) FindComputedFieldPosition(fieldName string) (line, column int) {
	// Computed fields can be <string name="...">, <int name="...">, etc.
	for _, elemType := range []string{"string", "int", "bool", "float"} {
		if line, col := p.FindElementPosition(elemType, "name", fieldName); line > 0 {
			return line, col
		}
	}
	return 0, 0
}

// FindContextForExpression finds the line number where an expression appears
// by searching for the expression text in the XML
func (p *PositionTracker) FindContextForExpression(expr string) (line, column int) {
	// Remove leading/trailing whitespace for searching
	searchExpr := strings.TrimSpace(expr)
	if len(searchExpr) > 50 {
		// For long expressions, use a shorter substring
		searchExpr = searchExpr[:50]
	}

	for i, lineContent := range p.lines {
		if strings.Contains(lineContent, searchExpr) {
			idx := strings.Index(lineContent, searchExpr)
			return i + 1, idx + 1
		}
	}
	return 0, 0
}
