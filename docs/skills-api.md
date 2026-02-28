# Skill System API Reference

Complete API reference for the Gollum skill system.

## Package: `skills`

```go
import "github.com/denkhaus/gollum/pkg/skills"
```

## Types

### Skill

Represents a parsed skill definition following ASOS v1.0 with Gollum extensions.

```go
type Skill struct {
    // Core ASOS v1.0 fields
    Name        string   // Required: Unique skill name
    Description string   // Optional: Brief description
    Version     string   // Optional: Semantic version

    // Tool configuration
    Tools      []string  // Allowed tools (empty = all)
    ToolScope  ToolScope // Tool access scope
    ToolFilter []string  // Tools to filter out

    // Type and classification
    Type SkillType // Skill type (agent, mcp, workflow)

    // Gollum-specific extensions
    Arguments     string // Argument specification for slash commands
    UserInvocable bool   // Can be invoked by user via slash command
    Priority      int    // Execution priority (higher = more important)
    AutoInvoke    bool   // Automatically invoke on matching context

    // Metadata
    Author   string
    Tags     []string
    Category string

    // Parsed content (not from YAML)
    FilePath    string // Path to the SKILL.md file
    Content     string // Raw content after frontmatter
    FullContent string // Complete file content including frontmatter
}
```

### SkillType

Skill type enumeration.

```go
type SkillType string

const (
    SkillTypeAgent    SkillType = "agent"    // Standard agent skill
    SkillTypeMcp      SkillType = "mcp"      // MCP server skill
    SkillTypeWorkflow SkillType = "workflow" // Workflow skill
)
```

#### Methods

```go
func (t SkillType) IsValid() bool
func (t SkillType) String() string
func ParseSkillType(s string) (SkillType, error)
```

### ToolScope

Tool access scope enumeration.

```go
type ToolScope string

const (
    ToolScopeAll      ToolScope = "all"       // Full access
    ToolScopeReadOnly ToolScope = "read-only" // Read-only access
    ToolScopeNone     ToolScope = "none"      // No access
    ToolScopeCustom   ToolScope = "custom"    // Custom restrictions
)
```

#### Methods

```go
func (s ToolScope) IsValid() bool
func (s ToolScope) String() string
func ParseToolScope(str string) (ToolScope, error)
```

### Skills

Collection of skills with utility methods.

```go
type Skills []*Skill
```

#### Methods

```go
func (s Skills) FindByName(name string) *Skill
func (s Skills) FilterByType(skillType SkillType) Skills
func (s Skills) FilterUserInvocable() Skills
func (s Skills) Names() []string
func (s Skills) String() string
```

### DiscoveryOptions

Configuration for skill discovery.

```go
type DiscoveryOptions struct {
    RootDir       string   // Starting directory (default: current)
    MaxDepth      int      // Directory depth limit (default: 10)
    IncludeHidden bool     // Include hidden directories
    IgnoredDirs   []string // Directories to skip
    Concurrency   int      // Worker count (default: 4)
}
```

### DiscoveryResult

Results from skill discovery.

```go
type DiscoveryResult struct {
    Skills       Skills  // Discovered skills
    Errors       []error // Errors encountered
    ScannedDirs  int     // Directories scanned
    ScannedFiles int     // Files checked
}
```

## SkillService Interface

Main interface for skill management.

```go
type SkillService interface {
    // Discover scans for skills in configured directories
    Discover(ctx context.Context) error

    // Get retrieves a skill by name (case-insensitive)
    Get(name string) (*Skill, error)

    // GetByPath retrieves a skill by file path
    GetByPath(path string) (*Skill, error)

    // List returns all discovered skills
    List() Skills

    // ListByType returns skills of a specific type
    ListByType(skillType SkillType) Skills

    // ListUserInvocable returns user-invocable skills
    ListUserInvocable() Skills

    // Validate validates a skill
    Validate(skill *Skill) error

    // Refresh rediscovers all skills
    Refresh(ctx context.Context) error

    // AddSearchPath adds a directory to search
    AddSearchPath(path string)

    // AddSearchPathAndDiscover adds path and triggers discovery
    AddSearchPathAndDiscover(ctx context.Context, path string) error

    // RemoveSearchPath removes a search path
    RemoveSearchPath(path string)

    // GetSearchPaths returns current search paths
    GetSearchPaths() []string
}
```

## Functions

### Parsing

```go
// Parse parses SKILL.md content into a Skill struct
func Parse(content, filePath string) (*Skill, error)

// ParseFile reads and parses a SKILL.md file from disk
func ParseFile(filePath string) (*Skill, error)
```

### Discovery

```go
// DiscoverInPath discovers skills in a specific path
func DiscoverInPath(ctx context.Context, path string, log *zap.Logger) (*DiscoveryResult, error)

// DiscoverMultiple discovers skills in multiple directories
func DiscoverMultiple(ctx context.Context, paths []string, log *zap.Logger) (*DiscoveryResult, error)
```

### Service Creation

```go
// NewService creates a new SkillService via DI
func NewService(injector do.Injector) (SkillService, error)
```

## Skill Methods

### Validation

```go
// Validate checks required fields and valid values
func (s *Skill) Validate() error
```

### Tool Access

