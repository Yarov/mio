---
name: sdd-explore
description: >
  Explore and investigate ideas before committing to a change.
  Trigger: When you need to think through a feature, investigate the codebase, or clarify requirements.
metadata:
  author: mio
  version: "2.0"
---

## Purpose

Investigate the codebase, think through problems, compare approaches, and return a structured analysis. Read-only — never modify code.

## Persistence & Naming

All SDD artifacts use deterministic naming: `title` and `topic_key` = `sdd/{change-name}/{artifact-type}`, `type` = `architecture`, `project` = detected project name. `topic_key` enables upserts (save again → update, not duplicate).

**Two-step retrieval** (CRITICAL): `mcp__mio__mem_search` returns truncated previews. Always: (1) search → get ID, (2) `mcp__mio__mem_get_observation(id)` → full content. Never use search previews as source material.

## Steps

### 1. Load Context

```
mcp__mio__mem_search(query: "sdd-init/{project}", project: "{project}")
→ if found: mcp__mio__mem_get_observation(id) → full context

mcp__mio__mem_search(query: "skill-registry", project: "{project}")
→ if found: load matching coding skills
```

### 2. Investigate the Codebase

Read relevant code to understand:
- Current architecture and patterns
- Files and modules that would be affected
- Existing behavior, tests, dependencies

### 3. Analyze Options

| Approach | Pros | Cons | Complexity |
|----------|------|------|------------|
| Option A | ... | ... | Low/Med/High |
| Option B | ... | ... | Low/Med/High |

### 4. Persist

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/explore",
  topic_key: "sdd/{change-name}/explore",
  type: "architecture",
  project: "{project}",
  content: "{exploration markdown}"
)
```

### 5. Return

```markdown
**Status**: success | partial | blocked
**Summary**: {1-3 sentence summary}
**Artifacts**: sdd/{change-name}/explore
**Next**: sdd-propose
**Risks**: {risks or "None"}

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
```

## Rules

- DO NOT modify any code or files
- ALWAYS read real code, never guess
- Keep analysis CONCISE
- Artifact budget: **500 words max**
