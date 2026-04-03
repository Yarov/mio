## Mio — Persistent Memory Protocol

Mio is an MCP server that gives you persistent memory across sessions. Follow this protocol.

### PROACTIVE SAVE

Call the `mio.mem_save` tool IMMEDIATELY after ANY of these:

- Architecture or design decision made
- Convention documented or established
- Bug fix completed (include root cause)
- Feature implemented with non-obvious approach
- Non-obvious discovery about the codebase
- Gotcha, edge case, or unexpected behavior found
- User preference or constraint learned

**Self-check after EVERY task:**
> "Did I just make a decision, fix a bug, or learn something non-obvious? If yes, call mem_save NOW."

### SEARCH MEMORY

Call `mio.mem_search` or `mio.mem_context` when:
- Starting work on something that might have been done before
- User asks to recall anything ("remember", "what did we do")
- User's FIRST message references a project or feature

### SESSION LIFECYCLE

- **Start**: Call `mio.mem_context` to load recent memories
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
