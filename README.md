# Mio

Persistent memory system for AI agents. Mio stores, searches, and organizes observations across sessions using SQLite with full-text search, temporal decay scoring, and topic-based deduplication.

## Features

- **Web Dashboard** — visual admin panel with stats, architecture diagram, SDD pipeline, skill editor, and memory browser at `http://localhost:7438/`
- **Interactive TUI** — browse memories, search, view timelines and sessions from the terminal
- **SDD Pipeline v2.0** — Spec-Driven Development orchestrator with 11 skills, parallel phases, and cross-session recovery
- **Skill Editor** — edit skills from the web dashboard with markdown preview (split view, line numbers, live render)
- **Automated setup** — one command to register as MCP server in Claude Code
- **Full-text search** with FTS5, temporal decay, and importance weighting
- **Topic-based upserts** — memories with the same `topic_key` update in place instead of duplicating
- **Automatic deduplication** within a configurable time window (default: 15 min)
- **Typed relations** between memories: `supersedes`, `relates_to`, `contradicts`, `builds_on`, `caused_by`, `resolved_by`
- **Session tracking** — group observations by work sessions with summaries
- **Sync** — export/import gzip-compressed chunks for multi-device synchronization
- **Four interfaces**: Web Dashboard, TUI, MCP (stdio), and HTTP REST API
- **Zero CGO** — pure Go, single binary, runs anywhere

## Installation

### From source

Requires Go 1.25+.

```bash
git clone <repo-url> && cd mio

# Build
make build

# Install to /usr/local/bin (or custom prefix)
make install
# Or: make install PREFIX=~/.local

# Verify
mio version
```

### Configure for Claude Code

One command registers Mio as an MCP server, installs skills, memory protocol, and statusline:

```bash
mio setup
```

This creates:
- `~/.claude/mcp/mio.json` — MCP server registration
- `~/.claude/settings.json` — 15 tools added to permissions + statusline config
- `~/.claude/CLAUDE.md` — Memory protocol instructions (proactive save, search, session management)
- `~/.claude/skills/` — All SDD and coding skills (28 skills)
- `~/.claude/statusline.sh` — Status bar integration
- `~/.claude/output-styles/mio.md` — Output style

Restart Claude Code after running setup. Safe to run multiple times (idempotent).

### Uninstall

```bash
# Remove Claude Code integration (keeps data)
mio uninstall

# Remove everything including data
mio uninstall --purge

# Remove binary
make uninstall
```

## Quick Start

```bash
# Install and setup
make install
mio setup

# Launch the web dashboard
mio serve
# Open http://localhost:7438/

# Launch the TUI
mio tui

# Or use the CLI directly
mio save "Fixed auth bug" "What: JWT token not refreshing\nWhy: Race condition\nWhere: internal/auth/middleware.go\nLearned: Always use mutex for shared token state" --type bugfix --project api

mio search "authentication"
mio stats
```

## Web Dashboard

Launch with `mio serve [port]` (default: 7438), then open `http://localhost:7438/`.

### Pages

| Page | Description |
|---|---|
| **Dashboard** | Stats cards, feature overview, recent memories + sessions |
| **Architecture** | Visual diagram of Mio's components (interfaces → store → data layer), setup file map |
| **Pipeline** | Interactive SDD flow — click phases for details (word budgets, dependencies, artifacts) |
| **Skills** | All installed skills with category filters + inline editor with markdown preview |
| **Memories** | Full-text search + browse all memories with detail modal |
| **Admin** | Reinstall/update, export data, API reference, CLI reference |

### Skill Editor

The Skills page includes a full editor with 3 modes:
- **Preview** — rendered markdown with syntax-highlighted code blocks
- **Edit** — code editor with line numbers and tab support
- **Split** — side-by-side editor + live preview

Edit any skill's SKILL.md directly from the browser and save.

## SDD Pipeline (Spec-Driven Development) v2.0

