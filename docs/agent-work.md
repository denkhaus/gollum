# Agent Task System - Complete Guide

## Overview

The built-in **Task Tool** allows spawning specialized subagents for complex, multi-step tasks. This document summarizes the complete mechanics of task creation, execution, and result retrieval.

---

## Task Tool Parameters

### Required Parameters

| Parameter       | Type   | Description                              |
| --------------- | ------ | ---------------------------------------- |
| `subagent_type` | string | Type of specialized agent to use         |
| `description`   | string | Short task description (3-5 words)       |
| `prompt`        | string | Detailed task instructions for the agent |

### Optional Parameters

| Parameter           | Type    | Description                                                             |
| ------------------- | ------- | ----------------------------------------------------------------------- |
| `model`             | string  | Model to use (`sonnet`, `opus`, `haiku`). Default: inherits from parent |
| `resume`            | string  | Agent ID to resume from (preserves full previous context)               |
| `run_in_background` | boolean | Run agent in background. Use TaskOutput to read results later           |

### Available Subagent Types

- `general-purpose` - Complex queries, multi-step tasks, code searches
- `Explore` - Fast codebase exploration (thoroughness: quick/medium/very thorough)
- `Plan` - Software architecture and implementation planning
- `claude-code-guide` - Claude Code / Claude Agent SDK documentation

---

## Return Mechanism

### 1. Synchronous Execution (Default)

When `run_in_background` is **not set** or `false`:

```xml
<Task subagent_type="Explore" description="Search API" prompt="Find all API endpoints">
</Task>
```

**Behavior:**
- Calling agent **waits** for subagent completion
- Result returned in `<result>` tag
- Subagent's final text response becomes the return value

**Flow:**
```
Caller → Task → Subagent works → Result → Caller
```

### 2. Asynchronous Execution (Background)

When `run_in_background="true"`:

```xml
<Task subagent_type="Explore" description="Long task" prompt="..." run_in_background="true">
</Task>
```

**Behavior:**
- Task starts immediately, returns `task_id`
- Subagent continues in background
- Caller must use **TaskOutput** to retrieve results later

**Flow:**
```
Caller → Task → task_id returned
Caller continues...
Caller → TaskOutput(task_id) → Result
```

---

## TaskOutput Tool

Retrieves output from a running or completed background task.

### Parameters

| Parameter | Type    | Default | Description                                             |
| --------- | ------- | ------- | ------------------------------------------------------- |
| `task_id` | string  | -       | ID from Task tool response                              |
| `block`   | boolean | true    | Wait for completion (true) or check status only (false) |
| `timeout` | number  | 30000   | Max wait time in ms (max 600000 = 10 min)               |

### Usage Examples

**Blocking wait (default):**
```xml
<TaskOutput task_id="agent-123" block="true" timeout="60000" />
```

**Non-blocking status check:**
```xml
<TaskOutput task_id="agent-123" block="false" />
```

---

## Communication Model

### Critical Limitation: One-Shot Only

**There is NO bidirectional communication during execution.**

```
Caller                Subagent
  |                       |
  |--- Task(prompt) ----->|
  |                       | (working)
  |                       |
  |<--- Result ------------|
  |                    (done)
```

### What is NOT possible

- ❌ Sending messages to a **running** subagent
- ❌ Bidirectional communication during execution
- ❌ Conversation between two agents
- ❌ Status updates (except via TaskOutput block=false)

### What IS possible

**1. Resume (after completion):**
```xml
<Task subagent_type="Explore" resume="agent-123" prompt="Continue with...">
</Task>
```
- Resumes **completed** agent with full previous context
- Not for communication during execution

**2. TaskOutput block=false (status check):**
```xml
<TaskOutput task_id="agent-123" block="false" />
```
- Read-only status check
- No communication, just inspection

---

## Background Tasks: Caller Responsibility

### No Automatic Reminders

**There is NO mechanism that reminds the caller about background tasks.**

When you start a background task:
```xml
<Task run_in_background="true" ... />
```

Returns:
```xml
<result>
  {"task_id": "agent-abc123", "status": "running"}
</result>
```

### Caller Must Remember

| Responsibility       | Mechanism                       |
| -------------------- | ------------------------------- |
| Remember task exists | ❌ No automatic reminder         |
| Store task_id        | 🔵 Caller must track it          |
| Retrieve result      | 🔵 Caller must invoke TaskOutput |
| Handle timeout       | 🔵 Caller must manage timing     |

