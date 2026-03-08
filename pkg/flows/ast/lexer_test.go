package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer_SimpleFunction(t *testing.T) {
	input := `EQ(context.pr.state, 'open')`
	tokens, err := Lex(input)

	assert.NoError(t, err)

	// Check token types and values, ignoring Pos
	expectedTypes := []TokenType{
		TokenIdent, TokenLParen, TokenIdent, TokenDot,
		TokenIdent, TokenDot, TokenIdent, TokenComma,
		TokenString, TokenRParen, TokenEOF,
	}
	expectedValues := []string{
		"EQ", "", "context", "", "pr", "", "state", "", "open", "", "",
	}

	assert.Len(t, tokens, len(expectedTypes))
	for i, tok := range tokens {
		assert.Equal(t, expectedTypes[i], tok.Type, "token %d type mismatch", i)
		assert.Equal(t, expectedValues[i], tok.Value, "token %d value mismatch", i)
	}
}

func TestLexer_NestedFunction(t *testing.T) {
	input := `AND(is_open, GT(changed_files, 10))`
	tokens, err := Lex(input)

	assert.NoError(t, err)

	// Find specific tokens
	hasAND := false
	hasGT := false
	hasNumber := false
	for _, tok := range tokens {
		if tok.Type == TokenIdent && tok.Value == "AND" {
			hasAND = true
		}
		if tok.Type == TokenIdent && tok.Value == "GT" {
			hasGT = true
		}
		if tok.Type == TokenNumber && tok.Value == "10" {
			hasNumber = true
		}
	}
	assert.True(t, hasAND, "should have AND token")
	assert.True(t, hasGT, "should have GT token")
	assert.True(t, hasNumber, "should have number token")
}
