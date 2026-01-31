# Trajectory Structure for Prompt Optimization

**Source:** Analysis of LangMEM usage patterns + Gollem integration
**Created:** 2025-01-31
**Updated:** 2025-01-31 (Gollem Type Integration)
**Purpose:** Define trajectory data structure for Go implementation

---

## Overview

A **Trajectory** represents a single conversation or interaction that the optimizer analyzes to identify prompt improvement opportunities.

**IMPORTANT:** Uses `gollem.Message` to avoid type duplication.

---

## Core Trajectory Definition

Based on LangMEM analysis and user requirements:

```go
// pkg/prompt/optimizer/types.go

package optimizer

import "github.com/m-mizutani/gollem"

// Trajectory represents a single conversation with optional feedback
type Trajectory struct {
    Messages []gollem.Message `json:"messages"` // ✅ Gollem Message (no duplication!)

    // Feedback contains optional evaluation data
    // Can be: nil, string, *Feedback, or *EditFeedback
    Feedback interface{} `json:"feedback,omitempty"`
}
```

---

## Gollem Message Structure

**Import from Gollem:**

```go
import "github.com/m-mizutani/gollem"

// Gollem Message (already defined - DO NOT re-define!)
type Message struct {
    Role     MessageRole      `json:"role"`
    Contents []MessageContent `json:"contents"`
    Name     string                 `json:"name,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Gollem MessageRole
const (
    RoleSystem    MessageRole = "system"
    RoleUser      MessageRole = "user"
    RoleAssistant MessageRole = "assistant"
    RoleTool      MessageRole = "tool"
)

// Gollem MessageContent Types
const (
    MessageContentTypeText         MessageContentType = "text"
    MessageContentTypeImage        MessageContentType = "image"
    MessageContentTypeToolCall     MessageContentType = "tool_call"
    MessageContentTypeToolResponse MessageContentType = "tool_response"
)
```

---

## Creating Messages with Gollem Helpers

---

## Feedback Types

### 1. Simple String Feedback

```go
import "github.com/m-mizutani/gollem"

// String feedback - free-form text
textContent1, _ := gollem.NewTextContent("Explain quantum computing")
textContent2, _ := gollem.NewTextContent("Quantum computing uses...")

trajectory := &Trajectory{
    Messages: []gollem.Message{
        {Role: gollem.RoleUser, Contents: []gollem.MessageContent{textContent1}},
        {Role: gollem.RoleAssistant, Contents: []gollem.MessageContent{textContent2}},
    },
    Feedback: "Response should include concrete examples",
}
```

### 2. Structured Feedback

```go
// Feedback provides structured evaluation
type Feedback struct {
    // Score is a numeric rating (0.0 - 1.0)
    Score float64 `json:"score,omitempty"`

    // Comment is free-form feedback text
    Comment string `json:"comment,omitempty"`

    // FailureModes identified issues
    FailureModes []string `json:"failure_modes,omitempty"`

    // Outcome indicates success/failure
    Outcome string `json:"outcome,omitempty"` // "success", "failure"
}

// Create messages with Gollem helpers
textContent, _ := gollem.NewTextContent("Tell me about Mars")

trajectory := &Trajectory{
    Messages: []gollem.Message{...},
    Feedback: &Feedback{
        Score:   0.7,
        Comment: "Needs more structure",
        FailureModes: []string{"style_mismatch", "incomplete"},
        Outcome: "failure",
    },
}
```

### 3. Edit-Based Feedback

```go
// EditFeedback provides revised response
type EditFeedback struct {
    // Revised is the corrected response
    Revised string `json:"revised"`

    // Edits are specific change annotations
    Edits []TextEdit `json:"edits,omitempty"`
}

type TextEdit struct {
    OldText string `json:"old_text"`
    NewText string `json:"new_text"`
    Reason  string `json:"reason,omitempty"`
}

