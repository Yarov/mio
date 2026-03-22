# Mio

Persistent memory system for AI agents. Mio stores, searches, and organizes observations across sessions using SQLite with full-text search, temporal decay scoring, and topic-based deduplication.

## Features

- **Full-text search** with FTS5, temporal decay, and importance weighting
- **Topic-based upserts** — memories with the same `topic_key` update in place instead of duplicating
- **Automatic deduplication** within a configurable time window (default: 15 min)
- **Typed relations** between memories: `supersedes`, `relates_to`, `contradicts`, `builds_on`, `caused_by`, `resolved_by`
- **Session tracking** — group observations by work sessions with summaries
- **Soft and hard deletes** — preserve data history or remove permanently
- **Sync** — export/import gzip-compressed chunks for multi-device synchronization
- **Two interfaces**: MCP (stdio, for agent integration) and HTTP REST API
- **Search analytics** — hit rate, latency tracking, access frequency

## Quick Start

```bash
# Build
make build

# Install to /usr/local/bin
make install

# Run MCP server (for Claude Code, Cursor, etc.)
mio mcp

# Run HTTP API server
mio serve        # default port 7438
mio serve 8080   # custom port

# Try the demo
make demo
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

## CLI Usage

```
mio <command> [args]
```

### Commands

```bash
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
mio serve [port]            # HTTP API server

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

# Export all data to a file
mio export backup.json

# Export only one project
mio export --project payments payments-backup.json
```

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

## Content Structure

Observations follow a structured format for consistency:

```
What: [what was done]
Why: [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
```

## MCP Server Integration

Mio implements the [Model Context Protocol](https://modelcontextprotocol.io/) for direct integration with AI agents.

### Claude Code

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "mio": {
      "command": "mio",
      "args": ["mcp"]
    }
  }
}
```

### Available MCP Tools

#### Memory Management

| Tool | Description |
|---|---|
| `mem_save` | Save an observation with title, type, content, project, scope, topic_key, importance |
| `mem_update` | Update an existing memory by ID |
| `mem_delete` | Soft or hard delete a memory |
| `mem_get_observation` | Get full content of a memory by ID |

#### Search & Retrieval

| Tool | Description |
|---|---|
| `mem_search` | Full-text search with temporal decay and importance weighting |
| `mem_context` | Get recent observations, optionally filtered by project |
| `mem_timeline` | Chronological view around a specific observation |

#### Sessions

| Tool | Description |
|---|---|
| `mem_session_start` | Register a new work session (returns UUID) |
| `mem_session_end` | End a session with optional summary |
| `mem_session_summary` | List recent sessions with observation counts |

#### Relations

| Tool | Description |
|---|---|
| `mem_relate` | Create a typed relation between two memories |
| `mem_relations` | Get all related observations for a memory |

#### Utilities

| Tool | Description |
|---|---|
| `mem_save_prompt` | Archive a user prompt |
| `mem_suggest_topic_key` | Generate a stable topic key from title and content |
| `mem_stats` | System statistics: totals, hit rate, latency, top projects |

### Tool Parameters

#### `mem_save`

| Parameter | Required | Description |
|---|---|---|
| `title` | yes | Brief title (action verb + what) |
| `type` | yes | One of the observation types |
| `content` | yes | Structured content (What/Why/Where/Learned) |
| `session_id` | no | Current session ID |
| `project` | no | Project name |
| `scope` | no | `project` (default), `personal`, or `global` |
| `topic_key` | no | Stable key for evolving topics (enables upsert) |
| `importance` | no | 0.0 to 1.0 (default: 0.5) |

#### `mem_search`

| Parameter | Required | Description |
|---|---|---|
| `query` | yes | Search query (full-text) |
| `project` | no | Filter by project |
| `type` | no | Filter by observation type |
| `limit` | no | Max results (default: 20) |

#### `mem_relate`

| Parameter | Required | Description |
|---|---|---|
| `from_id` | yes | Source observation ID |
| `to_id` | yes | Target observation ID |
| `type` | yes | `supersedes`, `relates_to`, `contradicts`, `builds_on`, `caused_by`, `resolved_by` |
| `strength` | no | 0.0 to 1.0 (default: 1.0) |

## HTTP API

Start with `mio serve [port]` (default: 7438).

### Endpoints

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

# Get recent context
curl "http://localhost:7438/context?project=api&limit=5"

# Get stats
curl http://localhost:7438/stats
```

## Search Scoring

Search results are ranked using a composite score:

1. **FTS5 relevance** — base text match score
2. **Temporal decay** — exponential decay with half-life of ~69 days (`e^(-0.01 * age_in_days)`)
3. **Importance boost** — `score *= (0.7 + 0.3 * importance)`
4. **Access frequency** — logarithmic boost: `score *= log2(access_count + 2)`

## Topic Keys

For evolving topics (e.g., "auth system design", "deploy pipeline"), use `topic_key` to keep a single living memory that updates in place:

```bash
# First save creates the memory
mio save "Auth system design" "Current: JWT with RS256" --type architecture --topic-key auth-system

# Later save with same topic_key UPDATES instead of creating a new one
mio save "Auth system design" "Current: JWT with RS256 + refresh tokens" --type architecture --topic-key auth-system
```

Use `mem_suggest_topic_key` to generate consistent keys from titles.

## Sync

Mio supports chunk-based synchronization for multi-device setups. Chunks are gzip-compressed JSONL files stored in `~/.mio/chunks/`.

```bash
# Export new data since last sync
mio sync

# Import pending chunks (from other devices)
mio sync --import

# Check sync status
mio sync --status
```

A manifest file (`~/.mio/chunks/manifest.json`) tracks which chunks have been processed to avoid duplicates.

## Database Schema

SQLite with WAL mode for concurrent reads. Single-writer pattern.

### Tables

- **sessions** — work sessions with project, directory, timestamps, summary
- **observations** — primary memory units with 20+ fields including type, scope, importance, sync_id, topic_key, normalized_hash
- **user_prompts** — archived user prompts linked to sessions
- **relations** — typed links between observations with strength (cascading deletes)
- **search_log** — search analytics (query, result count, latency)

### Indexes

Observations are indexed on: session_id, type, project, created_at, scope, sync_id, topic_key, deleted_at, normalized_hash, importance. Relations are indexed on both from_id and to_id.

## Architecture

```
cmd/mio/main.go          CLI entrypoint and command routing
internal/
  config/config.go        Configuration with defaults and env overrides
  store/store.go          SQLite store: CRUD, search, metrics, import/export
  mcp/mcp.go              MCP stdio server with 15 tools
  server/server.go        HTTP REST API server
  sync/
    sync.go               Chunk-based sync logic with manifest tracking
    transport.go           File transport (gzip-compressed JSONL)
```

## Dependencies

| Package | Purpose |
|---|---|
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO) |
| `github.com/mark3labs/mcp-go` | MCP protocol implementation |
| `github.com/google/uuid` | UUID generation for sync_id |

## Development

```bash
make build       # Compile to ./bin/mio
make test        # Run tests
make clean       # Remove binary and database
make demo        # Run demo with sample data
make run-mcp     # Build and run MCP server
make run-serve   # Build and run HTTP server
make install     # Copy to /usr/local/bin
```

## License

Private project.
