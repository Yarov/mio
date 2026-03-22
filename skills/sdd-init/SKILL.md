---
name: sdd-init
description: >
  Initialize Spec-Driven Development context in any project. Detects stack and bootstraps persistence.
  Trigger: When user wants to initialize SDD, says "sdd init", or starts a new structured development cycle.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Initialize the SDD context: detect project stack, conventions, and bootstrap the persistence backend.

> Read `skills/_shared/conventions.md` for persistence and naming rules.

## Steps

### 1. Detect Project Context

Read the project to understand:
- Tech stack (package.json, go.mod, pyproject.toml, etc.)
- Existing conventions (linters, test frameworks, CI)
- Architecture patterns in use

### 2. Build Skill Registry

Scan for available skills and project conventions. See `skills/skill-registry/SKILL.md` for the full scanning logic. Write `.atl/skill-registry.md` always.

### 3. Persist Project Context

**Mio mode** (default):
```
mcp__mio__mem_save(
  title: "sdd-init/{project-name}",
  topic_key: "sdd-init/{project-name}",
  type: "architecture",
  project: "{project-name}",
  content: "{detected project context}"
)
```

**Filesystem mode**: Create `openspec/config.yaml` with detected context:
```yaml
schema: spec-driven
context: |
  Tech stack: {detected}
  Architecture: {detected}
  Testing: {detected}
rules:
  specs:
    - Use Given/When/Then for scenarios
    - Use RFC 2119 keywords (MUST, SHALL, SHOULD, MAY)
  tasks:
    - Group by phase, hierarchical numbering
  apply:
    tdd: false
```

### 4. Return Summary

```markdown
## SDD Initialized

**Project**: {name}
**Stack**: {detected stack}
**Persistence**: mio | filesystem

### Next Steps
Ready for /sdd-explore or /sdd-propose.
```

## Rules

- NEVER create placeholder spec files
- ALWAYS detect the real tech stack, don't guess
- Keep context CONCISE — max 10 lines
