package shared

// OptimizerStrategy defines the optimization approach.
type OptimizerStrategy string

const (
	// StrategyGradient uses reflection phase (think/critique/recommend) then applies updates
	StrategyGradient OptimizerStrategy = "gradient"
	// StrategyMetaPrompt combines reflection and update in single phase
	StrategyMetaPrompt OptimizerStrategy = "metaprompt"
	// StrategyPromptMemory performs single-shot optimization
	StrategyPromptMemory OptimizerStrategy = "promptmemory"
	// StrategyUnknown represents an invalid/unknown strategy
	StrategyUnknown OptimizerStrategy = ""
)
