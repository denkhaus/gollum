# Step Result Notation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename step-level `<output>` to `<result>` to distinguish it from flow-root output interface.

**Architecture:** Rename `StepOutput` type to `StepResult`, change XML tag from `output` to `result`, and rename `assign` attribute to `assignTo`.

**Tech Stack:** Go, XML encoding, existing flow executor

---

## Chunk 1: Type Definition Changes

### Task 1: Rename StepOutput to StepResult in types.go

**Files:**
- Modify: `pkg/flows/types.go:488-499`

- [ ] **Step 1: Rename StepOutput struct and update XML tag**

Find:
```go
// StepOutput defines step output mapping
type StepOutput struct {
	Assign string       `xml:"assign,attr"`
	Paths  []OutputPath `xml:",any"`
}
```

Replace with:
```go
// StepResult defines step result mapping
type StepResult struct {
	AssignTo string       `xml:"assignTo,attr"`
	Paths    []ResultPath `xml:",any"`
}
```

- [ ] **Step 2: Rename OutputPath to ResultPath**

Find:
```go
// OutputPath maps a JSONPath to a field
type OutputPath struct {
	XMLName xml.Name
	Path    string `xml:"path,attr"`
	Assign  string `xml:"assign,attr"`
}
```

Replace with:
```go
// ResultPath maps a JSONPath to a field
type ResultPath struct {
	XMLName  xml.Name
	Path     string `xml:"path,attr"`
	AssignTo string `xml:"assignTo,attr"`
}
```

- [ ] **Step 3: Update Step.Output to Step.Result**

Find in Step struct (line ~473):
```go
	Output   *StepOutput        `xml:"output"`
```

Replace with:
```go
	Result   *StepResult        `xml:"result"`
```

- [ ] **Step 4: Commit**

```bash
git add pkg/flows/types.go
git commit -m "refactor(types): rename StepOutput to StepResult

- StepOutput → StepResult
- OutputPath → ResultPath
- XML tag: <output> → <result>
- assign attribute → assignTo
"
```

---

## Chunk 2: XSD Schema Updates

### Task 2: Update XSD schema for result notation

**Files:**
- Modify: `pkg/flows/linter/schema/flow.xsd`

- [ ] **Step 1: Update OutputMappingType to ResultMappingType**

Find (line 223):
```xml
<xs:complexType name="OutputMappingType">
    <xs:sequence minOccurs="0">
      <xs:any processContents="skip" maxOccurs="unbounded"/>
    </xs:sequence>
    <xs:attribute name="assign" type="xs:string"/>
</xs:complexType>
```

Replace with:
```xml
<xs:complexType name="ResultMappingType">
    <xs:sequence minOccurs="0">
      <xs:any processContents="skip" maxOccurs="unbounded"/>
    </xs:sequence>
    <xs:attribute name="assignTo" type="xs:string"/>
</xs:complexType>
```

- [ ] **Step 2: Update step element to use result instead of output**

Find in StepType (line 182):
```xml
<xs:element name="output" type="OutputMappingType" minOccurs="0"/>
```

Replace with:
```xml
<xs:element name="result" type="ResultMappingType" minOccurs="0"/>
```

- [ ] **Step 3: Verify XSD is valid**

```bash
xmllint --schema pkg/flows/linter/schema/flow.xsd --noout .gollum/flows/default/main.xml
```

Expected: No validation errors for migrated flows

- [ ] **Step 4: Commit**

```bash
git add pkg/flows/linter/schema/flow.xsd
git commit -m "refactor(xsd): update step output to result notation

- OutputMappingType → ResultMappingType
- <output> element → <result> element
- assign attribute → assignTo
"
```

---

## Chunk 3: Executor Code Updates

### Task 3: Update executor.go for StepResult

**Files:**
- Modify: `pkg/flows/executor/executor.go`

- [ ] **Step 1: Find all step.Output references**

Search for `step.Output` in executor.go - there should be ~5 occurrences

- [ ] **Step 2: Replace step.Output with step.Result**

