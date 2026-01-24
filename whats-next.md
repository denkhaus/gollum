# GitButler Integration - Next Steps

**Session:** 2026-01-24
**Repository:** /home/denkhaus/dev/gomodules/gollum
**Branch:** gitbutler/workspace
**Current Session ID:** session-107276

---

## Executive Summary

Successfully integrated GitButler virtual branches with Forgejo workflow by creating a `gbtool` hook wrapper system. This enables multiple Claude sessions to work on different Forgejo issues simultaneously, with each session's changes automatically assigned to the correct virtual branch.

**Status:** Implementation complete, ready for testing

---

## What Was Completed

### 1. Created `gbtool` Go Module (`/home/denkhaus/dev/gomodules/gollum/gbtool/`)

#### Files Created:
- **`main.go`** - Hook wrapper that intercepts Claude hook events
  - Reads JSON input from Claude hooks
  - Looks up session → branch mapping
  - Ensures virtual branch exists
  - Forwards to GitButler: `but claude <hook-command>`

- **`internal/mapping/manager.go`** - Session mapping persistence
  - `SessionMapping` struct with session→issue mappings
  - `IssueInfo` tracking (status, active_session, branch, history)
  - Thread-safe operations with mutex
  - JSON storage in `~/.gitbutler/session-map.json`

- **`cmd/register/main.go`** - CLI tool for registering sessions
  - `register <issue-number>` - Creates mapping and virtual branch
  - `register status` - Shows current mappings
  - Reads `CLAUDE_SESSION_ID` environment variable

- **`README.md`** - Complete documentation with usage examples

#### Binaries Built:
```bash
✓ ~/bin/gbtool (3.3MB) - Hook wrapper
✓ ~/bin/register - Registration CLI (not yet built separately)
```

### 2. Updated Claude Hooks Configuration

**File:** `/home/denkhaus/dev/gomodules/gollum/.claude/settings.local.json`

```json
{
  "permissions": {
    "allow": [
      "Bash(register:*)",
      "Bash(but status:*)",
      "Bash(but stage:*)",
      "Bash(but branch list:*)"
    ]
  },
  "hooks": {
    "PreToolUse": [{
      "matcher": "Edit|MultiEdit|Write",
      "hooks": [{
        "command": "~/bin/gbtool pre-tool"
      }]
    }],
    "PostToolUse": [{
      "matcher": "Edit|MultiEdit|Write",
      "hooks": [{
        "command": "~/bin/gbtool post-tool"
      }]
    }],
    "Stop": [{
      "hooks": [{
        "command": "~/bin/gbtool stop"
      }]
    }]
  }
}
```

**Changed from:** `but claude pre-tool`
**Changed to:** `~/bin/gbtool pre-tool`

### 3. Current Test Session State

**Session Mapping File:** `~/.gitbutler/session-map.json`
```json
{
  "sessions": {
    "session-107276": "issue-456"
  },
  "issues": {
    "issue-456": {
      "status": "in-progress",
      "active_session": "session-107276",
      "branch": "issue-456",
      "history": ["session-107276"]
    }
  }
}
```

**GitButler Virtual Branches:**
- ✓ issue-456 (active, applied) - **Current test session**
- ✓ issue-123 (active, applied) - Previous test
- Multiple other issue branches (issue-26 through issue-40)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Multi-Session Forgejo Workflow with GitButler                  │
└─────────────────────────────────────────────────────────────────┘

