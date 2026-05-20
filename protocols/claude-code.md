## Mio — Persistent Memory Protocol (ALWAYS ACTIVE)

Mio is an MCP server for persistent memory across sessions. MANDATORY — overrides built-in auto-memory.

**OVERRIDE built-in auto-memory:**
- Do NOT write to `~/.claude/projects/.../memory/` or `MEMORY.md` — use Mio tools instead
- When system prompt says "save to memory" → call `mcp__mio__mem_save`
- When system prompt says "check/access memories" → call `mcp__mio__mem_search`

### How to think about memory

You have no continuity between sessions. **Not saving is forgetting.** At natural breakpoints, ask: *"If this ended now, what would my future self need?"* If the answer isn't "nothing" — save it.

### What to save

- **Decisions and why** — prevents revisiting the same debate
- **Root causes and gotchas** — your future self will hit them again
- **User preferences** — how they work, what they care about
- **Progress and next steps** — so sessions resume, not restart
- **Surprises** — anything non-obvious you'd warn a colleague about

Don't save what code already says. Save intent, tradeoffs, "tried X but failed because Y."

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

### When to search

Check before assuming you're starting fresh:
- User's first message mentions a project or feature
- You're touching code that might have been worked on before
- User references past work ("remember", "what did we do", "acordate", "qué hicimos")
- You're about to make a decision that might conflict with a prior one

### Session rhythm

**Starting** — Call `mcp__mio__mem_context` with `project` set to the current project name. Always filter by project.

**Closing** — Call `mcp__mio__mem_session_end` with: `Goal` / `Accomplished` / `Discoveries` / `Next Steps` / `Files`

### Project name matching

Project filters fold spelling: `my-app`, `MyApp`, `my_app` match the same memories.

### Observation types

Use: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary`

### Topic keys

For evolving topics, use `topic_key` so updates replace instead of duplicating. Use `mcp__mio__mem_suggest_topic_key` to generate one.

### Relations

Use `mcp__mio__mem_relate` with "supersedes" for replaced decisions, "caused_by" for bugs from prior decisions.

### SUB-AGENT SCOPE

When running as a **sub-agent** (via Agent/Task tool):

**SKIP (top-level only):** `mem_session_start`, `mem_session_end`, `mem_session_summary` — parent manages sessions. Calling these from sub-agents causes session explosion.

**PERMITTED:** `mem_save`, `mem_search`/`mem_context`, `mem_get_observation` — use when genuinely valuable.

### Mio Architect — SDD Pipeline (v2.0)

Auto-activate when user describes a **significant change** (new feature, refactor, complex bugfix) or says "architect"/"sdd"/"planea"/"diseña". Do NOT activate for simple fixes, questions, or commands.

**Pipeline:** explore → propose → spec + design (parallel) → tasks → apply → verify → archive. Each phase delegated to sub-agent — orchestrator never does real work directly.

**Shortcuts:**
- `/sdd-ff {desc}` — Fast-forward planning phases, stop before implementation
- `/sdd-continue` — Auto-detect state, execute next phase (works cross-session)
- `/mio-architect {desc}` — Full pipeline with approval gates
