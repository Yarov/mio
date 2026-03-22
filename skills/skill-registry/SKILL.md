---
name: skill-registry
description: >
  Create or update the skill registry for the current project.
  Trigger: When user says "update skills", "skill registry", or after installing/removing skills.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Generate a catalog of all available skills that the orchestrator reads once per session to resolve skill paths for sub-agents.

## Steps

### 1. Scan User Skills

Glob for `*/SKILL.md` in all known skill directories:

**User-level:**
- `~/.claude/skills/`
- `~/.config/opencode/skills/`
- `~/.gemini/skills/`

**Project-level:**
- `{project}/.claude/skills/`
- `{project}/skills/`

**Skip**: `sdd-*`, `_shared`, `skill-registry`

For each skill, read frontmatter (first 10 lines) to extract name and trigger.

### 2. Scan Project Conventions

Check project root for:
- `AGENTS.md` / `agents.md` — read and extract all referenced paths
- `CLAUDE.md` (project-level only)
- `.cursorrules`

### 3. Write Registry

Create `.atl/skill-registry.md`:

```markdown
# Skill Registry

## User Skills

| Trigger | Skill | Path |
|---------|-------|------|
| {trigger} | {name} | {full path} |

## Project Conventions

| File | Path | Notes |
|------|------|-------|
| {file} | {path} | {notes} |
```

### 4. Persist

**Always** write `.atl/skill-registry.md`.

**If Mio available**, also save:
```
mcp__mio__mem_save(
  title: "skill-registry",
  topic_key: "skill-registry",
  type: "config",
  project: "{project}",
  content: "{registry markdown}"
)
```

## Rules

- ALWAYS write `.atl/skill-registry.md` regardless of persistence mode
- SKIP `sdd-*`, `_shared`, `skill-registry` when scanning
- Only read frontmatter (first 10 lines)
- If no skills found, write empty registry
- Add `.atl/` to `.gitignore` if not already listed
