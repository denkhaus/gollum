package ast

import (
	"fmt"
	"strconv"
)

// Parser holds parsing state
type Parser struct {
	tokens []Token
	pos    int
}

// ParseExpression parses an expression string into an AST
func ParseExpression(input string) (Expr, error) {
	tokens, err := Lex(input)
	if err != nil {
		return nil, err
	}

	p := &Parser{tokens: tokens}
	return p.parseExpr()
}

// parseExpr parses an expression
func (p *Parser) parseExpr() (Expr, error) {
	return p.parseCall()
}

// parseCall parses a function call: IDENT(args)
func (p *Parser) parseCall() (Expr, error) {
	if !p.peek(TokenIdent) {
		return nil, p.error("expected identifier")
	}

	funcName := p.current().Value
	p.advance() // consume function name

	if !p.consume(TokenLParen) {
		return nil, p.error("expected '(' after function name")
	}

	var args []Expr
	for !p.peek(TokenRParen) && !p.peek(TokenEOF) {
		arg, err := p.parseArg()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		if !p.consume(TokenComma) {
			break // no more args
		}
	}

	if !p.consume(TokenRParen) {
		return nil, p.error("expected ')' to close function call")
	}

	return &CallExpr{Func: funcName, Args: args}, nil
}

// parseArg parses a function argument
func (p *Parser) parseArg() (Expr, error) {
	tok := p.current()

	switch tok.Type {
	case TokenIdent:
		// Check for function call next
		if p.peekAhead(TokenLParen, 1) {
			return p.parseCall()
		}
		// Check for field reference (has dot after)
		if p.peekAhead(TokenDot, 1) {
			return p.parseFieldRef()
		}
		// Bare identifier - treat as simple field ref without prefix
		p.advance()
		return &FieldRef{Prefix: "", Path: []string{tok.Value}}, nil

	case TokenString:
		p.advance()
		return &StringLiteral{Value: tok.Value}, nil

	case TokenNumber:
		p.advance()
		val, _ := strconv.ParseFloat(tok.Value, 64)
		return &NumberLiteral{Value: val}, nil

	case TokenLParen:
		return p.parseCall()

	default:
		return nil, p.error("unexpected token: " + tok.Value)
	}
}

// parseFieldRef parses a field reference like "context.pr.state"
func (p *Parser) parseFieldRef() (Expr, error) {
	parts := []string{p.current().Value}
	p.advance()

	for p.consume(TokenDot) {
		if !p.peek(TokenIdent) {
			return nil, p.error("expected identifier after '.'")
		}
		parts = append(parts, p.current().Value)
		p.advance()
	}

	// Determine prefix (first part should be input/output/context/error)
	if len(parts) < 2 {
		return nil, p.error("field reference must have prefix (e.g., context.field)")
	}

	prefix := parts[0]
	if prefix != "input" && prefix != "output" && prefix != "context" && prefix != "error" {
		return nil, p.error("invalid field prefix: " + prefix + ", expected one of: input, output, context, error")
	}

	return &FieldRef{
		Prefix: prefix,
		Path:   parts[1:],
	}, nil
}

// Helper methods

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) peek(typ TokenType) bool {
	return p.current().Type == typ
}

func (p *Parser) peekAhead(typ TokenType, offset int) bool {
	pos := p.pos + offset
	if pos >= len(p.tokens) {
		return false
	}
	return p.tokens[pos].Type == typ
}

func (p *Parser) consume(typ TokenType) bool {
	if p.peek(typ) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) error(msg string) error {
	tok := p.current()
	return fmt.Errorf("parse error at %d: %s", tok.Pos, msg)
}
