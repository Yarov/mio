---
name: sdd-continue
description: >
  Detect current SDD state and execute the next phase automatically.
  Trigger: When user says "continue", "sdd-continue", "siguiente", "next", or wants to resume an SDD pipeline.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Automatically detect where an SDD pipeline left off and execute the next phase. The user doesn't need to know which phase is next — this skill figures it out.

## How It Works

### 1. Find Active Changes

Search Mio for all SDD artifacts:
```
mcp__mio__mem_search(query: "sdd/", project: "{project}", limit: 20)
```

Group results by change name. If multiple active changes exist, ask the user which one to continue.

### 2. Detect Current State

For the target change, check which artifacts exist:

```
Phase        | Artifact Key                    | Exists?
-------------|--------------------------------|--------
init         | sdd-init/{project}             | ?
explore      | sdd/{change}/explore           | ?
propose      | sdd/{change}/proposal          | ?
spec         | sdd/{change}/spec              | ?
design       | sdd/{change}/design            | ?
tasks        | sdd/{change}/tasks             | ?
apply        | sdd/{change}/apply-progress    | ?
verify       | sdd/{change}/verify-report     | ?
archive      | sdd/{change}/archive-report    | ?
```

### 3. Determine Next Phase

```
Decision tree:
├── No artifacts → "No active SDD pipeline found. Use /sdd-explore to start."
├── Has explore, no proposal → RUN sdd-propose
├── Has proposal, no spec AND no design → RUN sdd-spec + sdd-design (parallel)
├── Has proposal, has spec, no design → RUN sdd-design
├── Has proposal, no spec, has design → RUN sdd-spec
├── Has spec + design, no tasks → RUN sdd-tasks
├── Has tasks, no apply-progress → RUN sdd-apply (Phase 1)
├── Has apply-progress, tasks incomplete → RUN sdd-apply (next incomplete phase)
├── Has apply-progress, all tasks [x], no verify → RUN sdd-verify
├── Has verify PASS, no archive → RUN sdd-archive
├── Has verify FAIL → Show issues, ask user: "Fix and re-verify?"
├── Has archive → "SDD cycle complete for {change}."
```

### 4. Execute

Tell the user what phase is next and why:
```
"Change: {change-name}. Last completed: {phase}. Next: {next-phase}. Executing..."
```

Then delegate to the appropriate `/sdd-{phase}` skill.

### 5. After Execution

Report the result and ask:
```
"{phase} complete. [summary]. Continue to next phase?"
```

If user confirms, loop back to step 2.

## Handling Edge Cases

- **Multiple changes**: List them, ask user to pick
- **Blocked phase**: Show what's blocking, suggest resolution
- **Verify failed**: Show failing scenarios, offer to run sdd-apply for fixes
- **Context reset**: This skill recovers full state from Mio — that's its superpower

## Sub-Agent Scope (CRITICAL)
You are a SUB-AGENT. Do NOT call these tools — they are top-level only:
- mem_session_start, mem_session_end, mem_session_summary
Use mem_save (once per task), mem_search, mem_context, mem_get_observation as needed.

## Rules

- NEVER guess the state — always query Mio
- ALWAYS tell the user which phase you're running and why
- Respect the DAG: spec+design can be parallel, everything else is sequential
- If a phase fails, stop and report — don't silently continue
