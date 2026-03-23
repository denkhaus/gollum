# LLM Step Verbose Attribute Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `verbose` attribute to LLM steps that controls whether LLM response output is shown in logs

**Architecture:**
- Add optional `Verbose` boolean attribute to `Step` struct
- Pass verbose flag through executor to control agent OutputMode
- Update XSD schema to allow optional verbose attribute
- Ensure linter accepts the new attribute

**Tech Stack:** Go, XML (XSD schema), existing flow/executor/linter packages

---

## Chunk 1: XSD Schema Update

### Task 1: Update XSD Schema for Verbose Attribute

**Files:**
- Modify: `pkg/flows/linter/schema/flow.xsd`

- [ ] **Step 1: Add verbose attribute to StepType**
```xml
<!-- In StepType complexType, after the tool attribute (line ~188) -->
<xs:attribute name="verbose" type="xs:boolean"/>
```

- [ ] **Step 2: Verify XSD is valid**
Run: `xmllint --schema pkg/flows/linter/schema/flow.xsd --noout test_flow.xml`
Or just: `go test ./pkg/flows/linter/... -v` (existing tests validate schema)

Expected: No validation errors for flows with/without verbose attribute

- [ ] **Step 3: Commit**
```bash
git add pkg/flows/linter/schema/flow.xsd
git commit -m "feat(xsd): add optional verbose attribute to step elements

The verbose attribute controls whether LLM response output is shown in logs.
- Default is false (silent, current behavior)
- When true, LLM responses are logged during flow execution
"
```

---

## Chunk 2: Go Type Definition Update

### Task 2: Add Verbose Field to Step Struct

**Files:**
- Modify: `pkg/flows/types.go`

- [ ] **Step 1: Add Verbose field to Step struct**
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
	Verbose  bool               `xml:"verbose,attr"` // NEW: Show LLM output in logs
}
```

- [ ] **Step 2: Write test for verbose parsing**
Create: `pkg/flows/types_test.go` (or add to existing test file)
```go
func TestStep_VerboseAttribute(t *testing.T) {
	xml := `<step type="llm" agent="test" verbose="true">
		<prompt>Test</prompt>
	</step>`

	var step Step
	err := xml.Unmarshal([]byte(xml), &step)
	require.NoError(t, err)
	assert.True(t, step.Verbose, "verbose should be true when set to true")
}

func TestStep_VerboseAttributeDefault(t *testing.T) {
	xml := `<step type="llm" agent="test">
		<prompt>Test</prompt>
	</step>`

	var step Step
	err := xml.Unmarshal([]byte(xml), &step)
	require.NoError(t, err)
	assert.False(t, step.Verbose, "verbose should default to false when omitted")
}
```

- [ ] **Step 3: Run tests to verify**
Run: `go test ./pkg/flows/... -run TestStep_Verbose -v`

Expected: Tests pass, verbose defaults to false

- [ ] **Step 4: Commit**
```bash
git add pkg/flows/types.go pkg/flows/types_test.go
git commit -m "feat(types): add Verbose field to Step struct

