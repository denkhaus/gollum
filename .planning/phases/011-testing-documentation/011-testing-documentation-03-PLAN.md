---
phase: 011-testing-documentation
plan: 03
type: execute
wave: 1
depends_on: []
files_modified:
  - CLAUDE.md
  - guides/guide.golang.langfuse-tracing.md
autonomous: true

must_haves:
  truths:
    - CLAUDE.md needs Langfuse configuration and usage documentation
    - Knowledge base guidance should be in universal format
    - Documentation must follow existing patterns in CLAUDE.md
    - Path to knowledge base is project-relative or environment-determined
  artifacts:
    - path: CLAUDE.md
      provides: Langfuse configuration and usage examples
      min_lines: 300 (existing 250 + ~50 new)
    - path: guides/guide.golang.langfuse-tracing.md
      provides: Go-specific Langfuse tracing guidance
      min_lines: 200
      new_file: true
  key_links:
    - from: CLAUDE.md
      to: guide.golang.langfuse-tracing.md
      via: See Also reference
      pattern: \\[\\[guide\\.golang\\.langfuse-tracing\\]\\]
    - from: guide.golang.langfuse-tracing.md
      to: CLAUDE.md
      via: Related resources section
      pattern: \\[\\[.*\\]\\]
---

<objective>
Update CLAUDE.md and create knowledge base guidance for Langfuse tracing integration.

Purpose: Provide users with clear documentation on configuring and using Langfuse tracing in Gollum framework.
Output: Updated project CLAUDE.md and universal knowledge base guidance file.
</objective>

<execution_context>
@/home/denkhaus/.claude/get-shit-done/workflows/execute-plan.md
@/home/denkhaus/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@CLAUDE.md
@pkg/builtin/langfuse_hook.go
@guides/guide.general.prompt-optimization.md
@guides/guide.golang.testing.md
</context>

<tasks>

