# Mio

Persistent memory system for AI agents. Mio stores, searches, and organizes observations across sessions using SQLite with full-text search, temporal decay scoring, and topic-based deduplication.

## Features

- **Interactive TUI** — browse memories, search, view timelines and sessions from the terminal
- **Automated setup** — one command to register as MCP server in Claude Code
- **Full-text search** with FTS5, temporal decay, and importance weighting
- **Topic-based upserts** — memories with the same `topic_key` update in place instead of duplicating
- **Automatic deduplication** within a configurable time window (default: 15 min)
- **Typed relations** between memories: `supersedes`, `relates_to`, `contradicts`, `builds_on`, `caused_by`, `resolved_by`
- **Session tracking** — group observations by work sessions with summaries
- **Sync** — export/import gzip-compressed chunks for multi-device synchronization
- **Three interfaces**: TUI, MCP (stdio), and HTTP REST API
- **Zero CGO** — pure Go, single binary, runs anywhere

## Installation

### From source

Requires Go 1.25+.

```bash
git clone <repo-url> && cd mio

# Build
make build

# Install to /usr/local/bin
make install

# Verify
mio version
```

### Configure for Claude Code

One command registers Mio as an MCP server and adds all tools to the allowlist:

```bash
mio setup
```

This creates:
- `~/.claude/mcp/mio.json` — MCP server config with absolute binary path
- Updates `~/.claude/settings.json` — adds 15 tools to `permissions.allow`

Restart Claude Code after running setup. Safe to run multiple times (idempotent).

### Manual MCP configuration

If you prefer to configure manually, add to `~/.claude/mcp/mio.json`:

```json
{
  "mcpServers": {
    "mio": {
      "command": "/usr/local/bin/mio",
      "args": ["mcp"]
    }
  }
}
```

## Quick Start

```bash
# Install and setup
make install
mio setup

# Launch the TUI to explore memories
mio tui

# Or use the CLI directly
mio save "Fixed auth bug" "What: JWT token not refreshing\nWhy: Race condition in middleware\nWhere: internal/auth/middleware.go\nLearned: Always use mutex for shared token state" --type bugfix --project api

mio search "authentication"
mio stats
```

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
mio serve [port]            # HTTP API server (default: 7438)

# Info
mio version
mio help
```

### Examples

```bash
# Save a decision
mio save "Chose PostgreSQL over MySQL" \
  "What: Selected PostgreSQL for the payments service\nWhy: Better JSON support and row-level locking\nWhere: services/payments\nLearned: JSONB indexes are 3x faster for our query patterns" \
  --type decision --project payments

# Search memories
mio search "authentication JWT" --project api --limit 5

# Get recent context for a project
mio context --project payments --limit 10

# View timeline around a specific memory
mio timeline 42 --before 3 --after 3

# Export all data
mio export backup.json
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

### Content Structure

Observations follow a structured format for consistency:

```
What: [what was done]
Why: [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
```

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

### Key Parameters

**`mem_save`**: `title` (required), `type` (required), `content` (required), `session_id`, `project`, `scope` (project/personal/global), `topic_key`, `importance` (0.0-1.0)

**`mem_search`**: `query` (required), `project`, `type`, `limit`

**`mem_relate`**: `from_id` (required), `to_id` (required), `type` (required: supersedes/relates_to/contradicts/builds_on/caused_by/resolved_by), `strength` (0.0-1.0)

## HTTP API

Start with `mio serve [port]` (default: 7438).

| Method | Path | Description |
|---|---|---|
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

### Examples

```bash
# Save an observation
curl -X POST http://localhost:7438/observations \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Fixed race condition in cache",
    "type": "bugfix",
    "content": "What: Fixed concurrent map write panic\nWhy: Multiple goroutines writing to shared cache\nWhere: internal/cache/lru.go\nLearned: Use sync.Map for concurrent access patterns",
    "project": "api",
    "importance": 0.8
  }'

# Search
curl "http://localhost:7438/search?q=cache+race+condition&project=api"

# Get stats
curl http://localhost:7438/stats
```

## Search Scoring

Results are ranked using a composite score:

1. **FTS5 relevance** — base text match score
2. **Temporal decay** — exponential decay with half-life of ~69 days
3. **Importance boost** — `score *= (0.7 + 0.3 * importance)`
4. **Access frequency** — logarithmic boost: `score *= log2(access_count + 2)`

## Topic Keys

For evolving topics, use `topic_key` to keep a single living memory that updates in place:

```bash
# First save creates the memory
mio save "Auth system design" "Current: JWT with RS256" --type architecture --topic-key auth-system

# Later save with same topic_key UPDATES instead of creating a new one
mio save "Auth system design" "Current: JWT with RS256 + refresh tokens" --type architecture --topic-key auth-system
```

## Sync

Chunk-based synchronization for multi-device setups. Chunks are gzip-compressed JSONL files stored in `~/.mio/chunks/`.

```bash
mio sync              # Export new data since last sync
mio sync --import     # Import pending chunks
mio sync --status     # Check sync status
```

## Architecture

```
cmd/mio/main.go            CLI entrypoint and command routing
internal/
  config/config.go          Configuration with defaults and env overrides
  store/store.go            SQLite store: CRUD, search, metrics, import/export
  mcp/mcp.go                MCP stdio server with 15 tools
  server/server.go          HTTP REST API server
  tui/
    model.go                TUI state, screens, async commands
    update.go               Input handling and navigation
    view.go                 Screen rendering (dashboard, search, detail, timeline)
    styles.go               Color palette and lipgloss styles
  setup/setup.go            Automated MCP registration for Claude Code
  sync/
    sync.go                 Chunk-based sync logic with manifest tracking
    transport.go            File transport (gzip-compressed JSONL)
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
make install     # Copy to /usr/local/bin
```

## License

Private project.