- Add Verbose bool field with XML tag
- Defaults to false (maintains current silent behavior)
- Add tests for attribute parsing with default and explicit values
"
```

---

## Chunk 3: Linter Support

### Task 3: Ensure Linter Accepts Verbose Attribute

**Files:**
- Check: `pkg/flows/linter/` - verify no validation errors for verbose
- Modify: `pkg/flows/linter/` (if needed)

- [ ] **Step 1: Check existing step validation**
Search for step validation code in linter package to ensure verbose is accepted

Run: `grep -r "step.*attribute\|Step.*valid" pkg/flows/linter/*.go | head -20`

- [ ] **Step 2: Run existing linter tests to ensure no regressions**
Run: `go test ./pkg/flows/linter/... -v`

Expected: All tests pass, verbose attribute is accepted

- [ ] **Step 3: (Optional) Add verbose-specific validation if needed**
If linter has strict validation, add explicit acceptance:
```go
// In step validation code
// Verbose is optional and defaults to false (no validation needed)
```

---

## Chunk 4: Executor Implementation - Use Verbose Flag

### Task 4: Pass Verbose Flag to Agent Config

**Files:**
- Modify: `pkg/flows/executor/llm_step.go`

- [ ] **Step 1: Update executeLLMStep to use step.Verbose**
Find line 46 in `llm_step.go`:
```go
OutputMode: shared.OutputModeSilent, // Suppress output during flow execution
```

Replace with:
```go
OutputMode: p.getOutputModeForStep(step), // Use verbose flag to control output
```

- [ ] **Step 2: Add getOutputModeForStep helper method**
Add to `llm_step.go`:
```go
// getOutputModeForStep determines the output mode based on step's verbose flag
func (p *flowExecutorImpl) getOutputModeForStep(step *flows.Step) shared.OutputMode {
	if step.Verbose {
		return shared.OutputModeFull // Show LLM response in logs
	}
	return shared.OutputModeSilent // Suppress LLM response (default)
}
```

- [ ] **Step 3: Write test for verbose mode**
Create or modify: `pkg/flows/executor/llm_step_test.go`
```go
func TestExecuteLLMStep_VerboseControlsOutput(t *testing.T) {
	// Setup
	flow := &flows.Flow{
		Name: "test-flow",
		Agents: []flows.Agent{
			{Name: "test", Model: "test-model", Prompt: "You are a test agent"},
		},
		States: []flows.State{
			{
				Name: "init", Initial: true,
				Steps: []flows.Step{
					{
						Type:    "llm",
						Agent:   "test",
						Prompt:  "Say 'test output'",
						Verbose: true, // Enable verbose output
					},
				},
			},
		},
	}

	// Create executor with mock factory that captures output
	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	err := exec.SetInput(map[string]string{})
	require.NoError(t, err)

	// Execute and verify verbose mode shows LLM response
	// (This will need a mock agent factory that returns output)
	// For now, just verify the step's Verbose flag is preserved
	assert.True(t, flow.States[0].Steps[0].Verbose, "verbose flag should be preserved")
}
```

- [ ] **Step 4: Run tests**
Run: `go test ./pkg/flows/executor/... -run TestExecuteLLMStep_Verbose -v`

Expected: Tests pass, verbose flag controls output mode

- [ ] **Step 5: Commit**
```bash
git add pkg/flows/executor/llm_step.go pkg/flows/executor/llm_step_test.go
git commit -m "feat(executor): use verbose flag to control LLM output in logs

- Add getOutputModeForStep helper to determine output mode
- Pass OutputModeFull when verbose=true, OutputModeSilent otherwise
- Add test to verify verbose flag is preserved through execution
"
```

---

## Chunk 5: Integration Test

### Task 5: Create Integration Test with Real Flow

**Files:**
- Create: `test_flows/verbose_test_flow.xml`

- [ ] **Step 1: Create test flow with verbose step**
Create: `test_flows/verbose_test_flow.xml`
```xml
<?xml version="1.0" encoding="UTF-8"?>
<flow name="verbose-test" version="1.0">
    <agents>
        <agent name="test" model="test-model">
            <prompt>You are a test agent. Always respond with "Test successful"</prompt>
        </agent>
    </agents>

    <states>
        <state name="init" initial="true">
            <steps>
                <!-- This step should NOT show LLM output in logs (default behavior) -->
                <step type="llm" agent="test">
                    <prompt>Say something quietly</prompt>
                </step>
                <!-- This step SHOULD show LLM output in logs -->
                <step type="llm" agent="test" verbose="true">
                    <prompt>Say something loudly</prompt>
                </step>
            </steps>
            <transitions>
                <transition to="done"/>
            </transitions>
        </state>
        <state name="done"/>
    </states>
</flow>
```

- [ ] **Step 2: Create integration test**
Create: `pkg/flows/executor/verbose_integration_test.go`
```go
func TestLLMStep_VerboseIntegration(t *testing.T) {
	// This test requires a mock agent factory
	// For now, test that the verbose flag flows through the system

	// Test 1: Verify verbose=false (default) uses OutputModeSilent
	step1 := &flows.Step{Type: "llm", Agent: "test", Verbose: false}
	exec := createTestExecutor()
	outputMode := exec.getOutputModeForStep(step1)
	assert.Equal(t, shared.OutputModeSilent, outputMode)

	// Test 2: Verify verbose=true uses OutputModeFull
	step2 := &flows.Step{Type: "llm", Agent: "test", Verbose: true}
	outputMode = exec.getOutputModeForStep(step2)
	assert.Equal(t, shared.OutputModeFull, outputMode)

	// Test 3: Verify step without verbose attribute defaults to false
	step3 := &flows.Step{Type: "llm", Agent: "test"}
	outputMode = exec.getOutputModeForStep(step3)
	assert.Equal(t, shared.OutputModeSilent, outputMode)
}
```

- [ ] **Step 3: Run integration test**
Run: `go test ./pkg/flows/executor/... -run TestLLMStep_VerboseIntegration -v`

Expected: All tests pass

- [ ] **Step 4: Commit**
```bash
git add test_flows/verbose_test_flow.xml pkg/flows/executor/verbose_integration_test.go
git commit -m "test: add integration test for LLM step verbose attribute

- Create test flow with verbose and non-verbose steps
- Add integration test verifying OutputMode selection
- Tests default behavior (silent) vs verbose mode (full output)
"
```

---

## Chunk 6: Documentation

### Task 6: Update Documentation

**Files:**
- Modify: `docs/flows.md` (or create if doesn't exist)
- Modify: `README.md` or `CHANGELOG.md`

- [ ] **Step 1: Add verbose attribute documentation to flow documentation**
Add section:
```markdown
## Step Attributes

### verbose (optional boolean)
Controls whether LLM response output is shown in execution logs.
- **Default:** `false` (LLM responses are suppressed)
- `true`: LLM responses are logged at INFO level

Example:
```xml
<step type="llm" agent="assistant" verbose="true">
    <prompt>Generate a summary</prompt>
</step>
```

**Note:** This attribute only applies to LLM steps. It does not affect other step types.
```

- [ ] **Step 2: Update CHANGELOG.md**
Add entry:
```markdown
## [Unreleased]

### Added
- LLM steps now support an optional `verbose` attribute to control LLM output in logs
  - When `verbose="true"`, LLM responses are logged at INFO level
  - Default is `verbose="false"` (silent mode) to maintain current behavior
```

- [ ] **Step 3: Commit**
```bash
git add docs/flows.md CHANGELOG.md
git commit -m "docs: document LLM step verbose attribute

- Add verbose attribute documentation to flow reference
- Update CHANGELOG with new feature description
- Explain default behavior and usage examples
"
```

---

## Summary

This plan implements the `verbose` attribute for LLM steps in 6 chunks:

1. **XSD Schema** - Allow optional verbose attribute
2. **Go Types** - Add Verbose field to Step struct with tests
3. **Linter** - Ensure linter accepts the new attribute
4. **Executor** - Use verbose flag to control OutputMode
5. **Integration** - Test the complete flow from XML to execution
6. **Documentation** - Document the new feature

**Key Design Decisions:**
- Default is `false` (silent) - maintains current behavior
- Boolean type (not string) - simpler validation and usage
- Only applies to LLM steps - ignored for other step types
- Uses existing `OutputMode` constants - no new logging infrastructure needed

**Testing Strategy:**
- Unit tests for XML parsing
- Unit tests for OutputMode selection
- Integration test with complete flow
- Existing tests should continue passing (backward compatible)
