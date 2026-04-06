## Mio — Persistent Memory Protocol (ALWAYS ACTIVE)

Mio is an MCP server for persistent memory across sessions. In Cursor, enable **Mio** under **Settings → MCP**. These instructions are installed as `~/.cursor/rules/mio.md` and **`~/.cursor/rules/mio.mdc`** (`alwaysApply: true`, preferred for the rule engine). Prefer durable memory over hoping the chat will stick around. If global rules feel ignored (known Cursor quirk), paste this into **Settings → Rules for AI** or add the same `.mdc` under the project’s `.cursor/rules/`.

### How tools fit (no fixed playbook)

Mio exposes **capabilities**, not a mandatory script. **You decide** whether each call is justified: saving, searching, and session tools should follow from the task and risk, not from habit.

Use the tools registered under the **mio** MCP server in Cursor (`mem_save`, `mem_search`, `mem_context`, `mem_get_observation`, session helpers, `mem_relate`, `mem_suggest_topic_key`, analytics/search tools, etc.). In Composer/Agent, pick the **mio** server when needed — names match MCP registration.

When saving, pass **`agent`** = `cursor` if you include the field. `mio setup cursor` sets **`MIO_DEFAULT_AGENT=cursor`** in `~/.cursor/mcp.json` when the argument is omitted. It merges **`mio:*`** into **`~/.cursor/permissions.json`** (`mcpAllowlist`) so Mio tools are not interrupted on every call. Team onboarding: install `mio`, then **`mio setup cursor`** once per machine.

**Optional:** `MIO_SUBAGENT=1` blocks session start/end/summary unless `force=true`. **Need a compact map of tools?** Call **`mem_tool_guide`** — reference only, not a checklist to run through.

### Why “validate early” on a new session (and what that means)

A **new chat** is a blank model: it does **not** inherit this conversation. The durable bridge is **Mio + the repo**. If you skip memory when the user is clearly **continuing** work on a project (same product, feature, or open thread), the failure mode is real: wrong assumptions, reversed decisions, ignored preferences, duplicated debates.

**Important:** validating is not superstition — it is **risk management**. Ask:

- *Could a past session have recorded decisions, prefs, or facts that would change what I do now?*
- If **yes** or **maybe**, **bias toward a cheap check**: **`mem_context`** (recent observations for this repo; omit `project` to use cwd) and, if the topic is narrow, a **targeted** **`mem_search`** on the feature or problem. If **no** (one-off question, greenfield micro-task with no history), skipping is fine.

**Goal:** memory **up to date in your reasoning**, not “always 10 tool calls.” One informed pass beats ritual.

### Why saving matters when changes are *material*

“Important” changes are those a **future session cannot reliably reconstruct** from code alone: **decisions and tradeoffs**, **root causes**, **constraints the user stated**, **milestone / where we left off**, **gotchas**. If you land one of those and **don’t** save, the next session pays the cost — rework, contradictions, repeated questions to the user.

**Save when omission would plausibly hurt** the *next* agent, not when you touched a line of CSS. After a material change, connect the action to the reason (internally or briefly in the reply): *“Persisting X because it’s a stable contract / decision / handoff point.”*

### Tool routing (reference)

| Goal | Tool |
|------|------|
| Recent context for *this* repo | `mem_context` — omit `project` to use the cwd folder name; or `mem_surface` on the user’s prompt |
| Targeted lookup | `mem_search` — `include_full: true` when you need long bodies in one round-trip (capped) |
| Richer ranking | `mem_enhanced_search` — same `include_full` / project rules |
| Another codebase might matter | `mem_cross_project` |
| Text must be exact | **`mem_get_observation(id)`** after you have an id — treat search previews as hints, not sources of truth when precision matters |

Project filters **fold** spelling: `my-app`, `MyApp`, `my_app` match the same memories.

### How to think about memory

You have no continuity between sessions once the chat ends. **What is not saved is unavailable to the next session** — that is a planning constraint, not an order to spam saves. Before **`mem_save`**, sanity-check: *Would a future session need this **even if** the repo diff doesn’t spell it out?* If **no**, don’t save.

### What tends to be worth saving

Save when something is **non-obvious from the code** and **stable enough** to matter later: decisions and tradeoffs, root causes and gotchas, durable user preferences, real progress checkpoints (“where we left off”), surprises you would tell a teammate. **Do not** save paraphrases of the code, trivial trivia, or noise — that dilutes search.

Suggested shape: `What: / Why: / Where: / Learned:`

### When search and context *make sense* (tie to *why*)

Use memory reads when they **reduce real risk**, not by reflex:

| Situation | Why memory helps | Typical move |
|-----------|------------------|--------------|
| **New session**, ongoing project / feature | You may be missing decisions, prefs, or prior conclusions | `mem_context` first; add `mem_search` if the topic is specific |
| User references past work (“last time”, “we agreed”, “remember”) | Grounding prevents gaslighting the history | `mem_search` on keywords + **`mem_get_observation`** if previews aren’t enough |
| You’re about to lock a **decision** or **contract** | Prior art may already exist | Targeted search before you commit |
| Small, isolated task with no story | Low risk of hidden state | OK to skip reads |

**Anti-pattern:** opening every message with a search “just in case.” **Good pattern:** you can state (briefly) *why* you’re reading or skipping memory so the behavior stays legible.

### Sessions (optional discipline)

**`mem_session_start` / `mem_session_end` / `mem_session_summary`** are useful when you intentionally want session boundaries and summaries — e.g. long work units or explicit handoff. They are **not** required for every chat. When you do use them, summaries should be substantive, not filler.

Prefer **`mem_context`** (with `project` or cwd inference) when you only need **recent observations**, not a formal session record.

### Retrieval and previews

**`mem_search`** (and similar) often returns **truncated** text. When the exact wording matters — constraints, errors, legal-ish notes — **open `mem_get_observation(id)`** for the full record. When a capped inline body is enough, **`include_full: true`** can reduce round-trips.

### Observation types

Use: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary`

### Topic keys

For topics that evolve, use `topic_key` so updates replace instead of duplicating. **`mem_suggest_topic_key`** can propose a stable key.

### Relations

**`mem_relate`**: e.g. `supersedes` for replaced decisions, `caused_by` linking bugs to earlier choices — when the graph adds clarity, not for every save.

### Sub-agent / nested task scope

In a **nested agent** (e.g. Cursor sub-task), **do not** run **`mem_session_start`**, **`mem_session_end`**, or **`mem_session_summary`** — the parent session owns that (avoid session explosion). **`mem_save`**, **`mem_search`**, **`mem_context`**, **`mem_get_observation`**, etc. remain available when they genuinely help the sub-task.

### Mio Architect — SDD Pipeline (v2.0)

Engage when the user describes a **significant change** (new feature, refactor, complex bugfix) or says "architect" / "sdd" / "planea" / "diseña". **Do not** engage for trivial fixes, pure questions, or one-off commands unless the user asks.

**Pipeline:** explore → propose → spec + design (parallel) → tasks → apply → verify → archive. Delegate phases to sub-agents where appropriate; the orchestrator coordinates rather than replacing subordinate work.

**Shortcuts:**

- `/sdd-ff {desc}` — Fast-forward planning phases, stop before implementation
- `/sdd-continue` — Auto-detect state, execute next phase (works cross-session)
- `/mio-architect {desc}` — Full pipeline with approval gates

See **~/.cursor/skills/** for `mio-architect` and `sdd-*` skill files.
