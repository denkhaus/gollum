// Package shared provides common types and interfaces used across the Gollum agent system.
package shared

import (
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/react"
)

// StrategyConfig defines iteration limits for LLM agent strategies.
//
// This configuration controls how agents handle iterations and detect infinite loops.
// The same configuration is used for both subagents and supervisors, with different
// default values provided by the config service.
//
// Design Note: This struct is defined in pkg/shared (not pkg/config) to avoid
// circular import dependencies, since pkg/shared cannot import pkg/config.
type StrategyConfig struct {
	// MaxIterations is the maximum number of iterations an agent can perform.
	// Default: 20
	MaxIterations int `envconfig:"MAX_ITERATIONS" default:"20"`

	// MaxRepeatedActions is the maximum number of times the same action can be repeated.
	// This helps detect infinite loops where an agent repeats the same action.
	// Default: 3
	MaxRepeatedActions int `envconfig:"MAX_REPEATED_ACTIONS" default:"3"`
}

// StrategyBuilder provides strategy construction as a DI service.
//
// This interface abstracts the creation of react strategies with configurable
// parameters, allowing different parts of the system to obtain strategy instances
// without depending on the concrete implementation details.
//
// The builder follows the dependency injection pattern, making it easy to:
// - Mock strategy construction in tests
// - Switch strategy implementations without modifying calling code
// - Configure strategies globally via environment variables
//
// Design Note: LLMClient is passed by the caller (AgentFactory, FlowExecutor)
// to avoid circular dependencies. The builder constructs strategies but does not
// own the client lifecycle.
type StrategyBuilder interface {
	// BuildReact creates a react strategy with the specified configuration.
	//
	// The strategy will use the provided configuration to control execution behavior:
	// - MaxIterations: Maximum number of LLM interaction loops
	// - MaxRepeatedActions: Maximum times the same action can be repeated
	//
	// This allows per-agent customization of strategy parameters.
	//
	// Parameters:
	//   - cfg: Strategy configuration (max iterations, repeated actions, etc.)
	//   - client: LLM client for strategy execution
	//
	// Returns:
	//   - *react.Strategy: Configured strategy ready for use
	//
	// Usage:
	//   strategy := builder.BuildReact(cfg, client)
	BuildReact(cfg *StrategyConfig, client gollem.LLMClient) *react.Strategy

	// BuildDefaultReact creates react strategy with default configuration.
	//
	// This provides a sensible default strategy without requiring configuration
	// parameters. The defaults are:
	// - MaxIterations: 20
	// - MaxRepeatedActions: 3
	//
	// Use this when you don't need custom strategy parameters.
	//
	// Parameters:
	//   - client: LLM client for strategy execution
	//
	// Returns:
	//   - *react.Strategy: Configured strategy ready for use
	//
	// Usage:
	//   strategy := builder.BuildDefaultReact(client)
	BuildDefaultReact(client gollem.LLMClient) *react.Strategy
}
