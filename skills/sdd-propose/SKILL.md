---
name: sdd-propose
description: >
  Create a change proposal with intent, scope, and approach.
  Trigger: When creating or updating a proposal for a change.
metadata:
  author: mio
  version: "2.0"
---

## Purpose

Take exploration analysis (or direct user input) and produce a structured proposal document.

## Persistence & Naming

All SDD artifacts use deterministic naming: `title` and `topic_key` = `sdd/{change-name}/{artifact-type}`, `type` = `architecture`, `project` = detected project name. `topic_key` enables upserts (save again → update, not duplicate).

**Two-step retrieval** (CRITICAL): `mcp__mio__mem_search` returns truncated previews. Always: (1) search → get ID, (2) `mcp__mio__mem_get_observation(id)` → full content. Never use search previews as source material.

## Sub-Agent Scope (CRITICAL)
You are a SUB-AGENT. Do NOT call these tools — they are top-level only:
- mem_session_start, mem_session_end, mem_session_summary
Use mem_save (once per task), mem_search, mem_context, mem_get_observation as needed.

## Steps

### 1. Load Context & Dependencies

```
mcp__mio__mem_search(query: "sdd/{change-name}/explore", project: "{project}") → get ID (optional)
→ if found: mcp__mio__mem_get_observation(id) → full exploration

mcp__mio__mem_search(query: "skill-registry", project: "{project}")
→ if found: load matching coding skills
```

### 2. Write Proposal

```markdown
# Proposal: {Change Title}

## Intent
{What problem are we solving? Why?}

## Scope

### In Scope
- {Concrete deliverable 1}
- {Concrete deliverable 2}

### Out of Scope
- {What we're NOT doing}

## Approach
{High-level technical approach}

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `path/to/area` | New/Modified/Removed | {What changes} |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| {Risk} | Low/Med/High | {How we mitigate} |

## Rollback Plan
{How to revert if something goes wrong}

## Success Criteria
- [ ] {How do we know this succeeded?}
```

### 3. Persist (MANDATORY)

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/proposal",
  topic_key: "sdd/{change-name}/proposal",
  type: "architecture",
  project: "{project}",
  content: "{proposal markdown}"
)
```

If you skip this, sdd-spec CANNOT find your proposal and the pipeline BREAKS.

### 4. Return

```markdown
**Status**: success
**Summary**: Proposal created for {change-name}. {N} deliverables, {risk level} risk.
**Artifacts**: sdd/{change-name}/proposal
**Next**: sdd-spec and sdd-design (can run in parallel)
**Risks**: {risks or "None"}
```

## Rules

- Every proposal MUST have a rollback plan
- Every proposal MUST have success criteria
- Use concrete file paths in Affected Areas
- Keep it CONCISE — thinking tool, not a novel
- Artifact budget: **400 words max**
