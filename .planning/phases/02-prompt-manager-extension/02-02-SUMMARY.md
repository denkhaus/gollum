---
phase: 02-prompt-manager-extension
plan: 02
subsystem: template-rendering
tags: [go-templates, text-template, prompt-management, backward-compatibility]

# Dependency graph
requires:
  - phase: 02-prompt-manager-extension
    plan: 01
    provides: PromptManager with store integration, lazy bootstrap, embedded templates
provides:
  - Template rendering with text/template for dynamic prompt variables
  - Named template support for built-in prompts (systemprompt, supervisorprompt, etc.)
  - Backward-compatible wrapper methods for existing PromptManager API
  - GetPromptWithContext for convenient retrieve+render operations
affects: [prompt-optimization, agent-system, future-ui-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Named template pattern with define/end blocks
    - Base ID extraction from versioned prompt IDs
    - Template name mapping for built-in prompts
    - RenderContext with SubAgentContext, AgentContext, and Values

key-files:
  created: []
  modified:
    - pkg/prompt/manager/render.go
    - pkg/prompt/manager/backward_compat.go
    - pkg/prompt/manager/manager_test.go

key-decisions:
  - "Use text/template.Lookup for named template execution - templates use define/end blocks"
  - "Extract base ID from versioned prompt IDs for template name mapping (system@1.0.0 -> system)"
  - "GetCompacterPrompt accepts map[string]interface{} directly for template variable access"

patterns-established:
  - "Template rendering: Parse -> Lookup named template -> Execute with data"
  - "Context building: Values map + SubAgent fields + Agent fields combined into single data map"
  - "Fallback pattern: Try named template first, fall back to root template execution"

# Metrics
duration: 25min
completed: 2026-02-01
---

# Phase 2: Plan 2 - Template Rendering and Backward Compatibility Summary

**Template rendering with text/template supporting named templates, context variables, and backward-compatible wrapper methods for built-in prompts**

## Performance

- **Duration:** 25 min
- **Started:** 2026-02-01T13:27:58Z
- **Completed:** 2026-02-01T13:53:05Z
- **Tasks:** 3 (combined into single commit)
- **Files modified:** 3

## Accomplishments

- **Template rendering with text/template**: Prompt templates with `{{- define "name"}}` blocks are parsed and executed correctly using `template.Lookup()`
- **Context variable support**: RenderContext with SubAgentContext, AgentContext, and generic Values map for flexible template data
- **Backward compatibility preserved**: GetCompacterPrompt, GetSystemPrompt, GetSupervisorPrompt, GetSubagentPrompt work with new rendering system
- **Comprehensive test coverage**: Tests for rendering with variables, backward compatibility, nil contexts, and missing variables

## Task Commits

All tasks completed in single atomic commit:

1. **Tasks 1-3: Template rendering and backward compatibility** - `0682dbc` (feat)
   - Implemented RenderPrompt with text/template
   - Added GetPromptWithContext convenience method
   - Created backward-compatible wrapper methods
   - Added comprehensive tests

**Plan metadata:** (pending)

## Files Created/Modified

- `pkg/prompt/manager/render.go` (122 lines) - Template rendering implementation
  - RenderPrompt method with text/template parsing and named template support
  - GetPromptWithContext for retrieve + render in one call
  - buildTemplateData to combine Values, SubAgent, and Agent context
  - extractBaseID helper for versioned ID handling
  - templateNameMap mapping prompt IDs to template names

- `pkg/prompt/manager/backward_compat.go` (61 lines) - Backward-compatible wrapper methods
  - GetCompacterPrompt with proper map handling for template variables
  - GetSystemPrompt returning built-in system prompt
  - GetSupervisorPrompt returning built-in supervisor prompt
  - GetSubagentPrompt with role, description, and tool names

- `pkg/prompt/manager/manager_test.go` (584 lines) - Comprehensive test coverage
  - TestRenderPrompt_WithVariables, TestRenderPrompt_AgentContext, TestRenderPrompt_ValuesContext
  - TestGetPromptWithContext for convenience method
  - TestBackwardCompatibility_* for all four wrapper methods
  - TestBackwardCompatibility_GetCompacterPromptWithCustomTemplate for custom prompts
  - TestRenderPrompt_MissingVariable, TestRenderPrompt_NilContext, TestRenderPrompt_NilPrompt
  - TestRenderPrompt_AllContextFields for combined context usage

## Decisions Made

1. **Use text/template.Lookup for named templates**: Built-in prompt templates use `{{- define "name"}}...{{- end}}` syntax, requiring template name lookup rather than direct execution

2. **Extract base ID from versioned prompt IDs**: Prompt IDs from store include version (e.g., "system@1.0.0"), but template name mapping uses base ID ("system")

3. **GetCompacterPassPassPassAccepts map[string]interface{} directly**: When data is a map, use it directly as Values rather than wrapping in another map level, enabling `{{.Data}}` access instead of `{{.data.Data}}`

4. **Template name mapping for built-in prompts**: Hard-coded map of prompt ID to template name (system -> systemprompt, supervisor -> supervisorprompt, etc.)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed template rendering returning empty strings**
- **Found during:** Task 1 (initial test runs)
- **Issue:** Templates with `{{- define "name"}}...{{- end}}` blocks returned "\n" instead of content when executed directly
- **Fix:** Added template.Lookup() to find and execute named templates by name, with fallback to root template
- **Files modified:** pkg/prompt/manager/render.go
- **Verification:** All backward compatibility tests now pass with expected template content
- **Committed in:** 0682dbc

**2. [Rule 1 - Bug] Fixed base ID extraction for versioned prompt IDs**
- **Found during:** Task 1 (after template lookup fix)
- **Issue:** Prompt.ID from store is versioned ("system@1.0.0") but templateNameMap uses base ID ("system")
- **Fix:** Added extractBaseID helper function to extract base ID before template name lookup
- **Files modified:** pkg/prompt/manager/render.go
- **Verification:** Template name lookup correctly matches built-in prompts
- **Committed in:** 0682dbc

**3. [Rule 1 - Bug] Fixed GetCompacterPrompt data wrapping**
- **Found during:** Task 2 (compacter test failure)
- **Issue:** data was wrapped as `Values["data"]` but templates expected direct `Values["Data"]` access
- **Fix:** Check if data is map[string]interface{}, use directly as Values; otherwise wrap in "data" key
- **Files modified:** pkg/prompt/manager/backward_compat.go
- **Verification:** TestBackwardCompatibility_GetCompacterPromptWithCustomTemplate passes
- **Committed in:** 0682dbc

**4. [Rule 1 - Bug] Fixed supervisor template name mismatch**
- **Found during:** Task 3 (supervisor test failure)
- **Issue:** templateNameMap had "supervisor" but actual template defined "supervisorprompt"
- **Fix:** Updated templateNameMap entry from "supervisor" to "supervisorprompt"
- **Files modified:** pkg/prompt/manager/render.go
- **Verification:** TestBackwardCompatibility_GetSupervisorPrompt passes
- **Committed in:** 0682dbc

**5. [Rule 2 - Missing Critical] Updated test expectations for bootstrap behavior**
- **Found during:** Task 3 (backward compatibility test failures)
- **Issue:** Tests saved custom templates to store but built-in IDs trigger bootstrap from embedded templates, overriding test content
- **Fix:** Removed custom template saves from tests, let them use actual built-in templates; added separate test for custom prompts
- **Files modified:** pkg/prompt/manager/manager_test.go
- **Verification:** All backward compatibility tests pass with actual template content
- **Committed in:** 0682dbc

---

**Total deviations:** 5 auto-fixed (3 bugs, 1 template name fix, 1 test expectation fix)
**Impact on plan:** All auto-fixes necessary for correct functionality. Template rendering now works with Go's text/template package properly.

## Issues Encountered

1. **Template define/end blocks**: Initial implementation didn't handle Go template's named template pattern. Fixed by using template.Lookup().

2. **Versioned prompt IDs**: Store returns prompts with versioned IDs (e.g., "system@1.0.0") but template name mapping uses base IDs. Fixed with extractBaseID helper.

3. **Test bootstrap behavior**: Tests that save custom templates with built-in IDs get overridden by lazy bootstrap. Fixed by updating tests to use actual built-in templates.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Template rendering fully functional with text/template
- All backward-compatible methods working with new rendering system
- Built-in prompts (system, supervisor, compacter, subagent) properly render with template variables
- Custom prompts can use GetPromptWithContext for template rendering
- Ready for prompt optimization and A/B testing features

---
*Phase: 02-prompt-manager-extension*
*Completed: 2026-02-01*
