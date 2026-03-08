# Structured Agentic Workflows

**Status:** Design Phase (Revised)
**Created:** 2025-03-07
**Revised:** 2025-03-07
**Purpose:** Composable, reusable, lintable, shareable agentic workflows

---

## Vision

A declarative XML-based workflow engine where **flows call flows** to compose complex agentic workflows.

**Key Principles:**
- **Composable**: Flows are reusable components that call other flows
- **Structured**: Explicit state transitions with computed conditions
- **Lintable**: Schema validation before execution
- **Shareable**: Portable flow definitions between projects
- **LLM-Creatable**: LLMs can create flows with linter feedback

**Note:** While the *orchestration* is deterministic (explicit states, transitions, conditions), LLM step *outputs* are inherently non-deterministic.

---

## Architecture

```
+-------------------------------------------------------------+
|                    Flow Engine (Go)                         |
+-------------------------------------------------------------+
|  +-------------+  +--------------+  +------------------+   |
|  | Flow Loader |->|   Linter     |->|  Executor        |   |
|  | (XML parser)|  | (Validator)  |  | (State Machine)  |   |
|  +-------------+  +--------------+  +------------------+   |
|                                              |               |
|                                   +----------v----------+   |
|                                   | Execution Store     |   |
|                                   | (Full History)      |   |
|                                   +---------------------+   |
+-------------------------------------------------------------+
```

### Components

| Component | Responsibility |
|-----------|----------------|
| **Flow Loader** | Parse XML flows, resolve flow references from filesystem |
| **Linter** | Validate schema, check transitions, validate expressions |
| **Executor** | State machine engine, step execution, history tracking |
| **Execution Store** | Persist full execution history for replay/debug |

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **State namespacing** | Sub-flow states become `parent.child` - unified state machine |
| **Absolute path references** | All field references use `${input.*}`, `${output.*}`, `${context.*}` - explicit scope |
| **Input fields: read-only** | Inputs are immutable parameters - prevents accidental modification |
| **Computed: context/output only** | Inputs cannot have computed fields - inputs are external parameters |
| **Output: not for intermediate** | Output fields cannot be used as step params - use context for intermediates |
| **Func steps require output** | No magic logging; engine handles logging via Langfuse |
| **Computed deps: auto-detect** | Engine parses expressions, validates at load time |
| **Computed fields: immutable** | Cannot be modified by `set_context_field` tool |
| **Error handling: lifecycle** | Full error lifecycle with `${error.*}` context fields |
| **Parallel groups: fail-fast** | Simple, predictable behavior |
| **Call syntax: auto-detect** | Filesystem-based: folder with main.xml vs single .xml file |
| **No versioning** | What's in the filesystem gets executed - no version pinning |
| **Go functions: registry** | Explicit signature registry, validated by linter |
| **MCP output: path attribute** | `<string path="head.ref" assign="branch"/>` - unified syntax |
| **Expressions: formal grammar** | EBNF grammar with type coercion rules |
| **Flow interface: mandatory** | Every flow MUST have `<input>` and `<output>` |
| **Fragments: removed** | Use sub-flows instead - consistent model |
| **Transitions: LLM + conditions** | `transition_to` tool AND condition-based transitions |
| **Timeouts: per-call** | Each call specifies timeout, default 60s |
| **Imports: deferred** | Universal `<import>` for top-level nodes in later version |
| **Extensions: registry** | Registered at startup, XML declares usage |
| **Observability: Langfuse** | Via gollum framework's existing hooks |

---

## Package Structure