### Consequences

**If caller forgets:**
```xml
<!-- Task started -->
<Task run_in_background="true" ... />

<!-- Caller continues... -->
<!-- ... */

<!-- Caller completely forgets -->
<!-- Result NEVER retrieved -->
```

### Best Practices

**1. Immediate retrieval pattern:**
```xml
<!-- Start task, note task_id -->
<Task run_in_background="true" description="..." prompt="...">
  <!-- Returns: task_id = "agent-123" -->
</Task>

<!-- ... do other work ... -->

<!-- Later: retrieve result -->
<TaskOutput task_id="agent-123" block="true" />
```

**2. Manual tracking required**
The agent must self-manage:
- Remember: "I started task agent-123"
- Plan: "I must call TaskOutput after X steps"
- Execute: Actually call TaskOutput at appropriate time

---

## Workarounds and Patterns

### Multi-Step Interaction

Since true conversation isn't possible, use sequential tasks:

```xml
<!-- 1. Initial task with context -->
<Task description="Analyze code" prompt="Analyze pkg/tools/*.go">
  <!-- Returns analysis result -->
</Task>

<!-- 2. Process result, then follow-up task -->
<Task description="Deepen analysis"
      prompt="Based on previous: ${result}. Now also examine tests">
  <!-- New task with accumulated context -->
</Task>
```

### Parallel Execution

Launch multiple tasks in a single response:

```xml
<!-- All tasks start in parallel -->
<Task subagent_type="Explore" description="Search A" prompt="Search feature A" />

<Task subagent_type="Explore" description="Search B" prompt="Search feature B" />

<Task subagent_type="Explore" description="Search C" prompt="Search feature C" />
```

### Resume for Continuation

```xml
<!-- First run -->
<Task subagent_type="Explore" description="Big task" prompt="...">
  <!-- Returns task_id = "agent-456" -->
</Task>

<!-- Later resume with full context -->
<Task subagent_type="Explore" resume="agent-456" prompt="Continue with...">
</Task>
```

---

## Summary Table

| Aspect                   | Mechanism                                         |
| ------------------------ | ------------------------------------------------- |
| **Execution modes**      | Synchronous (default) / Asynchronous (background) |
| **Return value**         | Plain text from subagent's final response         |
| **Communication**        | One-shot only (prompt → result)                   |
| **Bidirectional chat**   | ❌ Not possible                                    |
| **Background reminders** | ❌ None, caller must remember                      |
| **task_id tracking**     | 🔵 Full caller responsibility                      |
| **Result retrieval**     | 🔵 Caller must invoke TaskOutput                   |
| **Resume**               | ✅ Possible after completion                       |
| **Parallel tasks**       | ✅ Launch multiple in single response              |

---

## Key Takeaways

1. **One-Shot Model**: Each task is a single prompt-response exchange
2. **No Chat**: Agents cannot "talk back and forth" during execution
3. **Caller Remembers**: Background tasks require manual tracking by caller
4. **Manual Retrieval**: Results must be explicitly fetched via TaskOutput
5. **Resume Available**: Can continue completed agents with preserved context
6. **Parallel Possible**: Multiple tasks can be launched simultaneously

---

## Example: Complete Workflow

```xml
<!-- 1. Start multiple background tasks in parallel -->
<Task subagent_type="Explore" description="Find APIs" prompt="List all API endpoints" run_in_background="true" />
<!-- Returns: task_id = "agent-001" -->

<Task subagent_type="Explore" description="Find models" prompt="List all data models" run_in_background="true" />
<!-- Returns: task_id = "agent-002" -->

<!-- 2. Caller continues with other work... -->

<!-- 3. Caller explicitly retrieves results -->
<TaskOutput task_id="agent-001" block="true" timeout="60000" />
<!-- Returns: API endpoints list -->

<TaskOutput task_id="agent-002" block="true" timeout="60000" />
<!-- Returns: Data models list -->

<!-- 4. Process results and potentially launch follow-up task -->
<Task subagent_type="Plan" description="Design integration" prompt="Design how to connect APIs and models found earlier">
</Task>
```

---

## Conclusion

The Task system is designed for **fire-and-forget** operations with manual result collection. It excels at:
- Parallel execution of independent tasks
- Heavy computation without blocking
- Codebase exploration and analysis

But requires careful planning from the calling agent for:
- Background task tracking
- Result retrieval timing
- Multi-step workflows (via sequential tasks)