trajectory := &Trajectory{
    Messages: []gollem.Message{...},
    Feedback: &EditFeedback{
        Revised: "Earth and Mars have many similarities and differences...",
        Edits: []TextEdit{
            {
                OldText: "Mars and Earth have many differences",
                NewText: "Earth and Mars have many similarities and differences",
                Reason: "Start with similarities before differences",
            },
        },
    },
}
```

### 4. No Feedback

```go
// Trajectory without feedback - implicit learning
userContent, _ := gollem.NewTextContent("...")
assistantContent, _ := gollem.NewTextContent("...")

trajectory := &Trajectory{
    Messages: []gollem.Message{
        {Role: gollem.RoleUser, Contents: []gollem.MessageContent{userContent}},
        {Role: gollem.RoleAssistant, Contents: []gollem.MessageContent{assistantContent}},
    },
    Feedback: nil, // Optimizer infers issues from conversation
}
```

---

## Tool Calls mit Gollem

```go
// Tool Call Content
toolContent, _ := gollem.NewToolCallContent(
    "call_123",                          // ID
    "search",                            // Name
    map[string]any{"query": "example"},  // Arguments
)

trajectory := &Trajectory{
    Messages: []gollem.Message{
        {Role: gollem.RoleUser, Contents: []gollem.MessageContent{userContent}},
        {Role: gollem.RoleAssistant, Contents: []gollem.MessageContent{toolContent}},
    },
}
```

---

## Trajectory Formatting (Gollem-Version)

```go
// formatSessions converts trajectories to LLM-readable format
func formatSessions(trajectories []*Trajectory) string {
    var sb strings.Builder

    for i, traj := range trajectories {
        sb.WriteString(fmt.Sprintf("## Session %d\n\n", i+1))

        for _, msg := range traj.Messages {
            role := strings.Title(string(msg.Role))

            for _, content := range msg.Contents {
                switch content.Type {
                case gollem.MessageContentTypeText:
                    if text, err := content.GetTextContent(); err == nil {
                        sb.WriteString(fmt.Sprintf("%s: %s\n\n", role, text.Text))
                    }

                case gollem.MessageContentTypeToolCall:
                    if tc, err := content.GetToolCallContent(); err == nil {
                        sb.WriteString(fmt.Sprintf("%s: Called tool '%s' with args %v\n\n",
                            role, tc.Name, tc.Arguments))
                    }

                case gollem.MessageContentTypeToolResponse:
                    if tr, err := content.GetToolResponseContent(); err == nil {
                        sb.WriteString(fmt.Sprintf("%s: Tool response: %v\n\n",
                            role, tr.Response))
                    }
                }
            }
        }

        if traj.Feedback != nil {
            sb.WriteString("### Feedback\n\n")
            switch fb := traj.Feedback.(type) {
            case string:
                sb.WriteString(fb)
            case *Feedback:
                sb.WriteString(fmt.Sprintf("Score: %.2f\n", fb.Score))
                sb.WriteString(fmt.Sprintf("Comment: %s\n", fb.Comment))
                if len(fb.FailureModes) > 0 {
                    sb.WriteString(fmt.Sprintf("Issues: %s\n",
                        strings.Join(fb.FailureModes, ", ")))
                }
            case *EditFeedback:
                sb.WriteString(fmt.Sprintf("Revised: %s\n", fb.Revised))
            }
            sb.WriteString("\n")
        }
    }

    return sb.String()
}
```

### Example Output

```
## Session 1

User: Tell me about Mars
Assistant: Mars is the fourth planet...
User: I wanted more about its moons

### Feedback

Score: 0.70
Comment: Should include details about moons when discussing planets
Issues: incomplete, missing_relevant_details

## Session 2

User: What are Mars' moons?
Assistant: Mars has two moons: Phobos and Deimos...

### Feedback

