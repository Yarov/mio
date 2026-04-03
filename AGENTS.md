# Mio Agent Skills

Load the relevant skill(s) BEFORE writing any code. Follow ALL patterns from the loaded skill.

## Skill Discovery Table

| Skill | Trigger | Path |
|-------|---------|------|
| mio-architect | When user wants to build a feature, refactor, fix a complex bug, or says "architect", "sdd" | `skills/mio-architect/SKILL.md` |
| sdd-explore | When you need to think through a feature, investigate the codebase, or clarify requirements | `skills/sdd-explore/SKILL.md` |
| sdd-propose | When creating or updating a proposal for a change | `skills/sdd-propose/SKILL.md` |
| sdd-spec | When writing or updating specs for a change | `skills/sdd-spec/SKILL.md` |
| sdd-design | When writing or updating the technical design for a change | `skills/sdd-design/SKILL.md` |
| sdd-tasks | When creating the task breakdown for a change | `skills/sdd-tasks/SKILL.md` |
| sdd-apply | When implementing one or more tasks from a change | `skills/sdd-apply/SKILL.md` |
| sdd-verify | When verifying a completed or partially completed change | `skills/sdd-verify/SKILL.md` |
| sdd-archive | When archiving a change after implementation and verification | `skills/sdd-archive/SKILL.md` |
| sdd-continue | When user says "continue", "sdd-continue", "siguiente", "next", or wants to resume | `skills/sdd-continue/SKILL.md` |
| sdd-ff | When user says "sdd-ff", "fast-forward", or wants to speed through planning | `skills/sdd-ff/SKILL.md` |
| sdd-init | When initializing SDD for a new project | `skills/sdd-init/SKILL.md` |
| github-pr | When creating PRs, writing PR descriptions, or using gh CLI | `skills/github-pr/SKILL.md` |
| pr-review | When reviewing PRs, pending issues, or says "pr review" | `skills/pr-review/SKILL.md` |
| skill-creator | When creating a new skill or documenting agent patterns | `skills/skill-creator/SKILL.md` |
| skill-registry | When updating skills or managing the skill registry | `skills/skill-registry/SKILL.md` |
| technical-review | When reviewing technical exercises or code assessments | `skills/technical-review/SKILL.md` |
| react-19 | When writing React components | `skills/react-19/SKILL.md` |
| nextjs-15 | When working with Next.js - routing, Server Actions, data fetching | `skills/nextjs-15/SKILL.md` |
| typescript | When writing TypeScript code - types, interfaces, generics | `skills/typescript/SKILL.md` |
| tailwind-4 | When styling with Tailwind CSS | `skills/tailwind-4/SKILL.md` |
| zod-4 | When using Zod for validation | `skills/zod-4/SKILL.md` |
| zustand-5 | When managing React state with Zustand | `skills/zustand-5/SKILL.md` |
| ai-sdk-5 | When building AI chat features with Vercel AI SDK | `skills/ai-sdk-5/SKILL.md` |
| playwright | When writing E2E tests | `skills/playwright/SKILL.md` |
| pytest | When writing Python tests | `skills/pytest/SKILL.md` |
| django-drf | When building REST APIs with Django | `skills/django-drf/SKILL.md` |

## Memory Protocol

Mio memory is always active via MCP. See `protocols/claude-code.md` for the full protocol. Key tools:
- `mem_save` - Save decisions, discoveries, bugs (proactive, don't wait for user)
- `mem_search` / `mem_context` - Retrieve prior context before starting work
- `mem_get_observation` - Fetch full content by ID (search returns truncated previews)

## Sub-Agent Rules

Delegated agents (via Agent tool) MUST NOT call session lifecycle tools:
- `mem_session_start`, `mem_session_end`, `mem_session_summary` are TOP-LEVEL ONLY
- Sub-agents may use: `mem_save` (once per task), `mem_search`, `mem_context`, `mem_get_observation`
