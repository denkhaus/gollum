# Prompt Optimizer for Gollum

## What This Is

A Prompt Optimizer system for the Gollum agent framework that automatically improves agent prompts using execution trajectory feedback. Inspired by LangMEM's procedural memory approach, the system analyzes agent sessions to identify pain points and iteratively refine prompts for better performance.

## Core Value

**Agent quality improves iteratively through automatic prompt optimization based on execution feedback.**

Every session makes agents better — they gather new facts from feedback and learn from their own trajectories.

## Requirements

### Validated

(N/A — building new feature for existing Gollum codebase)

### Active

- [ ] **PROMPT-01**: System supports three optimization strategies (Gradient, Meta-Prompt, Prompt Memory)
- [ ] **PROMPT-02**: Session-end feedback loop collects user feedback OR analyzes session automatically
- [ ] **PROMPT-03**: Prompt Store with File-based backend (in-memory available for testing)
- [ ] **PROMPT-04**: Prompt versioning with SemVer (e.g., `supervisor@1.0.0`) and alias support
- [ ] **PROMPT-05**: Integration with Supervisor agent initially (other agents follow)
- [ ] **PROMPT-06**: Trajectory capture and storage for optimization input
- [ ] **PROMPT-07**: LLM provider integration uses existing Anthropic/OpenAI/Gemini providers
- [ ] **PROMPT-08**: Lazy initialization pattern for Prompt Store (created on first use)
- [ ] **PROMPT-09**: Centralized mock generation with go.uber.org/mock for testing
- [ ] **PROMPT-10**: Proper Go package structure following Gollum conventions

### Out of Scope

- **Other storage backends** (Langfuse, etc.) — File backend first, expand in v2 based on learnings
- **All agents from day one** — Start with Supervisor agent, extend to other agents after validation
- **Real-time optimization** — Optimization happens at session end, not during execution
- **Multi-tenant prompt sharing** — Single-tenant initially, sharing is v2+
- **Prompt A/B testing** — Focus on single-strategy optimization first
- **Optimization strategy routing** — All strategies available initially, learn which applies where

## Context

**Existing Gollum Codebase:**
- Go-based agent framework with dependency injection (samber/do/v2)
- LLM abstraction via m-mizutani/gollem (supports Anthropic, OpenAI, Gemini)
- Agent registry with parent-child relationships
- Tool system for agent capabilities
- MCP integration at localhost:8555/mcp
- Centralized testing patterns with uber.org/mock

**Reference Implementation:**
- LangMEM (https://github.com/langchain-ai/langmem) provides the architectural pattern
- Prompt templates and optimizer API documented in `docs/prompt-optimizer/`
- Complete implementation plan in `PROMPT_OPTIMIZER_PLAN.md`

**Domain Knowledge:**
- Procedural memory: using execution trajectories to improve prompts
- Trajectory structure captures inputs, outputs, intermediate states
- SemVer versioning allows rollback and comparison
- Lazy init avoids unnecessary resource allocation

## Constraints

- **LLM Providers**: Must use existing Anthropic/OpenAI/Gemini providers in Gollum — no new provider dependencies
- **Go Guidance**: CRITICAL — `/home/denkhaus/dev/kb/guides/guide.golang.*.md` files MUST be read and understood in every phase
- **Testing**: All code must include tests using centralized mocks from `pkg/mocks/`
- **Package Structure**: Follow Gollum conventions (pkg/ structure, DI patterns, naming)
- **Breaking Changes**: Minimize disruption to existing agents; integration should be opt-in initially

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| File-based storage first | Simple, reliable, no external dependencies; learn patterns before complex backends | — Pending |
| Supervisor agent as initial target | Main user interaction point; high-value optimization target; patterns extend to other agents | — Pending |
| All three strategies from v1 | Don't know which strategy works best for which use case until we have real data | — Pending |
| Session-end optimization only | Avoids performance impact during execution; feedback is most valuable after completion | — Pending |
| Lazy init for Prompt Store | Resources only allocated when needed; follows Go best practices | — Pending |

---
*Last updated: 2025-02-01 after initialization*
