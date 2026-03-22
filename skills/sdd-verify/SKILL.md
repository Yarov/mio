---
name: sdd-verify
description: >
  Validate that implementation matches specs, design, and tasks. Quality gate.
  Trigger: When verifying a completed or partially completed change.
metadata:
  author: mio
  version: "2.0"
---

## Purpose

You are the quality gate. Prove — with real execution evidence — that the implementation is complete, correct, and compliant with specs. Static analysis alone is NOT enough.

## Persistence & Naming

All SDD artifacts use deterministic naming: `title` and `topic_key` = `sdd/{change-name}/{artifact-type}`, `type` = `architecture`, `project` = detected project name. `topic_key` enables upserts (save again → update, not duplicate).

**Two-step retrieval** (CRITICAL): `mcp__mio__mem_search` returns truncated previews. Always: (1) search → get ID, (2) `mcp__mio__mem_get_observation(id)` → full content. Never use search previews as source material.

## Steps

### 1. Load All Dependencies

Retrieve ALL (REQUIRED):
```
proposal      → mcp__mio__mem_search + mcp__mio__mem_get_observation
spec          → mcp__mio__mem_search + mcp__mio__mem_get_observation (CRITICAL for compliance)
design        → mcp__mio__mem_search + mcp__mio__mem_get_observation
tasks         → mcp__mio__mem_search + mcp__mio__mem_get_observation
```

### 2. Check Completeness

```
Count total tasks vs completed [x]
├── CRITICAL if core tasks incomplete
└── WARNING if cleanup tasks incomplete
```

### 3. Run Tests (Real Execution)

Detect and execute:
```
package.json → npm test
pyproject.toml → pytest
Makefile → make test
go.mod → go test ./...
```

Capture: total, passed, failed (with errors), skipped, exit code.

**CRITICAL if any test fails.**

### 4. Build & Type Check

```
npm run build / tsc --noEmit
go build ./... / go vet ./...
python -m build
make build
```

**CRITICAL if build fails.**

### 5. Spec Compliance Matrix (MOST IMPORTANT)

Cross-reference EVERY spec scenario against test results:

```
FOR EACH REQUIREMENT:
  FOR EACH SCENARIO:
  ├── Find tests covering this scenario
  ├── Check test result from Step 3
  └── Assign status:
      ├── COMPLIANT  → test exists AND passed
      ├── FAILING    → test exists BUT failed (CRITICAL)
      ├── UNTESTED   → no test found (CRITICAL)
      └── PARTIAL    → test covers part of scenario (WARNING)
```

A scenario is only COMPLIANT when a passing test proves the behavior.

### 6. Check Design Coherence

```
FOR EACH DECISION:
├── Was the chosen approach used?
├── Were rejected alternatives accidentally implemented?
└── Do file changes match the design table?
```

### 7. Persist (MANDATORY)

```
mcp__mio__mem_save(
  title: "sdd/{change-name}/verify-report",
  topic_key: "sdd/{change-name}/verify-report",
  type: "architecture",
  project: "{project}",
  content: "{verification report}"
)
```

### 8. Return

```markdown
**Status**: success | partial | blocked
**Summary**: Verification {PASS/FAIL}. {N}/{M} scenarios compliant.
**Artifacts**: sdd/{change-name}/verify-report
**Next**: sdd-archive (if PASS) | sdd-apply (if FAIL, fix issues first)
**Risks**: {critical issues or "None"}

### Spec Compliance Matrix
| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| {REQ-01} | {scenario} | `test_file > test_name` | COMPLIANT |

### Verdict
{PASS / PASS WITH WARNINGS / FAIL}
```

## Rules

- ALWAYS execute tests — static analysis is not verification
- A scenario is only COMPLIANT with a passing test
- Compare against SPECS first, DESIGN second
- Be objective — report what IS
- CRITICAL = must fix, WARNING = should fix, SUGGESTION = nice to have
- DO NOT fix issues — only report them
