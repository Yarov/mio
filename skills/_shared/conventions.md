# Mio SDD Conventions (shared reference)

> **NOTE**: As of v2.0, all critical conventions are inlined directly into each SKILL.md.
> This file exists as a reference only — sub-agents do NOT need to read it.

## Persistence Modes

| Mode | Read from | Write to | Project files |
|------|-----------|----------|---------------|
| `mio` (default) | Mio memory | Mio memory | Never |
| `filesystem` | `openspec/` directory | `openspec/` directory | Yes |
| `none` | N/A | N/A | Never (inline only) |

## Artifact Naming (Mio mode)

```
title:     sdd/{change-name}/{artifact-type}
topic_key: sdd/{change-name}/{artifact-type}
type:      architecture
project:   {detected project name}
```

## Artifact Types & Word Budgets

| Type | Produced By | Max Words |
|------|-------------|-----------|
| `explore` | sdd-explore | 500 |
| `proposal` | sdd-propose | 400 |
| `spec` | sdd-spec | 650 |
| `design` | sdd-design | 800 |
| `tasks` | sdd-tasks | 530 |
| `apply-progress` | sdd-apply | — |
| `verify-report` | sdd-verify | — |
| `archive-report` | sdd-archive | — |
| `state` | mio-architect | — |

## Two-Step Retrieval (CRITICAL)

`mem_search` (Mio MCP) returns truncated previews — in Claude Code the same tool appears as `mcp__mio__mem_search`. Always:
1. Search → get observation ID
2. `mem_get_observation(id)` (or `mcp__mio__mem_get_observation` in Claude Code) → full content

**Never use search previews as source material.**

## Return Envelope

Every phase MUST return:

```
**Status**: success | partial | blocked
**Summary**: 1-3 sentence summary
**Artifacts**: list of artifact keys/paths written
**Next**: next SDD phase to run
**Risks**: risks discovered or "None"
```

## Filesystem Mode (openspec/)

Only when explicitly requested:

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