Score: 0.95
Comment: Good response with proper details
```

---

## Annotated Trajectory (Tuple Format)

LangMEM uses a tuple format for annotated trajectories:

```python
# Python tuple format
trajectory = (
    [message1, message2, ...],  # Messages
    feedback_data               # Feedback
)
```

**Go Equivalent:**

```go
// AnnotatedTrajectory matches Python's tuple format
type AnnotatedTrajectory struct {
    Messages []Message
    Feedback interface{}
}

// Constructor for convenient creation
func NewAnnotatedTrajectory(messages []Message, feedback interface{}) *Trajectory {
    return &Trajectory{
        Messages: messages,
        Feedback: feedback,
    }
}
```

---

## Validation

```go
// Validate ensures trajectory data is well-formed
func (t *Trajectory) Validate() error {
    if len(t.Messages) == 0 {
        return fmt.Errorf("trajectory must have at least one message")
    }

    for i, msg := range t.Messages {
        if msg.Role == "" {
            return fmt.Errorf("message %d: role cannot be empty", i)
        }
        if msg.Content == "" && msg.ToolCalls == nil {
            return fmt.Errorf("message %d: must have content or tool calls", i)
        }
    }

    return nil
}

// ValidateFeedback ensures feedback is valid type
func (t *Trajectory) ValidateFeedback() error {
    if t.Feedback == nil {
        return nil // No feedback is valid
    }

    switch fb := t.Feedback.(type) {
    case string:
        if fb == "" {
            return fmt.Errorf("string feedback cannot be empty")
        }
    case *Feedback:
        if fb.Score < 0 || fb.Score > 1 {
            return fmt.Errorf("score must be between 0 and 1")
        }
    case *EditFeedback:
        if fb.Revised == "" {
            return fmt.Errorf("edit feedback must have revised text")
        }
    default:
        return fmt.Errorf("unsupported feedback type: %T", fb)
    }

    return nil
}
```

---

## Collection Structures

```go
// TrajectoryCollection manages multiple trajectories
type TrajectoryCollection struct {
    trajectories []*Trajectory
}

func NewTrajectoryCollection(trajs ...*Trajectory) *TrajectoryCollection {
    return &TrajectoryCollection{
        trajectories: trajs,
    }
}

// Filter returns filtered subset
func (tc *TrajectoryCollection) Filter(pred func(*Trajectory) bool) *TrajectoryCollection {
    var filtered []*Trajectory
    for _, t := range tc.trajectories {
        if pred(t) {
            filtered = append(filtered, t)
        }
    }
    return &TrajectoryCollection{trajectories: filtered}
}

// WithFeedback returns only trajectories with feedback
func (tc *TrajectoryCollection) WithFeedback() *TrajectoryCollection {
    return tc.Filter(func(t *Trajectory) bool {
        return t.Feedback != nil
    })
}

// WithFailure returns only trajectories marked as failures
func (tc *TrajectoryCollection) WithFailure() *TrajectoryCollection {
    return tc.Filter(func(t *Trajectory) bool {
        if fb, ok := t.Feedback.(*Feedback); ok {
            return fb.Outcome == "failure"
        }
        return false
    })
}

// Format returns LLM-readable string
func (tc *TrajectoryCollection) Format() string {
    return formatSessions(tc.trajectories)
}
```

---

## Metrics (Deferred)

As per user decision, metrics tracking is postponed:

```go
// Deferred for future implementation
type TrajectoryMetrics struct {
    Latency    time.Duration
    TokenCount int
    Cost       float64
    Timestamp  time.Time
}
```

---

## Summary

| Component | Type | Required | Description |
|-----------|------|----------|-------------|
| `Messages` | `[]Message` | Yes | Conversation history |
| `Role` | `string` | Yes | user/assistant/tool |
| `Content` | `string` | No* | Message text |
| `ToolCalls` | `[]ToolCall` | No* | Tool invocations |
| `Feedback` | `interface{}` | No | Evaluation data |

*One of Content or ToolCalls required

---

## See Also

- [LangMEM Prompt Templates](./01-langmem-prompt-templates.md)
- [LangMEM Architecture](./03-langmem-architecture.md)
