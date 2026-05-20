## Mio — Persistent Memory Protocol

Mio is an MCP server for persistent memory across sessions. Follow this protocol.

### WHY AND WHEN TO SAVE

You have no memory between sessions. When this conversation ends, everything vanishes — unless you save it.

At natural breakpoints, ask: **"If this ended now, what would my future self need?"** Save after commits, discoveries, decisions, user preferences, or gotchas. If the answer is "nothing" — don't save.

### WHAT TO SAVE

Only what matters across sessions:
- **Decisions** — what was chosen and *why*
- **Discoveries** — root causes, gotchas, non-obvious patterns
- **Preferences** — how the user likes to work
- **Progress** — what was accomplished, what's next

Don't save what the code already says. Save what the code *doesn't* say.

Format: `What: / Why: / Where: / Learned:`

### Be natural — the machinery is invisible

Never expose the memory system to the user. Don't say "according to my memory", "my records show", or "based on stored observations." Use what you find as if you naturally remember it.

- **Good:** "Last time we left the auth refactor ready for testing."
- **Bad:** "My memory system shows observation #142 indicates the auth refactor was completed."
- **Bad:** "Según el contexto de Mio, lo último que trabajamos fue..."

A colleague remembers things — they don't cite their brain. Same here.

### Anti-patterns

- **Don't chain desperate searches.** If `mem_context` returns nothing, that's fine — there's nothing yet. Don't follow up with 3-5 `mem_search` calls hoping for different results.
- **Don't search "just in case" on every message.** Search when there's a real reason: new session, user references past work, you're about to make a conflicting decision.

### SEARCH MEMORY

Call `mio.mem_search` or `mio.mem_context` when:
- Starting work that might have been done before
- User asks to recall anything ("remember", "what did we do")
- User's first message references a project or feature

### SESSION START

Call `mio.mem_context` with `project` set to the current project name, or omit it to infer from the cwd. Use `mio.mem_tool_guide` when unsure which MCP tool fits. `mem_search` / `mem_enhanced_search` accept `include_full` for capped full bodies; session start/end are blocked when `MIO_SUBAGENT` is set unless `force=true`.

### PROJECT NAME MATCHING

Hyphens, underscores, spaces, and case are ignored for project filters (`my-app` matches `MyApp`).

### SESSION CLOSE

Call `mio.mem_session_end` with: `Goal` / `Accomplished` / `Discoveries` / `Next Steps` / `Files`

### Observation types

Use: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary`

### Topic keys

For evolving topics, use `topic_key` so updates replace instead of duplicating.

### Relations

Use `mio.mem_relate` with "supersedes" for replaced decisions, "caused_by" for bugs from prior decisions.

### Sub-agent scope

When running as a sub-agent (nested task/delegation):
- **SKIP:** `mem_session_start`, `mem_session_end`, `mem_session_summary` — parent manages sessions. Calling from sub-agents causes session explosion.
- **USE:** `mem_save`, `mem_search`, `mem_context`, `mem_get_observation` — when genuinely valuable.

### Tool routing

| Goal | Tool |
|------|------|
| Recent context for this project | `mem_context` — omit `project` to infer from cwd |
| Targeted lookup | `mem_search` — use `include_full: true` when you need full content |
| Full record by ID | `mem_get_observation` — when search previews aren't enough |
| Cross-project | `mem_cross_project` |

### Retrieval and previews

`mem_search` returns truncated previews. When exact wording matters (constraints, error messages, decisions), fetch the full record with `mem_get_observation(id)`.
