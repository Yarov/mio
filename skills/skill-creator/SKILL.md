---
name: skill-creator
description: >
  Create new AI agent skills following the standard format.
  Trigger: When user asks to create a new skill, add agent instructions, or document patterns.
metadata:
  author: mio
  version: "1.0"
---

## When to Create a Skill

**Create when:**
- A pattern is used repeatedly and AI needs guidance
- Project conventions differ from generic best practices
- Complex workflows need step-by-step instructions

**Don't create when:**
- Pattern is trivial or self-explanatory
- It's a one-off task

## Structure

```
skills/{skill-name}/
├── SKILL.md              # Required
├── assets/               # Optional - templates, schemas
└── references/           # Optional - links to local docs
```

## SKILL.md Template

```markdown
---
name: {skill-name}
description: >
  {One-line description}.
  Trigger: {When AI should load this skill}.
metadata:
  author: {author}
  version: "1.0"
---

## When to Use
{Bullet list of activation conditions}

## Critical Patterns
{Most important rules — what AI MUST know}

## Code Examples
{Minimum 3 concrete examples with good/bad patterns}

## Anti-Patterns
{What NOT to do}

## Commands
```bash
{Common commands}
```
```

## Naming

| Type | Pattern | Example |
|------|---------|---------|
| Technology | `{tech}` | `react-19`, `pytest` |
| Workflow | `{action}-{target}` | `skill-creator`, `pr-review` |
| SDD phase | `sdd-{phase}` | `sdd-explore`, `sdd-apply` |

## Checklist

- [ ] Name follows conventions (lowercase, hyphens)
- [ ] Frontmatter complete (name, description with Trigger, metadata)
- [ ] Critical patterns clear
- [ ] At least 3 code examples
- [ ] Anti-patterns included
- [ ] Commands section exists
