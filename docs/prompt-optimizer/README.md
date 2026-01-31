# Prompt Optimizer Documentation

**Status:** ✅ **Ready for Implementation**
**Last Updated:** 2025-01-31
**Reference:** [LangMEM by LangChain](https://github.com/langchain-ai/langmem)
**Main Plan:** [`/home/denkhaus/dev/gomodules/gollum/PROMPT_OPTIMIZER_PLAN.md`](../../PROMPT_OPTIMIZER_PLAN.md)

---

## Overview

This directory contains documentation extracted from the LangMEM project, which serves as the reference implementation for our Go-based Prompt Optimizer.

## Purpose

The Prompt Optimizer enables agents to **learn from their interactions** by:
1. Analyzing conversation trajectories
2. Identifying prompt improvement opportunities
3. Automatically refining system prompts
4. Maintaining version history of optimized prompts

---

## Document Index

| File | Description | Status |
|------|-------------|--------|
| [01-langmem-prompt-templates.md](./01-langmem-prompt-templates.md) | Actual prompt templates for all three strategies | ✅ Complete |
| [02-langmem-optimizer-reference.md](./02-langmem-optimizer-reference.md) | API reference and usage examples | ✅ Complete |
| [03-langmem-architecture.md](./03-langmem-architecture.md) | System architecture and design patterns | ✅ Complete |
| [04-trajectory-structure.md](./04-trajectory-structure.md) | Trajectory data structure definition | ✅ Complete |
| [type-mapping-gollem.md](./type-mapping-gollem.md) | Type mapping between Gollem and Prompt Optimizer | ✅ Complete |

---

## Quick Reference

### Optimization Strategies

| Strategy | LLM Calls | Best For | Implementation Priority |
|----------|-----------|----------|-------------------------|
| **Gradient** | 2-10 | Complex improvements, thorough analysis | P0 (First) |
| **Meta-Prompt** | 1-5 | Balanced optimization | P1 (Second) |
| **Prompt Memory** | 1 | Simple adjustments | P2 (Third) |

### Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Package Structure** | `pkg/prompt/types.go` | Centralized type definitions |
| **Strategy Files** | Separate files | Clear separation, easier testing |
| **File Locking** | `syscall.Flock` | Native Unix locking |
| **Bootstrap Pattern** | Lazy Init | Default prompts loaded on first access |
| **Trajectory Fields** | Messages, Outcome, ToolCalls | Metrics postponed |
| **Tool-Calling** | Gollem Native Tools (think/critique/recommend) | LangMEM-style for Gradient strategy |
| **Versioning** | SemVer with version-in-ID | All changes tracked |
| **Store Method** | Only `SaveNewVersion()` | All storage operations versioned |

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    GOLLUM PROMPT OPTIMIZER ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │                    APPLICATION LAYER                                  │   │
│  │                                                                         │   │
│  │  ┌──────────────┐    ┌──────────────────┐    ┌──────────────────────┐ │   │
│  │  │Prompt        │    │Prompt           │    │Prompt Optimizer    │ │   │
│  │  │Manager       │◄───│Optimizer        │◄───│                    │ │   │
│  │  │(erweitert)    │    │                  │    │(uses strategies)    │ │   │
│  │  └───────┬───────┘    └──────────────────┘    └──────────────────────┘ │   │
│  └──────────┼──────────────────────────────────────────────────────────────────┘   │
│             │                                                                  │
│  ┌──────────▼─────────────────────────────────────────────────────────────────┐   │
│  │                    STRATEGY LAYER                                         │   │
│  │                                                                         │   │
│  │  ┌────────────────────────────────────────────────────────────────────┐ │   │
│  │  │  Gradient Strategy  │  Meta-Prompt  │  Prompt Memory              │ │   │
│  │  │  (gradient.go)       │  (metaprompt.go)   │  (memory.go)         │ │   │
│  │  └────────────────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│             │                                                                  │
│  ┌──────────▼─────────────────────────────────────────────────────────────────┐   │
│  │                    STORE LAYER                                             │   │
│  │                                                                         │   │
│  │  PromptStore Interface (store.go)                                        │   │
│  │  ┌─────────────┬──────────────┬──────────────┐                          │   │
│  │  │ InMemory    │ File         │ Future:      │                          │   │
│  │  │ Store       │ Store        │ Langfuse     │                          │   │
│  │  └─────────────┴──────────────┴──────────────┘                          │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 1: Store Layer (Foundation)
- [ ] `pkg/prompt/types.go` - Core types
- [ ] `pkg/prompt/store/store.go` - Interface
- [ ] `pkg/prompt/store/memory_store.go` - In-memory implementation
- [ ] `pkg/prompt/store/file_store.go` - File-based implementation with flock
- [ ] `pkg/prompt/store/provider.go` - DI provider

### Phase 2: Prompt Manager Extension
- [ ] Extend `pkg/prompt/manager.go` with Store integration
- [ ] Implement lazy init for default prompts
- [ ] Add context rendering for templates

### Phase 3: Prompt Optimizer
- [ ] `pkg/prompt/optimizer/types.go` - Trajectory structures
- [ ] `pkg/prompt/optimizer/strategies/gradient.go` - Gradient strategy
- [ ] `pkg/prompt/optimizer/strategies/metaprompt.go` - Meta-prompt strategy
- [ ] `pkg/prompt/optimizer/strategies/memory.go` - Prompt memory strategy
- [ ] `pkg/prompt/optimizer/templates.go` - Prompt templates

### Phase 4: Integration
- [ ] Update `pkg/config/service.go` - New configs
- [ ] Update `pkg/di/container.go` - Register services
- [ ] Update `CLAUDE.md` - Usage documentation

---

## Conventions

- All code examples in Go unless noted otherwise
- Python examples from LangMEM are preserved as-is
- Use `//` for single-line comments, `/* */` for multi-line
- Struct tags follow Go conventions: `json:"field_name"`

---

## Related Documentation

- **Main Plan:** `/home/denkhaus/dev/gomodules/gollum/PROMPT_OPTIMIZER_PLAN.md`
- **Knowledge Base:** `/home/denkhaus/dev/kb/guide.golang.*.md`
- **LangMEM Source:** https://github.com/langchain-ai/langmem