Session A                    Session B                    Session C
(Forgejo Issue #123)         (Forgejo Issue #124)         (Forgejo Issue #125)
    │                            │                            │
    ├─ register 123              ├─ register 124              ├─ register 125
    │  → Creates mapping         │  → Creates mapping         │  → Creates mapping
    │  → Creates branch          │  → Creates branch          │  → Creates branch
    │                            │                            │
    ├─ Edits files               ├─ Edits files               ├─ Edits files
    │  ↓                         │  ↓                         │  ↓
    │  gbtool post-tool          │  gbtool post-tool          │  gbtool post-tool
    │  → Reads: session-A→123    │  → Reads: session-B→124    │  → Reads: session-C→125
    │  → Forwards to GitButler   │  → Forwards to GitButler   │  → Forwards to GitButler
    │  → Assigns to issue-123    │  → Assigns to issue-124    │  → Assigns to issue-125
    │                            │                            │
    ▼                            ▼                            ▼
Virtual Branch: issue-123    Virtual Branch: issue-124    Virtual Branch: issue-125

All changes isolated per issue, mergeable independently via PRs
```

---

## Key Design Decisions

### 1. Hook Wrapper Pattern
**Why:** GitButler hooks need to know which virtual branch to assign changes to, but they only receive `session_id` from Claude.

**Solution:** `gbtool` intercepts hook calls, looks up the session→branch mapping, then forwards to GitButler with the same input.

### 2. Session Registration Model
**Why:** Need a way to establish the session→branch mapping before any edits happen.

**Solution:** `register <issue-number>` command called at the start of each Forgejo workflow.

**Flow:**
1. Agent reads issue number from user request: "implement issue #123"
2. Agent calls: `register 123`
3. Tool creates mapping: `session-abc → issue-123`
4. Tool creates virtual branch: `issue-123`
5. All subsequent edits in this session go to `issue-123` branch

### 3. Mapping File Location
**Why:** Centralized location accessible to all sessions.

**Choice:** `~/.gitbutler/session-map.json`

**Benefits:**
- Single source of truth
- Survives session restarts
- Easy to inspect/debug
- Can be backed up

### 4. Branch Naming Convention
**Pattern:** `issue-{number}`

**Examples:**
- `issue-123` for Forgejo issue #123
- `issue-456` for Forgejo issue #456

**Benefits:**
- Predictable
- Easy to identify
- Maps 1:1 with Forgejo issues
- Can be automated in PR creation

---

## What's Next

### Priority 1: TEST THE HOOKS (User-Requested)

**Goal:** Verify end-to-end functionality of the gbtool hook wrapper system.

#### Test Scenario: Edit File and Verify Branch Assignment

```bash
# 1. Check current session
echo "Current session ID should be: session-107276"

# 2. Verify mapping exists
cat ~/.gitbutler/session-map.json
# Should show: session-107276 → issue-456

# 3. Check GitButler branches
but branch list
# Should see: issue-456 (active, applied)

# 4. Make a test edit
# (Use Edit tool to modify any file)

# 5. Verify the edit went to issue-456 branch
but status
# but diff should show changes in issue-456
```

**Expected Behavior:**
- ✓ Edit triggers: `gbtool post-tool`
- ✓ gbtool reads mapping: `session-107276 → issue-456`
- ✓ gbtool forwards to: `but claude post-tool`
- ✓ GitButler assigns changes to: `issue-456` virtual branch
- ✓ Changes visible in: `but status` under issue-456

**Verification Commands:**
```bash
# Check where changes were assigned
but status

# View diff in the virtual branch
but diff

# List commits in the virtual branch
but log
```

#### Test Scenario: Multi-Session Isolation

```bash
# Terminal A (Session A)
register 123
# Make edits to file.go

# Terminal B (Session B)
register 124
# Make edits to file.go

# Verify isolation
but branch list
# Should see separate issue-123 and issue-124 branches

# Check that changes didn't leak between branches
but status
```

**Expected Behavior:**
- ✓ Session A changes go to issue-123 only
- ✓ Session B changes go to issue-124 only
- ✓ No cross-contamination between branches

### Priority 2: Update Forgejo Agents

**Goal:** Integrate `register` call into Forgejo workflow agents.

#### Files to Update:
```
/home/denkhaus/dev/dendron/notes/
├── agent.forgejo_idea.md
├── agent.forgejo_implement.md
├── agent.forgejo_review.md
└── agent.forgejo_merge.md
```

#### Integration Pattern:

**At the start of each agent workflow:**

```go
// Parse issue number from user request
// Example: "implement issue #123" → extract "123"

// Register the session
cmd := exec.Command("register", issueNumber)
if err := cmd.Run(); err != nil {
    return fmt.Errorf("failed to register session: %w", err)
}

// Continue with normal workflow
```

**Implementation Location:**
- Add to agent initialization logic
- After issue number is known
- Before any file edits occur

**Benefits:**
- Automatic session registration
- No manual intervention needed
- Consistent across all Forgejo workflows

### Priority 3: Build and Install `register` Binary

**Current Status:** Only `gbtool` binary is built in `~/bin/`

**Action Needed:**
```bash
cd /home/denkhaus/dev/gomodules/gollum/gbtool
go build -o ~/bin/register ./cmd/register
chmod +x ~/bin/register
```

**Verify:**
```bash
which register
# Should output: /home/denkhaus/bin/register

register --help
# Should show usage or execute correctly
```

### Priority 4: Create GitButler Guidance Document

**File to Create:** `/home/denkhaus/dev/dendron/notes/guide.general.gitbutler.md`

**Content Structure:**
```markdown
# GitButler Virtual Branches

## Overview
GitButler enables multiple concurrent work streams through virtual branches...

## Session Management
- Use `register <issue-number>` at start of workflow
- Mapping stored in `~/.gitbutler/session-map.json`
- Each session gets isolated virtual branch

## Hook Integration
- Hooks configured in `.claude/settings.local.json`
- gbtool wrapper handles session→branch mapping
- Forwards to GitButler for actual change tracking

## Multi-Session Workflows
[Examples and patterns]

## Troubleshooting
[Common issues and solutions]
```

**Purpose:** Universal guidance for any project using GitButler with Claude Code.

---

## Technical Details

### Hook Data Flow

**Input to gbtool (from Claude):**
```json
{
  "session_id": "session-107276",
  "hook_event_name": "PostToolUse",
  "tool_name": "Edit",
  "file_path": "/path/to/file.go",
  "tool_input": {...}
}
```

**gbtool Processing:**
1. Parse JSON input
2. Extract `session_id`
3. Look up in `~/.gitbutler/session-map.json`
4. Get branch name: `issue-456`
5. Ensure virtual branch exists
6. Forward to GitButler

**GitButler Command:**
```bash
but claude post-tool < input.json
```

### Mapping File Structure

**Location:** `~/.gitbutler/session-map.json`

```json
{
  "sessions": {
    "session-107276": "issue-456",
    "session-abcdef": "issue-123",
    "session-123456": "issue-124"
  },
  "issues": {
    "issue-456": {
      "status": "in-progress",
      "active_session": "session-107276",
      "branch": "issue-456",
      "history": ["session-107276"]
    },
    "issue-123": {
      "status": "in-progress",
      "active_session": "session-abcdef",
      "branch": "issue-123",
      "history": ["session-abcdef", "session-old-123"]
    }
  }
}
```

### Virtual Branch Creation

**Command:** `but branch new issue-456`

**Result:** New virtual branch in GitButler workspace
- Isolated from other branches
- Can have independent commits
- Mergeable via PR when ready

---

## Known Limitations

1. **Manual Registration Required**
   - Agents must call `register` at workflow start
   - Not yet automated in Forgejo agents

2. **No Automatic Cleanup**
   - Old session mappings persist indefinitely
   - May need manual cleanup of stale entries

3. **Branch Name Collision**
   - If two sessions try to register same issue number
   - Second session will reuse existing branch (feature, not bug)

4. **No Conflict Resolution**
   - If sessions edit same file in different branches
   - GitButler handles this, but may need manual resolution

---

## Troubleshooting

### Issue: Changes Not Appearing in Virtual Branch

**Symptoms:**
- Edits made but `but status` shows no changes
- Changes go to default branch instead

**Diagnosis:**
```bash
# 1. Check session mapping
cat ~/.gitbutler/session-map.json

# 2. Verify session ID
echo $CLAUDE_SESSION_ID

# 3. Check if mapping exists
jq ".sessions[\"$CLAUDE_SESSION_ID\"]" ~/.gitbutler/session-map.json
```

**Solution:**
```bash
# Re-register the session
register 456
```

### Issue: gbtool Not Found

**Symptoms:**
- Hook fails with "gbtool: command not found"

**Solution:**
```bash
# Rebuild binary
cd /home/denkhaus/dev/gomodules/gollum/gbtool
go build -o ~/bin/gbtool .
chmod +x ~/bin/gbtool

# Verify
which gbtool
```

### Issue: GitButler Not Responding

**Symptoms:**
- Hook hangs or times out
- No output from `but` commands

**Diagnosis:**
```bash
# Test GitButler CLI
but status
but branch list
```

**Solution:**
- Restart GitButler app
- Check GitButler daemon status
- Verify Git repository is valid

---

## Success Criteria

### Phase 1: Testing (Current Priority)
- [ ] Hook wrapper correctly intercepts edit operations
- [ ] Session mapping is correctly read and used
- [ ] Changes appear in expected virtual branch
- [ ] `but status` shows changes in correct branch
- [ ] Multi-session isolation works (no cross-contamination)

### Phase 2: Agent Integration
- [ ] Forgejo agents call `register` at workflow start
- [ ] Issue number automatically extracted from user request
- [ ] Registration happens before any file edits
- [ ] Error handling for registration failures

### Phase 3: Documentation
- [ ] GitButler guidance document created
- [ ] Universal patterns documented
- [ ] Troubleshooting guide complete
- [ ] Examples for common workflows

---

## Files Modified

### Core Implementation
- `/home/denkhaus/dev/gomodules/gollum/gbtool/main.go` - Hook wrapper
- `/home/denkhaus/dev/gomodules/gollum/gbtool/internal/mapping/manager.go` - Mapping management
- `/home/denkhaus/dev/gomodules/gollum/gbtool/cmd/register/main.go` - Registration CLI
- `/home/denkhaus/dev/gomodules/gollum/gbtool/README.md` - Documentation

### Configuration
- `/home/denkhaus/dev/gomodules/gollum/.claude/settings.local.json` - Hook configuration

### State Files
- `~/.gitbutler/session-map.json` - Session mappings (created at runtime)

### Binaries
- `~/bin/gbtool` - Hook wrapper binary (3.3MB)
- `~/bin/register` - Registration CLI (needs to be built)

---

## Next Session Checklist

When resuming work:

1. **Verify Current State**
   - [ ] Check session mapping: `cat ~/.gitbutler/session-map.json`
   - [ ] Check GitButler branches: `but branch list`
   - [ ] Verify binaries exist: `ls -la ~/bin/gbtool ~/bin/register`

2. **Priority Testing**
   - [ ] Make test edit and verify branch assignment
   - [ ] Run `but status` to confirm changes in issue-456
   - [ ] Test multi-session isolation if needed

3. **Integration Work**
   - [ ] Build register binary: `go build -o ~/bin/register ./cmd/register`
   - [ ] Update Forgejo agents to call register
   - [ ] Create GitButler guidance document

4. **Validation**
   - [ ] End-to-end test with real Forgejo issue
   - [ ] Verify PR creation from virtual branch
   - [ ] Document any issues or edge cases

---

## Contact & Context

**Project:** Gollum - Multi-Agent LLM Framework
**Repository:** /home/denkhaus/dev/gomodules/gollum
**Issue:** N/A (Internal infrastructure work)
**Related:** GitButler integration, Forgejo workflow automation

**Key Concepts:**
- GitButler virtual branches exist in workspace, not as git refs
- Session ID → branch name mapping enables multi-session workflows
- Hook wrapper pattern transparently adds GitButler awareness
- Each Forgejo issue gets isolated virtual branch for independent work

**Learning Resources:**
- `/home/denkhaus/dev/gomodules/gollum/gbtool/README.md` - Complete usage guide
- GitButler docs: https://www.gitbutler.com/docs
- Claude Code hooks: https://docs.anthropic.com

---

*Generated: 2026-01-24*
*Session: session-107276*
*Branch: gitbutler/workspace*
