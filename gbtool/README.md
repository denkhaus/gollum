# gbtool - GitButler Session Manager for Forgejo Workflows

Manages session-to-branch mappings for GitButler virtual branches in Forgejo issue workflows.

## Problem

GitButler's Claude hooks track changes by `session_id`, but don't know which virtual branch to assign them to. This tool bridges that gap.

## Solution

1. **Agent registers session** → `register <issue-number>` creates mapping
2. **Hook wrapper** → Reads mapping, ensures virtual branch exists, forwards to GitButler
3. **Multiple sessions** → Each can work on different issues simultaneously

## Structure

```
gbtool/
├── main.go              # Hook wrapper (installed as gbtool)
├── cmd/register/        # Registration CLI tool
├── internal/mapping/    # Mapping management
└── README.md
```

## Usage

### 1. Build and Install

```bash
cd gbtool
go build -o ~/bin/gbtool .
go build -o ~/bin/register ./cmd/register
```

### 2. Configure Claude Hooks

In `.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [{
      "command": "~/bin/gbtool pre-tool"
    }],
    "PostToolUse": [{
      "command": "~/bin/gbtool post-tool"
    }],
    "Stop": [{
      "command": "~/bin/gbtool stop"
    }]
  }
}
```

### 3. In Forgejo Agents

At the start of each issue workflow:

```go
// Register this session for issue-123
cmd := exec.Command("register", "123")
cmd.Run()
```

This creates the mapping:
```
~/.gitbutler/session-map.json
{
  "sessions": {
    "session-abc-123": "issue-123"
  },
  "issues": {
    "issue-123": {
      "status": "in-progress",
      "active_session": "session-abc-123",
      "branch": "issue-123"
    }
  }
}
```

### 4. Check Status

```bash
register status
```

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│  Forgejo Agent                                                  │
│                                                                 │
│  1. register 123                                               │
│     → Creates mapping: session-abc → issue-123                 │
│     → Creates virtual branch: issue-123                        │
│                                                                 │
│  2. Agent edits files                                          │
│     ↓                                                           │
│  3. Claude Hook: gbtool post-tool                              │
│     → Reads mapping: session-abc → issue-123                   │
│     → Ensures virtual branch exists                            │
│     → Forwards to: but claude post-tool                        │
│     → GitButler assigns changes to issue-123 branch            │
└─────────────────────────────────────────────────────────────────┘
```

## Multi-Session Support

Multiple agents can work on different issues simultaneously:

```
Session A → register 123 → issue-123 branch
Session B → register 124 → issue-124 branch
Session C → register 125 → issue-125 branch
```

Each session's changes are automatically assigned to the correct virtual branch.

## Resume Work Later

If you stop and come back later with a new session:

```
Session A → register 123 → issue-123 branch (reused!)
```

The tool recognizes the existing branch and continues work there.
