---
name: technical-review
description: >
  Review technical exercises and candidate submissions with structured scoring.
  Trigger: When reviewing technical exercises, code assessments, or take-home tests.
metadata:
  author: mio
  version: "1.0"
---

## When to Use

- Reviewing take-home technical exercises
- Evaluating code submissions for hiring
- Assessing technical tests before interviews

## Process

1. **Explore structure** — understand project layout
2. **Read key files** — models, views, tests, README, docker-compose
3. **Check for tests** — presence/absence is major signal for senior roles
4. **Look for red flags** — security issues, leaked data, no error handling
5. **Score each factor 0-10** with specific evidence
6. **Output as table** per candidate

## Evaluation Factors

| Factor | What to Look For | Red Flags |
|---|---|---|
| **Styling** | Consistent formatting, naming, file organization | Mixed styles, messy structure |
| **Technical expertise** | Correct primitives, sensible architecture, tradeoff awareness | Cargo-cult patterns, security gaps |
| **Code Quality** | Maintainability, testability, small functions, error handling | Giant functions, no tests |
| **Go beyond asked** | Tests, validation, docs, UX improvements | Scope creep, unrelated features |
| **Explanations** | README quality, design rationale, setup instructions | No context, missing setup |
| **Other notes** | Security, operational concerns, collaboration signals | Leaked secrets, hardcoded credentials |

## Red Flags Checklist

- [ ] Secrets/API keys in code
- [ ] Employer data exposed (AWS accounts, internal URLs)
- [ ] No tests at all (critical for senior roles)
- [ ] Copy-pasted code without understanding
- [ ] Missing README
- [ ] Security gaps (SQL injection, no validation)
- [ ] Giant functions (>50 lines)

## Output Format

```markdown
# Review: {Candidate Name}

| Factor | Score (0-10) | Notes |
|--------|-------------|-------|
| Styling | {N} | {observations} |
| Technical expertise | {N} | {observations} |
| Code Quality | {N} | {observations} |
| Go beyond asked | {N} | {observations} |
| Explanations | {N} | {observations} |
| Other notes | — | {strengths / concerns} |
| **TOTAL** | **{sum}** | |
```

## Comparative Template

```markdown
## Comparative Summary

| Factor | Candidate A | Candidate B |
|--------|------------|------------|
| Styling | {score} | {score} |
| Technical | {score} | {score} |
| Quality | {score} | {score} |
| Beyond asked | {score} | {score} |
| Explanations | {score} | {score} |
| **TOTAL** | **{N}** | **{N}** |

**Recommendation**: {clear recommendation with reasoning}
```
