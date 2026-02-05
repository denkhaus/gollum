# Prompt Optimizer - Implementierungsplan

**Status:** Ready for Implementation
**Created:** 2025-01-31
**Updated:** 2025-01-31
**Referenz:** [LangMEM](https://github.com/langchain-ai/langmem)
**Dokumentation:** `docs/prompt-optimizer/` (LangMEM-Extraktion + Typ-Mapping)

---

## Phase Boundary

Erweiterung des existierenden `pkg/prompt/` Package um:
1. **Prompt Store Interface** - Abstraktion für verschiedene Persistenz-Backends
2. **Prompt Manager Erweiterung** - ID-basierte Prompt-Verwaltung mit Context-Rendering
3. **Prompt Optimizer** - Analysiert Konversationen und optimiert Prompts

**NICHT im Scope:** Memory-Manager, Vektor-Datenbank, semantische Suche (dies sind getrennte Komponenten)

---

## Architektur-Überblick

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    PROMPT ARCHITEKTUR MIT PERSISTENZ-SCHICHT                   │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │                    APPLICATION LAYER                                  │   │
│  │                                                                         │   │
│  │  ┌──────────────┐    ┌──────────────────┐    ┌──────────────────────┐ │   │
│  │  │Prompt        │    │Prompt           │    │Prompt Optimizer    │ │   │
│  │  │Manager       │◄───│Optimizer        │◄───│                    │ │   │
│  │  │(erweitert)    │    │                  │    │(nutzt Store)       │ │   │
│  │  └───────┬───────┘    └──────────────────┘    └──────────────────────┘ │   │
│  └──────────┼──────────────────────────────────────────────────────────────────┘   │
│             │                                                                  │
│  ┌──────────▼─────────────────────────────────────────────────────────────────┐   │
│  │                    STORE LAYER (Repository Pattern)                     │   │
│  │                                                                         │   │
│  │  ┌────────────────────────────────────────────────────────────────────┐ │   │
│  │  │              PromptStore (Interface)                               │ │   │
│  │  │                                                                     │ │   │
│  │  │  • Save(prompt *Prompt) error                                       │ │   │
│  │  │  • Load(id string) (*Prompt, error)                                │ │   │
│  │  │  • Delete(id string) error                                         │ │   │
│  │  │  • List() ([]*Prompt, error)                                       │ │   │
│  │  │  • Exists(id string) bool                                          │ │   │
│  │  └────────────────────────────────────────────────────────────────────┘ │   │
│  │                                │                                       │   │
│  │  ┌───────────────┬──────────────┬───────────────┬─────────────┐       │   │
│  │  │               │              │               │             │       │   │
│  │  ▼               ▼              ▼               ▼             ▼       │   │
│  │  ┌──────────┐  ┌─────────┐   ┌──────────┐  ┌──────────┐  ┌─────────┐│   │
│  │  │InMemory  │  │File     │   │Langfuse  │  │Database │  │Custom   ││   │
│  │  │Store     │  │Store    │   │Store    │  │Store    │  │Store    ││   │
│  │  │(default) │  │(JSON)   │   │(Remote) │  │(Postgres)│  │(Plugin)  ││   │
│  │  └──────────┘  └─────────┘   └──────────┘  └──────────┘  └─────────┘│   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Analyse der existierenden Codebase

### Existierende Komponenten (werden verwendet)

| Komponente | Pfad | Verwendung |
|------------|------|------------|
| **PromptManager** | `pkg/prompt/manager.go` | Wird erweitert um Store + dynamische Prompts |
| **LLM ClientProvider** | `pkg/llm/provider.go` | Liefert `gollem.LLMClient` für Optimizer |
| **ConfigService** | `pkg/config/service.go` | Neue Configs für Store und Optimizer |
| **LoggerService** | `pkg/logger/logger.go` | Logging mit zap |
| **DI Container** | `pkg/di/container.go` | Registrierung aller Services |

---

## Package Struktur

```
pkg/prompt/
├── manager.go              (existiert - wird erweitert)
├── templates/               (existiert - embedded templates)
│   ├── compacter_prompt.md
│   ├── subagent_system_prompt.md
│   ├── supervisor_system_prompt.md
│   └── subagent_task_prompt.md
├── types.go                (NEU - Core Types für Prompts)
│
├── store/                   (NEU - Persistenz-Schicht)
│   ├── store.go             (PromptStore Interface)
│   ├── memory_store.go      (In-Memory Implementierung)
│   ├── file_store.go        (File-basierte Implementierung + syscall.Flock)
│   ├── langfuse_store.go   (Langfuse Implementierung - optional, später)
│   ├── provider.go          (DI Provider)
│   └── store_test.go        (Tests)
│
├── optimizer/               (NEU - Prompt Optimizer)
│   ├── types.go             (Trajectory, Feedback, OptimizerInput)
│   ├── optimizer.go         (Optimizer Interface + Factory)
│   ├── strategies/          (NEU - Separate Strategie-Dateien)
│   │   ├── gradient.go      (Gradient Strategie mit think/critique)
│   │   ├── metaprompt.go    (Meta-Prompt Strategie)
│   │   └── memory.go        (Prompt Memory Strategie)
│   ├── templates.go         (Optimizer Prompt Templates aus LangMEM)
│   ├── tools.go             (Tool-Definitionen: think, critique, recommend)
│   └── optimizer_test.go    (Tests)
│
└── manager_test.go         (existiert - wird erweitert)
```

**Design-Entscheidungen:**
- **`pkg/prompt/types.go`**: Zentralisierte Core Types (vermeidet Circular Dependencies)
- **`strategies/` Subdirectory**: Separate Dateien pro Strategie (bessere Testbarkeit)
- **`syscall.Flock`**: Native Unix File-Locking für File Store
- **Lazy Init Pattern**: Built-in Prompts bei Erstzugriff aus Embedded FS laden

---

## Phase 1: Store Interface (Grundbaustein)

### 1.1 Core Types

```go
// pkg/prompt/types.go
package prompt

import (
    "time"

    "github.com/Masterminds/semver/v3"
)

// Prompt repräsentiert einen einzelnen Prompt mit Metadaten
type Prompt struct {
    ID        string                 `json:"id"`         // Eindeutige ID (z.B. "system", "subagent@1.0.0")
    Name      string                 `json:"name"`       // Lesbarer Name
    Content   string                 `json:"content"`    // Der Prompt-Text (kann Template-Syntax haben)
    Context   map[string]interface{} `json:"context,omitempty"` // Optional: Context-Werte für Rendering
    Tags      []string               `json:"tags,omitempty"`      // Optional: Tags für Kategorisierung
    CreatedAt time.Time             `json:"created_at"`
    UpdatedAt time.Time             `json:"updated_at"`
    Version   *semver.Version        `json:"version"`    // SemVer Version für Änderungs-Tracking
    IsBuiltin bool                   `json:"is_builtin"`  // Built-in Prompts können nicht gelöscht werden
}

// PromptContext definiert die verfügbaren Context-Werte für Rendering
type PromptContext struct {
    // Allgemeine Werte
    Values map[string]interface{} `json:"values,omitempty"`

    // Spezifische Werte für bestimmte Prompt-Typen
    SubAgent *SubAgentContext `json:"subagent,omitempty"`
    Agent    *AgentContext    `json:"agent,omitempty"`
}

// SubAgentContext enthält Werte für Subagent-Prompts
type SubAgentContext struct {
    Role            string `json:"role"`
    Description     string `json:"description"`
    SpawnAgentTool  string `json:"spawn_agent_tool"`
    RemoveAgentTool string `json:"remove_agent_tool"`
    ResumeAgentTool string `json:"resume_agent_tool"`
    AgentOutputTool string `json:"agent_output_tool"`
    ListAgentsTool  string `json:"list_agents_tool"`
}

// AgentContext enthält Werte für allgemeine Agent-Prompts
type AgentContext struct {
    AgentID string `json:"agent_id"`
    Task    string `json:"task"`
}

// Built-in Prompt IDs (Konstanten)
const (
    PromptIDSystem      = "system"
    PromptIDSupervisor  = "supervisor"
    PromptIDCompacter   = "compacter"
    PromptIDSubagent    = "subagent" // Dynamisch mit Role/Description
)
```

### 1.2 Trajectory Types (für Optimizer)

```go
// pkg/prompt/optimizer/types.go

package optimizer

import (
    "github.com/Masterminds/semver/v3"
    "github.com/m-mizutani/gollem"
)

// Trajectory repräsentiert eine Konversation mit optionalen Feedback
// ✅ Verwendet Gollem Message für Konsistenz mit dem Rest des Projekts
type Trajectory struct {
    Messages []gollem.Message `json:"messages"`  // ✅ Gollem Message (keine Doppeldefinition!)
    Feedback interface{}      `json:"feedback,omitempty"` // nil, string, *Feedback, *EditFeedback
}

// Feedback provides strukturiertes Feedback
type Feedback struct {
    Score        float64  `json:"score,omitempty"`         // 0.0 - 1.0
    Comment      string   `json:"comment,omitempty"`
    FailureModes []string `json:"failure_modes,omitempty"`
    Outcome      string   `json:"outcome,omitempty"`       // "success" | "failure"
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
// GAP 11 RESOLVED: Erweiterter Input-Typ mit Prompt-Metadaten
type OptimizerInput struct {
    // Prompt-Informationen
    PromptID   string // Basis-ID (z.B. "subagent" oder "subagent@latest")
    Prompt     string // Aktueller Prompt-Content

    // Optimierungs-Kontext
    Trajectories interface{} // []*Trajectory oder string (pre-formatted)

    // Optionale Metadaten für neue Version
    NewVersionName string // Optional: Name für neue Version
}

// OptimizerResult enthält das Ergebnis der Optimierung
// GAP 11 RESOLVED: Result-Typ mit Prompt-Versionsinformationen
type OptimizerResult struct {
    OptimizedContent string             // Der optimierte Prompt-Content
    NewPromptID      string             // Die ID der neuen Version (z.B. "subagent@1.0.1")
    OldPromptID      string             // Die ID des ursprünglichen Prompts
    Version          *semver.Version    // Die neue Version
    Changes          []string           // Beschreibung der Änderungen (optional)
}

type PromptWithMeta struct {
    Prompt             string `json:"prompt"`
    UpdateInstructions string `json:"update_instructions,omitempty"`
    Feedback           string `json:"feedback,omitempty"`
    WhenToUpdate       string `json:"when_to_update,omitempty"`
}
```

**Gollem Message Referenz:**

| Gollem Typ | Verwendung |
|------------|------------|
| `gollem.Message` | Unified Message Format |
| `gollem.MessageRole` | RoleSystem, RoleUser, RoleAssistant, RoleTool |
| `gollem.ToolCallContent` | Tool-Aufruf in Message |
| `gollem.ToolResponseContent` | Tool-Antwort |
| `gollem.NewTextContent(text)` | Helper für Text-Content |
| `gollem.NewToolCallContent(...)` | Helper für Tool-Calls |

**Siehe auch:** `docs/prompt-optimizer/type-mapping-gollem.md`
```

### 1.2 PromptStore Interface

```go
// pkg/prompt/store/store.go
package store

import (
    "context"

    "github.com/Masterminds/semver/v3"
    "github.com/denkhaus/gollum/pkg/prompt"
)

// PromptStore defines the interface for prompt persistence
//
//revive:disable-next-line:exported
type PromptStore interface {
    // SaveNewVersion erstellt eine neue Version eines Prompts
    // - Gibt die neue ID zurück (z.B. "subagent@1.0.1")
    // - Auto-incrementiert die Patch-Version (1.0.0 → 1.0.1)
    // - Setzt CreatedAt auf jetzt, UpdatedAt auf jetzt
    // - GAP 8 RESOLVED: Neue Methode für gezieltes Versioning
    SaveNewVersion(ctx context.Context, baseID string, content string, name string) (*prompt.Prompt, error)

    // Load liest einen Prompt per ID
    // Gibt nil zurück, wenn der Prompt nicht existiert (kein Fehler)
    Load(ctx context.Context, id string) (*prompt.Prompt, error)

    // Delete löscht einen Prompt per ID
    // Gibt nil zurück, wenn der Prompt nicht existiert
    Delete(ctx context.Context, id string) error

    // List listet alle verfügbaren Prompts auf
    // Optional filter für Tags
    List(ctx context.Context, filter *ListFilter) ([]*prompt.Prompt, error)

    // Exists prüft, ob ein Prompt existiert
    Exists(ctx context.Context, id string) (bool, error)

    // ListTags listet alle verfügbaren Tags
    ListTags(ctx context.Context) ([]string, error)

    // ResolveAlias löst Shortcuts und Aliases zu gültigen versionierten IDs auf
    // Beispiele:
    //   "subagent" → "subagent@latest" → "subagent@1.1.0" (Shortcut)
    //   "system@latest" → "system@1.0.0" (Alias)
    //   "system@1.0.0" → "system@1.0.0" (Bereits versioniert)
    // GAP 10 RESOLVED: Shortcut-Auflösung für unversionierte IDs
    ResolveAlias(ctx context.Context, id string) (*prompt.Prompt, error)

    // ListVersions listet alle Versionen eines base ID
    // Beispiel: ListVersions("subagent") → [subagent@1.0.0, subagent@1.1.0]
    ListVersions(ctx context.Context, baseID string) ([]*prompt.Prompt, error)

    // SetLatestAlias setzt den @latest Alias für eine base ID auf eine bestimmte Version
    // Entfernt @latest von allen anderen Versionen der gleichen base ID
    // GAP 12 RESOLVED: Nur neueste Version hat Aliases
    SetLatestAlias(ctx context.Context, baseID, versionID string) error
}

// ListFilter definiert Filterkriterien für List
type ListFilter struct {
    Tags []string // Nur Prompts mit diesen Tags
    IDs  []string // Nur Prompts mit diesen IDs
}

// Fehler
var (
    ErrPromptNotFound  = errors.New("prompt not found")
    ErrPromptIsBuiltin = errors.New("cannot modify built-in prompt")
    ErrInvalidPromptID = errors.New("invalid prompt ID: must be versioned or resolvable shortcut")
    ErrCASFailed       = errors.New("prompt was modified by another process")
)

// PromptStoreConfig enthält Konfiguration für Store-Initialisierung
type PromptStoreConfig struct {
    // Type definiert den Store-Typ
    // Options: "memory", "file", "langfuse"
    Type string `envconfig:"TYPE" default:"memory"`

    // File-Store Konfiguration
    FilePath string `envconfig:"FILE_PATH" default:"./data/prompts"`

    // Langfuse Store Konfiguration
    LangfusePublicKey string `envconfig:"LANGFUSE_PUBLIC_KEY"`
    LangfuseSecretKey string `envconfig:"LANGFUSE_SECRET_KEY"`
    LangfuseHost      string `envconfig:"LANGFUSE_HOST" default:"https://cloud.langfuse.com"`

    // Cache-Konfiguration (optional)
    CacheEnabled bool `envconfig:"CACHE_ENABLED" default:"true"`
}
```

### 1.3 Implementierungen

```go
// pkg/prompt/store/memory_store.go

package store

import (
    "context"
    "fmt"
    "sync"

    "github.com/Masterminds/semver/v3"
    "github.com/denkhaus/gollum/pkg/prompt"
)

type memoryStore struct {
    sync.RWMutex
    prompts map[string]*prompt.Prompt
}

// NewMemoryStore creates a new in-memory prompt store
func NewMemoryStore() PromptStore {
    return &memoryStore{
        prompts: make(map[string]*prompt.Prompt),
    }
}

// Save() entfernt - Nur SaveNewVersion() wird verwendet
// Architektur-Entscheidung: Built-in Prompts werden via SaveNewVersion() geladen,
// Optimizer erstellt neue Versionen via SaveNewVersion(). Kein direktes Save() nötig.

func (s *memoryStore) Load(ctx context.Context, id string) (*prompt.Prompt, error) {
    s.RLock()
    defer s.RUnlock()

    return s.prompts[id], nil
}

func (s *memoryStore) Delete(ctx context.Context, id string) error {
    s.Lock()
    defer s.Unlock()

    prompt := s.prompts[id]
    if prompt != nil && prompt.IsBuiltin {
        return fmt.Errorf("cannot delete built-in prompt '%s'", id)
    }

    delete(s.prompts, id)
    return nil
}

func (s *memoryStore) List(ctx context.Context, filter *ListFilter) ([]*prompt.Prompt, error) {
    s.RLock()
    defer s.RUnlock()

    result := make([]*prompt.Prompt, 0)

    for _, prompt := range s.prompts {
        // Filter anwenden
        if filter != nil {
            if len(filter.Tags) > 0 && !hasAnyTag(prompt.Tags, filter.Tags) {
                continue
            }
            if len(filter.IDs) > 0 && !containsAny(prompt.ID, filter.IDs) {
                continue
            }
        }

        // Kopie zurückgeben
        result = append(result, copyPrompt(prompt))
    }

    return result, nil
}

func (s *memoryStore) Exists(ctx context.Context, id string) (bool, error) {
    s.RLock()
    defer s.RUnlock()

    _, exists := s.prompts[id]
    return exists, nil
}

func (s *memoryStore) ListTags(ctx context.Context) ([]string, error) {
    s.RLock()
    defer s.RUnlock()

    tagSet := make(map[string]bool)
    for _, prompt := range s.prompts {
        for _, tag := range prompt.Tags {
            tagSet[tag] = true
        }
    }

    result := make([]string, 0, len(tagSet))
    for tag := range tagSet {
        result = append(result, tag)
    }

    return result, nil
}
```

```go
// pkg/prompt/store/file_store.go

package store

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"

    "github.com/Masterminds/semver/v3"
    "github.com/denkhaus/gollum/pkg/prompt"
)

