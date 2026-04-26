package linter

import (
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"30s", 30 * time.Second, false},
		{"60s", 60 * time.Second, false},
		{"120s", 120 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"1h30m", 90 * time.Minute, false},
		{"500ms", 500 * time.Millisecond, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseTimeout(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatTimeout(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{60 * time.Second, "1m"},
		{90 * time.Second, "1m30s"},
		{120 * time.Second, "2m"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h30m"},
		{2 * time.Hour, "2h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatTimeout(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeout_InsufficientCallTimeout(t *testing.T) {
	// Create a flow that calls another flow with insufficient timeout
	flow := &flows.Flow{
		Name:   "caller",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref:     "test-flows/slow-flow", // Has a 60s step in slow-flow
						Timeout: "30s",                  // Insufficient
					},
				},
			},
		},
	}

	result := LintPath("testdata/test.xml", flow)

	// Should have timeout exceeded error
	assert.True(t, hasErrorCode(result, flows.ErrTimeoutExceeded), "should have timeout exceeded error")
}

func TestTimeout_SufficientCallTimeout(t *testing.T) {
	// Create a flow that calls another flow with sufficient timeout
	flow := &flows.Flow{
		Name:   "caller",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref:     "test-flows/slow-flow", // Has a 60s step in slow-flow
						Timeout: "120s",                 // Sufficient
					},
				},
			},
		},
	}

	result := LintPath("testdata/test.xml", flow)

	// Should NOT have timeout exceeded error
	assert.False(t, hasErrorCode(result, flows.ErrTimeoutExceeded), "should not have timeout exceeded error when timeout is sufficient")
}

func TestTimeout_ExactTimeoutMatch(t *testing.T) {
	// Create a flow that calls another flow with exactly the required timeout
	flow := &flows.Flow{
		Name:   "caller",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref:     "test-flows/slow-flow", // Has a 60s step in slow-flow
						Timeout: "60s",                  // Exact match
					},
				},
			},
		},
	}

	result := LintPath("testdata/test.xml", flow)

	// Should NOT have timeout exceeded error (exact match is OK)
	assert.False(t, hasErrorCode(result, flows.ErrTimeoutExceeded), "should not have timeout exceeded error when timeout matches exactly")
}

func TestTimeout_NoExplicitTimeout(t *testing.T) {
	// Create a flow that calls another flow without explicit timeout
	flow := &flows.Flow{
		Name:   "caller",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref:     "some-flow",
						Timeout: "", // No explicit timeout
					},
				},
			},
		},
	}

	result := LintPath("testdata/test.xml", flow)

	// Should NOT have timeout exceeded error (no timeout to validate)
	assert.False(t, hasErrorCode(result, flows.ErrTimeoutExceeded), "should not have timeout exceeded error when no explicit timeout")
}

func TestTimeout_InvalidTimeoutFormat(t *testing.T) {
	// Create a flow with invalid timeout format
	flow := &flows.Flow{
		Name:   "caller",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref:     "some-flow",
						Timeout: "invalid",
					},
				},
			},
		},
	}

	result := LintPath("testdata/test.xml", flow)

	// Should have invalid expression error
	assert.True(t, hasErrorCode(result, flows.ErrInvalidExpr), "should have invalid expression error for invalid timeout format")
}

func TestTimeout_MultipleStepsSummed(t *testing.T) {
	// Test that timeouts from multiple steps are summed
	// code-analysis/security-scan has a 120s step
	flow := &flows.Flow{
		Name:   "caller",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref:     "code-analysis/security-scan", // Has a 120s step
						Timeout: "60s",                         // Insufficient
					},
				},
			},
		},
	}

	result := LintPath("testdata/test.xml", flow)

	// Should have timeout exceeded error
	assert.True(t, hasErrorCode(result, flows.ErrTimeoutExceeded), "should have timeout exceeded error for security-scan")
}

func TestTimeout_RecursiveCallTimeout(t *testing.T) {
	// Test that nested call timeouts are included in the sum
	// This tests the recursive timeout calculation
	// code-analysis/main.xml calls complexity-check and security-scan
	flow := &flows.Flow{
		Name:   "caller",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref:     "code-analysis", // Calls complexity-check (60s) and security-scan (120s)
						Timeout: "60s",           // Insufficient for sum of nested calls
					},
				},
			},
		},
	}

	result := LintPath("testdata/test.xml", flow)

	// Should have timeout exceeded error
	// Note: This depends on how we calculate nested timeouts
	// If we use the explicit call timeouts from main.xml (120s + 180s = 300s), this should fail
	assert.True(t, hasErrorCode(result, flows.ErrTimeoutExceeded), "should have timeout exceeded error for recursive call")
}
