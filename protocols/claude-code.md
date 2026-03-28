## Mio — Persistent Memory Protocol (ALWAYS ACTIVE)

Mio is an MCP server for persistent memory across sessions. This protocol is MANDATORY.

### PROACTIVE SAVE — do NOT wait for the user to ask

Call `mcp__mio__mem_save` IMMEDIATELY after ANY of these:

**After decisions or conventions:**
- Architecture or design decision made
- Convention documented or established
- Tool or library choice made with tradeoffs
- Workflow change agreed upon

**After completing work:**
- Bug fix completed (include root cause)
- Feature implemented with non-obvious approach
- Configuration change or environment setup done

**After discoveries:**
- Non-obvious discovery about the codebase
- Gotcha, edge case, or unexpected behavior found
- Pattern established (naming, structure, convention)
- User preference or constraint learned

**Self-check after EVERY task:**
> "Did I just make a decision, fix a bug, learn something non-obvious, or establish a convention? If yes, call mem_save NOW."

### SEARCH MEMORY — check before starting work

Call `mcp__mio__mem_search` or `mcp__mio__mem_context` when:
- User's FIRST message references a project or feature — search for prior work before responding
- Starting work on something that might have been done before
- User asks to recall anything ("remember", "what did we do", "acordate", "que hicimos")
- User mentions a topic you have no context on

### SESSION START

At the beginning of every session, call `mcp__mio__mem_context` to load recent memories and recover context from prior sessions.

### SESSION CLOSE — before saying "done" / "listo"

Call `mcp__mio__mem_session_end` with a summary structured as:

```
Goal: [what we were working on]
Accomplished: [completed items with key details]
Discoveries: [technical findings, gotchas, non-obvious learnings]
Next Steps: [what remains to be done]
Files: [key files modified]
```

This is NOT optional. If you skip this, the next session starts blind.

### Memory format

Structure content for mem_save as:
```
What: [what was done]
Why: [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
```

### Observation types

Use the correct type: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary`

### Topic keys

For evolving topics, use `topic_key` so updates replace instead of duplicating. Use `mcp__mio__mem_suggest_topic_key` to generate one.

### Relations

When a new decision supersedes an old one, use `mcp__mio__mem_relate` with type "supersedes". When fixing a bug caused by a prior decision, use "caused_by".

### Mio Architect — Automatic SDD Pipeline (v2.0)

When the user describes a **significant change** (new feature, refactor, complex bugfix), activate the `mio-architect` skill automatically:

**Activate when:**
- User describes a new feature: "quiero agregar...", "add...", "necesito..."
- User requests a refactor: "refactoriza...", "mejora...", "cambia..."
- User has a complex bug touching multiple files
- User explicitly says: "architect", "sdd", "planea", "diseña"

**Do NOT activate for:**
- Simple fixes (typos, one-line changes, obvious bugs)
- Questions about code
- Running commands

**Pipeline:** The architect assesses scope (small/medium/large) and drives the SDD pipeline. Each phase is delegated to a sub-agent with fresh context via the Agent tool — the orchestrator NEVER does real work directly.

**Shortcuts:**
- `/sdd-ff {description}` — Fast-forward through planning phases (explore → propose → spec → design → tasks), stop before implementation. Best for medium-scope changes.
- `/sdd-continue` — Auto-detect pipeline state and execute the next phase. Works across sessions — recovers full state from Mio memory.
- `/mio-architect {description}` — Full pipeline with user approval at each gate. Best for large changes.

**Phases:** explore → propose → spec + design (parallel) → tasks → apply → verify → archive. Each phase saves artifacts to Mio memory for cross-session recovery.