```go
// HasTool checks if a tool is allowed
func (s *Skill) HasTool(toolName string) bool

// IsToolFiltered checks if a tool is filtered
func (s *Skill) IsToolFiltered(toolName string) bool
```

### User Invocation

```go
// IsUserInvocable checks slash command availability
func (s *Skill) IsUserInvocable() bool

// SlashCommand returns the slash command (e.g., "/skill-name")
func (s *Skill) SlashCommand() string
```

### Identity

```go
// DisplayName returns description or name
func (s *Skill) DisplayName() string

// ID returns lowercase unique identifier
func (s *Skill) ID() string

// Directory returns the skill's directory
func (s *Skill) Directory() string

// String returns string representation
func (s *Skill) String() string
```

### Serialization

```go
// ToMap converts skill to map for JSON serialization
func (s *Skill) ToMap() map[string]any
```

## Constants

```go
const (
    SkillFileName         = "SKILL.md"
    FrontmatterDelimiter  = "---"
    DefaultMaxDepth       = 10
    DefaultConcurrency    = 4
)

var DefaultIgnoredDirs = []string{
    ".git", ".svn", ".hg",
    "node_modules", "vendor",
    "__pycache__", ".cache",
    "dist", "build", "target",
    "bin", "tmp", "temp",
}
```

## Errors

```go
// Skill not found
func ErrSkillNotFound(name string) error

// Parse failure
func ErrParseFailed(filePath, reason string) error
func ErrParseFailedWithCause(filePath string, cause error) error

// Missing required field
func ErrMissingRequired(filePath, field string) error

// Invalid field value
func ErrInvalidValue(filePath, field, value string) error

// Empty field
func ErrEmptyField(filePath, field string) error

// Skill load failure
func ErrSkillLoadFailed(filePath string, err error) error

// Duplicate skill
func ErrDuplicateSkill(name, path1, path2 string) error
```

## InvokeSkillTool

Tool for executing skills at runtime.

### Tool Specification

```go
func (t *InvokeSkillTool) Spec() gollem.ToolSpec
```

Returns:
```go
gollem.ToolSpec{
    Name:        "invoke_skill",
    Description: "Executes a discovered skill by name...",
    Parameters: map[string]*gollem.Parameter{
        "name":         {Type: gollem.TypeString, Description: "..."},
        "input":        {Type: gollem.TypeString, Description: "..."},
        "context_mode": {Type: gollem.TypeString, Description: "..."},
        "model":        {Type: gollem.TypeString, Description: "..."},
    },
}
```

### Execution

```go
func (t *InvokeSkillTool) Run(ctx context.Context, args map[string]any) (map[string]any, error)
```

### Context Modes

```go
type ContextMode string

const (
    ContextModeInherited ContextMode = "inherited"
    ContextModeIsolated  ContextMode = "isolated"
)

func (m ContextMode) IsValid() bool
```

## Hook Events

```go
const (
    BeforeSkillInvoked hooks.HookType = "before_skill_invoked"
    AfterSkillInvoked  hooks.HookType = "after_skill_invoked"
    OnSkillError       hooks.HookType = "on_skill_error"
)
```

### Hook Context

```go
&hooks.HookContext{
    AgentID: senderID,
    Data: map[string]any{
        "skill_name":    string,
        "skill_type":    string,
        "context_mode":  string,
        "model":         string,
        "invoker_id":    string,
        "skill_path":    string,
        "skill_version": string,
    },
    ToolResult: map[string]any,  // AfterSkillInvoked only
    ToolError:  error,           // OnSkillError only
}
```

## Usage Examples

### Discovering Skills

```go
// Create service
svc := do.MustInvoke[skills.SkillService](injector)

// Discover skills
if err := svc.Discover(ctx); err != nil {
    log.Fatal(err)
}

// List all skills
for _, skill := range svc.List() {
    fmt.Printf("- %s: %s\n", skill.Name, skill.Description)
}
```

### Getting a Skill

```go
// Get by name (case-insensitive)
skill, err := svc.Get("Code-Reviewer")
if err != nil {
    // Skill not found
}

// Get by path
skill, err := svc.GetByPath("/path/to/skills/reviewer/SKILL.md")
```

### Parsing a Skill File

```go
// Parse from file
skill, err := skills.ParseFile("skills/my-skill/SKILL.md")
if err != nil {
    log.Fatal(err)
}

// Parse from content
content := `---
name: my-skill
---
# My Skill
Content here...`
skill, err := skills.Parse(content, "skills/my-skill/SKILL.md")
```

### Custom Discovery

```go
// Discover in specific paths
result, err := skills.DiscoverMultiple(ctx, []string{
    "/project/skills",
    "/shared/skills",
}, log)

fmt.Printf("Found %d skills\n", len(result.Skills))
for _, err := range result.Errors {
    fmt.Printf("Error: %v\n", err)
}
```

### Using with InvokeSkillTool

```go
// Via DI
provider := do.MustInvoke[tools.InvokeSkillToolProvider](injector)
tool := provider.CreateTool(senderID, agentFactory)

// Execute
result, err := tool.Run(ctx, map[string]any{
    "name":         "code-reviewer",
    "input":        "Review src/auth/",
    "context_mode": "isolated",
})
```
