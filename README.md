# Mio

**Persistent memory for AI coding agents.**

Mio gives your AI agents the ability to remember decisions, discoveries, and context across sessions. It works as an [MCP server](https://modelcontextprotocol.io) that any compatible agent can connect to, plus an HTTP dashboard and TUI for humans.

```
mio setup cursor        # one command, any agent
```

> Built with Go. Single binary. SQLite storage. Zero dependencies at runtime.

---

## Why Mio?

Every time you start a new session with an AI coding agent, it starts from zero. Mio fixes that:

- **Decisions persist** - Architecture choices, conventions, and tradeoffs survive across sessions
- **Bugs don't repeat** - Root causes and fixes are remembered
- **Context is automatic** - The agent loads relevant memories at session start
- **Works with any agent** - One memory store shared across Claude Code, Cursor, Gemini CLI, Codex CLI, and more

## Supported Agents

| Agent | MCP | Skills | Protocol | Status |
|-------|-----|--------|----------|--------|
| [Claude Code](https://claude.ai/code) | Own file | Yes | CLAUDE.md | Full support |
| [Cursor](https://cursor.sh) | Shared JSON | Yes | Rules | Full support |
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) | Shared JSON | Yes | GEMINI.md | Full support |
| [Codex CLI](https://github.com/openai/codex) | Shared JSON | Yes | agents.md | Full support |
| [VS Code Copilot](https://github.com/features/copilot) | Shared JSON | Yes | Instructions | Full support |
| [OpenCode](https://github.com/opencode-ai/opencode) | Shared JSON | - | - | MCP only |
| [Continue.dev](https://continue.dev) | Shared JSON | - | - | MCP only |
| [Kilo Code](https://kilocode.ai) | Shared JSON | - | - | MCP only |

## Quick Start

### Install

**macOS / Linux** (recommended):
```bash
curl -fsSL https://raw.githubusercontent.com/Yarov/mio/main/install.sh | sh
```

**Homebrew** (macOS / Linux):
```bash
brew install yarov/tap/mio
```

**Windows** (PowerShell):
```powershell
irm https://raw.githubusercontent.com/Yarov/mio/main/install.ps1 | iex
```

**From source** (requires Go 1.25+):
```bash
git clone https://github.com/Yarov/mio.git
cd mio
make install
```

**Manual download**: Grab a binary from [GitHub Releases](https://github.com/Yarov/mio/releases).

### Setup

```bash
# Configure for your agent (auto-detects installed agents)
mio setup                    # default: claude-code
mio setup cursor             # specific agent
mio setup --all              # all detected agents
mio setup --list             # show agents and status
```

Setup does everything automatically:
1. Registers Mio as MCP server in the agent's config
2. Installs the memory protocol (instructions for the agent)
3. Copies skills (reusable AI prompts)
4. Starts the HTTP dashboard (macOS: via launchd, auto-start on login)

### Verify

```bash
# Check the dashboard
open http://localhost:7438

# Check agent status
mio setup --list
```

### Restart your agent

After setup, restart your AI agent. Mio will be available as an MCP server and the agent will automatically start saving and searching memories.

---

## How It Works

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Claude Code  │     │   Cursor    │     │ Gemini CLI  │
│   (MCP)      │     │   (MCP)     │     │   (MCP)     │
└──────┬───────┘     └──────┬──────┘     └──────┬──────┘
       │                    │                    │
       └────────────┬───────┘────────────────────┘
                    │
              ┌─────▼─────┐
              │  mio mcp   │  MCP stdio server
              └─────┬──────┘
                    │
              ┌─────▼──────┐
              │  SQLite DB  │  ~/.mio/mio.db
              └─────┬──────┘
                    │
              ┌─────▼──────┐
              │ mio server  │  HTTP dashboard + API
              └────────────┘  http://localhost:7438
```

Each agent connects to Mio via MCP (stdio). Mio stores everything in a single SQLite database with full-text search (FTS5). The HTTP server runs as an independent process for the dashboard and REST API.

### What gets remembered

The agent automatically saves:
- **Decisions** - Architecture choices, tool selections, conventions
- **Bug fixes** - Root cause and solution
- **Discoveries** - Non-obvious codebase behaviors, gotchas
- **Patterns** - Naming conventions, project structure
- **Preferences** - How you like to work

### Memory format

Each memory is structured:
```
What: [what was done]
Why:  [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
```

---

## CLI Reference

### Memory Operations

```bash
# Save a memory directly
mio save "Fixed auth bug" "Root cause was expired JWT validation" --type bugfix --project myapp

# Search memories
mio search "authentication" --project myapp --limit 10

# Get recent context
mio context --project myapp --limit 20

# Show timeline around a memory
mio timeline 42 --before 5 --after 5

# View statistics
mio stats
```

### Agent Management

```bash
# Setup
mio setup                    # claude-code (default)
mio setup cursor             # specific agent
mio setup --all              # all detected agents
mio setup --list             # show status

# Uninstall
mio uninstall cursor         # specific agent
mio uninstall --all          # all agents
mio uninstall --purge        # also delete ~/.mio data
```

### Server

```bash
# Start HTTP dashboard + API (default port 7438)
mio server
mio server 8080              # custom port

# Start MCP server (used by agents, not typically run manually)
mio mcp
```

### Data Management

```bash
# Export/import
mio export backup.json
mio export --project myapp backup.json
mio import backup.json

# Cross-device sync
mio sync                     # export new chunk
mio sync --import            # import pending chunks
mio sync --status            # show sync status
```

### Interactive

```bash
# Terminal UI
mio tui
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MIO_DATA_DIR` | `~/.mio` | Data directory (database, chunks, logs) |

---

## MCP Tools

Mio exposes 16 MCP tools that agents use automatically:

| Tool | Description |
|------|-------------|
| `mem_save` | Save a memory with title, type, content, importance |
| `mem_search` | Full-text search with temporal decay scoring |
| `mem_update` | Update an existing memory |
| `mem_delete` | Soft or hard delete a memory |
| `mem_get_observation` | Get full memory by ID |
| `mem_context` | Get recent memories for context loading |
| `mem_timeline` | Chronological window around a memory |
| `mem_session_start` | Register a new session |
| `mem_session_end` | End session with summary |
| `mem_session_summary` | List recent sessions |
| `mem_save_prompt` | Archive a user prompt |
| `mem_relations` | Get related memories |
| `mem_relate` | Create a relation (supersedes, caused_by, etc.) |
| `mem_suggest_topic_key` | Generate stable key for evolving topics |
| `mem_stats` | Get system statistics |

See [docs/mcp-tools.md](docs/mcp-tools.md) for full parameter documentation.

---

## HTTP API

Base URL: `http://localhost:7438`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/` | Dashboard (HTML) |
| `GET` | `/health` | Health check |
| `GET` | `/agents` | List agents and status |
| `POST` | `/observations` | Save memory |
| `GET` | `/observations/{id}` | Get memory |
| `PUT` | `/observations/{id}` | Update memory |
| `DELETE` | `/observations/{id}` | Delete memory (`?hard=true`) |
| `GET` | `/search?q=...` | Search memories |
| `GET` | `/context` | Recent context |
| `GET` | `/timeline/{id}` | Timeline view |
| `GET` | `/relations/{id}` | Related memories |
| `POST` | `/sessions` | Create session |
| `PUT` | `/sessions/{id}/end` | End session |
| `GET` | `/sessions` | List sessions |
| `GET` | `/stats` | Statistics |
| `GET` | `/export` | Export all data |
| `POST` | `/import` | Import data |
| `GET` | `/skills` | List skills |
| `GET` | `/skills/{name}` | Get skill content |
| `PUT` | `/skills/{name}` | Update skill |
| `POST` | `/admin/setup?agent=...` | Setup agent |
| `POST` | `/admin/uninstall?agent=...` | Remove agent |

See [docs/api.md](docs/api.md) for request/response examples.

---

## Dashboard

The web dashboard at `http://localhost:7438` provides:

- **Overview** - Memory count, sessions, search hit rate, top projects
- **Memories** - Browse and search all stored memories
- **Sessions** - View session history and summaries
- **Skills** - Browse installed skills by category
- **Agents** - See which agents are installed/configured, setup or remove with one click
- **Admin** - Reinstall, export data, API reference

The dashboard runs as a persistent process (launchd on macOS) independent of any agent session.

---

## Skills

Mio ships with 27+ pre-built skills for common development tasks:

| Category | Skills |
|----------|--------|
| **SDD Pipeline** | sdd-init, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-apply, sdd-verify, sdd-archive, sdd-continue, sdd-ff |
| **Orchestrator** | mio-architect |
| **Engineering** | github-pr, pr-review, technical-review |
| **Coding** | react-19, typescript, nextjs-15, tailwind-4, zod-4, zustand-5, playwright, pytest, django-drf, ai-sdk-5 |
| **Meta** | skill-creator, skill-registry, find-skills |

Skills are markdown files with frontmatter that provide domain-specific instructions to agents. They are installed to each agent's skills directory during setup.

---

## Architecture

See [docs/architecture.md](docs/architecture.md) for full technical details.

```
mio/
├── cmd/mio/          # CLI entry point
├── internal/
│   ├── agents/       # Agent registry + 8 implementations
│   ├── config/       # Configuration
│   ├── mcp/          # MCP stdio server (16 tools)
│   ├── server/       # HTTP server + dashboard
│   ├── setup/        # Setup dispatcher
│   ├── store/        # SQLite store + FTS5 search
│   ├── sync/         # Cross-device sync
│   └── tui/          # Terminal UI (Bubble Tea)
├── protocols/        # Agent protocol templates
├── skills/           # Pre-built skills (27+)
├── docs/             # Documentation
└── Makefile
```

---

## Uninstall

```bash
# Remove from a specific agent
mio uninstall cursor

# Remove from all agents
mio uninstall --all

# Remove everything including data
mio uninstall --all --purge
```

---

## Development

```bash
# Build
make build

# Run tests
make test

# Run locally
make run-serve    # HTTP dashboard
make run-tui      # Terminal UI
make run-mcp      # MCP server

# Clean
make clean
```

### Requirements

- Go 1.25+
- macOS, Linux, or Windows

---

## License

MIT
