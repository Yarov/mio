---
name: sdd-apply
description: >
  Implement tasks from the change, writing actual code following specs and design.
  Trigger: When implementing one or more tasks from a change.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Receive specific tasks and implement them by writing actual code. Follow specs and design strictly.

> Read `skills/_shared/conventions.md` for persistence and naming rules.

## Steps

### 1. Load Skill Registry & All Dependencies

Retrieve ALL (REQUIRED):
```
proposal → mcp__mio__mem_search + mcp__mio__mem_get_observation
spec     → mcp__mio__mem_search + mcp__mio__mem_get_observation
design   → mcp__mio__mem_search + mcp__mio__mem_get_observation
tasks    → mcp__mio__mem_search + mcp__mio__mem_get_observation (keep ID for updates)
```

Load coding skills from registry matching the task (react-19, typescript, etc.)

### 2. Detect Implementation Mode

```
TDD detected?
├── config rules.apply.tdd = true
├── tdd skill installed
├── existing test patterns in codebase
└── Default: standard mode

TDD → Step 3a
Standard → Step 3b
```

### 3a. TDD Workflow (RED → GREEN → REFACTOR)

```
FOR EACH TASK:
├── 1. Read spec scenarios (acceptance criteria)
├── 2. RED — Write failing test
│   └── Run tests → confirm FAIL
├── 3. GREEN — Write minimum code to pass
│   └── Run tests → confirm PASS
├── 4. REFACTOR — Clean up, match conventions
│   └── Run tests → confirm STILL PASS
├── 5. Mark task [x] in tasks
└── 6. Note any deviations
```

### 3b. Standard Workflow

```
FOR EACH TASK:
├── Read spec scenarios + design decisions
├── Read existing code patterns
├── Write the code
├── Mark task [x]
└── Note any issues
```

### 4. Mark Tasks Complete

Change `- [ ]` to `- [x]` as you go:
```markdown
- [x] 1.1 Create `internal/auth/middleware.go`
- [x] 1.2 Add `AuthConfig` struct
- [ ] 1.3 Add auth routes  ← still pending
```

### 5. Persist Progress (MANDATORY)

Update tasks artifact:
```
mcp__mio__mem_update(id: {tasks-observation-id}, title: "sdd/{change-name}/tasks", content: "{updated tasks with [x]}")
```

Save progress report:
```
mcp__mio__mem_save(
  title: "sdd/{change-name}/apply-progress",
  topic_key: "sdd/{change-name}/apply-progress",
  type: "architecture",
  project: "{project}",
  content: "{progress report}"
)
```

### 6. Return Summary

```markdown
## Implementation Progress
**Change**: {change-name} | **Mode**: TDD/Standard

### Completed
- [x] {task descriptions}

### Files Changed
| File | Action | What |
|------|--------|------|
| `path/file` | Created | {brief} |

### Deviations from Design
{List or "None"}

### Status
{N}/{total} tasks complete. Ready for next batch / verify / blocked.
```

## Rules

- ALWAYS read specs before implementing — specs are acceptance criteria
- ALWAYS follow design decisions — don't freelance
- Match existing code patterns
- If design is wrong, NOTE IT — don't silently deviate
- If blocked, STOP and report back
- NEVER implement unassigned tasks
- In TDD mode, NEVER skip RED (the failing test)