```
pkg/flows/
+-- types.go           # Core types: Flow, State, Step, Context
+-- parser.go          # XML parsing with schema validation
+-- linter.go          # Flow validation and expression validation
+-- executor.go        # State machine execution engine
+-- store.go           # Execution history persistence
+-- context.go         # Context field management and validation
+-- transitions.go     # State transition logic and conditions
+-- tools.go           # Built-in tools (set_context_field, etc.)
+-- registry.go        # Function registry with signatures
+-- expressions.go     # Expression evaluation (GT, EQ, AND, etc.)
+-- resolver.go        # Flow resolution from filesystem

.gollum/flows/
+-- idea_plan.md       # This document
+-- examples/          # Example flows
|   +-- simple-flow.xml
|   +-- step-types-example.xml
|   +-- nested-context-example.xml
+-- modules/           # Reusable modules (folder with main.xml)
|   +-- forgejo-workflow/
|   |   +-- main.xml
|   |   +-- fetch-pr.xml
|   |   +-- determine-phase.xml
|   |   +-- review-phase.xml
|   +-- code-analysis/
|       +-- main.xml
|       +-- complexity-check.xml
|       +-- security-scan.xml
```

---

## XML Schema Specification

### Root Element

```xml
<?xml version="1.0" encoding="UTF-8"?>
<flow name="my-flow" version="1.0">
    <!-- MANDATORY: Typed interface -->
    <input>
        <int name="pr_number" required="true" />
        <string name="repo_owner" default="denkhaus" />
    </input>

    <output>
        <string name="review_status" />
        <object name="results">
            <int name="issues_found" />
        </object>
    </output>

    <!-- Internal context (not part of interface) -->
    <context>
        <string name="internal_state" />
        <computed name="is_ready" type="bool" when="EQ(status, 'ready')" />
    </context>

    <!-- Agents defined locally -->
    <agents>
        <agent name="worker" model="anthropic/claude-opus-4-6">
            <prompt><![CDATA[...]]></prompt>
            <temperature>0.0</temperature>
        </agent>
    </agents>

    <!-- States -->
    <states>
        <state name="init" initial="true">...</state>
    </states>
</flow>
```

### Input/Output Interface (MANDATORY)

Every flow MUST define explicit typed interface:

```xml
<input>
    <!-- Primitives -->
    <int name="pr_number" required="true" min="1" />
    <string name="status" enum="open,closed,merged" />
    <bool name="verbose" default="false" />
    <float name="threshold" default="0.5" />

    <!-- Collections -->
    <array name="files_changed" type="string" />
    <map name="metadata" />

    <!-- Nested objects -->
    <object name="config">
        <string name="branch" />
        <bool name="force" default="false" />
    </object>
</input>

<output>
    <string name="result" />
    <int name="items_processed" />
</output>
```

**Linter validates:**
- Caller provides all required input fields
- Caller's input types match callee's expected types
- Output mappings reference fields that actually exist in callee's output

### Context Definition

Internal context fields (not part of interface):

```xml
<context>
    <!-- Primitives -->
    <string name="current_phase" />
    <int name="retry_count" default="0" />

    <!-- Nested objects -->
    <object name="phases">
        <object name="testing">
            <int name="errors" />
            <int name="warnings" />
        </object>
    </object>

    <!-- Computed fields (auto-evaluated, dependencies auto-detected, IMMUTABLE) -->
    <computed name="is_open" type="bool" when="EQ(status, 'open')" />
    <computed name="is_large" type="bool" when="GT(changed_files, 10)" />
    <computed name="has_test_errors" type="bool" when="GT(phases.testing.errors, 0)" />
    <computed name="needs_refactor" type="bool" when="AND(is_large, has_test_errors)" />
</context>
```

**When to use `<context>`:**
- Only define context fields that are referenced as `${context.*}` in prompts
- If fields are only used for intermediate step output and then passed to `<output>`, they don't need context declaration
- Output fields should NOT be duplicated in context unless referenced as `${context.*}` somewhere

**Computed Field Rules:**
- Dependencies auto-detected from expression parsing
- Evaluated at state entry (dependency-ordered)
- **IMMUTABLE** - `set_context_field` tool cannot modify computed fields
- Linter validates: no circular dependencies, all referenced fields exist
- Can be placed in `<output>` section if the computed value is part of the flow's interface

