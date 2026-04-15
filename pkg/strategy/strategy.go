// Package strategy provides strategy builder for constructing LLM agent strategies.
package strategy

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/m-mizutani/gollem"
)

// StrategyType defines the type of strategy to use
type StrategyType string

const (
	StrategyTypeUnknown StrategyType = ""
	// StrategyTypeReact uses the ReAct (Reasoning and Acting) strategy
	StrategyTypeReact StrategyType = "react"

	// StrategyTypeSimple uses the simple strategy (original behavior)
	StrategyTypeSimple StrategyType = "simple"

	// StrategyTypeDefault is the default strategy type (react)
	StrategyTypeDefault StrategyType = StrategyTypeReact
)

// Builder provides strategy construction as a DI service.
//
// This interface abstracts the creation of strategies with configurable
// parameters for different use cases (supervisor, subagent, LLM step).
//
// Design Note: LLMClient is passed by the caller (AgentFactory, FlowExecutor)
// to avoid circular dependencies. The builder constructs strategies but does not
// own the client lifecycle.
type Builder interface {
	// BuildForSupervisor creates a strategy for supervisor agents with supervisor defaults
	BuildForSupervisor(client gollem.LLMClient, strategyType StrategyType) gollem.Strategy

	// BuildForSubAgent creates a strategy for subagent creation with subagent defaults
	BuildForSubAgent(client gollem.LLMClient, strategyType StrategyType) gollem.Strategy

	// BuildForLLMStep creates a strategy for flow LLM steps with LLM step defaults
	BuildForLLMStep(client gollem.LLMClient, strategyType StrategyType) gollem.Strategy

	// BuildReact creates a react strategy with the specified configuration
	BuildReact(cfg *config.StrategyConfig, client gollem.LLMClient) gollem.Strategy

	// BuildSimple creates a simple strategy
	BuildSimple(client gollem.LLMClient) gollem.Strategy
}
