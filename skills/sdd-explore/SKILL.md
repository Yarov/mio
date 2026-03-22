---
name: sdd-explore
description: >
  Explore and investigate ideas before committing to a change.
  Trigger: When you need to think through a feature, investigate the codebase, or clarify requirements.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Investigate the codebase, think through problems, compare approaches, and return a structured analysis. Read-only — never modify code.

> Read `skills/_shared/conventions.md` for persistence and naming rules.

## Steps

### 1. Load Skill Registry

Check for available coding skills (see conventions.md). Load any matching your task.

### 2. Load Project Context (optional)

```
mcp__mio__mem_search(query: "sdd-init/{project}", project: "{project}")
→ if found: mcp__mio__mem_get_observation(id) → full context
```

### 3. Investigate the Codebase

Read relevant code to understand:
- Current architecture and patterns
- Files and modules that would be affected
- Existing behavior, tests, dependencies

### 4. Analyze Options

| Approach | Pros | Cons | Complexity |
|----------|------|------|------------|
| Option A | ... | ... | Low/Med/High |
| Option B | ... | ... | Low/Med/High |

### 5. Persist

If tied to a named change:
```
mcp__mio__mem_save(
  title: "sdd/{change-name}/explore",
  topic_key: "sdd/{change-name}/explore",
  type: "architecture",
  project: "{project}",
  content: "{exploration markdown}"
)
```

### 6. Return

```markdown
## Exploration: {topic}

### Current State
{How the system works today}

### Affected Areas
- `path/to/file.ext` — {why}

### Approaches
1. **{Name}** — Pros: ... | Cons: ... | Effort: Low/Med/High
2. **{Name}** — Pros: ... | Cons: ... | Effort: Low/Med/High

### Recommendation
{Recommended approach and why}

### Risks
- {Risk 1}
```

## Rules

- DO NOT modify any code or files
- ALWAYS read real code, never guess
- Keep analysis CONCISE
