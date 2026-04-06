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