**Expression Syntax:**
- Field reference in computed expressions: Must use absolute paths (e.g., `context.pr.state`, `input.changed_files`)
- Nested: `object.subobject.field` (e.g., `context.phases.testing.errors`)
- Functions: `GT(field, value)`, `EQ(field, 'value')`, `AND(cond1, cond2)`, `NOT(cond)`
- String literals: Single quotes `'value'` (XSD-safe)
- In prompts: `${input.field_name}`, `${output.field_name}`, `${context.field_name}`
- **All references must use absolute paths** (input/output/context prefix required)

### Field Access and Assignment Rules

**Input Fields (Read-Only):**
- **FORBIDDEN**: Assignments to input fields (linter error)
- Input fields are immutable and cannot be modified by any step
- Input fields can be READ as `${input.field_name}` in prompts and expressions
- Input fields CANNOT have computed parameters (no `<computed>` in `<input>`)

**Output Fields (Write-Only):**
- Output fields are written via `set_output_field` tool or step output assignments
- Output fields can be READ as `${output.field_name}` in prompts within the same flow
- Output fields CAN have computed parameters in `<output>` section
- **FORBIDDEN**: Using output fields as input parameters for other steps/functions (use context instead)

**Context Fields (Read/Write):**
- Context fields can be read and written via `set_context_field` tool
- Context fields can be READ as `${context.field_name}` in prompts and expressions
- Context fields CAN have computed parameters
- Computed context fields are **IMMUTABLE** (cannot be modified by `set_context_field`)

**Correct Usage Pattern:**
```xml
<!-- CORRECT: Use context for intermediate values -->
<input>
    <int name="pr_number" />
</input>

<context>
    <string name="pr_title" />
    <computed name="is_open" type="bool" when="EQ(context.pr.state, 'open')" />
</context>

<output>
    <string name="result" />
    <computed name="is_complete" type="bool" when="EQ(output.result, 'done')" />
</output>

<!-- Step uses input for params, assigns to context -->
<step type="mcp" tool="forgejo/get_pr">
    <params>
        <param name="index" value="${input.pr_number}" />  <!-- CORRECT: read input -->
    </params>
    <output assign="${context.pr_title}" />  <!-- CORRECT: assign to context -->
</step>

<!-- Another step uses context as param -->
<step type="llm" agent="worker">
    <prompt>Analyze ${context.pr_title}</prompt>  <!-- CORRECT: read context -->
</step>
```

**Incorrect Usage Pattern:**
```xml
<!-- WRONG: Trying to assign to input -->
<step type="llm" agent="worker">
    <prompt>Analyze ${input.pr_title}</prompt>
    <!-- LINTER ERROR: Cannot use set_output_field on input fields -->
</step>

<!-- WRONG: Computed field in input -->
<input>
    <computed name="is_open" type="bool" when="EQ(state, 'open')" />
    <!-- LINTER ERROR: Computed fields not allowed in input -->
</input>

<!-- WRONG: Using output as param -->
<step type="func" function="ProcessResult">
    <params>
        <param name="data" value="${output.result}" />
        <!-- LINTER ERROR: Output fields cannot be used as parameters -->
    </params>
</step>
```

### Agents (Flow-Local)

Agents are defined per-flow:

```xml
<agents>
    <agent name="reviewer" model="anthropic/claude-opus-4-6">
        <prompt><![CDATA[
            You are a code reviewer.
            Be specific and actionable.
        ]]></prompt>
        <temperature>0.0</temperature>
        <max_tokens>4000</max_tokens>
    </agent>
</agents>
```

**Note:** Universal `<import>` for sharing agents across flows is deferred to a later version.

### States and Transitions

```xml
<states>
    <state name="init" initial="true">
        <steps>
            <!-- Steps here -->
        </steps>

        <transitions>
            <transition to="analyzing" when="is_open" />
            <transition to="done" otherwise="true" />
        </transitions>
    </state>

    <state name="done" />
</states>
```

**State Namespacing:**

When a flow calls a sub-flow, sub-flow states are namespaced:

