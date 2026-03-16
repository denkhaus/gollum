# Function Type Transformation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the redundant `<step type="func" function="...">` syntax into a clearer distinction between built-in functions (`builtin="..."`) and extension functions (`extension="..."`).

**Architecture:**
- Replace the single `function` attribute with two mutually exclusive attributes: `builtin` and `extension`
- Built-in functions are hardcoded in the executor (e.g., `assign`)
- Extension functions are executed via Yaegi runner from packages declared in `<extensions>`
- Keep `type="mcp"` and `type="llm"` unchanged

**Tech Stack:** Go 1.23, XML parsing, encoding/xml, XSD schema validation

---

## Chunk 1: Core Type Definition Changes

### Task 1: Update Step Struct in types.go

**Files:**
- Modify: `pkg/flows/types.go:384-399`

**Changes:**
- Remove `Function string` field
- Add `Builtin string` field
- Add `Extension string` field

- [ ] **Step 1: Write failing test for new struct fields**

Create a test file `pkg/flows/types_step_test.go`:

```go
package flows

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStep_BuiltinAndExtensionFields(t *testing.T) {
	// Test builtin step
	step := &Step{
		Type:    "func",
		Builtin: "assign",
	}
	assert.Equal(t, "assign", step.Builtin)
	assert.Equal(t, "", step.Extension)

	// Test extension step
	step2 := &Step{
		Type:      "func",
		Extension: "GetPRDiff",
	}
	assert.Equal(t, "GetPRDiff", step2.Extension)
	assert.Equal(t, "", step2.Builtin)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows -run TestStep_BuiltinAndExtensionFields -v`
Expected: FAIL with "unknown field Builtin/Extension"

- [ ] **Step 3: Update Step struct definition**

In `pkg/flows/types.go`, line 384-399, replace:

```go
// Step is a single execution step
type Step struct {
	XMLName  xml.Name
	Type     string             `xml:"type,attr"`
	Name     string             `xml:"name,attr"`
	Agent    string             `xml:"agent,attr"`
	Function string             `xml:"function,attr"`
	Tool     string             `xml:"tool,attr"`
	Prompt   string             `xml:"prompt"`
	Cmd      string             `xml:"cmd"`
	Tools    string             `xml:"tools"`
	Timeout  string             `xml:"timeout"`
	Params   []StepParam        `xml:"params>param"`
	OnError  *OnErrorTransition `xml:"on-error"`
	Retry    *Retry             `xml:"retry"`
	Output   *StepOutput        `xml:"output"`
}
```

With:

```go
// Step is a single execution step
type Step struct {
	XMLName  xml.Name
	Type     string             `xml:"type,attr"`
	Name     string             `xml:"name,attr"`
	Agent    string             `xml:"agent,attr"`
	Builtin  string             `xml:"builtin,attr"`
	Extension string            `xml:"extension,attr"`
	Tool     string             `xml:"tool,attr"`
	Prompt   string             `xml:"prompt"`
	Cmd      string             `xml:"cmd"`
	Tools    string             `xml:"tools"`
	Timeout  string             `xml:"timeout"`
	Params   []StepParam        `xml:"params>param"`
	OnError  *OnErrorTransition `xml:"on-error"`
	Retry    *Retry             `xml:"retry"`
	Output   *StepOutput        `xml:"output"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows -run TestStep_BuiltinAndExtensionFields -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/types.go pkg/flows/types_step_test.go
git commit -m "feat(step): replace Function attribute with Builtin and Extension

Breaking change:
- Use builtin=\"assign\" instead of function=\"assign\"
- Use extension=\"GetPRDiff\" instead of function=\"GetPRDiff\"

