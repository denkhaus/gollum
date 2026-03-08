package ast

import (
	"unicode"
)

// TokenType represents a token type
type TokenType int

const (
	TokenIdent TokenType = iota
	TokenString
	TokenNumber
	TokenBool
	TokenLParen
	TokenRParen
	TokenComma
	TokenDot
	TokenEOF
	TokenError
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lex tokenizes an expression string
func Lex(input string) ([]Token, error) {
	var tokens []Token
	pos := 0

	for pos < len(input) {
		ch := input[pos]

		// Skip whitespace
		if unicode.IsSpace(rune(ch)) {
			pos++
			continue
		}

		// Identifiers and keywords
		if isIdentStart(ch) {
			start := pos
			pos++
			for pos < len(input) && isIdentPart(input[pos]) {
				pos++
			}
			tokens = append(tokens, Token{
				Type:  TokenIdent,
				Value: input[start:pos],
				Pos:   start,
			})
			continue
		}

		// String literals (single quoted)
		if ch == '\'' {
			pos++ // skip opening quote
			start := pos
			for pos < len(input) && input[pos] != '\'' {
				if input[pos] == '\\' && pos+1 < len(input) {
					pos++ // skip escaped char
				}
				pos++
			}
			if pos >= len(input) {
				return nil, &LexError{Pos: pos, Msg: "unterminated string"}
			}
			tokens = append(tokens, Token{
				Type:  TokenString,
				Value: input[start:pos],
				Pos:   start,
			})
			pos++ // skip closing quote
			continue
		}

		// Numbers
		if isDigit(ch) || (ch == '-' && pos+1 < len(input) && isDigit(input[pos+1])) {
			start := pos
			if ch == '-' {
				pos++
			}
			for pos < len(input) && isDigit(input[pos]) {
				pos++
			}
			if pos < len(input) && input[pos] == '.' {
				pos++
				for pos < len(input) && isDigit(input[pos]) {
					pos++
				}
			}
			tokens = append(tokens, Token{
				Type:  TokenNumber,
				Value: input[start:pos],
				Pos:   start,
			})
			continue
		}

		// Single character tokens
		switch ch {
		case '(':
			tokens = append(tokens, Token{Type: TokenLParen, Pos: pos})
		case ')':
			tokens = append(tokens, Token{Type: TokenRParen, Pos: pos})
		case ',':
			tokens = append(tokens, Token{Type: TokenComma, Pos: pos})
		case '.':
			tokens = append(tokens, Token{Type: TokenDot, Pos: pos})
		default:
			return nil, &LexError{Pos: pos, Msg: string([]byte{ch})}
		}
		pos++
	}

	tokens = append(tokens, Token{Type: TokenEOF, Pos: pos})
	return tokens, nil
}

// isIdentStart returns true if ch can start an identifier
func isIdentStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// isIdentPart returns true if ch can be part of an identifier
func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}

// isDigit returns true if ch is a digit
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// LexError represents a lexing error
type LexError struct {
	Pos int
	Msg string
}

func (e *LexError) Error() string {
	return "lex error at " + string(rune(e.Pos)) + ": " + e.Msg
}
