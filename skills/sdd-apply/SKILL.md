---
name: sdd-apply
description: >
  Implement tasks from the change, writing actual code following specs and design.
  Trigger: When implementing one or more tasks from a change.
metadata:
  author: mio
  version: "2.0"
---

## Purpose

Receive specific tasks and implement them by writing actual code. Follow specs and design strictly.

## Persistence & Naming

All SDD artifacts use deterministic naming: `title` and `topic_key` = `sdd/{change-name}/{artifact-type}`, `type` = `architecture`, `project` = detected project name. `topic_key` enables upserts (save again → update, not duplicate).

**Two-step retrieval** (CRITICAL): `mcp__mio__mem_search` returns truncated previews. Always: (1) search → get ID, (2) `mcp__mio__mem_get_observation(id)` → full content. Never use search previews as source material. Skipping full retrieval produces incomplete implementations.

## Sub-Agent Scope (CRITICAL)
You are a SUB-AGENT. Do NOT call these tools — they are top-level only:
- mem_session_start, mem_session_end, mem_session_summary
Use mem_save (once per task), mem_search, mem_context, mem_get_observation as needed.

## Steps

### 1. Load All Dependencies

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

Change `- [ ]` to `- [x]` as you go.

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

### 6. Return

```markdown
**Status**: success | partial | blocked
**Summary**: {N}/{total} tasks complete for {change-name}.
**Artifacts**: sdd/{change-name}/tasks, sdd/{change-name}/apply-progress
**Next**: sdd-verify (if all done) | sdd-apply (next batch)
**Risks**: {deviations or blockers or "None"}
```

## Rules

- ALWAYS read specs before implementing — specs are acceptance criteria
- ALWAYS follow design decisions — don't freelance
- Match existing code patterns
- If design is wrong, NOTE IT — don't silently deviate
- If blocked, STOP and report back
- NEVER implement unassigned tasks
- In TDD mode, NEVER skip RED (the failing test)
