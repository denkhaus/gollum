// Package strategy provides strategy builder for constructing LLM agent strategies.
package strategy

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/m-mizutani/gollem"
)

// Builder provides strategy construction as a DI service.
//
// This interface abstracts the creation of strategies with configurable
// parameters, allowing different parts of the system to obtain strategy instances
// without depending on the concrete implementation details.
//
// Design Note: LLMClient is passed by the caller (AgentFactory, FlowExecutor)
// to avoid circular dependencies. The builder constructs strategies but does not
// own the client lifecycle.
type Builder interface {
	// BuildReact creates a react strategy with the specified configuration.
	BuildReact(cfg *config.StrategyConfig, client gollem.LLMClient) gollem.Strategy

	// BuildDefaultReact creates react strategy with default configuration.
	// Defaults from config service: MaxIterations=20, MaxRepeatedActions=3.
	BuildDefaultReact(client gollem.LLMClient) gollem.Strategy
}