This makes the distinction between built-in and extension functions
explicit and removes the redundancy of type=\"func\" function=\"...\""
```

---

## Chunk 2: Executor Logic Update

### Task 2: Update executeFuncStep in executor.go

**Files:**
- Modify: `pkg/flows/executor/executor.go:505-530`

- [ ] **Step 1: Write failing test for builtin execution**

In `pkg/flows/executor/func_step_test.go`, add:

```go
func TestExecuteFuncStep_BuiltinAttribute(t *testing.T) {
	// Create a test flow
	flow := &flows.Flow{
		Name: "test-flow",
		Input: &flows.InputBlock{
			Ints: []flows.FieldDef{{Name: "value", Type: flows.TypeInt}},
		},
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{{Name: "result", Type: flows.TypeInt}},
		},
		States: []flows.State{
			{
				Name:     "test-state",
				Initial:  true,
				Steps: []flows.Step{
					{
						Type:    "func",
						Builtin: "assign",
						Params: []flows.StepParam{
							{Name: "from", Value: "${input.value}"},
							{Name: "to", Value: "${output.result}"},
						},
					},
				},
			},
		},
	}

	exec := createTestExecutor(t, flow, map[string]string{"value": "42"})

	err := exec.Execute(context.Background())
	require.NoError(t, err)

	result, err := exec.GetOutputField("result")
	require.NoError(t, err)
	assert.Equal(t, 42, result)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -run TestExecuteFuncStep_BuiltinAttribute -v`
Expected: FAIL (executor still checks `Function` field)

- [ ] **Step 3: Update executeFuncStep to use Builtin/Extension**

In `pkg/flows/executor/executor.go`, line 505-530, replace:

```go
func (p *flowExecutorImpl) executeFuncStep(_ context.Context, step *flows.Step, stateName string) error {
	// Check for built-in "assign" function
	if step.Function == "assign" {
		return p.executeAssignStep(step, stateName)
	}

	// Use Yaegi runner from extension service for other functions
	funcRunner := p.extService.GetFuncRunner()
	if funcRunner == nil {
		return fmt.Errorf("extension service not available")
	}

	// Build args from params
	args := make(map[string]any)
	for _, param := range step.Params {
		value := p.substituteTemplate(param.Value)
		args[param.Name] = value
	}

	// Execute via Yaegi
	result, err := funcRunner.ExecuteFunc(step.Function, args)
	if err != nil {
		return &FuncError{
			Function: step.Function,
			Step:     stateName,
			Err:      err,
		}
	}
```

With:

```go
func (p *flowExecutorImpl) executeFuncStep(_ context.Context, step *flows.Step, stateName string) error {
	// Handle built-in functions
	if step.Builtin != "" {
		switch step.Builtin {
		case "assign":
			return p.executeAssignStep(step, stateName)
		default:
			return fmt.Errorf("unknown built-in function: %s", step.Builtin)
		}
	}

	// Handle extension functions
	if step.Extension != "" {
		funcRunner := p.extService.GetFuncRunner()
		if funcRunner == nil {
			return fmt.Errorf("extension service not available")
		}

		// Build args from params
		args := make(map[string]any)
		for _, param := range step.Params {
			value := p.substituteTemplate(param.Value)
			args[param.Name] = value
		}

		// Execute via Yaegi
		result, err := funcRunner.ExecuteFunc(step.Extension, args)
		if err != nil {
			return &FuncError{
				Function: step.Extension,
				Step:     stateName,
				Err:      err,
			}
		}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -run TestExecuteFuncStep_BuiltinAttribute -v`
Expected: PASS

- [ ] **Step 5: Update existing tests to use new syntax**

In `pkg/flows/executor/func_step_test.go`, replace all `function="assign"` with `builtin="assign"`:

```bash
sed -i 's/function="assign"/builtin="assign"/g' pkg/flows/executor/func_step_test.go
```

- [ ] **Step 6: Run all executor tests**

Run: `go test ./pkg/flows/executor -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/executor/executor.go pkg/flows/executor/func_step_test.go
git commit -m "feat(executor): use Builtin/Extension attributes instead of Function

Updated executeFuncStep to check step.Builtin and step.Extension
instead of step.Function."
```

---

## Chunk 3: XSD Schema Update

### Task 3: Update flow.xsd schema

**Files:**
- Modify: `pkg/flows/linter/schema/flow.xsd:165`

- [ ] **Step 1: Update XSD schema**

In `pkg/flows/linter/schema/flow.xsd`, around line 165, replace:

```xml
<xs:attribute name="function" type="xs:string"/>
```

With:

```xml
<xs:attribute name="builtin" type="xs:string"/>
<xs:attribute name="extension" type="xs:string"/>
```

- [ ] **Step 2: Verify schema is valid XML**

Run: `xmllint --noout pkg/flows/linter/schema/flow.xsd`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/flows/linter/schema/flow.xsd
git commit -m "feat(schema): replace function attribute with builtin/extension"
```

---

## Chunk 4: Linter Validation

### Task 4: Add validation for mutually exclusive attributes

**Files:**
- Create: `pkg/flows/linter/func_checker.go`

- [ ] **Step 1: Write test for validation**

Create `pkg/flows/linter/func_checker_test.go`:

```go
package linter

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestFuncChecker_MutuallyExclusive(t *testing.T) {
	tests := []struct {
		name        string
		step        flows.Step
		expectError bool
	}{
		{
			name: "builtin only - valid",
			step: flows.Step{
				Type:    "func",
				Builtin: "assign",
			},
			expectError: false,
		},
		{
			name: "extension only - valid",
			step: flows.Step{
				Type:      "func",
				Extension: "GetPRDiff",
			},
			expectError: false,
		},
		{
			name: "both builtin and extension - invalid",
			step: flows.Step{
				Type:      "func",
				Builtin:   "assign",
				Extension: "GetPRDiff",
			},
			expectError: true,
		},
		{
			name:        "neither builtin nor extension - invalid",
			step:        flows.Step{Type: "func"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linter := NewTestLinter()
			linter.AddStep(&tt.step)

			err := linter.CheckSteps()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/linter -run TestFuncChecker_MutuallyExclusive -v`
Expected: FAIL (validation not implemented yet)

- [ ] **Step 3: Implement func checker**

Create `pkg/flows/linter/func_checker.go`:

```go
package linter

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
)

// checkFuncSteps validates function steps
func (l *Linter) checkFuncSteps() {
	for _, state := range l.flow.States {
		for i := range state.Steps {
			step := &state.Steps[i]
			if step.Type != "func" {
				continue
			}

			// Check that exactly one of builtin or extension is set
			hasBuiltin := step.Builtin != ""
			hasExtension := step.Extension != ""

			if hasBuiltin && hasExtension {
				l.addError(l.errorWithStep(
					fmt.Sprintf("step cannot have both 'builtin' and 'extension' attributes"),
					"step",
					step.Name,
				))
				continue
			}

			if !hasBuiltin && !hasExtension {
				l.addError(l.errorWithStep(
					fmt.Sprintf("step must have either 'builtin' or 'extension' attribute"),
					"step",
					step.Name,
				))
				continue
			}

			// Validate built-in function name
			if hasBuiltin {
				if !isValidBuiltin(step.Builtin) {
					l.addError(l.errorWithStep(
						fmt.Sprintf("unknown built-in function: %s", step.Builtin),
						"step",
						step.Name,
					))
				}
			}
		}
	}
}

// isValidBuiltin checks if the function name is a valid built-in
func isValidBuiltin(name string) bool {
	validBuiltins := map[string]bool{
		"assign": true,
	}
	return validBuiltins[name]
}
```

- [ ] **Step 4: Update CheckSteps to call func checker**

In `pkg/flows/linter/linter.go`, add to the CheckSteps method:

```go
func (l *Linter) CheckSteps() error {
	// ... existing checks ...
	l.checkFuncSteps()
	// ... rest of checks ...
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./pkg/flows/linter -run TestFuncChecker_MutuallyExclusive -v`
Expected: PASS

- [ ] **Step 6: Run all linter tests**

Run: `go test ./pkg/flows/linter -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/linter/func_checker.go pkg/flows/linter/func_checker_test.go
git commit -m "feat(linter): add validation for builtin/extension attributes

- Validates that exactly one of builtin or extension is set
- Validates built-in function names against known list
- Reports clear error messages for invalid configurations"
```

---

## Chunk 5: Update Example Flows

### Task 5: Transform example flows to use new syntax

**Files:**
- Modify: `.gollum/flows/**/*.xml`

- [ ] **Step 1: Transform review-phase.xml**

In `.gollum/flows/forgejo-workflow/review-phase.xml`, line 52, replace:

```xml
<step type="func" function="GetPRDiff">
```

With:

```xml
<step type="func" extension="GetPRDiff">
```

- [ ] **Step 2: Verify flow still parses**

Run: `go test ./pkg/flows/parser -run TestParseSimpleFlow -v`
Expected: PASS

- [ ] **Step 3: Update all other flows**

Find all flows using old syntax and update:

```bash
# Find all flows with function attribute
grep -r 'function="' .gollum/flows/ | grep -v ".git"

# Manually update each one to use builtin="" or extension=""
```

- [ ] **Step 4: Test all example flows**

Run: `go test ./pkg/flows/parser -run TestParseAndLintAllFlows -v`
Expected: All flows parse and lint successfully

- [ ] **Step 5: Commit**

```bash
git add .gollum/flows/
git commit -m "feat(flows): migrate to builtin/extension syntax

Updated all example flows to use the new syntax:
- builtin=\"assign\" for built-in assign function
- extension=\"GetPRDiff\" for extension functions"
```

---

## Chunk 6: Documentation and Migration Guide

### Task 6: Create migration documentation

**Files:**
- Create: `docs/migration-guide-function-syntax.md`

- [ ] **Step 1: Write migration guide**

Create `docs/migration-guide-function-syntax.md`:

```markdown
# Function Syntax Migration Guide

## Breaking Change

The `function` attribute has been replaced with two mutually exclusive attributes:
- `builtin` - for built-in functions (hardcoded in executor)
- `extension` - for extension functions (loaded via Yaegi)

## Old Syntax

```xml
<step type="func" function="assign">
<step type="func" function="GetPRDiff">
```

## New Syntax

```xml
<step type="func" builtin="assign">
<step type="func" extension="GetPRDiff">
```

## Built-in Functions

Currently available built-in functions:
- `assign` - Assign a value from one field to another

## Extension Functions

Extension functions must be declared in the `<extensions>` block:

```xml
<extensions>
    <function name="GetPRDiff" package="github.com/denkhaus/gollum/flows/forgejo">
        <description>Get unified diff for PR</description>
    </function>
</extensions>
```

## Migration Steps

1. Replace `function="assign"` with `builtin="assign"`
2. Replace all other `function="..."` with `extension="..."`
3. Ensure extension functions are declared in `<extensions>` block
4. Run linter to validate: `gollum lint <flow-file>`
```

- [ ] **Step 2: Commit**

```bash
git add docs/migration-guide-function-syntax.md
git commit -m "docs: add migration guide for function syntax transformation"
```

---

## Completion Checklist

- [ ] All tests pass: `go test ./pkg/flows/... -v`
- [ ] All example flows parse and lint successfully
- [ ] XSD schema validates correctly
- [ ] Migration guide is complete
- [ ] No breaking changes missed (all references to `Function` updated)

---

## Testing Strategy

### Unit Tests
- `pkg/flows/types_step_test.go` - Test new struct fields
- `pkg/flows/executor/func_step_test.go` - Test builtin/extension execution
- `pkg/flows/linter/func_checker_test.go` - Test validation rules

### Integration Tests
- `pkg/flows/parser/all_flows_test.go` - Verify all example flows parse
- `pkg/flows/linter/all_flows_lint_test.go` - Verify all flows pass linter

### Manual Testing
1. Create a test flow with builtin function
2. Create a test flow with extension function
3. Create a test flow with both (should error)
4. Create a test flow with neither (should error)