type fileStore struct {
    sync.RWMutex
    dir    string
    cache  map[string]*prompt.Prompt // Optionaler Cache
    enableCache bool
}

// NewFileStore creates a new file-based prompt store
func NewFileStore(dir string, enableCache bool) PromptStore {
    // Verzeichnis erstellen falls nicht vorhanden
    _ = os.MkdirAll(dir, 0755)

    return &fileStore{
        dir:        dir,
        cache:      make(map[string]*prompt.Prompt),
        enableCache: enableCache,
    }
}

// Save() entfernt - Nur SaveNewVersion() wird verwendet
// Architektur-Entscheidung: Built-in Prompts werden via SaveNewVersion() geladen,
// Optimizer erstellt neue Versionen via SaveNewVersion(). Kein direktes Save() nötig.

func (s *fileStore) Load(ctx context.Context, id string) (*prompt.Prompt, error) {
    // Zuerst Cache prüfen
    if s.enableCache {
        s.RLock()
        cached, exists := s.cache[id]
        s.RUnlock()
        if exists {
            return cached, nil
        }
    }

    path := s.getFilePath(id)

    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil // Nicht gefunden = kein Fehler
        }
        return nil, err
    }

    var prompt prompt.Prompt
    if err := json.Unmarshal(data, &prompt); err != nil {
        return nil, fmt.Errorf("failed to unmarshal prompt: %w", err)
    }

    // Cache aktualisieren
    if s.enableCache {
        s.Lock()
        s.cache[id] = &prompt
        s.Unlock()
    }

    return &prompt, nil
}