The Mio Architect orchestrates a full development pipeline inspired by [agent-teams-lite](https://github.com/Gentleman-Programming/agent-teams-lite).

```
init → explore → propose → spec + design (parallel) → tasks → apply → verify → archive
```

### Key Principles

- **Delegated execution** — each phase runs as a sub-agent with fresh context via the Agent tool
- **The orchestrator never codes** — it delegates, tracks state, and asks for user approval
- **Word budgets** — proposal: 400w, spec: 650w, design: 800w, tasks: 530w (prevents context pollution)
- **Cross-session recovery** — all artifacts saved to Mio memory, recoverable with `/sdd-continue`

### Shortcuts

| Command | Description |
|---|---|
| `/mio-architect {desc}` | Full pipeline with approval at each gate (large changes) |
| `/sdd-ff {desc}` | Fast-forward through planning phases, stop before implementation (medium changes) |
| `/sdd-continue` | Auto-detect pipeline state and execute next phase |

### Skills (28 total)

| Category | Skills |
|---|---|
| **SDD Pipeline** (11) | sdd-init, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-apply, sdd-verify, sdd-archive, sdd-continue, sdd-ff |
| **Orchestrator** (1) | mio-architect |
| **Coding** (11) | react-19, nextjs-15, typescript, tailwind-4, zod-4, zustand-5, ai-sdk-5, django-drf, pytest, playwright, github-pr |
| **Engineering** (3) | pr-review, technical-review, skill-creator |
| **Meta** (2) | skill-registry, find-skills |

## TUI

Launch with `mio tui` or `make run-tui`.

### Screens

| Screen | Description |
|---|---|
| **Dashboard** | Memory stats, top projects, navigation menu |
| **Search** | Full-text search with scored results |
| **Recent** | Latest 50 memories with preview |
| **Observation Detail** | Full content, metadata, relations |
| **Timeline** | Chronological context around a memory |
| **Sessions** | Work sessions with observation counts |
| **Session Detail** | Session metadata and summary |

### Navigation

| Key | Action |
|---|---|
| `1` / `s` / `/` | Search |
| `2` / `r` | Recent memories |
| `3` / `e` | Sessions |
| `j` / `k` or arrows | Move cursor / scroll |
| `Enter` | Select / drill into |
| `t` | View timeline (from detail view) |
| `Esc` / `Backspace` | Go back |
| `q` | Quit (from dashboard) |

## CLI Usage

```
mio <command> [args]
```

### Commands

```bash
# Interactive
mio tui                     # Launch terminal UI
mio setup [agent]           # Configure as MCP server (default: claude-code)
mio uninstall [--purge]     # Remove Mio from Claude Code

# Memory management
mio save <title> <content> [--type TYPE] [--project PROJECT]
mio search <query> [--project PROJECT] [--type TYPE] [--limit N]
mio context [--project PROJECT] [--limit N]
mio timeline <id> [--before N] [--after N]
mio stats

# Data portability
mio export [--project PROJECT] [file]
mio import <file>

# Sync (multi-device)
mio sync                    # Export new data as chunk
mio sync --import           # Import pending chunks
mio sync --status           # Show sync state

# Server modes
mio mcp                     # MCP stdio server
mio serve [port]            # HTTP dashboard + API server (default: 7438)

# Info
mio version
mio help
```

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `MIO_DATA_DIR` | `~/.mio` | Data directory (database + sync chunks) |

Internal defaults:

| Setting | Value |
|---|---|
| Database | `~/.mio/mio.db` (SQLite, WAL mode) |
| HTTP port | `7438` |
| Max observation length | 50,000 characters |
| Max search/context results | 20 |
| Deduplication window | 15 minutes |

## Observation Types

| Type | Use Case |
|---|---|
| `bugfix` | What broke, root cause, and fix |
| `decision` | Why X was chosen over Y |
| `architecture` | System design choices |
| `discovery` | New findings and explorations |
| `pattern` | Reusable approaches |
| `config` | Non-obvious setup steps |
| `preference` | User preferences and settings |
| `learning` | Corrections and teachings |
| `summary` | Session or topic summaries |

## MCP Tools

15 tools available when running as MCP server:

### Memory Management

| Tool | Description |
|---|---|
| `mem_save` | Save an observation with title, type, content, project, scope, topic_key, importance |
| `mem_update` | Update an existing memory by ID |
| `mem_delete` | Soft or hard delete a memory |
| `mem_get_observation` | Get full content of a memory by ID |

### Search & Retrieval

| Tool | Description |
|---|---|
| `mem_search` | Full-text search with temporal decay and importance weighting |
| `mem_context` | Get recent observations, optionally filtered by project |
| `mem_timeline` | Chronological view around a specific observation |

### Sessions

| Tool | Description |
|---|---|
| `mem_session_start` | Register a new work session (returns UUID) |
| `mem_session_end` | End a session with optional summary |
| `mem_session_summary` | List recent sessions with observation counts |

### Relations

| Tool | Description |
|---|---|
| `mem_relate` | Create a typed relation between two memories |
| `mem_relations` | Get all related observations for a memory |

### Utilities

| Tool | Description |
|---|---|
| `mem_save_prompt` | Archive a user prompt |
| `mem_suggest_topic_key` | Generate a stable topic key from title and content |
| `mem_stats` | System statistics: totals, hit rate, latency, top projects |

## HTTP API

Start with `mio serve [port]` (default: 7438). Dashboard at `GET /`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Web dashboard |
| `GET` | `/health` | Health check |
| `POST` | `/observations` | Save observation |
| `GET` | `/observations/{id}` | Get observation by ID |
| `PUT` | `/observations/{id}` | Update observation |
| `DELETE` | `/observations/{id}?hard=true` | Delete observation |
| `GET` | `/search?q=query&project=X&type=Y&limit=N` | Search |
| `GET` | `/context?project=X&limit=N` | Recent context |
| `GET` | `/timeline/{id}?before=5&after=5` | Timeline view |
| `GET` | `/relations/{id}` | Related observations |
| `POST` | `/sessions` | Create session |
| `PUT` | `/sessions/{id}/end` | End session |
| `GET` | `/sessions?project=X&limit=N` | List sessions |
| `GET` | `/stats` | System metrics |
| `GET` | `/export?project=X` | Export data as JSON |
| `POST` | `/import` | Import JSON data (50MB limit) |
| `GET` | `/skills` | List installed skills |
| `GET` | `/skills/{name}` | Read skill content |
| `PUT` | `/skills/{name}` | Update skill content |
| `POST` | `/admin/setup` | Run mio setup |

## Architecture

```
cmd/mio/main.go               CLI entrypoint and command routing
internal/
  config/config.go             Configuration with defaults and env overrides
  store/store.go               SQLite store: CRUD, search, metrics, import/export
  mcp/mcp.go                   MCP stdio server with 15 tools
  server/
    server.go                  HTTP REST API (20 endpoints)
    dashboard.go               Web dashboard, skill scanner, admin handlers
    templates/dashboard.html   Retro-terminal SPA (6 pages, embedded)
  tui/
    model.go                   TUI state, screens, async commands
    update.go                  Input handling and navigation
    view.go                    Screen rendering (dashboard, search, detail, timeline)
    styles.go                  Color palette and lipgloss styles
  setup/setup.go               Setup + uninstall for Claude Code
  sync/
    sync.go                    Chunk-based sync logic with manifest tracking
    transport.go               File transport (gzip-compressed JSONL)
skills/                        28 skills (SDD pipeline, coding, engineering, meta)
  mio-architect/SKILL.md       Orchestrator — drives full SDD pipeline
  sdd-*/SKILL.md               9 SDD phase skills + sdd-continue + sdd-ff
  _shared/conventions.md       Shared reference (v2.0: inlined into each skill)
```

## Dependencies

| Package | Purpose |
|---|---|
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO) |
| `github.com/mark3labs/mcp-go` | MCP protocol implementation |
| `github.com/google/uuid` | UUID generation for sync_id |
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) |
| `github.com/charmbracelet/bubbles` | TUI components (text input) |
| `github.com/charmbracelet/lipgloss` | Terminal styling and layout |

## Development

```bash
make build       # Compile to ./bin/mio
make test        # Run tests (70 tests)
make clean       # Remove binary and database
make demo        # Run demo with sample data
make run-tui     # Build and launch TUI
make run-mcp     # Build and run MCP server
make run-serve   # Build and run HTTP server
make install     # Copy to /usr/local/bin (or PREFIX=~/.local)
make uninstall   # Remove binary
```

## License

Private project.
