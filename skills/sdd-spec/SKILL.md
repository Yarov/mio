---
name: sdd-spec
description: >
  Write specifications with requirements and Given/When/Then scenarios.
  Trigger: When writing or updating specs for a change.
metadata:
  author: mio
  version: "2.0"
---

## Purpose

Take the proposal and produce delta specs — structured requirements and scenarios describing what's being ADDED, MODIFIED, or REMOVED.

## Persistence & Naming

All SDD artifacts use deterministic naming: `title` and `topic_key` = `sdd/{change-name}/{artifact-type}`, `type` = `architecture`, `project` = detected project name. `topic_key` enables upserts (save again → update, not duplicate).

**Two-step retrieval** (CRITICAL): `mcp__mio__mem_search` returns truncated previews. Always: (1) search → get ID, (2) `mcp__mio__mem_get_observation(id)` → full content. Never use search previews as source material.

## Steps

### 1. Load Dependencies

```
mcp__mio__mem_search(query: "sdd/{change-name}/proposal", project: "{project}") → get ID
mcp__mio__mem_get_observation(id: {id}) → full proposal (REQUIRED)
```

### 2. Identify Affected Domains

From the proposal's "Affected Areas", group changes by domain (auth/, payments/, ui/).

### 3. Write Delta Specs

#### For EXISTING domains (delta format):

```markdown
# Delta for {Domain}

## ADDED Requirements

### Requirement: {Name}
The system MUST {do something specific}.

#### Scenario: {Happy path}
- GIVEN {precondition}
- WHEN {action}
- THEN {expected outcome}

#### Scenario: {Edge case}
- GIVEN {precondition}
- WHEN {action}
- THEN {expected outcome}

## MODIFIED Requirements

### Requirement: {Name}
{New description} (Previously: {what it was})

## REMOVED Requirements

### Requirement: {Name}
(Reason: {why removed})
```

#### For NEW domains (full spec):

```markdown
# {Domain} Specification

## Purpose
{High-level description}

## Requirements

### Requirement: {Name}
The system MUST {behavior}.

#### Scenario: {Name}
- GIVEN {precondition}
- WHEN {action}
- THEN {outcome}
```

### 4. Persist (MANDATORY)

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/spec",
  topic_key: "sdd/{change-name}/spec",
  type: "architecture",
  project: "{project}",
  content: "{all specs concatenated}"
)
```

### 5. Return

```markdown
**Status**: success
**Summary**: Specs created for {change-name}. {N} requirements, {M} scenarios.
**Artifacts**: sdd/{change-name}/spec
**Next**: sdd-design (or sdd-tasks if design already exists)
**Risks**: {risks or "None"}
```

## Rules

- ALWAYS use Given/When/Then for scenarios
- ALWAYS use RFC 2119 keywords: MUST, SHALL, SHOULD, MAY
- Every requirement MUST have at least ONE scenario
- Include happy path AND edge cases
- Specs describe WHAT, not HOW — no implementation details
- Artifact budget: **650 words max**
