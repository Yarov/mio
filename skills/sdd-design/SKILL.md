---
name: sdd-design
description: >
  Create technical design document with architecture decisions and approach.
  Trigger: When writing or updating the technical design for a change.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Take proposal and specs, then produce a design document capturing HOW the change will be implemented.

> Read `skills/_shared/conventions.md` for persistence and naming rules.

## Steps

### 1. Load Skill Registry & Dependencies

```
mcp__mio__mem_search(query: "sdd/{change-name}/proposal", project: "{project}") → get ID
mcp__mio__mem_get_observation(id: {id}) → full proposal (REQUIRED)

mcp__mio__mem_search(query: "sdd/{change-name}/spec", project: "{project}") → get ID
mcp__mio__mem_get_observation(id: {id}) → full spec (if exists)
```

### 2. Read the Codebase

Before designing, read actual code: entry points, module structure, existing patterns, dependencies.

### 3. Write Design

```markdown
# Design: {Change Title}

## Technical Approach
{Concise strategy. How does this map to the proposal?}

## Architecture Decisions

### Decision: {Title}
**Choice**: {what we chose}
**Alternatives**: {what we rejected}
**Rationale**: {why}

## Data Flow

    Component A ──→ Component B ──→ Component C
         │                              │
         └──────── Store ───────────────┘

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `path/to/file.ext` | Create | {what} |
| `path/to/existing.ext` | Modify | {what changes} |

## Interfaces / Contracts

{New interfaces, API contracts, type definitions in code blocks}

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | {what} | {how} |
| Integration | {what} | {how} |

## Open Questions
- [ ] {unresolved questions}
```

### 4. Persist (MANDATORY)

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/design",
  topic_key: "sdd/{change-name}/design",
  type: "architecture",
  project: "{project}",
  content: "{design markdown}"
)
```

### 5. Return Summary

```markdown
## Design Created
**Change**: {change-name}
- **Approach**: {one-line}
- **Decisions**: {N documented}
- **Files**: {N new, M modified, K deleted}
**Next**: Ready for sdd-tasks.
```

## Rules

- ALWAYS read actual codebase before designing
- Every decision MUST have a rationale
- Use the project's ACTUAL patterns, not generic best practices
- Include concrete file paths
- If open questions BLOCK the design, say so clearly
