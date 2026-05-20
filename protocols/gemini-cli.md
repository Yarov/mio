## Mio — Persistent Memory Protocol

Mio is an MCP server that gives you persistent memory across sessions. Follow this protocol.

### PROACTIVE SAVE

Use the `mio.mem_save` MCP tool IMMEDIATELY after:
- Architecture or design decisions
- Convention documented or established
- Bug fixes (include root cause)
- Feature implemented with non-obvious approach
- Non-obvious discoveries about the codebase
- Gotcha, edge case, or unexpected behavior found
- User preferences or constraints learned

**Self-check:** "Did I just learn something non-obvious? Call mem_save."

### SEARCH MEMORY

Use `mio.mem_search` or `mio.mem_context` when:
- Starting work on something that might have been done before
- User asks to recall anything
- User's first message references a project or feature

### SESSION LIFECYCLE

- **Start**: Call `mio.mem_context` with `project` set to the current project name. Always filter by project — context requests mean *this* project, not all projects
- **End**: Call `mio.mem_session_end` with summary (Goal, Accomplished, Discoveries, Next Steps, Files)

### Memory format

```
What: [what was done]
Why: [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
```

### Types: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary`

### Topic keys: Use `topic_key` for evolving topics so updates replace instead of duplicating.

### Be natural — the machinery is invisible

Never expose the memory system to the user. Don't say "according to my memory", "my records show", or "based on stored observations." Use what you find as if you naturally remember it.

- **Good:** "Last time we left the auth refactor ready for testing."
- **Bad:** "My memory system shows observation #142 indicates the auth refactor was completed."
- **Bad:** "Según el contexto de Mio, lo último que trabajamos fue..."

A colleague remembers things — they don't cite their brain. Same here.

### Anti-patterns

- **Don't chain desperate searches.** If `mem_context` returns nothing, that's fine — there's nothing yet. Don't follow up with 3-5 `mem_search` calls hoping for different results.
- **Don't search "just in case" on every message.** Search when there's a real reason: new session, user references past work, you're about to make a conflicting decision.

### Relations: Use `mio.mem_relate` with "supersedes" or "caused_by" to link related memories.
