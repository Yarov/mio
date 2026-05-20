# Mio — Project Context

Read this before touching any file. This is the source of truth for what Mio is, how it works, and how to work on it.

## What Mio is

Mio is a **persistent memory system for AI coding agents**. It's a single Go binary that gives agents (Claude Code, Cursor, Gemini CLI, Codex CLI, VS Code Copilot, OpenCode, Continue.dev, Kilo Code) the ability to remember across sessions.

Without Mio, every conversation starts from zero. With Mio, agents remember decisions, preferences, bugs, progress, and context — like a colleague who was there yesterday.

## Core principle

**Mio should be invisible to the user.** The agent should sound like it naturally remembers things, not like it's querying a database. "Last time we left the auth refactor ready for testing" — never "According to my memory system, observation #142 indicates..."

This principle drives everything: protocol wording, tool design, hook behavior.

## Architecture

```
AI Agents (Claude Code, Cursor, Gemini, ...)
    | MCP (stdio, JSON-RPC)
    v
mio mcp — 29 tools (10 eager, 19 deferred)
    |
    v
SQLite Store — FTS5 search, WAL mode, TF-IDF embeddings
~/.mio/mio.db
    |
    +-- mio server — HTTP dashboard + REST API on :7438 (launchd, always running)
    +-- mio tui — Bubble Tea terminal UI (8 screens)
    +-- mio CLI — save, search, context, export, import, setup, sync
```

Each agent spawns its own `mio mcp` process (ephemeral). All state lives in SQLite. Multiple MCP processes coexist safely via WAL.

## Directory map

| Path | What it is |
|------|-----------|
| `cmd/mio/main.go` | CLI entry point, command routing |
| `internal/agents/` | 8 agent implementations + registry + helpers. `agent.go` defines the interface |
| `internal/mcp/` | MCP server, 29 tools. `mcp.go` is the main file |
| `internal/mcp/tool_guide.go` | Tool routing guide returned by `mem_tool_guide` |
| `internal/mcp/tool_routing.go` | Model routing tools (get/set) |
| `internal/mcp/tool_style.go` | Output style toggle tools (get/set) |
| `internal/store/` | SQLite store, FTS5 search, dedup, scoring, migrations |
| `internal/store/routing.go` | Model routing persistence + config |
| `internal/store/output_style.go` | Output style toggle persistence |
| `internal/server/` | HTTP server + dashboard + REST API |
| `internal/tui/` | Bubble Tea TUI (model/view/update) |
| `internal/sync/` | Cross-device sync (file, git, S3 transports) |
| `internal/config/` | Config struct + defaults |
| `protocols/` | Agent-specific instruction templates — **these are functional code** |
| `skills/` | 27+ domain skills (SDD pipeline, coding patterns, PR/review) |
| `hooks/` | User-prompt-submit hook for session continuity |
| `output-styles/` | Claude Code output style (tone, language, format) |
| `docs/` | Architecture, MCP tools reference, plans |

## Data model

**Observation** — the core memory unit:
- `id`, `sync_id` (UUID for cross-device)
- `type`: bugfix, decision, architecture, discovery, pattern, config, preference, learning, summary
- `title` (3-200 chars, action verb + what), `content` (10-50K, What/Why/Where/Learned)
- `project`, `scope` (project/personal/global), `topic_key` (stable key for upsert)
- `importance` (0-1), `access_count`, `revision_count`
- `normalized_hash` (SHA256 for dedup within 15-min window)
- `agent` label, `consolidated` flag, timestamps

**Session** — work session container with summary (Goal/Accomplished/Discoveries/Next Steps)

**Relation** — links between observations: supersedes, caused_by, builds_on, relates_to, contradicts, resolved_by

## MCP tools (29 total)

**Eager (10, always in context):** mem_save, mem_search, mem_context, mem_get_observation, mem_session_start, mem_session_end, mem_session_summary, mem_tool_guide, mem_surface

**Deferred (19, loaded on-demand):** mem_update, mem_delete, mem_timeline, mem_save_prompt, mem_relations, mem_relate, mem_suggest_topic_key, mem_stats, mem_cross_project, mem_enhanced_search, mem_consolidate, mem_summarize, mem_gc, mem_graph, mem_agent_knowledge, mem_routing_get, mem_routing_set, mem_style_get, mem_style_set

### Tool design rules

- `mem_save` and `mem_relate` use **enum validation** for `type`, `scope`, and relation `type` — invalid values are rejected at the MCP layer
- `mem_save` and `mem_session_start` return **JSON responses** (`{"id": N, "sync_id": "..."}` / `{"session_id": "..."}`) — not prose strings
- Numeric params handle string coercion (some MCP clients send numbers as strings)
- `mem_update` is for direct corrections only — prefer `mem_save` with `topic_key` for evolving topics (handles versioning automatically)
- `mem_surface` is for background context without a specific query — surfaces relevant memories from conversation text

## Setup flow

