package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArithmeticFunctions tests arithmetic function evaluation
func TestArithmeticFunctions(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		scope    map[string]any
		expected any
		wantErr  bool
	}{
		// ADD tests
		{
			name:     "ADD two integers",
			expr:     "ADD(input.a, input.b)",
			scope:    map[string]any{"input": map[string]any{"a": 5, "b": 3}},
			expected: int64(8),
			wantErr:  false,
		},
		{
			name:     "ADD with floats",
			expr:     "ADD(input.x, input.y)",
			scope:    map[string]any{"input": map[string]any{"x": 2.5, "y": 1.5}},
			expected: 4.0,
			wantErr:  false,
		},
		{
			name:     "ADD negative numbers",
			expr:     "ADD(input.val, -5)",
			scope:    map[string]any{"input": map[string]any{"val": 10}},
			expected: int64(5),
			wantErr:  false,
		},

		// SUB tests
		{
			name:     "SUB two integers",
			expr:     "SUB(input.a, input.b)",
			scope:    map[string]any{"input": map[string]any{"a": 10, "b": 3}},
			expected: int64(7),
			wantErr:  false,
		},
		{
			name:     "SUB results in negative",
			expr:     "SUB(input.a, input.b)",
			scope:    map[string]any{"input": map[string]any{"a": 5, "b": 10}},
			expected: int64(-5),
			wantErr:  false,
		},

		// MUL tests
		{
			name:     "MUL two integers",
			expr:     "MUL(input.a, input.b)",
			scope:    map[string]any{"input": map[string]any{"a": 6, "b": 7}},
			expected: int64(42),
			wantErr:  false,
		},
		{
			name:     "MUL with floats",
			expr:     "MUL(input.x, input.y)",
			scope:    map[string]any{"input": map[string]any{"x": 2.5, "y": 4}},
			expected: 10.0,
			wantErr:  false,
		},
		{
			name:     "MUL by zero",
			expr:     "MUL(input.a, 0)",
			scope:    map[string]any{"input": map[string]any{"a": 5}},
			expected: int64(0),
			wantErr:  false,
		},
		{
			name:     "MUL negative",
			expr:     "MUL(input.a, input.b)",
			scope:    map[string]any{"input": map[string]any{"a": -3, "b": 4}},
			expected: int64(-12),
			wantErr:  false,
		},

		// DIV tests
		{
			name:     "DIV two integers",
			expr:     "DIV(input.a, input.b)",
			scope:    map[string]any{"input": map[string]any{"a": 10, "b": 2}},
			expected: int64(5),
			wantErr:  false,
		},
		{
			name:     "DIV with floats",
			expr:     "DIV(input.x, input.y)",
			scope:    map[string]any{"input": map[string]any{"x": 10.0, "y": 4.0}},
			expected: 2.5,
			wantErr:  false,
		},
		{
			name:     "DIV by zero - error",
			expr:     "DIV(input.a, input.b)",
			scope:    map[string]any{"input": map[string]any{"a": 5, "b": 0}},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "DIV negative result",
			expr:     "DIV(input.a, input.b)",
			scope:    map[string]any{"input": map[string]any{"a": 10, "b": 3}},
			expected: int64(3),
			wantErr:  false,
		},

		// Complex expressions
		{
			name:     "ADD and MUL combined",
			expr:     "ADD(MUL(input.a, 2), input.b)",
			scope:    map[string]any{"input": map[string]any{"a": 5, "b": 3}},
			expected: int64(13),
			wantErr:  false,
		},
		{
			name:     "MUL and SUB combined",
			expr:     "MUL(SUB(input.a, input.b), 2)",
			scope:    map[string]any{"input": map[string]any{"a": 10, "b": 3}},
			expected: int64(14),
			wantErr:  false,
		},
		{
			name:     "nested arithmetic",
			expr:     "MUL(ADD(input.a, input.b), DIV(input.c, 2))",
			scope:    map[string]any{"input": map[string]any{"a": 5, "b": 3, "c": 8}},
			expected: int64(32),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseExpression(tt.expr)
			require.NoError(t, err)

			result, err := Evaluate(parsed, tt.scope)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestArithmeticWithComputedFields tests arithmetic in computed field expressions
func TestArithmeticWithComputedFields(t *testing.T) {
	scope := map[string]any{
		"input": map[string]any{
			"base":     100,
			"bonus":    20,
			"penalty":  5,
			"multiplier": 2,
		},
	}

	tests := []struct {
		name     string
		expr     string
		expected any
	}{
		{"ADD base and bonus", "ADD(input.base, input.bonus)", int64(120)},
		{"SUB penalty from base", "SUB(input.base, input.penalty)", int64(95)},
		{"MUL by multiplier", "MUL(input.base, input.multiplier)", int64(200)},
		{"DIV base by 4", "DIV(input.base, 4)", int64(25)},
		{"complex calculation", "ADD(MUL(input.base, 2), DIV(input.bonus, 4))", int64(205)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseExpression(tt.expr)
			require.NoError(t, err)

			result, err := Evaluate(parsed, scope)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