Find pattern:
```go
if step.Output != nil {
    if step.Output.Assign != "" {
        fieldName := extractFieldName(step.Output.Assign)
```

Replace with:
```go
if step.Result != nil {
    if step.Result.AssignTo != "" {
        fieldName := extractFieldName(step.Result.AssignTo)
```

Do this for all occurrences in:
- Shell step result handling
- MCP tool result handling
- Path-based output handling

- [ ] **Step 3: Update OutputPath to ResultPath in loops**

Find:
```go
for _, path := range step.Output.Paths {
    fieldName := extractFieldName(path.Assign)
```

Replace with:
```go
for _, path := range step.Result.Paths {
    fieldName := extractFieldName(path.AssignTo)
```

- [ ] **Step 4: Run tests to verify compilation**

```bash
go test ./pkg/flows/executor/... -v
```

Expected: Compilation errors in tests (fixed in next chunk)

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/executor/executor.go
git commit -m "refactor(executor): update step.Output to step.Result"
```

---

## Chunk 4: Linter Updates

### Task 4: Update linter for StepResult

**Files:**
- Modify: `pkg/flows/linter/output_bindings.go`
- Modify: `pkg/flows/linter/position.go`
- Modify: `pkg/flows/linter/output_bindings_test.go`

- [ ] **Step 1: Update output_bindings.go**

Search and replace `StepOutput` with `StepResult`
Search and replace `OutputPath` with `ResultPath`

- [ ] **Step 2: Update position.go**

Search and replace `StepOutput` with `StepResult`

- [ ] **Step 3: Update output_bindings_test.go**

Search and replace:
- `StepOutput{` → `StepResult{`
- `OutputPath{` → `ResultPath{`
- `Assign:` → `AssignTo:`

- [ ] **Step 4: Run linter tests**

```bash
go test ./pkg/flows/linter/... -v
```

Expected: Should pass after replacements

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/linter/
git commit -m "refactor(linter): update StepOutput to StepResult"
```

---

## Chunk 5: Test Fixture Updates

### Task 5: Update all test files with StepResult

**Files:**
- Modify: `pkg/flows/executor/shell_step_test.go`
- Modify: `pkg/flows/executor/mcp_step_test.go`
- Modify: `pkg/flows/executor/func_step_test.go`
- Modify: `pkg/flows/executor/func_step_integration_test.go`
- Modify: `pkg/flows/executor/llm_step_test.go`
- Modify: `pkg/flows/executor/call_step_test.go`

- [ ] **Step 1: Global replace in test files**

```bash
# In each test file, replace:
sed -i 's/StepOutput{/StepResult{/g' pkg/flows/executor/*_test.go
sed -i 's/OutputPath{/ResultPath{/g' pkg/flows/executor/*_test.go
sed -i 's/Assign:/AssignTo:/g' pkg/flows/executor/*_test.go
```

Or do manually with editor find-replace

- [ ] **Step 2: Run all executor tests**

```bash
go test ./pkg/flows/executor/... -v
```

Expected: All tests should pass

- [ ] **Step 3: Run types tests**

```bash
go test ./pkg/flows/... -v
```

Expected: All tests should pass

- [ ] **Step 4: Commit**

```bash
git add pkg/flows/
git commit -m "refactor(tests): update StepOutput to StepResult in fixtures"
```

---

## Chunk 6: XML Flow Migration

### Task 6: Migrate all flow XML files

**Files:**
- Modify: All `.xml` files in `.gollum/flows/`

- [ ] **Step 1: List all affected XML files**

```bash
rg '<output>' .gollum/flows/ -l
```

This will show files with step-level `<output>` blocks

- [ ] **Step 2: Create migration script**

Create: `scripts/migrate-step-output.sh`

```bash
#!/bin/bash
# Migrate step-level <output> to <result>

for file in $(find .gollum/flows/ -name "*.xml"); do
    # Only modify step-level output (inside <step> tags)
    # Preserve flow-level <output> blocks

    # Use perl for multiline replacement
    perl -i -0pe 's/<output>\s*<string path="([^"]+)" assign="([^"]+)"\/>\s*<\/output>/<result>\n        <string path="$1" assignTo="$2"\/>\n    <\/result>/g' "$file"

    perl -i -0pe 's/<output>\s*<int path="([^"]+)" assign="([^"]+)"\/>\s*<\/output>/<result>\n        <int path="$1" assignTo="$2"\/>\n    <\/result>/g' "$file"

    perl -i -0pe 's/<output>\s*<string path="([^"]+)" assign="\$\{output\.([^}]+)\}"\/>\s*<\/output>/<result>\n        <string path="$1" assignTo="output.$2"\/>\n    <\/result>/g' "$file"
done
```

- [ ] **Step 3: Run migration**

```bash
chmod +x scripts/migrate-step-output.sh
./scripts/migrate-step-output.sh
```

- [ ] **Step 4: Verify migration**

```bash
git diff .gollum/flows/
```

Check that:
- Step-level `<output>` → `<result>`
- `assign=` → `assignTo=`
- Flow-level `<output>` is preserved

- [ ] **Step 5: Test flows still parse**

```bash
go run ./cmd/gollum/ flow validate .gollum/flows/default/main.xml
```

Expected: Flows should parse successfully

- [ ] **Step 6: Commit**

```bash
git add .gollum/flows/ scripts/migrate-step-output.sh
git commit -m "refactor(flows): migrate step-level <output> to <result>

- Rename <output> to <result> in all steps
- Change assign to assignTo attribute
- Preserve flow-level <output> blocks
"
```

---

## Chunk 7: Documentation Updates

### Task 7: Update documentation

**Files:**
- Modify: `.gollum/flows/idea_plan.md`
- Create: `docs/flows/step-results.md`

- [ ] **Step 1: Update idea_plan.md**

Search for `<output>` examples in steps and update to `<result>` with `assignTo`

- [ ] **Step 2: Create step-results documentation**

Create: `docs/flows/step-results.md`

```markdown
# Step Results

Steps can capture their output using the `<result>` tag.

## Syntax

```xml
<step type="shell" name="run-test">
    <cmd><![CDATA[go test ./...]]></cmd>
    <result>
        <string path="stdout" assignTo="test_output"/>
        <int path="exit_code" assignTo="exit_code"/>
    </result>
</step>
```

## Attributes

- `path` - Source path in the result (e.g., `stdout`, `exit_code`, JSON paths)
- `assignTo` - Target field (supports `output.fieldName`, `context.fieldName`, or bare `fieldName`)

## Single Field Shortcut

For steps that return a single value:

```xml
<step type="llm" agent="worker">
    <prompt>Analyze this PR</prompt>
    <result assignTo="output.analysis"/>
</step>
```

## Flow Output vs Step Result

- **Flow Output (`<output>`)**: Defines the flow's public interface at flow root
- **Step Result (`<result>`)**: Captures individual step execution results
```

- [ ] **Step 3: Update other docs with examples**

```bash
rg '<output>' docs/ -l
```

Update any step-level output examples

- [ ] **Step 4: Commit**

```bash
git add docs/
git commit -m "docs: update step result notation documentation"
```

---

## Chunk 8: Final Integration Tests

### Task 8: Run full integration tests

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: All tests pass

- [ ] **Step 2: Build and test actual flows**

```bash
just build
./gollum flow run .gollum/flows/default/main.xml --input '{"test": "value"}'
```

Expected: Flows execute correctly with new notation

- [ ] **Step 3: Lint all flows**

```bash
./gollum flow lint .gollum/flows/
```

Expected: No errors

- [ ] **Step 4: Final commit if needed**

```bash
git add -A
git commit -m "test: ensure all integration tests pass with StepResult"
```

---

## Migration Summary

| Component | Change |
|-----------|--------|
| Type | `StepOutput` → `StepResult` |
| XML Tag | `<output>` → `<result>` |
| Attribute | `assign` → `assignTo` |
| Path Type | `OutputPath` → `ResultPath` |

## Backward Compatibility Note

This is a **breaking change**. Existing flows using step-level `<output>` will need to migrate to `<result>`. A migration script is provided in Chunk 5.
