---
name: github-pr
description: >
  Create high-quality Pull Requests with conventional commits.
  Trigger: When creating PRs, writing PR descriptions, or using gh CLI.
metadata:
  author: mio
  version: "1.0"
---

## PR Title = Conventional Commit

```
<type>(<scope>): <short description>

feat     New feature
fix      Bug fix
docs     Documentation
refactor Code refactoring
test     Adding tests
chore    Maintenance
```

## PR Description Structure

```markdown
## Summary
- 1-3 bullet points explaining WHAT and WHY

## Changes
- List main changes

## Testing
- [ ] Tests added/updated
- [ ] Manual testing done

Closes #123
```

## Atomic Commits

```bash
# ✅ One thing per commit
git commit -m "feat(user): add User model"
git commit -m "feat(user): add UserService"
git commit -m "test(user): add UserService tests"

# ❌ Everything in one commit
git commit -m "add user feature"
```

## gh CLI Examples

```bash
# Basic PR
gh pr create \
  --title "feat(auth): add OAuth2 login" \
  --body "## Summary
- Add Google OAuth2 authentication

Closes #42"

# Complex description with HEREDOC
gh pr create --title "feat(dashboard): add analytics" --body "$(cat <<'EOF'
## Summary
- Add real-time analytics dashboard

## Changes
- Created AnalyticsProvider
- Added chart components

## Testing
- [x] Unit tests
- [x] Manual testing

Closes #123
EOF
)"

# Draft PR
gh pr create --draft --title "wip: refactor auth"

# With reviewers and labels
gh pr create \
  --title "feat(api): add rate limiting" \
  --reviewer "user1,user2" \
  --label "enhancement,api"
```

## Anti-Patterns

```bash
# ❌ Vague titles
gh pr create --title "fix bug"
gh pr create --title "update"

# ✅ Specific
gh pr create --title "fix(auth): prevent session timeout on idle"
```

## Quick Reference

| Task | Command |
|------|---------|
| Create PR | `gh pr create -t "type: desc" -b "body"` |
| Draft PR | `gh pr create --draft` |
| Add reviewer | `--reviewer user1,user2` |
| Link issue | `Closes #123` in body |
| Merge squash | `gh pr merge --squash` |
| View status | `gh pr status` |

## Keywords
github, pull request, pr, conventional commits, gh cli