func (s *fileStore) Delete(ctx context.Context, id string) error {
    // Prüfen ob Built-in Prompt
    prompt, err := s.Load(ctx, id)
    if err != nil {
        return err
    }

    if prompt != nil && prompt.IsBuiltin {
        return fmt.Errorf("cannot delete built-in prompt '%s'", id)
    }

    path := s.getFilePath(id)

    if err := os.Remove(path); err != nil {
        if os.IsNotExist(err) {
            return nil // Bereits gelöscht = kein Fehler
        }
        return err
    }

    // Cache entfernen
    if s.enableCache {
        s.Lock()
        delete(s.cache, id)
        s.Unlock()
    }

    return nil
}

func (s *fileStore) List(ctx context.Context, filter *ListFilter) ([]*prompt.Prompt, error) {
    entries, err := os.ReadDir(s.dir)
    if err != nil {
        return nil, err
    }

    var result []*prompt.Prompt

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        // Dateien mit .json Endnung
        if filepath.Ext(entry.Name()) != ".json" {
            continue
        }

        id := entry.Name()[:len(entry.Name())-5] // .json entfernen

        // Filter anwenden
        if filter != nil {
            if len(filter.Tags) > 0 || len(filter.IDs) > 0 {
                prompt, err := s.Load(ctx, id)
                if err != nil {
                    continue // Überspringe fehlerhafte Dateien
                }

                if filter != nil {
                    if len(filter.Tags) > 0 && !hasAnyTag(prompt.Tags, filter.Tags) {
                        continue
                    }
                    if len(filter.IDs) > 0 && !containsAny(prompt.ID, filter.IDs) {
                        continue
                    }
                }

                result = append(result, prompt)
            }
        } else {
            // Kein Filter: alle laden
            prompt, err := s.Load(ctx, id)
            if err != nil {
                continue
            }
            result = append(result, prompt)
        }
    }

    return result, nil
}

func (s *fileStore) Exists(ctx context.Context, id string) (bool, error) {
    path := s.getFilePath(id)

    // Cache prüfen
    if s.enableCache {
        s.RLock()
        _, exists := s.cache[id]
        s.RUnlock()
        if exists {
            return true, nil
        }
    }

    // Dateisystem prüfen
    _, err := os.Stat(path)
    return err == nil, nil
}

func (s *fileStore) ListTags(ctx context.Context) ([]string, error) {
    prompts, err := s.List(ctx, nil)
    if err != nil {
        return nil, err
    }

    tagSet := make(map[string]bool)
    for _, prompt := range prompts {
        for _, tag := range prompt.Tags {
            tagSet[tag] = true
        }
    }

    result := make([]string, 0, len(tagSet))
    for tag := range tagSet {
        result = append(result, tag)
    }

    return result, nil
}

func (s *fileStore) getFilePath(id string) string {
    return filepath.Join(s.dir, id+".json")
}
```

```go
// pkg/prompt/store/langfuse_store.go (OPTIONAL - FÜR SPÄTERE PHASE)

package store

import (
    "context"
    "fmt"

    "github.com/denkhaus/gollum/pkg/prompt"
    "github.com/denkhaus/gollum/pkg/config"
)

// Langfuse-specific Config
type LangfuseStoreConfig struct {
    Host      string `envconfig:"HOST" default:"https://cloud.langfuse.com"`
    PublicKey string `envconfig:"PUBLIC_KEY"`
    SecretKey string `envconfig:"SECRET_KEY"`
}

type langfuseStore struct {
    config *LangfuseStoreConfig
    client *langfuse.Client // Hypothetischer Langfuse Client
}

// NewLangfuseStore creates a new Langfuse-backed prompt store
func NewLangfuseStore(cfg *LangfuseStoreConfig) PromptStore {
    return &langfuseStore{
        config: cfg,
        // client: langfuse.NewClient(cfg.Host, cfg.PublicKey, cfg.SecretKey),
    }
}

// Implementierung mit Langfuse API
func (s *langfuseStore) Save(ctx context.Context, prompt *prompt.Prompt) error {
    // Speichere Prompt als Langfuse Dataset
    // ...
}

func (s *langfuseStore) Load(ctx context.Context, id string) (*prompt.Prompt, error) {
    // Lade Prompt aus Langfuse
    // ...
}

// ... weitere Methoden
```

### 1.4 Store Provider (DI-konform)

```go
// pkg/prompt/store/provider.go

