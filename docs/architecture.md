# Architecture

## Overview

Mio is a persistent memory system for AI coding agents. It has four main interfaces:

```
                    ┌────────────────────────────────┐
                    │         AI Agents               │
                    │  Claude, Cursor, Gemini, ...    │
                    └──────────┬─────────────────────┘
                               │ MCP (stdio)
                    ┌──────────▼─────────────────────┐
                    │         mio mcp                 │
                    │   MCP Server (16 tools)         │
                    └──────────┬─────────────────────┘
                               │
          ┌────────────────────▼────────────────────┐
          │              SQLite Store                │
          │   FTS5 search · WAL mode · Triggers     │
          │            ~/.mio/mio.db                 │
          └──┬──────────────┬──────────────┬────────┘
             │              │              │
    ┌────────▼───┐  ┌──────▼──────┐  ┌───▼────────┐
    │ mio server │  │   mio tui   │  │  mio CLI   │
    │ HTTP + UI  │  │ Bubble Tea  │  │  commands   │
    │ :7438      │  │ alt-screen  │  │  save/search│
    └────────────┘  └─────────────┘  └────────────┘
```

## Components

### 1. MCP Server (`internal/mcp/`)

The MCP (Model Context Protocol) server is the primary interface for AI agents. It runs on stdio and exposes 16 tools:

- **Memory CRUD**: save, get, update, delete
- **Search**: full-text search with scoring
- **Context**: recent memories, timeline
- **Sessions**: start, end, summary
- **Relations**: link memories together
- **Utilities**: suggest topic key, stats

Each agent spawns its own `mio mcp` process. The process is ephemeral — it lives and dies with the agent session. All state is in the SQLite database, so multiple MCP processes can safely coexist.

### 2. SQLite Store (`internal/store/`)

The storage layer uses SQLite with aggressive optimizations:

```sql
PRAGMA journal_mode = WAL;      -- concurrent reads during writes
PRAGMA busy_timeout = 5000;     -- retry on lock contention
PRAGMA synchronous = NORMAL;    -- balance safety vs speed
PRAGMA cache_size = -20000;     -- 20MB page cache
PRAGMA foreign_keys = ON;
PRAGMA temp_store = MEMORY;
```

#### Schema

```
sessions
├── id (TEXT PK)
├── project, directory
├── started_at, ended_at
└── summary

observations
├── id (INTEGER PK AUTOINCREMENT)
├── sync_id (UUID, for cross-device sync)
├── session_id → sessions.id
├── type (bugfix|decision|architecture|...)
├── title, content
├── project, scope
├── topic_key (stable key for upsert)
├── normalized_hash (dedup)
├── importance (0.0-1.0)
├── access_count, last_accessed
├── revision_count, duplicate_count
├── created_at, updated_at, deleted_at
└── [FTS5: title, content, tool_name, type, project]

relations
├── from_id → observations.id
├── to_id → observations.id
├── type (supersedes|caused_by|builds_on|...)
└── strength (0.0-1.0)

user_prompts
├── id, sync_id
├── session_id → sessions.id
├── content, project
└── created_at
```

#### Search Algorithm

Search uses SQLite FTS5 with a composite scoring function:

```
score = fts_rank × importance_boost × recency_boost
```

- **fts_rank**: BM25 relevance score from FTS5
- **importance_boost**: `1.0 + (importance - 0.5)` (range 0.5 to 1.5)
- **recency_boost**: Temporal decay — recent memories score higher

Results are filtered to exclude soft-deleted observations and sorted by composite score.

#### Deduplication

Each observation gets a `normalized_hash` computed from normalized title + content. On save:

1. If `topic_key` matches an existing observation → **upsert** (update in place, increment revision_count)
2. If `normalized_hash` matches within `DedupeWindow` (15 min) → **skip** (increment duplicate_count)
3. Otherwise → **insert** new observation

#### Validation

Observations must pass:
- Title: 3-200 characters
- Content: 10-50,000 characters
- Type: one of the valid types
- Importance: 0.0-1.0

### 3. HTTP Server (`internal/server/`)

The HTTP server runs as an **independent process** separate from any MCP session. This is a key architectural decision:

```
Agent session 1 (mio mcp) ──→ dies when agent closes
Agent session 2 (mio mcp) ──→ dies when agent closes
mio server                 ──→ always running (launchd)
```

On macOS, `mio setup` installs a launchd plist (`~/Library/LaunchAgents/com.mio.server.plist`) with `KeepAlive: true`. The server auto-starts on login and is restarted if it crashes.

As a fallback, `mio mcp` checks if the server is running on startup and spawns a detached `mio server` process if not.

The server provides:
- **Dashboard**: Single-page HTML app with retro cyberpunk design
- **REST API**: 22 endpoints for full memory management
- **Agent management**: Setup/uninstall agents via API
- **PID file**: `~/.mio/server.pid` for process coordination

### 4. Agent System (`internal/agents/`)

Agents are implemented via a Go interface:

```go
type Agent interface {
    Name() string
    DisplayName() string
    Detect() bool
    Setup(binPath string) error
    Uninstall(purge bool) error
    Status() AgentStatus
}
```

Each agent registers itself via `init()`. The registry provides:
- `Get(name)` — look up by name
- `All()` — all registered agents
- `DetectInstalled()` — scan system for installed agents
- `StatusAll()` — status of every agent

#### Setup Flow

