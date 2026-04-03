---
name: sdd-ff
description: >
  Fast-forward through SDD planning phases to reach implementation quickly.
  Trigger: When user says "sdd-ff", "fast-forward", "skip to implementation", or wants to speed through planning.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Fast-forward through SDD planning phases (explore → propose → spec → design → tasks) without pausing for user approval between each phase. Stops at sdd-apply so the user can review tasks before implementation begins.

## When to Use

- Medium-scope changes where the approach is clear
- User wants structure but doesn't want to approve every phase
- Resuming a pipeline that's partway through planning

## How It Works

### 1. Receive Change Description

```
/sdd-ff {change-description}
```

Generate a slug from the description:
- "add dark mode" → `add-dark-mode`
- "fix auth bug" → `fix-auth-bug`

### 2. Check Existing State

```
mcp__mio__mem_search(query: "sdd/{change-name}/", project: "{project}")
```

Determine which planning phases are already done.

### 3. Run Remaining Planning Phases

Execute phases sequentially without pausing, using the Agent tool for each with fresh context:

```
Phase order:
1. sdd-explore  (if no explore artifact)
2. sdd-propose  (if no proposal artifact)
3. sdd-spec + sdd-design  (parallel, if missing)
4. sdd-tasks    (if no tasks artifact)
```

After each phase:
- Verify the artifact was saved to Mio (search for it)
- Log a one-line status to the user
- Continue to the next phase immediately

### 4. Stop and Present Tasks

Once all planning phases are done, retrieve the tasks artifact and present it:

```markdown
## Fast-Forward Complete: {change-name}

### Pipeline Summary
| Phase | Status | Key Detail |
|-------|--------|------------|
| explore | done | {1-line from summary} |
| propose | done | {scope, risk level} |
| spec | done | {N requirements, M scenarios} |
| design | done | {N decisions, M file changes} |
| tasks | done | {N tasks in M phases} |

### Implementation Tasks
{Full tasks content from Mio}

### Ready to Implement
Run `/sdd-apply {change-name}` or `/sdd-continue` to start implementation.
```

### 5. Persist State

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/state",
  topic_key: "sdd/{change-name}/state",
  type: "architecture",
  project: "{project}",
  content: "Phase: tasks-complete. Ready for apply. Change: {change-name}."
)
```

## Sub-Agent Scope (CRITICAL)
You are a SUB-AGENT. Do NOT call these tools — they are top-level only:
- mem_session_start, mem_session_end, mem_session_summary
Use mem_save (once per task), mem_search, mem_context, mem_get_observation as needed.

## Rules

- ALWAYS delegate each phase to a sub-agent with fresh context
- NEVER skip to implementation — tasks must exist before apply
- If ANY phase blocks or fails, STOP and report to the user
- Log a one-line status per phase so the user sees progress
- spec + design run in parallel when both are missing
- Artifact budgets from each skill still apply (400w proposal, 650w spec, 800w design, 530w tasks)
