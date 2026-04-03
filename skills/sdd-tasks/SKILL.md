---
name: sdd-tasks
description: >
  Break down a change into an implementation task checklist by phases.
  Trigger: When creating the task breakdown for a change.
metadata:
  author: mio
  version: "2.0"
---

## Purpose

Take proposal, specs, and design, then produce concrete, actionable implementation tasks organized by phase.

## Persistence & Naming

All SDD artifacts use deterministic naming: `title` and `topic_key` = `sdd/{change-name}/{artifact-type}`, `type` = `architecture`, `project` = detected project name. `topic_key` enables upserts (save again → update, not duplicate).

**Two-step retrieval** (CRITICAL): `mcp__mio__mem_search` returns truncated previews. Always: (1) search → get ID, (2) `mcp__mio__mem_get_observation(id)` → full content. Never use search previews as source material.

## Sub-Agent Scope (CRITICAL)
You are a SUB-AGENT. Do NOT call these tools — they are top-level only:
- mem_session_start, mem_session_end, mem_session_summary
Use mem_save (once per task), mem_search, mem_context, mem_get_observation as needed.

## Steps

### 1. Load Dependencies

Retrieve all three (REQUIRED):
```
proposal → mcp__mio__mem_search + mcp__mio__mem_get_observation
spec     → mcp__mio__mem_search + mcp__mio__mem_get_observation
design   → mcp__mio__mem_search + mcp__mio__mem_get_observation
```

### 2. Analyze & Write Tasks

```markdown
# Tasks: {Change Title}

## Phase 1: Foundation
- [ ] 1.1 {Create `path/to/file.ext` with X}
- [ ] 1.2 {Add Y struct to `path/to/file.ext`}

## Phase 2: Core Implementation
- [ ] 2.1 {Implement Z in `path/to/file.ext`}
- [ ] 2.2 {Connect A to B}

## Phase 3: Testing
- [ ] 3.1 {Write tests for scenario X}
- [ ] 3.2 {Write tests for edge case Y}

## Phase 4: Cleanup
- [ ] 4.1 {Update docs}
- [ ] 4.2 {Remove temporary code}
```

### Task Quality

| Criteria | Good | Bad |
|----------|------|-----|
| **Specific** | "Create `internal/auth/middleware.go` with JWT validation" | "Add auth" |
| **Actionable** | "Add `ValidateToken()` to `AuthService`" | "Handle tokens" |
| **Verifiable** | "Test: POST /login returns 401 without token" | "Make sure it works" |
| **Small** | One file or one logical unit | "Implement the feature" |

### Phase Order

```
Phase 1: Foundation — types, interfaces, config (things others depend on)
Phase 2: Core — main logic, business rules
Phase 3: Integration — connect components, routes, wiring
Phase 4: Testing — unit, integration, e2e (verify spec scenarios)
Phase 5: Cleanup — docs, remove dead code
```

### 3. Persist (MANDATORY)

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/tasks",
  topic_key: "sdd/{change-name}/tasks",
  type: "architecture",
  project: "{project}",
  content: "{tasks markdown}"
)
```

### 4. Return

```markdown
**Status**: success
**Summary**: {N} tasks in {M} phases for {change-name}.
**Artifacts**: sdd/{change-name}/tasks
**Next**: sdd-apply
**Risks**: {risks or "None"}
```

## Rules

- ALWAYS reference concrete file paths
- Tasks ordered by dependency — Phase 1 before Phase 2
- Testing tasks reference specific spec scenarios
- Each task completable in ONE session
- Use hierarchical numbering: 1.1, 1.2, 2.1
- If project uses TDD, integrate RED → GREEN → REFACTOR tasks
- Artifact budget: **530 words max**
