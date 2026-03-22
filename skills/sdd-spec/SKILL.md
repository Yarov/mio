---
name: sdd-spec
description: >
  Write specifications with requirements and Given/When/Then scenarios.
  Trigger: When writing or updating specs for a change.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

Take the proposal and produce delta specs — structured requirements and scenarios describing what's being ADDED, MODIFIED, or REMOVED.

> Read `skills/_shared/conventions.md` for persistence and naming rules.

## Steps

### 1. Load Skill Registry & Dependencies

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

### 5. Return Summary

```markdown
## Specs Created
**Change**: {change-name}

| Domain | Type | Requirements | Scenarios |
|--------|------|-------------|-----------|
| {domain} | Delta/New | {N added, M modified} | {total} |

**Next**: Ready for sdd-design.
```

## Rules

- ALWAYS use Given/When/Then for scenarios
- ALWAYS use RFC 2119 keywords: MUST, SHALL, SHOULD, MAY
- Every requirement MUST have at least ONE scenario
- Include happy path AND edge cases
- Specs describe WHAT, not HOW — no implementation details