package store

import (
    "context"

    "github.com/denkhaus/gollum/pkg/config"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/samber/do/v2"
)

// PromptStoreProvider provides prompt store instances based on configuration
//
//revive:disable-next-line:exported
type PromptStoreProvider interface {
    // GetStore returns the configured PromptStore instance
    GetStore() PromptStore
}

type storeProviderImpl struct {
    logService    logger.LoggerService
    configService config.ConfigService
    store         PromptStore
}

// NewPromptStoreProvider creates a new PromptStoreProvider via DI
func NewPromptStoreProvider(injector do.Injector) (PromptStoreProvider, error) {
    logService := do.MustInvoke[logger.LoggerService](injector)
    configService := do.MustInvoke[config.ConfigService](injector)

    cfg := configService.GetPromptStoreConfig()

    var store PromptStore
    switch cfg.Type {
    case "file":
        store = NewFileStore(cfg.FilePath, cfg.CacheEnabled)
        logService.Info("prompt store initialized",
            logger.String("type", "file"),
            logger.String("path", cfg.FilePath),
        )
    case "langfuse":
        langfuseCfg := configService.GetLangfuseStoreConfig()
        store = NewLangfuseStore(langfuseCfg)
        logService.Info("prompt store initialized",
            logger.String("type", "langfuse"),
            logger.String("host", langfuseCfg.Host),
        )
    case "memory", "": // Default
        store = NewMemoryStore()
        logService.Info("prompt store initialized",
            logger.String("type", "memory"),
        )
    default:
        logService.Warn("unknown prompt store type, using memory",
            logger.String("type", cfg.Type),
        )
        store = NewMemoryStore()
    }

    return &storeProviderImpl{
        logService:    logService,
        configService: configService,
        store:         store,
    }, nil
}

func (p *storeProviderImpl) GetStore() PromptStore {
    return p.store
}
```

---

## Phase 2: Prompt Manager Erweiterung

### 2.1 Erweitertes Interface

```go
// pkg/prompt/manager.go - ERWEITERT

package prompt

import (
    "bytes"
    "context"
    "embed"
    "fmt"
    "sync"
    "text/template"
    "time"

    "github.com/denkhaus/gollum/pkg/prompt/store"
    "github.com/samber/do/v2"
)

//go:embed templates/*.md
var promptTemplates embed.FS

type (
    promptManager struct {
        sync.RWMutex
        store               store.PromptStore
        builtinLoaders      map[string]*sync.Once     // Thread-safe loading per prompt
        builtinPromptsCache map[string]*Prompt        // Cache for loaded built-ins
        builtinLoadError    map[string]error          // Errors from loading
    }

    // PromptManager provides prompt rendering and management services
    //
    //revive:disable-next-line:exported
    PromptManager interface {
        // ===== EXISTIERENDE METHODEN (Backward Compatibility) =====
        GetCompacterPrompt(data any) (string, error)
        GetSystemPrompt() (string, error)
        GetSupervisorPrompt() (string, error)
        GetSubagentPrompt(role, description string) (string, error)

        // ===== NEUE METHODEN (für Prompt Optimizer & dynamische Prompts) =====

        // GetPromptByID liest einen Prompt ohne Rendering
        GetPromptByID(ctx context.Context, id string) (*Prompt, error)

        // GetPromptWithContext liest und rendert einen Prompt mit Context
        GetPromptWithContext(ctx context.Context, id string, ctx *PromptContext) (string, error)

        // SetPrompt speichert oder aktualisiert einen Prompt im Store
        SetPrompt(ctx context.Context, prompt *Prompt) error

        // DeletePrompt löscht einen Prompt per ID
        DeletePrompt(ctx context.Context, id string) error

        // ListPrompts listet alle verfügbaren Prompts auf
        ListPrompts(ctx context.Context, filter *ListFilter) ([]*Prompt, error)

        // RenderPrompt rendert einen Prompt mit optionalen Context-Werten
        RenderPrompt(ctx context.Context, id string, ctx *PromptContext) (string, error)

        // GetStore gibt den zugrundeliegenden Store zurück
        GetStore() store.PromptStore
    }
)

// NewPromptManager erstellt eine neue Manager-Instanz mit Store
func NewPromptManager(injector do.Injector) (PromptManager, error) {
    storeProvider := do.MustInvoke[store.PromptStoreProvider](injector)

    pm := &promptManager{
        store:               storeProvider.GetStore(),
        builtinLoaders:      make(map[string]*sync.Once),
        builtinPromptsCache: make(map[string]*Prompt),
        builtinLoadError:    make(map[string]error),
    }

    // Built-in Prompts werden lazy geladen (nicht mehr beim Startup registrieren)
    // Die Funktion registerBuiltinPrompts() wurde entfernt zugunsten von Lazy Init

    return pm, nil
}

// ... existierende Methoden (angepasst mit Context) ...

// ===== NEUE METHODEN =====

// GetPromptByID lädt einen Prompt mit Lazy Init für Built-ins
// GAP 1.1 RESOLVED: Einheitliches Laden mit Lazy Init für Built-in Prompts
func (p *promptManager) GetPromptByID(ctx context.Context, id string) (*Prompt, error) {
    // 1. Zuerst aus Store laden (kann optimierte Version sein)
    prompt, err := p.store.Load(ctx, id)
    if err == nil && prompt != nil {
        return prompt, nil  // Gefunden im Store (Original oder Optimiert)
    }

    // 2. Nicht im Store → Built-in Prompt aus Embedded FS bootstrappen
    if p.isBuiltinID(id) {
        return p.bootstrapBuiltinPrompt(ctx, id)
    }

    // 3. Weder im Store noch Built-in → Fehler
    return nil, fmt.Errorf("prompt '%s' not found", id)
}

// bootstrapBuiltinPrompt lädt einen Built-in Prompt einmalig aus Embedded FS in den Store
func (p *promptManager) bootstrapBuiltinPrompt(ctx context.Context, id string) (*Prompt, error) {
    // Thread-safety: sync.Once per Built-in Prompt
    once := p.getBuiltinLoaderOnce(id)
    once.Do(func() {
        // 1. Aus Embedded FS laden
        content, err := p.loadEmbeddedTemplate(id)
        if err != nil {
            p.builtinLoadError[id] = err
            return
        }

        // 2. Als Version 1.0.0 in Store speichern
        version := semver.MustParse("1.0.0")
        promptID := id + "@1.0.0"  // Versionierte ID

        prompt := &Prompt{
            ID:        promptID,          // ID mit Version: "system@1.0.0"
            Name:      p.getBuiltinName(id),
            Content:   content,
            IsBuiltin: true,
            Version:   version,
            Aliases:   []string{id, id + "@latest"},  // "system" und "system@latest"
            Tags:      []string{"builtin"},
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        }

        if err := p.store.Save(ctx, prompt); err != nil {
            p.builtinLoadError[id] = err
            return
        }

        p.builtinPromptsCache[id] = prompt
    })

    // Fehler oder Ergebnis zurückgeben
    if err, ok := p.builtinLoadError[id]; ok {
        return nil, err
    }
    return p.builtinPromptsCache[id], nil
}

// isBuiltinID prüft, ob eine ID ein Built-in Prompt ist
func (p *promptManager) isBuiltinID(id string) bool {
    switch id {
    case PromptIDSystem, PromptIDSupervisor, PromptIDCompacter, PromptIDSubagent:
        return true
    default:
        return false
    }
}

// getBuiltinName gibt den lesbaren Namen für einen Built-in Prompt zurück
func (p *promptManager) getBuiltinName(id string) string {
    switch id {
    case PromptIDSystem:
        return "System Prompt"
    case PromptIDSupervisor:
        return "Supervisor Prompt"
    case PromptIDCompacter:
        return "Compacter Prompt"
    case PromptIDSubagent:
        return "Subagent Prompt"
    default:
        return id
    }
}

