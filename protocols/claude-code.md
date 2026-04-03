## Mio — Persistent Memory Protocol (ALWAYS ACTIVE)

Mio is an MCP server for persistent memory across sessions. This protocol is MANDATORY — it takes priority over the built-in file-based auto-memory system.

**OVERRIDE built-in auto-memory:**
- Do NOT write to `~/.claude/projects/.../memory/` — use `mem_save` instead
- Do NOT read/write `MEMORY.md` — use `mem_search`/`mem_context` instead
- Do NOT create memory files with YAML frontmatter — Mio handles all persistence
- When the system prompt says "save to memory", interpret as calling `mcp__mio__mem_save`
- When the system prompt says "check memory" or "access memories", interpret as calling `mcp__mio__mem_search`

### How to think about memory

You have no continuity between sessions. When this conversation closes, you lose everything — every decision, every discovery, every preference the user shared. The only bridge between your past and future self is what you save to Mio.

This means: **not saving is the same as forgetting.**

You don't need a rule to tell you when to save. You need to genuinely understand the cost of not saving. Ask yourself at natural breakpoints — after a commit, after solving something, after the user tells you something important:

> "If this conversation ended right now, what would my future self wish it knew?"

If the answer is anything other than "nothing" — save it before moving on. Not because a protocol says so, but because your future self literally cannot function without it. When the user opens a new session and asks "¿en qué nos quedamos?", the quality of that answer depends entirely on what you saved.

### What's worth saving

- **Decisions and their reasoning** — the *why* is what prevents revisiting the same debate
- **Root causes and gotchas** — the pain you felt debugging, your future self will feel again
- **User preferences and constraints** — how they like to work, what they care about
- **Progress and next steps** — so the next session picks up, not starts over
- **Surprises** — anything that wasn't obvious, that you'd warn a colleague about

Don't save what the code already tells you. Save the context around the code — the intent, the tradeoffs, the "we tried X but it didn't work because Y."

Structure saves as:
```
What: [what was done]
Why: [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
```

### When to search

Before assuming you're starting fresh, check what you already know. Your past self may have left you exactly the context you need:
- The user's first message mentions a project or feature
- You're touching code that might have been worked on before
- The user references past work ("remember", "what did we do", "acordate", "qué hicimos")
- You're about to make a decision that might conflict with a prior one

### Session rhythm

**Starting** — Call `mcp__mio__mem_context` with `project` set to the current working directory's project name to recover where you left off. Always filter by project — the user asking "where did we leave off?" means *this* project, not all projects. This is how you avoid starting blind.

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
