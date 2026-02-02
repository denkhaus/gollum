# Phase 3: Prompt Optimizer - Context

**Gathered:** 2026-02-02
**Status:** Ready for planning

<domain>
## Phase Boundary

LLM-based prompt optimization system that analyzes agent execution trajectories and iteratively improves prompts using three strategies (Gradient, Meta-Prompt, Prompt Memory). The optimizer is invoked on-demand after session completion, not as a background service.

</domain>

<decisions>
## Implementation Decisions

### Trajectory Data Capture
- **Data source**: Full message history from the current session being optimized (gollem.Message)
- **Format**: `Trajectory` struct with `Messages []gollem.Message` and optional `Feedback interface{}`
- **Message types**: Include User, Assistant, Tool calls, and Tool responses (full conversation context)
- **Feedback options**:
  - Simple string: Free-form user feedback
  - Structured `Feedback`: Score (0.0-1.0), Comment, FailureModes, Outcome
  - `EditFeedback`: Revised response with specific edits
  - `nil`: No feedback - optimizer infers issues from conversation
- **Reference example**: [LangMEM procedural_memory.ipynb](https://github.com/langchain-ai/langmem/blob/main/examples/intro_videos/procedural_memory.ipynb)

### Optimization Trigger
- **When**: After every agent session ends
- **User prompt**: Agent asks user if optimization should be executed for this session
- **Invocation**: On-demand only, not background processing
- **Scope**: Optimizes the current session's trajectory only

### Strategy Selection
- **Default**: System can use a default strategy if not specified
- **Auto-selection**: System may auto-select best strategy based on context (optional capability)
- **Manual override**: User can specify which strategy to use
- **Three strategies**:
  - **Gradient**: 2-10 LLM calls, reflection loop with think/critique tools, most thorough
  - **Meta-Prompt**: 1-5 LLM calls, combined reflection and update, balanced
  - **Prompt Memory**: 1 LLM call, simple single-shot optimization, fastest

### Strategy Differences (from docs)
- **Gradient Strategy**: Separates analysis from application. Uses reflection phase (think/critique) → recommendation → update phase. Best for complex improvements requiring thorough analysis.
- **Meta-Prompt Strategy**: Reflection and update in single phase. More direct approach, moderate cost. Good for balanced optimization needs.
- **Prompt Memory Strategy**: Single LLM call with simple metaprompt. Limited pattern learning ability. Fastest option for simple adjustments.

### LLM Integration
- **Provider**: Use existing Anthropic/OpenAI/Gemini providers in Gollum (m-mizutani/gollem)
- **No new dependencies**: Must use existing LLM abstraction layer
- **Tool calling**: Gradient strategy uses Gollem native tools (think/critique/recommend)

### Output Storage
- **Method**: Use `SaveNewVersion()` from PromptStore (always versioned)
- **Versioning**: SemVer with version-in-ID (e.g., "supervisor@1.0.0" → "supervisor@1.1.0")
- **Change description**: Include description of what changed and why

### Claude's Discretion
- Exact configuration values (max/min reflection steps, etc.)
- Error handling and retry logic
- Performance optimization and caching
- Testing strategy details

</decisions>

<specifics>
## Specific Ideas

- Reference implementation: LangMEM's `create_prompt_optimizer` and `create_multi_prompt_optimizer` APIs
- Trajectory format from Jupyter notebook example: `optimizer.invoke({"prompt": current_prompt, "trajectories": [(result["messages"], feedback)]})`
- Use Gollem's `Message` types directly to avoid duplication
- Prompt templates for all three strategies are documented in `docs/prompt-optimizer/01-langmem-prompt-templates.md`
- Type mapping between Gollem and Prompt Optimizer is in `docs/prompt-optimizer/type-mapping-gollem.md`

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 03-prompt-optimizer*
*Context gathered: 2026-02-02*
