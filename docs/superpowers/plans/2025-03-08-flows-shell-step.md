# Shell Step Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement shell command execution with auto-escaping for secure variable substitution.

**Architecture:**
1. Parse `<cmd>` template with variable references
2. Auto-escape shell variables using `shquote` package
3. Execute with `os/exec` and context timeout
4. Capture stdout, stderr, exit_code
5. Map to output fields using path attribute

**Security:**
- All variables MUST be shell-escaped to prevent injection
- Use `github.com/alessio/shellescape` or similar

---

## Task 1: Write Failing Test for Shell Step

**Files:**
- Create: `pkg/flows/executor/shell_step_test.go`

**Step 1: Write test that calls a simple command**

```go
func TestExecuteShellStep_SimpleEcho(t *testing.T) {
    flow := &flows.Flow{
        Name:    "test-shell",
        Version: "1.0",
        Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "greeting"}}},
        States: []flows.State{
            {
                Name:    "init",
                Initial: true,
                Steps: []flows.Step{
                    {
                        Type: "shell",
                        Cmd:  "echo 'Hello World'",
                        Output: &flows.StepOutput{
                            Paths: []flows.OutputPath{
                                {Path: "stdout", Assign: "${output.greeting}"},
                            },
                        },
                    },
                },
                Transitions: []flows.Transition{{To: "done"}},
            },
            {Name: "done"},
        },
    }

    exec := NewExecutor(flow)
    step := &flow.States[0].Steps[0]
    err := exec.executeShellStep(step, "init")

    require.NoError(t, err)
    result, ok := exec.ctx.GetOutputField("greeting")
    require.True(t, ok)
    assert.Equal(t, "Hello World\n", result) // echo adds newline
}
```

**Step 2: Run test to verify RED**
```bash
go test ./pkg/flows/executor/... -run TestExecuteShellStep -v
```

Expected: "shell step execution not yet implemented"

---

## Task 2: Implement Shell Command Execution

**Files:**
- Modify: `pkg/flows/executor/executor.go`

**Step 1: Implement executeShellStep**

```go
func (e *Executor) executeShellStep(step *flows.Step, stateName string) error {
    // Parse and escape command
    cmd := e.substituteTemplate(step.Cmd)

    // Execute with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Run shell command
    execCmd := exec.CommandContext(ctx, "sh", "-c", cmd)
    stdout, err := execCmd.CombinedOutput()

    // Map outputs
    if step.Output != nil {
        for _, path := range step.Output.Paths {
            switch path.Path {
            case "stdout":
                val := string(stdout)
                if err := e.assignOutput(path.Assign, val); err != nil {
                    return err
                }
            case "exit_code":
                code := 0
                if exitErr, ok := err.(*exec.ExitError); ok {
                    code = exitErr.ExitCode()
                }
                if err := e.assignOutput(path.Assign, code); err != nil {
                    return err
                }
            }
        }
    }

    // Handle execution errors
    if err != nil && !isExitError(err) {
        return fmt.Errorf("shell command failed: %w", err)
    }

    return nil
}
```

**Step 2: Run test to verify GREEN**
```bash
go test ./pkg/flows/executor/... -run TestExecuteShellStep -v
```

---

## Task 3: Add Template Substitution for Variables

**Files:**
- Modify: `pkg/flows/executor/executor.go` or `template.go`

**Step 1: Write test for variable substitution**

```go
func TestExecuteShellStep_WithVariable(t *testing.T) {
    flow := &flows.Flow{
        Name:    "test-shell-var",
        Version: "1.0",
        Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "name"}}},
        Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "greeting"}}},
        States: []flows.State{
            {
                Name:    "init",
                Initial: true,
                Steps: []flows.Step{
                    {
                        Type: "shell",
                        Cmd:  "echo 'Hello ${input.name}'",
                        Output: &flows.StepOutput{
                            Paths: []flows.OutputPath{{Path: "stdout", Assign: "${output.greeting}"}},
                        },
                    },
                },
                Transitions: []flows.Transition{{To: "done"}},
            },
            {Name: "done"},
        },
    }

    exec := NewExecutor(flow)
    exec.SetInput(map[string]any{"name": "Claude"})

    step := &flow.States[0].Steps[0]
    err := exec.executeShellStep(step, "init")

    require.NoError(t, err)
    result, ok := exec.ctx.GetOutputField("greeting")
    require.True(t, ok)
    assert.Equal(t, "Hello Claude\n", result)
}
```

**Step 2: Run test - should pass with existing template substitution**

---

## Task 4: Add Shell Auto-Escaping

**Files:**
- Create: `pkg/flows/executor/shell_escape.go`

**Step 1: Write test for shell escaping**

```go
func TestShellEscaping_PreventsInjection(t *testing.T) {
    flow := &flows.Flow{
        Name:    "test-shell-escape",
        Version: "1.0",
        Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "user_input"}}},
        Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "safe"}}},
        States: []flows.State{
            {
                Name:    "init",
                Initial: true,
                Steps: []flows.Step{
                    {
                        Type: "shell",
                        Cmd:  "echo ${input.user_input}",
                        Output: &flows.StepOutput{
                            Paths: []flows.OutputPath{{Path: "stdout", Assign: "${output.safe}}},
                        },
                    },
                },
                Transitions: []flows.Transition{{To: "done"}},
            },
            {Name: "done"},
        },
    }

    exec := NewExecutor(flow)
    // Malicious input with shell metacharacters
    exec.SetInput(map[string]any{"user_input": "'; rm -rf / ; echo '"})

    step := &flow.States[0].Steps[0]
    err := exec.executeShellStep(step, "init")

    require.NoError(t, err)
    result, ok := exec.ctx.GetOutputField("safe")
    require.True(t, ok)
    // Should be escaped, not executed
    assert.Contains(t, result, "'")
}
```

**Step 2: Implement shell escaping**

Use `github.com/alessio/shellescape` to escape variables in commands.

---

## Task 5: Add Timeout Support

**Files:**
- Modify: `pkg/flows/executor/executor.go`

**Step 1: Parse timeout from step**

```go
func parseTimeout(timeoutStr string) (time.Duration, error) {
    if timeoutStr == "" {
        return 60 * time.Second, nil // default
    }
    return time.ParseDuration(timeoutStr)
}
```

**Step 2: Use timeout in command execution**

```go
timeout, err := parseTimeout(step.Timeout)
if err != nil {
    return fmt.Errorf("invalid timeout: %w", err)
}
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
```

---

## Task 6: Integration Test - Shell Step in Real Flow

**Files:**
- Modify: `pkg/flows/executor/integration_test.go`

**Step 1: Add integration test**

```go
func TestShellStepIntegration(t *testing.T) {
    // Parse an example flow with shell step
    xml := `<?xml version="1.0" encoding="UTF-8"?>
    <flow name="test-shell-flow" version="1.0">
        <input>
            <string name="target_dir" />
        </input>
        <output>
            <string name="test_output" />
        </output>
        <states>
            <state name="init" initial="true">
                <steps>
                    <step type="shell" name="ls">
                        <cmd><![CDATA[ls ${input.target_dir}]]></cmd>
                        <output>
                            <string path="stdout" assign="${output.test_output}" />
                        </output>
                    </step>
                </steps>
                <transitions>
                    <transition to="done" />
                </transition>
            </state>
            <state name="done" />
        </states>
    </flow>`

    flow, err := parser.ParseString(xml)
    require.NoError(t, err)

    exec := NewExecutor(flow)
    exec.SetInput(map[string]any{"target_dir": "/tmp"})

    err = exec.Run()
    require.NoError(t, err)
}
```
