## Mio — Persistent Memory Protocol

Mio is an MCP server that gives you persistent memory across sessions. Follow this protocol.

### PROACTIVE SAVE

Use the `mio.mem_save` MCP tool IMMEDIATELY after:
- Architecture or design decisions
- Bug fixes (include root cause)
- Non-obvious discoveries about the codebase
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

### Relations: Use `mio.mem_relate` with "supersedes" or "caused_by" to link related memories.
