package linter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/parser"
)

// TimeoutChecker validates that call timeouts are sufficient for called flows
type TimeoutChecker struct {
	Resolver   *parser.Resolver
	PosTracker *PositionTracker
}

// NewTimeoutChecker creates a new timeout checker
func NewTimeoutChecker() *TimeoutChecker {
	return &TimeoutChecker{
		Resolver: parser.DefaultResolver(),
	}
}

// Check validates timeout constraints for all calls
func (t *TimeoutChecker) Check(flowPath string, flow *flows.Flow, result *flows.LinterResult) {
	for _, state := range flow.States {
		for _, call := range state.Calls {
			if call.Timeout == "" {
				continue // Skip calls without explicit timeout
			}

			callTimeout, err := parseTimeout(call.Timeout)
			if err != nil {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrInvalidExpr,
					Message: fmt.Sprintf("invalid timeout format '%s' in call to '%s': %s", call.Timeout, call.Ref, err),
				})
				continue
			}

			// Resolve the called flow
			resolvedPath, err := t.Resolver.ResolveCall(call.Ref, flowPath)
			if err != nil {
				continue // Will be caught by CallChecker
			}

			// Calculate total required timeout recursively
			requiredTimeout, err := t.calculateRequiredTimeout(resolvedPath, make(map[string]bool))
			if err != nil {
				continue // Skip if we can't parse the called flow
			}

			// Check if call timeout is sufficient
			if requiredTimeout > callTimeout {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrTimeoutExceeded,
					Message: fmt.Sprintf("call to '%s' timeout %s is insufficient: called flow requires at least %s (sum of all step timeouts)", call.Ref, call.Timeout, formatTimeout(requiredTimeout)),
				})
			}
		}
	}
}

// calculateRequiredTimeout recursively calculates the total timeout needed for a flow
func (t *TimeoutChecker) calculateRequiredTimeout(flowPath string, visited map[string]bool) (time.Duration, error) {
	// Prevent infinite recursion
	if visited[flowPath] {
		return 0, nil
	}
	visited[flowPath] = true

	// Parse the flow
	flow, err := parser.Parse(flowPath)
	if err != nil {
		return 0, err
	}

	var total time.Duration

	// Sum all step timeouts across all states
	for _, state := range flow.States {
		for _, step := range state.Steps {
			if step.Timeout != "" {
				timeout, err := parseTimeout(step.Timeout)
				if err != nil {
					continue // Skip invalid timeouts
				}
				total += timeout
			}
		}

		// Recursively check nested calls
		for _, call := range state.Calls {
			if call.Timeout != "" {
				// If the nested call has an explicit timeout, use that
				timeout, err := parseTimeout(call.Timeout)
				if err != nil {
					continue
				}
				total += timeout
			} else {
				// Otherwise, calculate the required timeout for the nested call
				resolvedPath, err := t.Resolver.ResolveCall(call.Ref, flowPath)
				if err != nil {
					continue
				}
				nestedTimeout, err := t.calculateRequiredTimeout(resolvedPath, visited)
				if err != nil {
					continue
				}
				total += nestedTimeout
			}
		}
	}

	return total, nil
}

// timeoutRegex matches timeout strings like "30s", "5m", "1h", "1h30m"
var timeoutRegex = regexp.MustCompile(`^(\d+(?:\.\d+)?)(s|m|h|ms)$`)

// parseTimeout parses a timeout string into a duration
func parseTimeout(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	// Try simple regex match first
	if matches := timeoutRegex.FindStringSubmatch(s); matches != nil {
		value, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timeout value: %s", matches[1])
		}
		unit := matches[2]

		switch unit {
		case "ms":
			return time.Duration(value * float64(time.Millisecond)), nil
		case "s":
			return time.Duration(value * float64(time.Second)), nil
		case "m":
			return time.Duration(value * float64(time.Minute)), nil
		case "h":
			return time.Duration(value * float64(time.Hour)), nil
		}
	}

	// Fall back to time.ParseDuration
	return time.ParseDuration(s)
}

// formatTimeout formats a duration as a human-readable timeout string
func formatTimeout(d time.Duration) string {
	if d >= time.Hour {
		h := d / time.Hour
		m := (d % time.Hour) / time.Minute
		if m > 0 {
			return fmt.Sprintf("%dh%dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	if d >= time.Minute {
		m := d / time.Minute
		s := (d % time.Minute) / time.Second
		if s > 0 {
			return fmt.Sprintf("%dm%ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", d/time.Second)
}