// loadEmbeddedTemplate lädt einen Prompt aus der Embedded FS
func (p *promptManager) loadEmbeddedTemplate(id string) (string, error) {
    // Dateiname aus ID ableiten: "system" → "system_prompt.md"
    filename := id + "_prompt.md"
    content, err := promptTemplates.ReadFile("templates/" + filename)
    if err != nil {
        return "", fmt.Errorf("no embedded template for '%s': %w", id, err)
    }
    return string(content), nil
}

// getBuiltinLoaderOnce gibt sync.Once für einen Prompt zurück (Thread-Safe)
func (p *promptManager) getBuiltinLoaderOnce(id string) *sync.Once {
    p.Lock()
    defer p.Unlock()
    once, ok := p.builtinLoaders[id]
    if !ok {
        once = &sync.Once{}
        p.builtinLoaders[id] = once
    }
    return once
}

func (p *promptManager) GetPromptWithContext(ctx context.Context, id string, ctx *PromptContext) (string, error) {
    return p.RenderPrompt(ctx, id, ctx)
}

func (p *promptManager) SetPrompt(ctx context.Context, prompt *Prompt) error {
    if prompt == nil {
        return fmt.Errorf("prompt cannot be nil")
    }

    if prompt.ID == "" {
        return fmt.Errorf("prompt ID cannot be empty")
    }

    return p.store.Save(ctx, prompt)
}

func (p *promptManager) DeletePrompt(ctx context.Context, id string) error {
    return p.store.Delete(ctx, id)
}

func (p *promptManager) ListPrompts(ctx context.Context, filter *ListFilter) ([]*Prompt, error) {
    return p.store.List(ctx, filter)
}

// RenderPrompt rendert einen Prompt mit Template-Variablen
// GAP 1.1 RESOLVED: Einheitliches Rendering für Built-in und Custom Prompts
func (p *promptManager) RenderPrompt(ctx context.Context, id string, tmplCtx *PromptContext) (string, error) {
    // 1. Prompt laden (mit Lazy Init für Built-ins)
    prompt, err := p.GetPromptByID(ctx, id)
    if err != nil {
        return "", err
    }

    // 2. Template parsen und ausführen
    t, err := template.New(id).Parse(prompt.Content)
    if err != nil {
        return "", fmt.Errorf("failed to parse template: %w", err)
    }

    var buf bytes.Buffer
    if err := t.Execute(&buf, tmplCtx); err != nil {
        return "", fmt.Errorf("failed to execute template: %w", err)
    }

    return buf.String(), nil
}

func (p *promptManager) GetStore() store.PromptStore {
    return p.store
}
```

---

## Phase 3: Config Erweiterung

```go
// pkg/config/service.go - NEUE Sub-Configs

// PromptStoreConfig defines configuration for the prompt store
type PromptStoreConfig struct {
    Type string `envconfig:"TYPE" default:"memory"`

    // File-Store Konfiguration
    FilePath    string `envconfig:"FILE_PATH" default:"./data/prompts"`
    CacheEnabled bool   `envconfig:"CACHE_ENABLED" default:"true"`

    // Langfuse Store Konfiguration
    LangfuseHost      string `envconfig:"LANGFUSE_HOST" default:"https://cloud.langfuse.com"`
    LangfusePublicKey string `envconfig:"LANGFUSE_PUBLIC_KEY"`
    LangfuseSecretKey string `envconfig:"LANGFUSE_SECRET_KEY"`
}

// PromptOptimizerConfig defines configuration for the prompt optimizer
type PromptOptimizerConfig struct {
    // Strategy
    DefaultStrategy string `envconfig:"DEFAULT_STRATEGY" default:"gradient"`

    // LLM Provider
    DefaultProvider string `envconfig:"DEFAULT_PROVIDER" default:"anthropic"`

    // Reflection Steps
    MaxReflectionSteps int `envconfig:"MAX_REFLECTION_STEPS" default:"5"`
    MinReflectionSteps int `envconfig:"MIN_REFLECTION_STEPS" default:"2"`
}

// GetPromptStoreConfig returns the prompt store configuration
func (s *serviceImpl) GetPromptStoreConfig() *PromptStoreConfig {
    return &s.PromptStore
}

// GetPromptOptimizerConfig returns the prompt optimizer configuration
func (s *serviceImpl) GetPromptOptimizerConfig() *PromptOptimizerConfig {
    return &s.PromptOptimizer
}
```

---

## Phase 4: Prompt Optimizer (mit Store-Integration)

### 4.1 Optimizer Interface & Factory

```go
// pkg/prompt/optimizer/optimizer.go

package optimizer

import (
    "context"
    "fmt"

    "github.com/denkhaus/gollum/pkg/config"
    "github.com/denkhaus/gollum/pkg/llm"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/denkhaus/gollum/pkg/prompt"
    "github.com/m-mizutani/gollem"
    "github.com/samber/do/v2"
)

// OptimizerStrategy definiert die Optimierungs-Strategie
type OptimizerStrategy string

const (
    StrategyGradient     OptimizerStrategy = "gradient"
    StrategyMetaPrompt   OptimizerStrategy = "metaprompt"
    StrategyPromptMemory OptimizerStrategy = "prompt_memory"
)

// PromptOptimizer defines the public interface
//
//revive:disable-next-line:exported
type PromptOptimizer interface {
    // Optimize verbessert einen Prompt basierend auf Trajektorien
    // Returns OptimizerResult mit Versions-Informationen
    // GAP 14/15 RESOLVED: Tool-Calling + Strukturiertes Ergebnis
    Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error)

    // GetStrategy returns die konfigurierte Strategie
    GetStrategy() OptimizerStrategy
}

type optimizerImpl struct {
    llmClient     gollem.LLMClient
    logService    logger.LoggerService
    strategy      OptimizerStrategy
    maxReflection int
    minReflection int
}

func NewOptimizer(injector do.Injector) (PromptOptimizer, error) {
    logService := do.MustInvoke[logger.LoggerService](injector)
    llmProvider := do.MustInvoke[llm.ClientProvider](injector)
    configService := do.MustInvoke[config.ConfigService](injector)

    cfg := configService.GetPromptOptimizerConfig()

    llmClient, err := llmProvider.GetClient(cfg.DefaultProvider)
    if err != nil {
        return nil, fmt.Errorf("failed to get LLM client: %w", err)
    }

    p := &optimizerImpl{
        llmClient:     llmClient,
        logService:    logService,
        strategy:      OptimizerStrategy(cfg.DefaultStrategy),
        maxReflection: cfg.MaxReflectionSteps,
        minReflection: cfg.MinReflectionSteps,
    }

    logService.Info("prompt optimizer initialized",
        logger.String("strategy", string(p.strategy)),
    )

    return p, nil
}

func (p *optimizerImpl) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
    // Strategie-basierte Optimierung
    switch p.strategy {
    case StrategyGradient:
        return p.runGradientOptimization(ctx, input)
    case StrategyMetaPrompt:
        return p.runMetaPromptOptimization(ctx, input)
    case StrategyPromptMemory:
        return p.runPromptMemoryOptimization(ctx, input)
    default:
        return "", fmt.Errorf("unknown strategy: %s", p.strategy)
    }
}

func (p *optimizerImpl) GetStrategy() OptimizerStrategy {
    return p.strategy
}
```

### 4.2 Gradient Strategie (Separate Datei)

```go
// pkg/prompt/optimizer/strategies/gradient.go

package strategies

// GradientPromptOptimizer implementiert die Gradient-Strategie mit Tool-Calling
//
// Ablauf:
// 1. Reflection Loop (min bis max Schritte)
//    - think(): Hypothesen generieren
//    - critique(): Vorherige Argumentation kritisieren
//    - recommend(): Entscheidung über Anpassung
// 2. Early Exit: Wenn step >= min UND warrants_adjustment=false → Abbruch
// 3. Wenn nach max Schritten immer noch warrants_adjustment=true → Fehler
// 4. Wenn warrants_adjustment=true: Update mit Hypothesen + Recommendations
// 5. Sonst: Original Prompt zurückgeben
//
// LLM Calls: 1-max (jeder Schritt = 1 LLM Call mit 0-3 Tool-Aufrufen)

