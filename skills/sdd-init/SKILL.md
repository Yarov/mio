---
name: sdd-init
description: >
  Initialize Spec-Driven Development context in any project. Detects stack and bootstraps persistence.
  Trigger: When user wants to initialize SDD, says "sdd init", or starts a new structured development cycle.
metadata:
  author: mio
  version: "2.0"
---

## Purpose

Initialize the SDD context: detect project stack, conventions, and bootstrap the persistence backend.

## Persistence & Naming

All SDD artifacts use deterministic naming: `title` and `topic_key` = `sdd/{change-name}/{artifact-type}`, `type` = `architecture`, `project` = detected project name. `topic_key` enables upserts (save again → update, not duplicate). Exception: this skill uses `sdd-init/{project-name}` as topic_key.

**Two-step retrieval** (CRITICAL): `mcp__mio__mem_search` returns truncated previews. Always: (1) search → get ID, (2) `mcp__mio__mem_get_observation(id)` → full content. Never use search previews as source material.

## Steps

### 1. Detect Project Context

Read the project to understand:
- Tech stack (package.json, go.mod, pyproject.toml, etc.)
- Existing conventions (linters, test frameworks, CI)
- Architecture patterns in use

### 2. Build Skill Registry

Scan for available skills and project conventions:
1. `mcp__mio__mem_search(query: "skill-registry", project: "{project}")` → if found, get full content
2. Fallback: read `.atl/skill-registry.md` from project root
3. If neither exists: scan skill directories and write `.atl/skill-registry.md`

### 3. Persist Project Context

```
mcp__mio__mem_save(
  title: "sdd-init/{project-name}",
  topic_key: "sdd-init/{project-name}",
  type: "architecture",
  project: "{project-name}",
  content: "{detected project context — max 10 lines}"
)
```

### 4. Return

```markdown
**Status**: success
**Summary**: SDD initialized for {project}. Stack: {detected}. Persistence: mio.
**Artifacts**: sdd-init/{project-name}
**Next**: sdd-explore or sdd-propose
**Risks**: None
```

## Rules

- NEVER create placeholder spec files
- ALWAYS detect the real tech stack, don't guess
- Keep context CONCISE — max 10 lines
- Artifact budget: **200 words max**
