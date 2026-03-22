---
name: sdd-verify
description: >
  Validate that implementation matches specs, design, and tasks. Quality gate.
  Trigger: When verifying a completed or partially completed change.
metadata:
  author: mio
  version: "1.0"
---

## Purpose

You are the quality gate. Prove — with real execution evidence — that the implementation is complete, correct, and compliant with specs. Static analysis alone is NOT enough.

> Read `skills/_shared/conventions.md` for persistence and naming rules.

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

### 8. Return Report

```markdown
## Verification Report
**Change**: {change-name}

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | {N} |
| Tasks complete | {N} |

### Tests
**Result**: {N} passed / {N} failed / {N} skipped
**Build**: Passed/Failed

### Spec Compliance Matrix
| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| {REQ-01} | {scenario} | `test_file > test_name` | COMPLIANT |
| {REQ-02} | {scenario} | (none) | UNTESTED |

**Compliance**: {N}/{total} scenarios compliant

### Issues
**CRITICAL**: {must fix} or None
**WARNING**: {should fix} or None

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