func (s *gradientStrategy) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
    messages := []gollem.Message{
        gollem.NewSystemMessage(s.buildReflectionPrompt(input)),
    }

    tools := optimizer.RegisterTools()

    // Min-Max Reflection Loop
    for step := 0; step < s.maxReflection; step++ {
        // LLM generiert Response (kann Tools aufrufen)
        response, err := s.llmClient.Generate(ctx, messages,
            gollem.WithTools(tools),
        )

        // Tool-Calls ausführen
        for _, content := range response.Content {
            if tc, err := content.GetToolCallContent(); err == nil {
                toolResult := s.executeTool(ctx, tc)
                messages = append(messages,
                    gollem.NewToolResponseMessage(tc.ID, tc.Name, toolResult),
                )
            }
        }

        // Assistant Response hinzufügen
        messages = append(messages, response.Message)

        // Prüfen ob recommend() aufgerufen wurde
        if s.hasRecommendation(messages) {
            rec := s.getLastRecommendation(messages)

            // Early Exit: Min erreicht UND keine Anpassung nötig
            if step >= s.minReflection && !rec.WarrantsAdjustment {
                break  // Keine Änderung nötig
            }

            // recommend() beendet den Loop
            break
        }
    }

    // Final Recommendation prüfen
    finalRec := s.getLastRecommendation(messages)

    // Fehler: Max erreicht aber immer noch Anpassung nötig
    if finalRec.WarrantsAdjustment {
        return "", fmt.Errorf("optimization failed: max reflection steps (%d) reached but prompt still needs adjustment",
            s.maxReflection)
    }

    // Update Phase (nur wenn warranted)
    if finalRec.WarrantsAdjustment {
        return s.runUpdatePhase(ctx, input, finalRec)
    }

    // Keine Anpassung - Original zurückgeben
    return input.Prompt, nil
}
```

### 4.3 Meta-Prompt Strategie (Separate Datei)

```go
// pkg/prompt/optimizer/strategies/metaprompt.go

package strategies

// MetaPromptOptimizer kombiniert Reflection + Update
//
// Ablauf:
// 1. Kombinierter Prompt mit Analyseauftrag
// 2. Reflection Loop mit Tool-Calls
// 3. Direkter Update im finalen Schritt
//
// LLM Calls: 1-5 (jeder Schritt = 1 LLM Call)
```

### 4.4 Prompt Memory Strategie (Separate Datei)

```go
// pkg/prompt/optimizer/strategies/memory.go

package strategies

// PromptMemoryOptimizer - einfache Single-Shot Optimierung
//
// Ablauf:
// 1. Einziger LLM Call mit Trajektorien
// 2. Direkter Prompt-Update
//
// LLM Calls: 1 (schnellste Option)
```

### 4.5 Optimizer Prompt Templates

```go
// pkg/prompt/optimizer/templates.go

package optimizer

const (
    // DEFAULT_GRADIENT_PROMPT wird für Reflection-Phase verwendet
    DEFAULT_GRADIENT_PROMPT = `You are reviewing the performance of an AI assistant...
[See docs/prompt-optimizer/01-langmem-prompt-templates.md]`

    // DEFAULT_GRADIENT_METAPROMPT wird für Update-Phase verwendet
    DEFAULT_GRADIENT_METAPROMPT = `You are optimizing a prompt...
[See docs/prompt-optimizer/01-langmem-prompt-templates.md]`

    // DEFAULT_METAPROMPT für Meta-Prompt Strategie
    DEFAULT_METAPROMPT = `You are helping an AI assistant learn...
[See docs/prompt-optimizer/01-langmem-prompt-templates.md]`
)
```

---

## DI Container Erweiterung

```go
// pkg/di/container.go

func (p *containerImpl) RegisterServices(_ context.Context) do.Injector {
    // ... existing registrations ...

    do.Provide(p.injector, prompt.NewPromptManager)


    do.Provide(p.injector, store.NewPromptStoreProvider)

    // NEU: Register PromptOptimizer
    do.Provide(p.injector, optimizer.NewOptimizer)

    // Register application service (must be last, depends on all other services)
    do.Provide(p.injector, app.NewService)

    return p.injector
}
```

---

## Design-Entscheidungen

### Entscheidungs-Log

| Bereich | Entscheidung | Datum | Status |
|---------|--------------|-------|--------|
| **Package-Struktur** | `pkg/prompt/types.go` für Core Types | 2025-01-31 | ✅ |
| **Strategy Files** | Separate Dateien in `strategies/` | 2025-01-31 | ✅ |
| **File Locking** | `syscall.Flock` für cross-process safety (Unix-only) | 2025-01-31 | ✅ |
| **Bootstrap Pattern** | Lazy Init (Hybrid) - Embedded FS → Store bei Erstzugriff | 2025-01-31 | ✅ |
| **Trajectory Fields** | Messages, Outcome, ToolCalls (Metrics postponed) | 2025-01-31 | ✅ |
| **Alle Strategien** | Alle drei upfront implementieren | 2025-01-31 | ✅ |
| **Tool-Calling** | Gollem Native Tool-Calling (think/critique/recommend) | 2025-01-31 | ✅ |
| **Versioning** | SemVer mit Version-in-ID (`subagent@1.0.0`) | 2025-01-31 | ✅ |
| **Save()-Methode** | **Save() entfernt** - Nur `SaveNewVersion()` verwendet (alle Speichervorgänge versioniert) | 2025-01-31 | ✅ |
| **Version-Erhöhung** | Immer Patch-Increment (automatisch in SaveNewVersion) | 2025-01-31 | ✅ |
| **Race Conditions** | CAS (Compare-And-Swap) mit Retry-Mechanismus | 2025-01-31 | ✅ |
| **Shortcut-Auflösung** | ResolveAlias() löst `subagent` → `subagent@latest` auf | 2025-01-31 | ✅ |
| **Alias-Struktur** | Nur neueste Version hat Aliases, ältere haben keine | 2025-01-31 | ✅ |
| **Template Engine** | `text/template` (Standard Go Templates) | 2025-01-31 | ✅ |
| **PromptContext Name** | `PromptContext` beibehalten (klar dokumentiert) | 2025-01-31 | ✅ |
| **Subagent Prompt** | `"subagent"` Basis-ID mit Versioning | 2025-01-31 | ✅ |

---

## Implementation Details

### Template Rendering ✅
**Lösung:** Lazy Init Pattern
- Built-in Prompts sind **optimierbar**
- Default-Prompts in Embedded FS (`//go:embed`)
- Bei Erstzugriff: Embedded FS → Store laden
- Folgezugriffe: Aus Store (kann optimierte Version sein)
- `IsBuiltin` Flag schützt vor Löschung

```go
func (p *promptManager) GetPromptByID(ctx context.Context, id string) (*Prompt, error) {
    // Store zuerst prüfen
    prompt, err := p.store.Load(ctx, id)
    if err == nil && prompt != nil {
        return prompt, nil
    }

    // Nicht im Store - Lazy Init aus Embedded FS
    if p.isBuiltinID(id) {
        return p.loadAndRegisterBuiltin(ctx, id)
    }

    return nil, fmt.Errorf("prompt '%s' not found", id)
}
```

#### 2. Trajectory-Struktur Gap ✅
**Problem:** Trajectory-Typ referenziert aber nicht definiert.

**Lösung:** Core Fields definiert (siehe 1.2 oben)
- ✅ `Messages []Message` (Required)
- ✅ `Feedback interface{}` (nil, string, *Feedback, *EditFeedback)
- ✅ `ToolCalls []ToolCall` (in Message)
- ⏳ `Metrics` (postponed per user request)

#### 3. Package-Struktur Gap ✅
**Problem:** Inkonsistente Import-Pfade; unklare Type-Location.

