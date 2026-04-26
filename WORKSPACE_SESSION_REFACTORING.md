# Workspace/Session Refactoring Plan

## Key Insight: SessionID as UUID

The ACP traffic shows that SessionIDs are actually UUIDs:
- `646eadf1-c3b7-4655-acaa-1f9f825f9ee9`
- `64c1c394-c07a-4bb7-b259-dd4553b06870`

**Decision:** Change SessionID from `string` to `uuid.UUID` throughout the codebase for:
1. Consistency with ChannelID (already UUID)
2. Type safety and validation
3. No string conversions needed
4. Simpler comparisons

**Changes needed:**
- `pkg/channel/types.go:Message.SessionID` - Change from `string` to `uuid.UUID`
- `pkg/shared/acp.go:ACPSession.SessionID` - Change from `acp.SessionID` to `uuid.UUID`
- `pkg/acp/service.go` - Remove string conversions at ACP boundaries
- ACP protocol layer - Convert `acppkg.SessionID` (string) to `uuid.UUID` at entry/exit

## Problem Analysis

### Current Architecture (Centralized Workspace)

```
┌─────────────────────────────────────────────────────────────────┐
│                    GLOBAL Workspace Service                      │
│  currentPath: "/home/user/project"                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ├─── Agent 1 (uses global workspace)
                              ├─── Agent 2 (uses global workspace)
                              └─── Agent 3 (uses global workspace)
```

**Issues:**
1. All agents share the same workspace path
2. ACP multi-session support requires different `cwd` per session
3. No way to have agents work in different directories simultaneously

### Required Architecture (Session-Based Workspace)

```
┌─────────────────────────────────────────────────────────────────┐
│                         ACP Service                              │
│  (ChannelID: uuid-1)                                            │
└─────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
      Session 1            Session 2        Session 3
    (session-abc)        (session-def)    (session-ghi)
      cwd: /a/             cwd: /b/         cwd: /c/
         │                   │                │
         ▼                   ▼                ▼
    Agent 1              Agent 2          Agent 3
    (works in /a/)       (works in /b/)   (works in /c/)
```

## Required Changes

### 1. Prompt Rendering - Session-Aware WorkspaceContext

**File:** `pkg/prompt/manager/render.go`

**Change:** When rendering prompts for ACP sessions, use session-specific `cwd` instead of global workspace.

```go
// New function to get workspace context for a session
func (p *promptManager) GetWorkspaceContextForSession(sessionID string) *shared.WorkspaceContext {
    // Get session from ACP store
    if session, ok := p.acpStore.Get(sessionID); ok {
        return &shared.WorkspaceContext{
            CurrentPath: session.Cwd,
            SkillsXML:   p.skillsService.GetSkillsXML(),
            Skills:      p.skillsService.GetSkillInfos(),
        }
    }
    // Fallback to global workspace for non-ACP sessions
    return &shared.WorkspaceContext{
        CurrentPath: p.workspaceService.GetCurrentWorkspace(),
        SkillsXML:   p.skillsService.GetSkillsXML(),
        Skills:      p.skillsService.GetSkillInfos(),
    }
}
```

### 2. Channel Facade - Pass Session ID to Agent Creation

**File:** `pkg/channel/facade.go`

**Change:** Pass session ID through the chain so agents know their workspace.

```go
func (f *channelFacade) SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (*InputResult, error) {
    // ... existing code ...
    // sessionID is already passed - need to use it for workspace context
}
```

### 3. Supervisor Factory - Use Session Workspace

**File:** `pkg/agents/factory.go`

**Change:** Accept session ID and use session-specific workspace.

```go
func (p *defaultAgentFactory) CreateSupervisorAgentForSession(ctx context.Context, sessionID string, opts ...shared.SupervisorAgentOption) (shared.Agent, *shared.AgentConfig, error) {
    // Get workspace context for this specific session
    workspaceCtx := p.promptManager.GetWorkspaceContextForSession(sessionID)

    systemPrompt, err := p.promptManager.GetPromptWithContext(ctx,
        prompt.PromptIDSupervisorSystem,
        &prompt.RenderContext{
            Workspace: workspaceCtx,
        },
    )
    // ... rest of creation
}
```

### 4. Registry - Keep Global but Allow Session Overrides

**File:** `pkg/registry/registry.go`

**Change:** Keep global workspace context for TUI/non-ACP sessions, but ACP sessions use their own.

```go
// No major changes needed - registry manages agents, not workspaces
// Each agent will have its workspace context set at creation time
```

## Implementation Priority

1. **HIGH**: Add `GetWorkspaceContextForSession()` to PromptManager
2. **HIGH**: Update `CreateSupervisorAgent()` to accept optional session ID
3. **MEDIUM**: Update ChannelFacade to pass session ID through
4. **MEDIUM**: Update agent creation calls to use session workspace
5. **LOW**: Deprecate global Workspace Service for ACP sessions

## Migration Notes

- TUI continues using global Workspace Service (no changes)
- ACP sessions use session-specific `cwd` from `ACPSession.Cwd`
- Backward compatibility maintained through fallback to global workspace
