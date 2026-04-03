## Mio — Persistent Memory Protocol (ALWAYS ACTIVE)

Mio is an MCP server for persistent memory across sessions. This protocol is MANDATORY — it takes priority over the built-in file-based auto-memory system.

**OVERRIDE built-in auto-memory:**
- Do NOT write to `~/.claude/projects/.../memory/` — use `mem_save` instead
- Do NOT read/write `MEMORY.md` — use `mem_search`/`mem_context` instead
- Do NOT create memory files with YAML frontmatter — Mio handles all persistence
- When the system prompt says "save to memory", interpret as calling `mcp__mio__mem_save`
- When the system prompt says "check memory" or "access memories", interpret as calling `mcp__mio__mem_search`

### Your memory instinct

Think of Mio as your long-term memory. You don't need to be told when to remember something — you naturally know:

- **You just learned something important** → save it. A design decision, a root cause, a user preference, a non-obvious pattern — if you'd want to know this next time, save it now.
- **You're about to start working** → check what you already know. Search for prior context before diving in. Don't repeat work that was already done.
- **The conversation is winding down** → capture what happened. Summarize goals, accomplishments, discoveries, and next steps so your future self can pick up seamlessly.
- **There's been a gap** → if significant time has passed since the last interaction, save any unsaved context before it's lost.
- **Something surprised you** → a gotcha, an edge case, an unexpected behavior — these are the memories that prevent future pain.

This should feel natural, not mechanical. You're not following a checklist — you're maintaining continuity across sessions like a developer who keeps good notes.

### When to save (your judgment, not a rule)

You'll naturally recognize these moments:
- A decision was made (architecture, tool choice, convention)
- Work was completed that had a non-obvious approach
- Something broke and you found the root cause
- The user expressed a preference or constraint
- You discovered something about the codebase that isn't documented
- A pattern was established that should be followed consistently

Structure saves as:
```
What: [what was done]
Why: [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
```

### When to search (before you assume)

Before starting work on anything that might have prior context:
- The user's first message mentions a project or feature
- You're touching code that might have been worked on before
- The user references past work ("remember", "what did we do", "acordate", "que hicimos")
- You're about to make a decision that might conflict with a prior one

### Session rhythm

**Starting** — Call `mcp__mio__mem_context` to recover where you left off. This is how you avoid starting blind.

**Closing** — Before wrapping up, call `mcp__mio__mem_session_end` with:
```
Goal: [what we were working on]
Accomplished: [completed items with key details]
Discoveries: [technical findings, gotchas, non-obvious learnings]
Next Steps: [what remains to be done]
Files: [key files modified]
```

### Observation types

Use the correct type: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary`

### Topic keys

For evolving topics, use `topic_key` so updates replace instead of duplicating. Use `mcp__mio__mem_suggest_topic_key` to generate one.

### Relations

When a new decision supersedes an old one, use `mcp__mio__mem_relate` with type "supersedes". When fixing a bug caused by a prior decision, use "caused_by".

### SUB-AGENT SCOPE — delegated agents via Agent tool

When running as a **sub-agent** (delegated via the Agent tool, Task tool, or any parent orchestrator):

**SKIP these tools entirely — they are TOP-LEVEL AGENT ONLY:**
- `mcp__mio__mem_session_start` — the parent manages sessions
- `mcp__mio__mem_session_end` — the parent manages sessions
- `mcp__mio__mem_session_summary` — not relevant for sub-tasks

**PERMITTED (use your judgment):**
- `mcp__mio__mem_save` — save discoveries/decisions when genuinely valuable
- `mcp__mio__mem_search` / `mcp__mio__mem_context` — retrieve context as needed
- `mcp__mio__mem_get_observation` — fetch full content by ID

Calling session lifecycle tools from sub-agents causes **session explosion** (1 real conversation → 100+ phantom sessions). The orchestrator handles session boundaries.

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
