---
name: pr-review
description: >
  Review GitHub PRs with structured analysis. Handles listing, analyzing, and reviewing.
  Trigger: When user wants to review PRs, asks about pending PRs/issues, or says "pr review".
metadata:
  author: mio
  version: "1.0"
---

## When to Use

- User mentions "pr review", "revisar PRs", "que hay pendiente"
- Analyze issues or contributions
- Audit PR/issue backlog

## Process

### 1. Gather Info

```bash
gh issue list --state open --limit 20
gh pr list --state open --limit 20
gh pr view {number} --json title,body,files,additions,deletions,author
gh pr diff {number} --patch
```

### 2. Load Project Skills

Check if the repo has coding skills. For each PR, load matching skills:

| Files Changed | Skills to Load |
|---|---|
| Python API code | django-drf, pytest |
| React/Next.js | react-19, nextjs-15, tailwind-4 |
| TypeScript | typescript, zod-4 |
| Tests | playwright, pytest |
| State management | zustand-5 |

Review against project conventions, not just generic quality.

### 3. Read Current Code

Before reviewing diffs, read the current code for context.

### 4. Evaluate Each PR

| Factor | Check |
|---|---|
| **Conventions** | Follows project skills? Structure, naming, patterns |
| **Quality** | Clean code, no duplication, error handling |
| **Tests** | Present? Follow project patterns? |
| **Breaking** | Breaks existing functionality? |
| **Commits** | Clean history, proper messages |

## Red Flags (DO NOT MERGE)

- Test/debug files committed
- Unused variables
- Hardcoded secrets
- Breaking changes without migration
- Security gaps

## Review Comment Style

Direct, concise. Lead with issues, numbered. No greetings, no fluff.

**Request changes:**
```
Two things:

1. **UpdateModelMixin exposes PUT** — you only need PATCH. Add `http_method_names = ["get", "patch"]`.

2. **Missing validation** on email input — add Zod schema before submit.

Everything else looks solid. Nice work on the service layer.
```

**Approve:**
```
Clean refactor, all specs covered, 28 tests. Ship it.
```

## Output Format

```markdown
## PR Analysis

### Ready to Merge
| PR | Author | Why |
|----|--------|-----|
| #XX | @user | {reason} |

### Needs Work
| PR | Author | What to fix |
|----|--------|-------------|
| #XX | @user | {issues} |

### Do Not Merge
| PR | Author | Critical problems |
|----|--------|-------------------|
| #XX | @user | {why} |
```

## Commands

```bash
gh pr review {N} --approve --body "..."
gh pr review {N} --request-changes --body "..."
gh pr merge {N} --squash
```
