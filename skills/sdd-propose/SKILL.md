---
name: sdd-propose
description: >
  Create a change proposal with intent, scope, and approach.
  Trigger: When creating or updating a proposal for a change.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Take exploration analysis (or direct user input) and produce a structured proposal document.

> Read `skills/_shared/conventions.md` for persistence and naming rules.

## Steps

### 1. Load Skill Registry

Check for available coding skills. Load any matching your task.

### 2. Retrieve Dependencies

```
mcp__mio__mem_search(query: "sdd/{change-name}/explore", project: "{project}") → get ID (optional)
→ if found: mcp__mio__mem_get_observation(id) → full exploration
```

### 3. Write Proposal

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

### 4. Persist (MANDATORY)

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

### 5. Return Summary

```markdown
## Proposal Created
**Change**: {change-name}
- **Intent**: {one-line}
- **Scope**: {N deliverables}
- **Risk Level**: Low/Medium/High
**Next**: Ready for sdd-spec or sdd-design.
```

## Rules

- Every proposal MUST have a rollback plan
- Every proposal MUST have success criteria
- Use concrete file paths in Affected Areas
- Keep it CONCISE — thinking tool, not a novel