```
agent.Setup(binPath)
  │
  ├─ Write MCP config
  │    ├─ Own file (Claude Code): ~/.claude/mcp/mio.json
  │    └─ Shared JSON (others): merge into existing config
  │
  ├─ Install protocol
  │    ├─ Using <!-- BEGIN:mio --> / <!-- END:mio --> markers
  │    └─ Idempotent: update on re-run, clean remove on uninstall
  │
  ├─ Copy skills (if agent supports them)
  │
  └─ Agent-specific extras
       ├─ Claude Code: allowlist, statusline, output-style, launchd
       └─ Others: minimal setup
```

### 5. Sync System (`internal/sync/`)

File-based sync for cross-device memory sharing:

```
~/.mio/chunks/
├── manifest.json        # index of all chunks
├── chunk-001.jsonl.gz   # compressed JSONL
├── chunk-002.jsonl.gz
└── ...
```

**Export**: Creates a new chunk with all data since the last export. Each observation has a `sync_id` (UUID) for cross-device deduplication.

**Import**: Reads pending chunks and merges into local database, skipping duplicates by `sync_id`.

### 6. TUI (`internal/tui/`)

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Provides 8 screens:

- Dashboard (stats + menu)
- Search (query input)
- Search Results (scored list)
- Observation Detail (full content + relations)
- Timeline (chronological context)
- Sessions (list)
- Session Detail (observations in session)

### 7. Skills System

Skills are markdown files with YAML frontmatter:

```yaml
---
name: react-19
description: React 19 patterns with React Compiler
metadata:
  version: "1.0"
---

## Patterns

Content the agent follows when working with React 19...
```

Skills are stored in `skills/` in the project and copied to each agent's skills directory during setup. The dashboard can browse and edit skills.

## Data Flow

### Agent saves a memory

```
Agent calls mem_save
  → MCP server receives JSON-RPC
  → Validate (title, content, type)
  → Compute normalized_hash
  → Check topic_key for upsert
  → Check hash for dedup within window
  → INSERT into observations table
  → FTS5 trigger updates search index
  → Return {id, sync_id}
```

### Agent searches memories

```
Agent calls mem_search("auth bug")
  → MCP server receives JSON-RPC
  → FTS5 MATCH query with BM25 ranking
  → Apply importance boost + recency decay
  → Log search to search_log
  → Increment access_count on returned results
  → Return sorted results with scores
```

### Agent starts a session

```
Agent calls mem_context (session start)
  → Return last N observations for project
  → Agent has full context from prior sessions

Agent works... calls mem_save as needed

Agent calls mem_session_end
  → Save summary with goal/accomplished/discoveries
  → Next session starts with this context
```

## File Layout

```
mio/
├── cmd/mio/main.go           # CLI entry point, command routing
├── internal/
│   ├── agents/
│   │   ├── agent.go           # Interface + types
│   │   ├── registry.go        # Global registry
│   │   ├── helpers.go         # Shared MCP/protocol/skills helpers
│   │   ├── claude_code.go     # Claude Code (full support)
│   │   ├── cursor.go          # Cursor
│   │   ├── gemini_cli.go      # Gemini CLI
│   │   ├── codex_cli.go       # Codex CLI
│   │   ├── vscode_copilot.go  # VS Code Copilot
│   │   ├── opencode.go        # OpenCode
│   │   ├── continue_dev.go    # Continue.dev
│   │   └── kilo_code.go       # Kilo Code
│   ├── config/config.go       # Config struct + defaults
│   ├── mcp/mcp.go             # MCP server + 16 tools
│   ├── server/
│   │   ├── server.go          # HTTP routes + handlers
│   │   ├── dashboard.go       # Dashboard + skills + admin handlers
│   │   └── templates/
│   │       └── dashboard.html # Single-page dashboard
│   ├── setup/setup.go         # Thin dispatcher → agents
│   ├── store/
│   │   ├── store.go           # SQLite store + all queries
│   │   └── store_test.go      # Store tests
│   ├── sync/
│   │   ├── sync.go            # Export/import chunks
│   │   └── sync_test.go       # Sync tests
│   └── tui/tui.go             # Bubble Tea TUI
├── protocols/                  # Protocol templates per agent
├── skills/                     # 27+ pre-built skills
├── statusline.sh              # Claude Code statusline script
├── output-styles/mio.md       # Claude Code output style
├── Makefile                   # Build targets
└── go.mod                     # Dependencies
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO) |
| `github.com/mark3labs/mcp-go` | MCP protocol implementation |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/bubbles` | TUI components |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/google/uuid` | UUID generation for sync_id |

## Design Decisions

1. **SQLite over Postgres/Redis** - Single file, zero ops, concurrent reads via WAL. Good enough for millions of memories on a single machine.

2. **FTS5 over external search** - Built into SQLite, zero config, fast BM25 ranking. No need for Elasticsearch or similar.

3. **MCP over custom protocol** - Standard protocol supported by Claude Code, Cursor, and others. One implementation, many agents.

4. **Independent HTTP server** - Decoupled from MCP sessions. Survives agent restarts. Managed by launchd on macOS.

5. **Markers for protocol injection** - `<!-- BEGIN:mio -->` / `<!-- END:mio -->` enables idempotent install/update/remove without touching user content.

6. **Agent interface pattern** - Adding a new agent is ~60-100 LOC implementing 5 methods. No changes to core code.

7. **File-based sync** - Simple, offline-first. No server dependency. Future: cloud sync as optional upgrade.

8. **Pure Go SQLite** - `modernc.org/sqlite` compiles without CGO. Single binary, cross-compile anywhere.
