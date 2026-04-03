---
name: sdd-archive
description: >
  Archive a completed change after verification. Merge delta specs and close the SDD cycle.
  Trigger: When archiving a change after implementation and verification.
metadata:
  author: mio
  version: "2.0"
---

## Purpose

Merge delta specs into main specs (source of truth), then archive the change. Complete the SDD cycle.

## Persistence & Naming

All SDD artifacts use deterministic naming: `title` and `topic_key` = `sdd/{change-name}/{artifact-type}`, `type` = `architecture`, `project` = detected project name. `topic_key` enables upserts (save again → update, not duplicate).

**Two-step retrieval** (CRITICAL): `mcp__mio__mem_search` returns truncated previews. Always: (1) search → get ID, (2) `mcp__mio__mem_get_observation(id)` → full content. Never use search previews as source material.

## Sub-Agent Scope (CRITICAL)
You are a SUB-AGENT. Do NOT call these tools — they are top-level only:
- mem_session_start, mem_session_end, mem_session_summary
Use mem_save (once per task), mem_search, mem_context, mem_get_observation as needed.

## Steps

### 1. Load All Dependencies

Retrieve ALL and record observation IDs for traceability:
```
proposal      → mcp__mio__mem_search + mcp__mio__mem_get_observation
spec          → mcp__mio__mem_search + mcp__mio__mem_get_observation
design        → mcp__mio__mem_search + mcp__mio__mem_get_observation
tasks         → mcp__mio__mem_search + mcp__mio__mem_get_observation
verify-report → mcp__mio__mem_search + mcp__mio__mem_get_observation
```

### 2. Check Verification

**NEVER archive a change with CRITICAL issues in its verify-report.** If the verdict is FAIL, stop and report back.

### 3. Sync Delta Specs (filesystem mode only)

If using filesystem persistence, merge deltas into main specs:
```
FOR EACH delta spec:
├── ADDED → append to main spec
├── MODIFIED → replace matching requirement
└── REMOVED → delete matching requirement
```

Preserve all requirements NOT mentioned in the delta.

In Mio mode: skip filesystem sync — artifacts live in Mio memory.

### 4. Persist Archive Report (MANDATORY)

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/archive-report",
  topic_key: "sdd/{change-name}/archive-report",
  type: "architecture",
  project: "{project}",
  content: "# Archive: {change-name}\n\n## Artifacts\n- proposal: obs#{id}\n- spec: obs#{id}\n- design: obs#{id}\n- tasks: obs#{id}\n- verify: obs#{id}\n\n## Specs Synced\n{domains and changes}\n\n## Completed: {date ISO}"
)
```

### 5. Return

```markdown
**Status**: success
**Summary**: Change {change-name} archived. SDD cycle complete.
**Artifacts**: sdd/{change-name}/archive-report
**Next**: Ready for next change
**Risks**: None

### Artifact Lineage
| Phase | Observation ID |
|-------|---------------|
| proposal | #{id} |
| spec | #{id} |
| design | #{id} |
| tasks | #{id} |
| verify | #{id} |
| archive | #{id} |
```

## Rules

- NEVER archive with CRITICAL verification issues
- The archive is an AUDIT TRAIL — never delete or modify archived changes
- Always include observation IDs for full traceability
- Use ISO date format (YYYY-MM-DD) for archive timestamps