<task type="auto">
  <name>Update CLAUDE.md with Langfuse integration documentation</name>
  <files>CLAUDE.md</files>
  <action>
    Add new "Langfuse Tracing" section to CLAUDE.md after the Prompt Optimizer section:

    ## Langfuse Tracing

    The LangfuseHook provides comprehensive tracing for LLM interactions, tool executions, and agent lifecycle events through Langfuse SDK integration.

    ### Configuration

    Environment variables for Langfuse tracing:

    - `GOLLUM_LANGFUSE_ENABLED`: Enable/disable tracing (default: "false")
    - `GOLLUM_LANGFUSE_HOST`: Langfuse server URL (default: "https://cloud.langfuse.com")
    - `GOLLUM_LANGFUSE_PUBLIC_KEY`: Langfuse public key for authentication
    - `GOLLUM_LANGFUSE_SECRET_KEY`: Langfuse secret key for authentication
    - `GOLLUM_LANGFUSE_FLUSH_INTERVAL`: Flush interval in milliseconds (default: "1000")
    - `GOLLUM_LANGFUSE_MAX_QUEUE_SIZE`: Max queue size before flush (default: "100")

    Example:
    ```bash
    export GOLLUM_LANGFUSE_ENABLED=true
    export GOLLUM_LANGFUSE_HOST=https://cloud.langfuse.com
    export GOLLUM_LANGFUSE_PUBLIC_KEY=pk-lf-...
    export GOLLUM_LANGFUSE_SECRET_KEY=sk-lf-...
    ```

    ### Hook Registration in main.go

    Register LangfuseHook with HookManager during initialization:

    ```go
    import (
        "github.com/denkhaus/gollum/pkg/builtin"
        "github.com/denkhaus/gollum/pkg/hooks"
    )

    func main() {
        // ... DI container setup ...

        // Get LangfuseHook from DI
        langfuseHook := do.MustInvoke[*builtin.LangfuseHook](injector)

        // Get HookManager from DI
        hm := do.MustInvoke[HookManager](injector)

        // Register all Langfuse tracing hooks
        if err := builtin.RegisterLangfuseHooks(hm, langfuseHook); err != nil {
            log.Fatal("Failed to register Langfuse hooks", zap.Error(err))
        }
    }
    ```

    ### Tracing Enable/Disable Control

    - Set `GOLLUM_LANGFUSE_ENABLED=false` to disable tracing
    - When disabled, hooks are no-ops (no performance overhead)
    - Enable at runtime by updating config (if supported)

    ### Viewing Traces in Langfuse UI

    1. Navigate to your Langfuse instance (e.g., https://cloud.langfuse.com)
    2. Select your project/org
    3. View traces in the "Traces" tab
    4. Filter by session ID, agent ID, or time range
    5. Inspect individual spans for LLM requests, tool calls, and agent events

    Ensure this section:
    - Follows existing CLAUDE.md formatting (## heading, ### subheading)
    - Uses code blocks with syntax highlighting (```bash, ```go)
    - Includes environment variable examples
    - Provides copy-paste ready code snippets
    - Links to knowledge base guidance for deeper details
  </action>
  <verify>grep -A 20 "## Langfuse Tracing" CLAUDE.md</verify>
  <done>CLAUDE.md contains complete Langfuse integration section</done>
</task>

<task type="auto">
  <name>Create knowledge base guidance for Langfuse tracing</name>
  <files>guides/guide.golang.langfuse-tracing.md</files>
  <action>
    Create knowledge base guidance file at project-relative path:
    `guides/guide.golang.langfuse-tracing.md`

    Note: The exact path may be environment-specific. Common locations:
    - For this project: `guides/guide.golang.langfuse-tracing.md`
    - For universal kb: `$KB_PATH/guides/guide.golang.langfuse-tracing.md`
    - For Obsidian vault: `$VAULT_PATH/guides/guide.golang.langfuse-tracing.md`

    Create the file with this structure:

    ```markdown
    ---
    title: Golang Langfuse Tracing Guidelines
    description: Langfuse SDK integration patterns for observability in Go agent frameworks
    updated: [current date]
    created: [current date]
    tags:
      - golang
      - tracing
      - langfuse
      - observability
      - hooks
    ---

    # Langfuse Tracing in Go

    ## What is Langfuse Tracing?

    - Open-source observability platform for LLM applications
    - Captures traces, spans, and metrics for agent operations
    - Provides insights into prompt performance, token usage, and latency
    - Enables debugging of multi-agent interactions and tool chains

    ## Core Concepts

    ### Trace Hierarchy
    - Trace: Top-level container for a single session/operation
    - Root Span: Session-level parent span
    - Child Spans: LLM calls, tool executions, agent lifecycle events
    - Span Types: LLM (generation), Tool (execution), Agent (lifecycle), Custom (user-defined)

    ### Hook-Based Integration
    - Hooks provide natural injection points for span creation
    - Non-invasive: No changes to core agent logic required
    - Lazy initialization: Client created only when tracing enabled
    - Thread-safe: Concurrent agent execution supported

    ## Implementation Patterns

    ### Hook Structure Pattern

    Following logging_hook.go pattern:

    ```go
    type LangfuseHook struct {
        log        logger.LoggerService
        config     *config.LangfuseConfig
        client     *langfuse.Langfuse
        clientMu   *sync.Mutex           // Protects lazy client init
        traceCtxs  map[uuid.UUID]*TraceContext
        traceCtxsMu *sync.RWMutex        // Protects traceCtxs map
    }
    ```

    Key principles:
    - Lazy client initialization (getClient() with mutex)
    - Thread-safe trace context access (RWMutex for reads, Mutex for writes)
    - No-op hooks when tracing disabled
    - Graceful degradation on client errors

    ### Trace Context Lifecycle

    ```go
    type TraceContext struct {
        TraceID   string                    // Langfuse trace ID
        RootSpan  interface{}                // SDK root span
        Spans     map[string]interface{}     // Child spans by ID
        SessionID uuid.UUID                 // Session ID
        CreatedAt time.Time                  // Creation timestamp
    }
    ```

    Lifecycle:
    1. Session start: Create TraceContext, generate TraceID
    2. Operation: Create child spans, store in Spans map
    3. Session end: End root span, flush to backend, remove context

    ### Span Creation Patterns

    **LLM Spans:**
    - Create in BeforeLLMRequest with model and input
    - Update in AfterLLMResponse with output and usage
    - Mark error in OnLLMError with error details

    **Tool Spans:**
    - Create in BeforeToolExecution with tool name and args
    - Update in AfterToolExecution with result
    - Mark error in OnToolError with error details

    **Agent Spans:**
    - Create in BeforeAgentSpawn/BeforeAgentRemove
    - Update in AfterAgentSpawn/AfterAgentRemove
    - Track hierarchy via ParentAgentID/NewAgentID

    ## Testing Patterns

    ### Mock Langfuse Client

    For testing without actual Langfuse backend:

    ```go
    // Create mock client that implements required methods
    type mockLangfuseClient struct {
        t *testing.T
        traces []*traceRecord
    }

    func (m *mockLangfuseClient) StartTrace(ctx context.Context, name string) Trace {
        trace := &traceRecord{
            ID:   uuid.New().String(),
            Name:  name,
            Spans: []spanRecord{},
        }
        m.traces = append(m.traces, trace)
        return trace
    }
    ```

    ### Test Concurrent Access

    ```go
    func TestConcurrentTraceContext(t *testing.T) {
        hook := &LangfuseHook{
            traceCtxs:   make(map[uuid.UUID]*TraceContext),
            traceCtxsMu: &sync.RWMutex{},
        }

        var wg sync.WaitGroup
        for i := 0; i < 10; i++ {
            wg.Add(1)
            go func(sessionID uuid.UUID) {
                defer wg.Done()
                tc := hook.createTraceContext(sessionID)
                assert.NotNil(t, tc)
            }(uuid.New())
        }
        wg.Wait()
    }
    ```

    ## Best Practices

    ### Configuration
    - Default to disabled: Opt-in tracing avoids overhead
    - Validate credentials on init: Fail fast on bad config
    - Lazy client creation: No SDK calls when disabled

    ### Performance
    - Batch flush operations: Send traces in groups
    - Async flush when possible: Don't block agent execution
    - Background cleanup: Remove old trace contexts periodically

    ### Error Handling
    - Log but don't fail: Tracing errors are non-fatal
    - Preserve partial data: Capture what you can on errors
    - Clean up resources: Always remove trace contexts

    ### Thread Safety
    - Use RWMutex for read-heavy maps: Allow concurrent reads
    - Minimize lock duration: Copy data before releasing
    - Document lock ordering: Avoid deadlocks with multiple locks

    ## Common Pitfalls

    ### Memory Leaks
    - Always remove trace contexts on session end
    - Clean up orphaned contexts during shutdown
    - Use defer for cleanup in error paths

    ### Race Conditions
    - Release traceCtxsMu before calling propagateTraceID
    - Don't hold locks while calling external code
    - Use unique span IDs (UUID, not counters)

    ### Missing Spans
    - Create spans in before-hooks, update in after-hooks
    - Store span ID in HookContext.Data for correlation
    - Handle missing span IDs gracefully (return early, don't panic)

    ## Additional Resources

    - Langfuse Go SDK: https://github.com/git-hulk/langfuse-go
    - Langfuse Documentation: https://langfuse.com/docs
    - Hook patterns: [[guide.golang.testing]]
    - Prompt optimization: [[guide.general.prompt-optimization]]

    Ensure guidance is:
    - Universal (no project-specific paths or details)
    - Bullet-point format for easy scanning
    - Code examples using ```go and ```bash
    - Cross-references to related guidance
    ```

    Save this file to the appropriate guides directory for your environment.
  </action>
  <verify>test -f guides/guide.golang.langfuse-tracing.md && cat guides/guide.golang.langfuse-tracing.md | head -50</verify>
  <done>Knowledge base guidance created with complete Langfuse tracing patterns</done>
</task>

<task type="auto">
  <name>Add See Also reference in CLAUDE.md</name>
  <files>CLAUDE.md</files>
  <action>
    Add "See Also" section at end of CLAUDE.md:

    ## See Also

    - [Langfuse Tracing Guide](guides/guide.golang.langfuse-tracing.md) - Comprehensive guidance on Langfuse integration patterns
    - [Prompt Optimization](#prompt-optimizer) - Automatic prompt improvement using execution feedback
    - [Go Testing Guide](guides/guide.golang.testing.md) - Unit and integration testing patterns

    Note: Use relative paths or wiki-style links that work across environments.
    For GitHub/Forgejo hosted docs: `../blob/main/guides/guide.golang.langfuse-tracing.md`
    For local Obsidian: `[[guide.golang.langfuse-tracing]]`
    For this project: `guides/guide.golang.langfuse-tracing.md`

    Add this section after all other content, maintaining consistent formatting.
  </action>
  <verify>grep -A 10 "## See Also" CLAUDE.md</verify>
  <done>CLAUDE.md includes cross-references to related documentation</done>
</task>

</tasks>

<verification>
Overall phase checks:
- CLAUDE.md updated with Langfuse configuration section
- Environment variables documented with defaults
- Hook registration example provided
- Knowledge base guidance created with universal patterns
- Path references are project-relative or environment-agnostic
- Cross-references added between documentation
- No hardcoded absolute paths like `/home/denkhaus/dev/kb/`
</verification>

<success_criteria>
1. CLAUDE.md contains "## Langfuse Tracing" section
2. All 6 environment variables documented with examples
3. Hook registration code snippet provided
4. guide.golang.langfuse-tracing.md created in project guides directory
5. Guidance uses universal patterns (no gollum-specific paths)
6. Path references are environment-agnostic
7. See Also section added to CLAUDE.md
</success_criteria>

<output>
After completion, create `.planning/phases/011-testing-documentation/011-testing-documentation-03-SUMMARY.md`
</output>