`mio setup [agent]` does:
1. Write MCP config (own file for Claude Code, shared JSON for others)
2. Install protocol via `<!-- BEGIN:mio -->` / `<!-- END:mio -->` markers (idempotent)
3. Copy skills to agent's skills directory
4. Agent-specific extras (Claude Code: allowlist, statusline, output-style, launchd)
5. Install launchd plist for HTTP server (macOS, KeepAlive: true)

## Protocols are functional code

**This is the most important rule for working on Mio.**

Protocol files (`protocols/*.md`) are not documentation. They are the **control interface** for agent behavior. Every line exists because an agent failed without it:

- The "Be natural — the machinery is invisible" section with Good/Bad examples? Agents were saying "According to my memory system..." without it.
- The "Don't chain desperate searches" line? Agents were doing 5 consecutive searches when `mem_context` returned empty.
- The sub-agent scope rules? Sub-agents were creating session explosions.
- The tool routing table? Agents were calling `mem_search` for everything instead of using `mem_context` for startup.

**Rules for editing protocols:**
1. Understand what specific failure each line prevents before touching it
2. Never compress, simplify, or "optimize" protocol text without confirming the protection is no longer needed
3. The Good/Bad examples are behavioral guardrails, not decoration
4. Test changes in a real agent session before committing
5. If base.md has a section that an agent-specific protocol doesn't, that's likely a bug — check if it was accidentally removed
6. Keep base.md and agent-specific protocols in sync — base.md is the canonical source, agent protocols add agent-specific overrides

## SDD pipeline and Architect

Mio includes an SDD (Spec-Driven Development) pipeline orchestrated by `mio-architect`:

**Pipeline:** explore -> propose -> spec + design (parallel) -> tasks -> apply -> verify -> archive

Each phase has its own skill file in `skills/sdd-*/SKILL.md`. The architect (`skills/mio-architect/SKILL.md`) is a **dispatch loop** — it delegates to sub-agents, never does implementation directly.

### Architect rules (hard-won)

- **NEVER does real work** — delegates everything via Agent tool
- **NEVER uses Edit, Write, or Bash** to modify project files — only Read/Grep/Glob for inspection + Mio MCP for state
- **Small changes (1-2 files):** Skip SDD, tell the user, and END. Does NOT implement.
- **Medium changes (3-5 files):** Uses `/sdd-ff` (fast-forward through planning, stop at apply)
- **Large changes (6+ files):** Full pipeline with user approval at each gate
- **Max 2 verify retries** — after 2 failures, stops and asks user for guidance

### Entry points

| Entry | Use case |
|-------|---------|
| `/mio-architect {desc}` | Full pipeline with approval gates |
| `/sdd-ff {desc}` | Fast-forward planning, stop before implementation |
| `/sdd-continue` | Auto-detect state, execute next phase (works cross-session) |

### Sub-agent conventions (`skills/_shared/conventions.md`)

- Every phase returns: Status (success/partial/blocked), Artifact key, What's next, Risks
- After `mem_save`, verify by calling `mem_search` for the `topic_key`
- Use `mem_save` with `topic_key` for upserts (not `mem_update`)
- `sdd-continue` and `sdd-ff` are **top-level entry points**, NOT sub-agents of the architect

## Hook behavior

`hooks/user-prompt-submit.sh` — runs on every user prompt:
- On session start (no state file): prompts agent to load context
- After 15min inactivity: gentle reminder to persist important context
- Uses atomic writes (`mv` from temp file) for state file safety
- Threshold configurable via `MIO_INACTIVITY_THRESHOLD` env var
- Validates state file content (guards against corruption)

## Output style

`output-styles/mio.md` — Mexican Spanish dev tone:
- Spanish responses when user writes Spanish, English otherwise
- Code/commits/PRs always in English
- Direct, no-BS tone ("va", "sale", "chido", "no mames")
- After user confirms: execute + one-line result, no re-stating
- Autonomy: propose before big changes (new files, architecture, deletions, >3 files), execute directly on small/obvious ones (typos, imports, formatting)

## Build and run

```bash
make build          # Compile with version/commit ldflags
make install        # Copy to /usr/local/bin/mio
make test           # Run all tests
mio setup claude-code  # Install for Claude Code
mio setup cursor       # Install for Cursor
mio tui             # Launch terminal UI
mio server 7438     # Start HTTP dashboard
```

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `MIO_DATA_DIR` | `~/.mio` | Data directory |
| `MIO_DEFAULT_AGENT` | `claude-code` | Default agent label for saves |
| `MIO_SUBAGENT` | (unset) | If `1`/`true`/`yes`, blocks session tools |
| `MIO_SYNC_TRANSPORT` | `file` | Sync backend: file, git, s3 |
| `MIO_INACTIVITY_THRESHOLD` | `900` | Hook inactivity threshold in seconds |

## Dependencies

- Go 1.25+
- `modernc.org/sqlite` — Pure Go SQLite (no CGO, single binary)
- `github.com/mark3labs/mcp-go` — MCP protocol
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/google/uuid` — UUIDs for sync
