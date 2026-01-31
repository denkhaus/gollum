# Type Mapping: Gollem vs. Prompt Optimizer

**Created:** 2025-01-31
**Purpose:** Vermeidung von Doppeldefinitionen durch Nutzung von Gollem-Typen

---

## Zusammenfassung

Gollem stellt bereits alle grundlegenden Message- und Tool-Typen bereit. Wir sollten diese verwenden statt eigene Definitionen zu erstellen.

---

## Type Mapping

### ✅ Verwende Gollem-Typen (NICHT neu definieren)

| Unser Typ | Gollem Typ | Package | Verwendung |
|-----------|------------|---------|------------|
| `Message` | `gollem.Message` | `github.com/m-mizutani/gollem` | Unified Message Format |
| `Role` | `gollem.MessageRole` | `github.com/m-mizutani/gollem` | RoleSystem, RoleUser, RoleAssistant, RoleTool |
| `ToolCall` | `gollem.ToolCallContent` | `github.com/m-mizutani/gollem` | Tool-Aufruf in Message |
| `ToolResponse` | `gollem.ToolResponseContent` | `github.com/m-mizutani/gollem` | Tool-Antwort |
| `ToolSpec` | `gollem.ToolSpec` | `github.com/m-mizutani/gollem` | Tool-Spezifikation |

### 🆕 Neu zu definieren (Prompt-Optimizer spezifisch)

| Typ | Package | Grund |
|-----|---------|-------|
| `Trajectory` | `pkg/prompt/optimizer` | Konversations-Historie mit Feedback |
| `Feedback` | `pkg/prompt/optimizer` | Bewertung/Feedback auf Trajektorie |
| `EditFeedback` | `pkg/prompt/optimizer` | Revisions-basiertes Feedback |
| `OptimizerInput` | `pkg/prompt/optimizer` | Optimizer-Input-Struktur |
| `Prompt` | `pkg/prompt` | Prompt-Struktur (unabhängig von Gollem) |
| `PromptContext` | `pkg/prompt` | Template-Rendering-Context |

---

## Aktualisierte Trajectory-Definition

```go
// pkg/prompt/optimizer/types.go

package optimizer

import (
    "github.com/m-mizutani/gollem"
)

// Trajectory repräsentiert eine Konversation mit optionalen Feedback
type Trajectory struct {
    Messages []gollem.Message  `json:"messages"`  // ✅ Gollem Typ
    Feedback interface{}       `json:"feedback,omitempty"`
}

// Feedback provides strukturiertes Feedback
type Feedback struct {
    Score        float64  `json:"score,omitempty"`
    Comment      string   `json:"comment,omitempty"`
    FailureModes []string `json:"failure_modes,omitempty"`
    Outcome      string   `json:"outcome,omitempty"` // "success" | "failure"
}

// EditFeedback provides Revisions-Feedback
type EditFeedback struct {
    Revised string     `json:"revised"`
    Edits   []TextEdit `json:"edits,omitempty"`
}

type TextEdit struct {
    OldText string `json:"old_text"`
    NewText string `json:"new_text"`
    Reason  string `json:"reason,omitempty"`
}

// OptimizerInput ist der Input zur Optimierung
type OptimizerInput struct {
    Trajectories interface{} `json:"trajectories"` // []*Trajectory oder string
    Prompt       interface{} `json:"prompt"`       // string oder *PromptWithMeta
}

type PromptWithMeta struct {
    Prompt             string `json:"prompt"`
    UpdateInstructions string `json:"update_instructions,omitempty"`
    Feedback           string `json:"feedback,omitempty"`
    WhenToUpdate       string `json:"when_to_update,omitempty"`
}
```

---

## Gollem Message Konstanten

```go
// Aus Gollem importieren
import "github.com/m-mizutani/gollem"

// Verfügbare Roles
gollem.RoleSystem    // "system"
gollem.RoleUser      // "user"
gollem.RoleAssistant // "assistant"
gollem.RoleTool      // "tool"

// Message Content Types
gollem.MessageContentTypeText         // "text"
gollem.MessageContentTypeImage        // "image"
gollem.MessageContentTypeToolCall     // "tool_call"
gollem.MessageContentTypeToolResponse // "tool_response"
```

---

## Helper-Funktionen für Message-Erstellung

Gollem stellt bereits Helper bereit:

```go
// Text-Message
content, _ := gollem.NewTextContent("Hello")
msg := gollem.Message{
    Role:     gollem.RoleUser,
    Contents: []gollem.MessageContent{content},
}

// Tool-Call
toolContent, _ := gollem.NewToolCallContent("call_123", "search", map[string]any{
    "query": "example",
})
msg := gollem.Message{
    Role:     gollem.RoleAssistant,
    Contents: []gollem.MessageContent{toolContent},
}

// Tool-Response
responseContent, _ := gollem.NewToolResponseContent("call_123", "search", map[string]any{
    "result": "found",
}, false)
msg := gollem.Message{
    Role:     gollem.RoleTool,
    Contents: []gollem.MessageContent{responseContent},
}
```

---

## Aktualisierte formatSessions Funktion

```go
// formatSessions konvertiert Trajectories zu LLM-lesbarem Format
func formatSessions(trajectories []*Trajectory) string {
    var sb strings.Builder

    for i, traj := range trajectories {
        sb.WriteString(fmt.Sprintf("## Session %d\n\n", i+1))

        for _, msg := range traj.Messages {
            // ✅ Nutze Gollem Message
            role := string(msg.Role)

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
            // Feedback formatting...
        }
    }

    return sb.String()
}
```

---

## Zu aktualisierende Dateien

### 1. PROMPT_OPTIMIZER_PLAN.md

**Entferne:**
- Definition von `Message`, `Role`, `ToolCall`

**Ersetze mit:**
```go
// ✅ Gollem Message verwenden
import "github.com/m-mizutani/gollem"

type Trajectory struct {
    Messages []gollem.Message  // ✅ Importiert aus Gollem
    Feedback interface{}
}
```

### 2. docs/prompt-optimizer/04-trajectory-structure.md

**Aktualisiere:**
- Remove duplicate Message/Role definitions
- Add Gollem import section
- Update examples to use `gollem.Message`

### 3. pkg/prompt/optimizer/types.go (bei Implementation)

```go
// ✅ Korrekte Import-Struktur
package optimizer

import (
    "github.com/m-mizutani/gollem"
)

// Trajectory verwendet Gollem Message
type Trajectory struct {
    Messages []gollem.Message
    Feedback interface{}
}

// Restliche Types bleiben gleich...
```

---

## Vorteile der Gollem-Typ-Nutzung

1. **DRY Prinzip**: Keine Doppeldefinitionen
2. **Cross-Provider Compatibility**: Gollem Message-Format ist provider-agnostisch
3. **Wartbarkeit**: Änderungen an Gollem profitieren automatisch unseren Optimizer
4. **Konsistenz**: Gleiche Message-Typen im ganzen Projekt
5. **Weniger Code**: Weniger eigene Typ-Definitionen und Validierung

---

## Checkliste für Implementation

- [ ] PROMPT_OPTIMIZER_PLAN.md aktualisieren (Gollem-Typen verwenden)
- [ ] docs/prompt-optimizer/04-trajectory-structure.md aktualisieren
- [ ] Imports in allen Strategie-Dateien korrigieren
- [ ] formatSessions() Helper aktualisieren
- [ ] Gollem Message Helper in Beispielen verwenden
- [ ] Tests mit Gollem Message-Typen schreiben
