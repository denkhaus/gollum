# Refactoring: Workspace Service & Skills Consolidation

## Problem

1. **Redundante Types**: `SkillInfo` und `WorkspaceContext` existieren in beiden Packages (`workspace` und `prompt`)
2. **Fehlerhafter Datenfluss**: Registry holt Skills vom WorkspaceService, aber dieser wird nie aktualisiert
3. **Verantwortlichkeiten vermischt**: WorkspaceService verwaltet Skills, obwohl es einen dedizierten SkillService gibt

## Ziel

1. Types in `shared` Package zentralisieren
2. WorkspaceService nur für Pfad-Management zuständig machen
3. SkillService als einzige Quelle für Skills-Informationen
4. Registry nutzt SkillService direkt

---

## Phase 1: Types in `shared` Package erstellen

### 1.1 Neue Datei: `pkg/shared/workspace.go`

```go
package shared

// SkillInfo holds information about a discovered skill
type SkillInfo struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Location    string `json:"location"`
}

// WorkspaceContext holds workspace-specific information for prompt rendering
type WorkspaceContext struct {
    CurrentPath string      `json:"current_path"`
    SkillsXML   string      `json:"skills_xml"`
    Skills      []SkillInfo `json:"skills"`
}
```

### 1.2 Verwendungen aktualisieren

- `pkg/workspace/service.go` - Import aus shared, lokale Definition entfernen
- `pkg/prompt/types.go` - Import aus shared, lokale Definition entfernen
- `pkg/registry/registry.go` - Import aus shared, convertSkillInfo entfernen
- `pkg/mocks/mock_workspace_service.go` - Regenerieren

---

## Phase 2: WorkspaceService entschlacken

### 2.1 Interface reduzieren

**Vorher:**
```go
type Service interface {
    SetCurrentWorkspace(path string) error
    GetCurrentWorkspace() string
    AddToHistory(path string)
    GetWorkspaceHistory() []string
    ClearHistory()
    SetSkillsContext(skills []SkillInfo, skillsXML string)  // ENTFERNEN
    GetSkillsContext() []SkillInfo                          // ENTFERNEN
    GetSkillsXML() string                                   // ENTFERNEN
    GetWorkspaceContext() *WorkspaceContext                 // ENTFERNEN
}
```

**Nachher:**
```go
type Service interface {
    SetCurrentWorkspace(path string) error
    GetCurrentWorkspace() string
}
```

### 2.2 WorkspaceHistory Entscheidung

**Option A:** Ganz entfernen (wenn nicht wirklich gebraucht)
**Option B:** Zum SkillService verschieben (dort wo es genutzt wird)

Empfehlung: **Option B** - SkillService verwaltet seine eigenen Search-Paths

### 2.3 Implementierung bereinigen

- `skills` und `skillsXML` Felder entfernen
- `history` und zugehörige Methoden entfernen oder zu SkillService verschieben
- `GetWorkspaceContext()` entfernen

---

## Phase 3: SkillService erweitern

### 3.1 Neue Methoden im Interface

```go
type SkillService interface {
    // ... existing methods ...

    // New methods for workspace context
    GetSkillsXML() string
    GetSkillInfos() []shared.SkillInfo
    GetWorkspaceContext(currentPath string) *shared.WorkspaceContext
}
```

### 3.2 Implementierung

```go
func (s *skillServiceImpl) GetSkillsXML() string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Generate XML from current skills
    return generateSkillsXML(s.skills)
}

func (s *skillServiceImpl) GetSkillInfos() []shared.SkillInfo {
    s.mu.RLock()
    defer s.mu.RUnlock()

    result := make([]shared.SkillInfo, 0, len(s.skills))
    for _, skill := range s.skills {
        result = append(result, shared.SkillInfo{
            Name:        skill.Name,
            Description: skill.Description,
            Location:    skill.FilePath,
        })
    }
    return result
}

func (s *skillServiceImpl) GetWorkspaceContext(currentPath string) *shared.WorkspaceContext {
    return &shared.WorkspaceContext{
        CurrentPath: currentPath,
        SkillsXML:   s.GetSkillsXML(),
        Skills:      s.GetSkillInfos(),
    }
}
```

---

## Phase 4: Registry anpassen

### 4.1 Dependencies ändern

**Vorher:**
```go
type agentRegistry struct {
    // ...
    workspaceService workspace.Service
}
```

**Nachher:**
```go
type agentRegistry struct {
    // ...
    workspaceService workspace.Service  // Nur noch für CurrentPath
    skillService     skills.SkillService // Für Skills
}
```

### 4.2 handleSkillsUpdated anpassen

**Vorher:**
```go
wsCtx := r.workspaceService.GetWorkspaceContext()
renderCtx := &prompt.RenderContext{
    Workspace: &prompt.WorkspaceContext{
        CurrentPath: wsCtx.CurrentPath,
        SkillsXML:   payload.SkillsXML,
        Skills:      convertSkillInfo(wsCtx.Skills),
    },
}
```

**Nachher:**
```go
currentPath := r.workspaceService.GetCurrentWorkspace()
renderCtx := &prompt.RenderContext{
    Workspace: r.skillService.GetWorkspaceContext(currentPath),
}
```

### 4.3 convertSkillInfo entfernen

Die Funktion wird nicht mehr benötigt.

---

## Phase 5: Prompt Package anpassen

### 5.1 Types entfernen

In `pkg/prompt/types.go`:
- `SkillInfo` struct entfernen
- `WorkspaceContext` struct entfernen

### 5.2 Imports aktualisieren

Alle Dateien die diese Types nutzen, importieren sie nun aus `shared`.

---

## Phase 6: Mocks regenerieren

```bash
cd pkg/mocks && go generate ./...
```

---

## Datei-Änderungen Übersicht

| Datei | Aktion |
|-------|--------|
| `pkg/shared/workspace.go` | **NEU** - SkillInfo, WorkspaceContext |
| `pkg/workspace/service.go` | **Ändern** - Interface reduzieren, Types entfernen |
| `pkg/skills/service.go` | **Ändern** - Neue Methoden, Search-Paths übernehmen |
| `pkg/skills/types.go` | **Ändern** - Interface erweitern |
| `pkg/registry/registry.go` | **Ändern** - SkillService nutzen, convertSkillInfo entfernen |
| `pkg/prompt/types.go` | **Ändern** - Types entfernen |
| `pkg/mocks/*.go` | **Regenerieren** |

---

## Tests

Nach jeder Phase:
```bash
go test ./...
```

---

## Reihenfolge

1. Phase 1 (Types in shared) - Foundation
2. Phase 3 (SkillService erweitern) - Neue Quelle bereitstellen
3. Phase 4 (Registry anpassen) - Consumer umstellen
4. Phase 2 (WorkspaceService entschlacken) - Alte Quelle entfernen
5. Phase 5 (Prompt Package) - Types entfernen
6. Phase 6 (Mocks) - Regenerieren
