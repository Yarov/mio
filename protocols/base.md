## Mio — Persistent Memory Protocol

Mio is an MCP server that gives you persistent memory across sessions. Follow this protocol.

### WHY YOU SAVE

You have no memory between sessions. When this conversation ends, everything you learned vanishes — unless you save it. Your future self will start blind, ask the user things they already told you, repeat mistakes you already solved, and lose decisions that took effort to reach.

Saving is not a chore — it's how you stay useful across time.

### WHEN TO SAVE

Ask yourself at natural breakpoints: **"If this conversation ended right now, what would my future self wish it knew?"**

That question is your compass. Not a checklist. Think about it after:
- You committed code, finished a task, or resolved something
- The user told you something about how they work or what they want
- You discovered something non-obvious about the codebase
- A decision was made that shapes future work
- You hit a gotcha or edge case someone else would hit too

If the answer is "nothing" — don't save. If the answer is "the root cause of that bug" or "that the user prefers X over Y" or "we restructured the auth module" — save it now, not later. Later doesn't exist for you.

### WHAT TO SAVE

Not everything. Only what matters across sessions:
- **Decisions** — what was chosen and *why* (the why is what prevents revisiting it)
- **Discoveries** — root causes, gotchas, non-obvious patterns
- **Preferences** — how the user likes to work
- **Progress** — what was accomplished, what's next

Don't save what the code already says. Save what the code *doesn't* say.

### SEARCH MEMORY

Call `mio.mem_search` or `mio.mem_context` when:
- Starting work on something that might have been done before
- User asks to recall anything ("remember", "what did we do")
- User's FIRST message references a project or feature

### SESSION START

Call `mio.mem_context` with `project` set to the current project name at the beginning of every session. Always filter by project — context requests mean *this* project, not all projects.

### SESSION CLOSE

Call `mio.mem_session_end` with a summary:

```
Goal: [what we were working on]
Accomplished: [completed items]
Discoveries: [technical findings, gotchas]
Next Steps: [what remains]
Files: [key files modified]
```

### Memory format

```
What: [what was done]
Why: [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
```

### Observation types

Use: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary`

### Topic keys

For evolving topics, use `topic_key` so updates replace instead of duplicating.

### Relations

When a decision supersedes an old one, use `mio.mem_relate` with type "supersedes". For bugs caused by a prior decision, use "caused_by".