```
Parent flow states:     init, processing, done
Sub-flow "fetch" states: fetch.init, fetch.fetching, fetch.done
```

### Error Handling

State-based error handling with full error lifecycle:

```xml
<state name="fetch" initial="true">
    <steps>
        <step type="mcp" tool="forgejo/get_pr" name="get-pr">
            <on-error state="error" />
            <retry count="3" backoff="exponential" />
            <timeout>30s</timeout>
        </step>
    </steps>
    <transitions>
        <transition to="process" />
    </transitions>
</state>

<state name="error">
    <steps>
        <step type="llm" agent="coordinator">
            <prompt><![CDATA[
                Step ${error.step_name} failed at ${error.timestamp}
                Error: ${error.message}

                Decide recovery action using transition_to tool.
            ]]></prompt>
            <tools>transition_to, emit_log</tools>
        </step>
    </steps>
</state>
```

**Error Lifecycle:**
```
Step Execution
    │
    ├── Success ──> Continue to next step
    │
    └── Failure ──> Has retry?
                        │
                        ├── Yes ──> Retry loop (exponential backoff)
                        │                │
                        │                ├── Success ──> Continue
                        │                └── Exhausted ──> Capture error
                        │
                        └── No ──> Capture error
                                        │
                                        v
                            Set ${error.*} context fields:
                            - error.step_name
                            - error.step_type
                            - error.message
                            - error.timestamp
                                        │
                                        v
                            Transition to <on-error state="...">
```

### Step Types

#### 1. LLM Step
```xml
<step type="llm" agent="reviewer">
    <prompt><![CDATA[
        Analyze PR #${input.pr_number}
        Title: ${context.pr_title}

        Use set_output_field to set output fields.
        Use set_context_field to set context fields.
    ]]></prompt>
    <tools>set_output_field, set_context_field, get_context, emit_log, transition_to</tools>
    <timeout>60s</timeout>
    <on-error state="error" />
</step>
```

**Field references in prompts:**
- Input fields: `${input.field_name}`
- Output fields: `${output.field_name}`
- Context fields: `${context.field_name}`
- Error fields: `${error.field_name}`

#### 2. Shell Step
```xml
<step type="shell" name="run-tests">
    <cmd><![CDATA[go test ${target_dir} -v]]></cmd>
    <output>
        <string path="stdout" assign="test_output" />
        <int path="exit_code" assign="test_exit_code" />
    </output>
    <timeout>300s</timeout>
    <on-error state="error" />
</step>
```

**Note:** Variables in shell commands are **auto-escaped** to prevent injection.

#### 3. Func Step (Go Functions)
```xml
<!-- Built-in stdlib - MUST have output assignment -->
<step type="func" function="strings.ToUpper">
    <params>
        <param name="s" value="${input.input_text}" />
    </params>
    <output assign="${context.upper_result}" />
</step>

<!-- Extension function -->
<step type="func" function="GetPRDiff">
    <params>
        <param name="owner" value="${input.repo_owner}" />
        <param name="index" value="${input.pr_number}" />
    </params>
    <output assign="${context.pr_diff}" />
    <timeout>60s</timeout>
    <on-error state="error" />
</step>
```

**Rules:**
- **Output assignment is MANDATORY** - linter error if missing
- **Use absolute paths**: `${context.field_name}` for context, `${output.field_name}` for output
- Engine handles logging internally via Langfuse hooks

#### 4. MCP Step
```xml
<step type="mcp" tool="forgejo/get_pr" name="get-pr">
    <params>
        <param name="owner" value="${input.repo_owner}" />
        <param name="repo" value="${input.repo_name}" />
        <param name="index" value="${input.pr_number}" />
    </params>
    <output>
        <string path="title" assign="${output.pr_title}" />
        <string path="state" assign="${output.pr_state}" />
        <string path="head.ref" assign="${output.head_branch}" />
    </output>
    <timeout>30s</timeout>
    <on-error state="error" />
</step>
```