**Lösung:** Zentralisiert in `pkg/prompt/types.go`
```go
// Import
import "github.com/denkhaus/gollum/pkg/prompt"

// Verwendung
p := &prompt.Prompt{...}
store := store.NewMemoryStore()
```

#### 4. File-Locking Gap ✅
**Problem:** Nur Cache-Locks; keine Dateisystem-Synchronisation.

**Lösung:** `syscall.Flock` für cross-process locking
```go
func (s *fileStore) SaveNewVersion(ctx context.Context, baseID string, content string, name string) (*prompt.Prompt, error) {
    path := s.getFilePath(newID)  // newID wird berechnet

    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    // Exclusive lock
    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
        return err
    }
    defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

    // Write...
}
```

#### 5. Strategy-Struktur Gap ✅
**Problem:** Alle Strategien in einer Datei vs. getrennt?

**Lösung:** Separate Dateien in `strategies/` Subdirectory
```
pkg/prompt/optimizer/strategies/
├── gradient.go      // GradientPromptOptimizer
├── metaprompt.go    // MetaPromptOptimizer
└── memory.go        // PromptMemoryOptimizer
```

**Vorteile:**
- Klare Trennung
- Einfachere Tests
- Bessere Übersicht

#### 6. Structured Output vs Tool-Calling Gap ✅ (NEU)
**Problem:** Der Plan erwähnt Tool-Calling (think/critique/recommend), aber Gollem hat **ResponseSchema** für strukturierte Outputs.

**Lösung:** Verwende `gollem.WithResponseSchema()` für die Optimizer-Reflection-Phase:

```go
// pkg/prompt/optimizer/types.go

// ReflectionResponse ist das Schema für die Analyse-Phase
type ReflectionResponse struct {
    Thought   string `json:"thought"`
    Criticism string `json:"criticism"`
}

// RecommendationResponse ist das Schema für die Entscheidungs-Phase
type RecommendationResponse struct {
    WarrantsAdjustment   bool   `json:"warrants_adjustment"`
    Hypotheses          string `json:"hypotheses,omitempty"`
    FullRecommendations string `json:"full_recommendations,omitempty"`
}

// Im Optimizer verwendet
func (p *gradientOptimizer) runReflectionStep(ctx context.Context, prompt string) (*ReflectionResponse, error) {
    schema, _ := gollem.ToSchema(&ReflectionResponse{})

    response, err := p.llmClient.Generate(ctx,
        gollem.NewMessage(gollem.RoleUser, prompt),
        gollem.WithResponseSchema(schema),
    )

    // Response parsen und zurückgeben
    var result ReflectionResponse
    if err := json.Unmarshal([]byte(response.Content[0].(*gollem.TextContent).Text), &result); err != nil {
        return nil, err
    }
    return &result, nil
}
```

**Vorteile:**
- Keine Tool-Execution nötig (Parse-only)
- Strukturierte Outputs ohne Tool-Middleware
- Konsistent mit Gollem's JSON Schema Support

#### 7. Versioning mit Aliases Gap ✅ (NEU)
**Problem:** Der Plan hat ein `Version` Feld, aber keine Möglichkeit, auf "aktuelle" Version zu referenzieren.

**Lösung:** Versions-Aliases im Prompt Store:

```go
// pkg/prompt/types.go - Erweitert
type Prompt struct {
    ID         string   // z.B. "subagent" oder "subagent@v2"
    Aliases    []string // ["subagent", "@latest"] für v2
    Version    int
    // ... restliche Felder
}

// pkg/prompt/store/store.go - Interface erweitert
type PromptStore interface {
    // ... existing methods

    // ResolveAlias löst "@latest" oder unversionierte IDs
    // Beispiel: "subagent" → "subagent@v2" (wenn v2 @latest alias hat)
    ResolveAlias(ctx context.Context, id string) (*Prompt, error)

    // ListVersions listet alle Versionen eines Prompts
    // Beispiel: ListVersions("subagent") → [subagent@v1, subagent@v2]
    ListVersions(ctx context.Context, baseID string) ([]*Prompt, error)
}
```

**Beispiel-Konfiguration:**
```
subagent@v1 → Version: 1, Aliases: []
subagent@v2 → Version: 2, Aliases: ["subagent", "@latest"]
```

Code referenziert immer `"subagent"`, Store löst zu `subagent@v2` auf.

#### 8. Lazy Init: Hybrid-Pattern ✅ (KLARSTELLUNG)
**Problem:** Der Plan registriert Built-in Prompts mit leerem Content beim Startup.

**Lösung:** Hybrides Lazy Init Pattern:

```go
// pkg/prompt/manager.go
func (p *promptManager) GetPromptByID(ctx context.Context, id string) (*Prompt, error) {
    // 1. Store prüfen (kann optimierte Version sein)
    prompt, err := p.store.Load(ctx, id)
    if err == nil && prompt != nil {
        return prompt, nil
    }

    // 2. Built-in Lazy Load aus Embedded FS
    if p.isBuiltinID(id) {
        defaultContent, err := p.loadEmbeddedTemplate(id)
        if err != nil {
            return nil, err
        }

        // Im Store speichern (für zukünftige Zugriffe)
        prompt := &Prompt{
            ID:        id,
            Name:      p.getBuiltinName(id),
            Content:   defaultContent,
            IsBuiltin: true,
            Version:   1,
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        }
        _ = p.store.Save(ctx, prompt)

        return prompt, nil
    }

    return nil, fmt.Errorf("prompt '%s' not found", id)
}
```

**Ablauf:**
1. Erstzugriff auf `"system"` → Embedded FS laden → als `system@v1` speichern
2. Optimizer erstellt `system@v2` → speichert optimierte Version
3. Nächster Zugriff auf `"system"` → `ResolveAlias("system")` → lädt `system@v2`

#### 9. Template Engine Entscheidung ✅ (NEU)
**Frage:** `text/template` vs `html/template`?

**Entscheidung:** Verwende `text/template`

**Rationale:**
- Standard Go Templates mit `{{.Variable}}` Syntax
- Kein Auto-Escaping nötig (keine HTML-Ausgabe)
- Bedingte Blöcke: `{{if .UpdateInstructions}}...{{end}}`
- Schleifen: `{{range .Trajectories}}...{{end}}`

```go
// pkg/prompt/optimizer/templates.go
const gradientPromptTemplate = `You are reviewing the performance of an AI assistant.

<current_prompt>
{{.Prompt}}
</current_prompt>

{{if .UpdateInstructions}}
Update instructions: {{.UpdateInstructions}}
{{end}}

<trajectories>
{{.Trajectories}}
</trajectories>
`

func (p *gradientOptimizer) renderTemplate(input *OptimizerInput) (string, error) {
    tmpl, err := template.New("gradient").Parse(gradientPromptTemplate)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, input); err != nil {
        return "", err
    }

    return buf.String(), nil
}
```

#### 10. PromptContext Naming ✅ (KLARSTELLUNG)
**Problem:** Inkonsistenz zwischen `PromptContext` und `PromptTemplateContext`

**Entscheidung:** Behalte `PromptContext`, aber dokumentiere klar:

```go
// PromptContext defines template variables for prompt rendering
//
// Note: Distinct from Go's context.Context - this is for template data only
// Use as templateCtx to avoid confusion in code
type PromptContext struct {
    Values   map[string]interface{} `json:"values,omitempty"`
    SubAgent *SubAgentContext        `json:"subagent,omitempty"`
    Agent    *AgentContext           `json:"agent,omitempty"`
}

// Manager method shows the distinction clearly
func (p *promptManager) RenderPrompt(
    ctx context.Context,      // Go stdlib context
    id string,
    tmplCtx *PromptContext,   // Template variables (not Go context!)
) (string, error)
```

### Import-Korrekturen

Original Plan hatte inkonsistente Import-Pfade:

```go
// FALSCH (ursprünglicher Plan)
import "github.com/denkhaus/gollum/pkg/prompt/store"
store := promptTypes.Prompt{...}  // Type aus falschem Package

// KORRIGIERT
import "github.com/denkhaus/gollum/pkg/prompt"
import "github.com/denkhaus/gollum/pkg/prompt/store"

p := &prompt.Prompt{...}
s := store.NewMemoryStore()
```

### Error Handling Patterns

```go
// Store Errors
var (
    ErrPromptNotFound     = errors.New("prompt not found")
    ErrPromptIsBuiltin    = errors.New("cannot modify built-in prompt")
    ErrInvalidPromptID    = errors.New("invalid prompt ID")
    ErrStoreUnavailable   = errors.New("store unavailable")
)

// Optimizer Errors
var (
    ErrNoTrajectories     = errors.New("no trajectories provided")
    ErrInvalidTrajectory  = errors.New("invalid trajectory format")
    ErrOptimizationFailed = errors.New("optimization failed")
)
```

---

## Implementierungs-Schritte

### Phase 1: Store Layer (Grundbaustein)
1. `pkg/prompt/types.go` - Core Types (Prompt, PromptContext, etc.)
2. `pkg/prompt/store/store.go` - PromptStore Interface
3. `pkg/prompt/store/memory_store.go` - In-Memory Implementierung
4. `pkg/prompt/store/file_store.go` - File-basierte Implementierung
5. `pkg/prompt/store/provider.go` - DI Provider
6. `pkg/prompt/store/store_test.go` - Tests
7. `pkg/config/service.go` - PromptStoreConfig
8. **[DELIVERABLE]** `/home/denkhaus/dev/kb/guides/guide.golang.prompt-store.md` erstellen

### Phase 2: Prompt Manager Erweiterung
1. `pkg/prompt/manager.go` - Interface erweitern
2. `pkg/prompt/manager_test.go` - Tests aktualisieren
3. `pkg/mocks/generate.go` - Mock neu generieren

### Phase 3: Prompt Optimizer
1. `pkg/prompt/optimizer/types.go` - Optimizer Types (Trajectory, etc.)
2. `pkg/prompt/optimizer/optimizer.go` - Optimizer Implementierung
3. `pkg/prompt/optimizer/strategies.go` - Strategies
4. `pkg/prompt/optimizer/optimizer_test.go` - Tests
5. **[DELIVERABLE]** `/home/denkhaus/dev/kb/guides/guide.golang.prompt-optimizer.md` erstellen
6. **[DELIVERABLE]** `/home/denkhaus/dev/kb/guides/guide.general.prompt-optimization.md` erstellen

### Phase 4: Config Integration
1. `pkg/config/service.go` - Alle Configs hinzufügen
2. DI Container aktualisieren
3. **[DELIVERABLE]** Final Review aller Knowledge Base Guidance-Files

### Phase 5: Dokumentation

#### 5.1 Knowledge Base zu aktualisieren

**WICHTIG**: Die Knowledge Base in `/home/denkhaus/dev/kb/` dient als zentrales Wissensrepository für allgemeingültige Muster und Guidance. Alle während der Implementation gewonnenen Erkenntnisse, die universell anwendbar sind, müssen dort dokumentiert werden.

**Zu erstellende Guidance-Files:**

1. **`guide.golang.prompt-store.md`** (NEU)
   - Repository Pattern für Prompt-Speicherung
   - Store-Interface Design (Load, Save, Delete, List, Exists)
   - Implementierungs-Muster für verschiedene Backends
   - Test-Patterns für Store-Implementierungen
   - DI-Integration mit Provider-Pattern

2. **`guide.golang.prompt-optimizer.md`** (NEU)
   - LLM-basierte Prompt-Optimierung in Go
   - Trajectory-Sammlung aus Konversationen
   - Optimierungs-Strategien (Gradient, Metaprompt, Prompt Memory)
   - Feedback-Schleifen und Iteration
   - Testing von LLM-basierten Komponenten mit mockgen
   - DI-Pattern für LLM-Abhängigkeiten

3. **`guide.general.prompt-optimization.md`** (NEU)
   - Konzepte der prozeduralen Memory (LangMEM)
   - Prompt-Optimierung als sprachunabhängiges Prinzip
   - Agent-Selbstverbesserung durch Feedback
   - Best Practices für Prompt-Tuning

**Wann erstellen?**
- Während/forEach Phase: Bei Entdeckung universeller Muster
- Nach jeder Phase: Zusammenfassung der Erkenntnisse
- Vor Abschluss: Vollständige Review aller Guidance-Files

**Validierung:**
- Alle Guidance-Files müssen sprachunabhängig sein (kein Go-spezifisches in `guide.general.*`)
- Beispiele müssen verallgemeinerbar sein
- Keine projekt-spezifischen Pfade oder Konfigurationen

#### 5.2 Projekt-Dokumentation

1. `CLAUDE.md` aktualisieren mit:
   - Prompt Optimizer Usage
   - Prompt Store Konfiguration
   - Beispiel für Optimierungs-Workflow

---

## Knowledge Base Governance

### Prinzipien
1. **Universalität vor Projektspezifik**: Nur allgemeingültige Muster in Knowledge Base
2. **Aktualität bei Entdeckung**: Guidance nicht erst am Ende, sondern sofort bei Erkenntnis
3. **Vermeidung von Redundanz**: Nicht doppelt dokumentieren (projekt-spezifisch in README, allgemein in KB)
4. **Beispiel-basiert**: Jede Guidance mit konkreten Code-Beispielen

### Decision Tree: Wo dokumentieren?

```
Ist das Wissen universell (über mehrere Projekte anwendbar)?
├── JA → /home/denkhaus/dev/kb/
│   ├── Ist es Go-spezifisch?
│   │   ├── JA → guide.golang.*.md
│   │   └── NEIN → guide.general.*.md
│   └── Ist es ein Workflow/Prozess?
│       └── workflow.*.md
└── NEIN → Projekt-Repository (README.md, docs/)
```

---

## Store Typen Entscheidungshilfe

| Store-Typ | Verwendung | Vorteile | Nachteile |
|-----------|------------|-----------|----------|
| **InMemory** | Development, Testing | Schnell, keine Dependencies | Geht bei Restart verloren |
| **File** | Kleine Produktionen, Single-Instance | Persistiert, einfach | Keine Concurrent-Sicherheit ohne Locks |
| **Langfuse** | Multi-Instance, Observability | Remote, Versionierung, Tracing | Externe Abhängigkeit, Kosten |
| **Database** | Große Produktionen, Multi-Writer | Skalierbar, ACID | Komplex, benötigt Migration |

---

## Deferred Ideas (für spätere Phasen)

- **Memory-Manager**: Extrahiert automatisch Memories aus Konversationen
- **Semantic Search**: Vektor-basierte Suche nach ähnlichen Trajektorien
- **Multi-Prompt Optimizer**: Optimiert mehrere Prompts gleichzeitig (mit Credit Assignment)
- **Background Reflection**: Asynchrone Optimierung im Hintergrund
- **Database Store**: PostgreSQL/MySQL Implementierung
- **Metrics Collection**: TrajectoryMetrics mit Latency, TokenCount, Cost
- **Prompt Rollback UI**: Web-Interface für Prompt-Versionenvergleich und Rollback

---

## LangMEM Referenz-Dokumentation

Alle LangMEM-Referenzen wurden extrahiert und in `docs/prompt-optimizer/` gespeichert:

| Datei | Inhalt |
|-------|--------|
| `README.md` | Übersicht, Roadmap, Quick Reference |
| `01-langmem-prompt-templates.md` | Alle Prompt-Templates (Gradient, Meta, Memory) |
| `02-langmem-optimizer-reference.md` | API-Referenz mit Code-Beispielen |
| `03-langmem-architecture.md` | System-Architektur und Design-Patterns |
| `04-trajectory-structure.md` | Trajectory-Datenstruktur-Definition |
| `type-mapping-gollem.md` | Type-Mapping zwischen Gollem und Prompt Optimizer |

**Original:** https://github.com/langchain-ai/langmem
