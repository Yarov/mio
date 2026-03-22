# Mio SDD Conventions (shared across all SDD skills)

## Persistence Modes

| Mode | Read from | Write to | Project files |
|------|-----------|----------|---------------|
| `mio` (default) | Mio memory | Mio memory | Never |
| `filesystem` | `openspec/` directory | `openspec/` directory | Yes |

Default: if Mio MCP tools are available → use `mio`. Otherwise → return results inline.

`filesystem` mode is only used when the user explicitly requests file-based persistence.

## Artifact Naming (Mio mode)

All SDD artifacts use deterministic naming:

```
title:     sdd/{change-name}/{artifact-type}
topic_key: sdd/{change-name}/{artifact-type}
type:      architecture
project:   {detected project name}
```

### Artifact Types

| Type | Produced By |
|------|-------------|
| `explore` | sdd-explore |
| `proposal` | sdd-propose |
| `spec` | sdd-spec |
| `design` | sdd-design |
| `tasks` | sdd-tasks |
| `apply-progress` | sdd-apply |
| `verify-report` | sdd-verify |
| `archive-report` | sdd-archive |

**Exception**: `sdd-init` uses `sdd-init/{project-name}` as topic_key.

## Two-Step Retrieval (CRITICAL)

`mcp__mio__mem_search` returns **truncated previews** (not full content). You MUST always:

```
Step 1: Search → get observation ID
  mcp__mio__mem_search(query: "sdd/{change-name}/{type}", project: "{project}")

Step 2: Get full content (REQUIRED)
  mcp__mio__mem_get_observation(id: {id from step 1})
```

**Never use search previews as source material.**

## Saving Artifacts

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/{artifact-type}",
  topic_key: "sdd/{change-name}/{artifact-type}",
  type: "architecture",
  project: "{project}",
  content: "{full markdown}"
)
```

`topic_key` enables upserts — saving again updates, not duplicates.

## Updating Artifacts

When you have the observation ID (e.g., marking tasks complete):

```
mcp__mio__mem_update(id: {observation-id}, title: "...", content: "{updated content}")
```

## Skill Registry Loading

Every SDD skill MUST check for available coding skills before starting work:

```
1. mcp__mio__mem_search(query: "skill-registry", project: "{project}")
   → if found: mcp__mio__mem_get_observation(id) → full registry
2. Fallback: read .atl/skill-registry.md from project root
3. If neither exists: proceed without skills (not an error)
4. Load skills matching your task (React code → react-19, tests → pytest, etc.)
```

## Filesystem Mode (openspec/)

Only when explicitly requested. Structure:

```
openspec/
├── config.yaml
├── specs/{domain}/spec.md
└── changes/{change-name}/
    ├── proposal.md
    ├── specs/{domain}/spec.md
    ├── design.md
    ├── tasks.md
    └── verify-report.md
```

## Return Envelope

Every phase MUST return:

```markdown
**Status**: success | partial | blocked
**Summary**: 1-3 sentence summary
**Artifacts**: list of artifact keys/paths written
**Next**: next SDD phase to run
**Risks**: risks discovered or "None"
```