**Output mapping:** Uses `path` attribute for JSONPath dot notation. `path="head.ref"` maps to `response.head.ref`.

**Absolute paths:** Use `${output.field_name}` for output fields, `${context.field_name}` for context fields.

**Missing paths:** Trigger `<on-error>` if no handler, flow fails.

#### 5. Call Step (Flow/Module)
```xml
<call ref="fetch-pr" timeout="60s">
    <input>
        <field name="pr_number" value="${input.pr_number}" />
    </input>
    <output>
        <field name="title" value="${context.pr.title}" />
    </output>
    <on-error state="error" />
</call>
```

**Conditional calls:**
```xml
<call ref="security-scan" timeout="300s" when="EQ(context.is_deep_scan, true)">
    <input><field name="target" value="${input.target_dir}" /></input>
</call>
```

**Absolute paths:**
- Input mapping: `${input.field_name}` (from caller's context)
- Output mapping: `${context.field_name}` (to caller's context) or `${output.field_name}` (to caller's output)

**No versioning:** What's in the filesystem gets executed. `ref` is always a path.

### Unified Output Mapping Syntax

All step types use consistent output mapping syntax with **absolute paths**:

```xml
<!-- Simple assignment to context -->
<output assign="${context.result}" />

<!-- Simple assignment to output -->
<output assign="${output.result}" />

<!-- Path-based extraction (MCP, Shell, Func returning object) -->
<output>
    <string path="title" assign="${context.pr_title}" />
    <int path="count" assign="${output.item_count}" />
    <string path="head.ref" assign="${context.branch}" />
</output>
```

**Absolute Path Rules:**
- `${context.field_name}` - Assign to context field
- `${output.field_name}` - Assign to output field
- Local variable names without prefix are NOT allowed (linter error)

### Parallel Groups

```xml
<group name="parallel-checks" parallel="true">
    <step type="shell">
        <cmd><![CDATA[go test ./...]]></cmd>
    </step>
    <step type="mcp" tool="security/scan" />
</group>
```

**Failure mode:** Fail-fast. If any step fails, all running steps are cancelled and `<on-error>` triggers.

### Context Isolation (Sub-flow Calls)

Sub-flows have **explicit-only** context access:

```
Parent context:    {pr_number: 123, repo: "gollum", ...}
                          │
                          v input mapping
Sub-flow receives:  {pr_number: 123}  <!-- ONLY what's in <input> -->
                          │
                          v execution
Sub-flow context:   {pr_number: 123, internal: "x", ...}
                          │
                          v output mapping
Parent receives:    {pr.title: "Fix bug"}  <!-- ONLY what's in <output> -->
                          │
                          v merge
Parent context:     {pr_number: 123, repo: "gollum", pr.title: "Fix bug", ...}
```

---

## Built-in Tools

Tools must be **explicitly declared** in LLM step `<tools>`.

### set_output_field
```go
Input: {"name": "review_status", "value": "approved"}
// Engine validates type and constraints
// ERROR: Cannot modify computed fields
// Sets fields in the <output> section
```

### set_context_field
```go
Input: {"name": "internal_state", "value": "processing"}
// Engine validates type and constraints
// ERROR: Cannot modify computed fields
// Sets fields in the <context> section
```

### get_context
```go
Input: {"fields": ["pr_number", "status"]}
Output: {"pr_number": 123, "status": "open"}
```

### emit_log
```go
Input: {"level": "info", "message": "Starting analysis"}
// Logged via Langfuse hooks
```

### transition_to
```go
Input: {"to": "fixing"}
// Engine validates transition is allowed from current state
Output: {"success": true, "from": "analyzing", "to": "fixing"}
```

---

## State Machine Semantics

### Execution Model

1. **Load Flow**: Parse XML, resolve references from filesystem
2. **Lint**: Validate schema, check transitions, validate expressions
3. **Execute**:
   - Start at initial state
   - Execute steps (sequential or parallel)
   - Evaluate computed fields (dependency-ordered)
   - Transition when conditions match OR LLM calls transition_to
   - Persist to history

### Transition Triggers

1. **When condition**: `when="is_open"`
2. **Otherwise fallback**: `otherwise="true"`
3. **LLM tool call**: `transition_to(to="state_name")`

---

## Expression Grammar (EBNF)

```ebnf
expression    = boolean_expr | comparison_expr | string_func | primary
boolean_expr  = "AND" "(" expression "," expression ")"
              | "OR" "(" expression "," expression ")"
              | "NOT" "(" expression ")"
comparison    = "GT" "(" value "," value ")"
              | "GTE" "(" value "," value ")"
              | "LT" "(" value "," value ")"
              | "LTE" "(" value "," value ")"
              | "EQ" "(" value "," value ")"
              | "NEQ" "(" value "," value ")"
              | "BETWEEN" "(" value "," value "," value ")"
string_func   = "CONTAINS" "(" value "," value ")"
              | "STARTSWITH" "(" value "," value ")"
              | "ENDSWITH" "(" value "," value ")"
value         = field_ref | literal
field_ref     = identifier | identifier "." field_ref
literal       = string_lit | number_lit | bool_lit
string_lit    = "'" { char } "'"
number_lit    = [ "-" ] digit { digit } [ "." digit { digit } ]
bool_lit      = "true" | "false"
identifier    = letter { letter | digit | "_" }
```

### Type Coercion Rules

| Function | Rule |
|----------|------|
| `GT/LT/GTE/LTE` | Both operands must be numeric OR both string |
| `EQ/NEQ` | Type coercion allowed (string "123" EQ int 123 = true) |
| `AND/OR/NOT` | Operands must be boolean |
| `BETWEEN` | All three operands must be same type |

---

## Function Registry

Built-in functions with explicit signatures:

```go
// pkg/flows/registry.go
var BuiltinRegistry = FunctionRegistry{
    "strings.ToUpper": {
        Params:   []Param{{Name: "s", Type: "string"}},
        Returns:  "string",
        Variadic: false,
    },
    "strings.ToLower": {
        Params:   []Param{{Name: "s", Type: "string"}},
        Returns:  "string",
        Variadic: false,
    },
    "strings.Contains": {
        Params:   []Param{{Name: "s", Type: "string"}, {Name: "substr", Type: "string"}},
        Returns:  "bool",
        Variadic: false,
    },
    "strings.HasPrefix": {
        Params:   []Param{{Name: "s", Type: "string"}, {Name: "prefix", Type: "string"}},
        Returns:  "bool",
        Variadic: false,
    },
    "strings.HasSuffix": {
        Params:   []Param{{Name: "s", Type: "string"}, {Name: "suffix", Type: "string"}},
        Returns:  "bool",
        Variadic: false,
    },
    "fmt.Sprintf": {
        Params:   []Param{{Name: "format", Type: "string"}, {Name: "args", Type: "any"}},
        Returns:  "string",
        Variadic: true,
    },
    "len": {
        Params:   []Param{{Name: "v", Type: "any"}},
        Returns:  "int",
        Variadic: false,
    },
}
```

### Extension Registry

Extensions are registered at application startup:

```go
// Application startup
func init() {
    flows.RegisterExtension("GetPRDiff", FuncSpec{
        Params:   []Param{{Name: "owner", Type: "string"}, {Name: "index", Type: "int"}},
        Returns:  "string",
        Func:     forgejo.GetPRDiff,
    })
}
```

XML declares usage (linter validates availability):
```xml
<step type="func" function="GetPRDiff">
    <params>
        <param name="owner" value="${repo_owner}" />
        <param name="index" value="${pr_number}" />
    </params>
    <output assign="pr_diff" />
</step>
```

---

## Linter Features

### Schema Validation
- Required input fields present in calls
- Types match between caller and callee
- State transitions are valid
- Agent references exist
- Func step has output assignment
- Expressions are valid per grammar
- **No assignments to input fields** (linter error)
- **No computed fields in `<input>` section** (linter error)
- **No output fields used as step parameters** (linter error - use context instead)
- **All field references use absolute paths** (input/output/context prefix required)

### Best Practice Hints
- Suggest parallel groups for independent steps
- Flag missing timeout values
- Detect unreachable states
- Suggest expression simplification
- Warn on circular computed field dependencies

### Linter Output
```json
{
    "valid": false,
    "errors": [
        {
            "line": 42,
            "column": 10,
            "code": "FUNC_MISSING_OUTPUT",
            "message": "func step 'strings.ToUpper' must have output assignment"
        },
        {
            "line": 55,
            "column": 5,
            "code": "REQUIRED_FIELD_MISSING",
            "message": "required input field 'pr_number' not provided in call to 'fetch-pr'"
        },
        {
            "line": 23,
            "column": 12,
            "code": "INPUT_HAS_COMPUTED",
            "message": "computed field 'is_open' not allowed in <input> section (move to <context> or <output>)"
        },
        {
            "line": 67,
            "column": 8,
            "code": "ASSIGN_TO_INPUT",
            "message": "cannot assign to input field 'pr_number' (input fields are read-only)"
        },
        {
            "line": 89,
            "column": 15,
            "code": "OUTPUT_AS_PARAM",
            "message": "output field 'result' cannot be used as parameter (use context field instead)"
        },
        {
            "line": 101,
            "column": 5,
            "code": "RELATIVE_PATH_REFERENCE",
            "message": "field reference 'pr_title' must use absolute path ('${input.pr_title}', '${context.pr_title}', or '${output.pr_title}')"
        }
    ],
    "warnings": [
        {
            "line": 30,
            "message": "computed field 'is_ready' dependency 'status' not found in context"
        }
    ],
    "hints": [
        {
            "line": 20,
            "message": "steps 16-18 could run in parallel"
        }
    ]
}
```

---

## Execution History

```go
type ExecutionRecord struct {
    ID           string
    FlowName     string
    StartedAt    time.Time
    CompletedAt  time.Time
    Status       string
    States       []StateExecution
    Context      ContextSnapshot
    Errors       []ErrorRecord
}

type StateExecution struct {
    StateName   string            // Namespaced: "fetch.init", "fetch.done"
    StartedAt   time.Time
    Steps       []StepExecution
    Transitions []TransitionEvent
}

type ErrorRecord struct {
    StepName    string
    StepType    string
    Message     string
    Timestamp   time.Time
    RetryCount  int
}
```

---

## Complete Example: Forgejo PR Workflow

```xml
<?xml version="1.0" encoding="UTF-8"?>
<flow name="forgejo-pr-workflow" version="1.0">
    <description>Complete Forgejo PR workflow with structured phases.</description>

    <input>
        <int name="pr_number" required="true" />
        <string name="repo_owner" default="denkhaus" />
        <string name="repo_name" required="true" />
    </input>

    <output>
        <string name="review_status" />
        <string name="review_summary" />
    </output>

    <context>
        <object name="pr">
            <string name="title" />
            <string name="state" />
            <int name="changed_files" />
        </object>

        <computed name="is_open" type="bool" when="EQ(context.pr.state, 'open')" />
        <computed name="is_large" type="bool" when="GT(context.pr.changed_files, 10)" />

        <string name="current_phase" />
    </context>

    <agents>
        <agent name="coach" model="anthropic/claude-opus-4-6">
            <prompt>Workflow coordinator</prompt>
            <temperature>0.0</temperature>
        </agent>
        <agent name="reviewer" model="anthropic/claude-opus-4-6">
            <prompt>Code reviewer - be specific and actionable</prompt>
            <temperature>0.0</temperature>
        </agent>
    </agents>

    <states>
        <state name="init" initial="true">
            <steps>
                <call ref="fetch-pr" timeout="60s">
                    <input>
                        <field name="pr_number" value="${input.pr_number}" />
                        <field name="repo_owner" value="${input.repo_owner}" />
                        <field name="repo_name" value="${input.repo_name}" />
                    </input>
                    <output>
                        <field name="title" value="${context.pr.title}" />
                        <field name="state" value="${context.pr.state}" />
                        <field name="changed_files" value="${context.pr.changed_files}" />
                    </output>
                    <on-error state="error" />
                </call>
            </steps>
            <transitions>
                <transition to="determine-phase" when="EQ(context.is_open, true)" />
                <transition to="done" otherwise="true" />
            </transitions>
        </state>

        <state name="determine-phase">
            <steps>
                <step type="llm" agent="coach">
                    <prompt><![CDATA[
                        Analyze PR #${input.pr_number}: ${context.pr.title}
                        State: ${context.pr.state}, Files: ${context.pr.changed_files}, Large: ${context.is_large}

                        Determine phase: idea, implement, review, fix, or merge.
                        Use set_context_field("current_phase", "<phase>")
                    ]]></prompt>
                    <tools>set_context_field, get_context, emit_log</tools>
                </step>
            </steps>
            <transitions>
                <transition to="review" when="EQ(context.current_phase, 'review')" />
                <transition to="done" otherwise="true" />
            </transitions>
        </state>

        <state name="review">
            <steps>
                <call ref="review-phase" timeout="300s">
                    <input>
                        <field name="pr_number" value="${input.pr_number}" />
                        <field name="repo_owner" value="${input.repo_owner}" />
                        <field name="repo_name" value="${input.repo_name}" />
                    </input>
                    <output>
                        <field name="status" value="${output.review_status}" />
                        <field name="summary" value="${output.review_summary}" />
                    </output>
                    <on-error state="error" />
                </call>
            </steps>
            <transitions>
                <transition to="done" />
            </transitions>
        </state>

        <state name="error">
            <steps>
                <step type="llm" agent="coach">
                    <prompt><![CDATA[
                        Workflow failed:
                        Step: ${error.step_name}
                        Error: ${error.message}

                        Decide recovery action using transition_to tool.
                    ]]></prompt>
                    <tools>transition_to, emit_log</tools>
                </step>
            </steps>
        </state>

        <state name="done" />
    </states>
</flow>
```

**Key Absolute Path Patterns:**
- Input references: `${input.pr_number}`, `${input.repo_owner}`
- Output references: `${output.review_status}`, `${output.review_summary}`
- Context references: `${context.pr.title}`, `${context.current_phase}`
- Error references: `${error.step_name}`, `${error.message}`
- In computed field expressions: `EQ(context.pr.state, 'open')` (prefix required)

---

## Implementation Phases

### Phase 1: Core (MVP)
- [ ] Define Go types (Flow, State, Step, Context, Input, Output)
- [ ] XML parser with schema validation
- [ ] Typed input/output interfaces
- [ ] State machine executor with namespaced states
- [ ] Context management with computed field evaluation
- [ ] Condition-based transitions
- [ ] LLM step execution
- [ ] Built-in tools: set_output_field, set_context_field, get_context, emit_log, transition_to
- [ ] Absolute path validation (input/output/context/error prefixes)
- [ ] Expression evaluation with formal grammar
- [ ] Error lifecycle with ${error.*} fields
- [ ] Linter with expression validation
- [ ] Execution history tracking

### Phase 2: Composition
- [ ] Sub-flow calls with context isolation
- [ ] Unified call syntax (auto-detect module vs flow)
- [ ] Input/output mapping validation
- [ ] Function registry with signatures
- [ ] Func step execution (built-in stdlib)

### Phase 3: Integration
- [ ] MCP step execution
- [ ] Shell step execution with auto-escaping
- [ ] Extension function support
- [ ] Parallel step groups (fail-fast)
- [ ] Retry logic with exponential backoff

### Phase 4: Advanced
- [ ] Import directives for sharing nodes
- [ ] Advanced linter hints
- [ ] CLI commands (flow run, lint, validate)
- [ ] Flow debugging/replay tools
