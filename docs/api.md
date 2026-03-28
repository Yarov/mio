# HTTP API Reference

Base URL: `http://localhost:7438`

All responses are JSON. Error responses follow the format `{"error": "message"}`.

---

## Health

### GET /health

```bash
curl http://localhost:7438/health
```

```json
{"status": "ok", "version": "0.1.0"}
```

---

## Observations

### POST /observations

Save a new memory.

```bash
curl -X POST http://localhost:7438/observations \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Fixed auth token validation",
    "type": "bugfix",
    "content": "What: Fixed JWT expiry check\nWhy: Tokens were accepted after expiry\nWhere: internal/auth/jwt.go\nLearned: Always validate exp claim",
    "project": "myapp",
    "importance": 0.8
  }'
```

```json
{"id": 42, "sync_id": "a1b2c3d4-..."}
```

### GET /observations/{id}

```bash
curl http://localhost:7438/observations/42
```

```json
{
  "ID": 42,
  "SyncID": "a1b2c3d4-...",
  "Type": "bugfix",
  "Title": "Fixed auth token validation",
  "Content": "What: Fixed JWT expiry check...",
  "Project": "myapp",
  "Importance": 0.8,
  "AccessCount": 3,
  "CreatedAt": "2026-03-27T10:30:00Z"
}
```

### PUT /observations/{id}

```bash
curl -X PUT http://localhost:7438/observations/42 \
  -H "Content-Type: application/json" \
  -d '{"title": "Updated title", "content": "Updated content"}'
```

```json
{"status": "updated"}
```

### DELETE /observations/{id}

Soft delete (recoverable):

```bash
curl -X DELETE http://localhost:7438/observations/42
```

Hard delete (permanent):

```bash
curl -X DELETE http://localhost:7438/observations/42?hard=true
```

```json
{"status": "deleted"}
```

---

## Search

### GET /search

Full-text search with scoring.

| Param | Description |
|-------|-------------|
| `q` | Search query (required) |
| `project` | Filter by project |
| `type` | Filter by observation type |
| `limit` | Max results |

```bash
curl "http://localhost:7438/search?q=authentication&project=myapp&limit=5"
```

```json
[
  {
    "ID": 42,
    "Type": "bugfix",
    "Title": "Fixed auth token validation",
    "Content": "What: Fixed JWT expiry check...",
    "Score": 3.45,
    "CreatedAt": "2026-03-27T10:30:00Z"
  }
]
```

---

## Context

### GET /context

Get recent observations (for session start context loading).

| Param | Description |
|-------|-------------|
| `project` | Filter by project |
| `limit` | Max results (default 20) |

```bash
curl "http://localhost:7438/context?project=myapp&limit=10"
```

---

## Timeline

### GET /timeline/{id}

Chronological context around an observation.

| Param | Description |
|-------|-------------|
| `before` | Entries before (default 5) |
| `after` | Entries after (default 5) |

```bash
curl "http://localhost:7438/timeline/42?before=3&after=3"
```

```json
[
  {"ID": 40, "Title": "...", "IsFocus": false},
  {"ID": 41, "Title": "...", "IsFocus": false},
  {"ID": 42, "Title": "Fixed auth token validation", "IsFocus": true},
  {"ID": 43, "Title": "...", "IsFocus": false}
]
```

---

## Relations

### GET /relations/{id}

```bash
curl http://localhost:7438/relations/42
```

```json
[
  {
    "ID": 38,
    "Title": "Auth middleware design",
    "RelationType": "caused_by",
    "Strength": 1.0
  }
]
```

---

## Sessions

### POST /sessions

```bash
curl -X POST http://localhost:7438/sessions \
  -H "Content-Type: application/json" \
  -d '{"project": "myapp", "directory": "/home/user/myapp"}'
```

```json
{"session_id": "a1b2c3d4"}
```

### PUT /sessions/{id}/end

```bash
curl -X PUT http://localhost:7438/sessions/a1b2c3d4/end \
  -H "Content-Type: application/json" \
  -d '{"summary": "Goal: Fix auth bugs\nAccomplished: Fixed JWT validation\nFiles: internal/auth/jwt.go"}'
```

```json
{"status": "ended"}
```

### GET /sessions

```bash
curl "http://localhost:7438/sessions?project=myapp&limit=5"
```

---

## Statistics

### GET /stats

```bash
curl http://localhost:7438/stats
```

```json
{
  "TotalObservations": 127,
  "TotalSessions": 23,
  "TotalSearches": 456,
  "SearchHitRate": 0.87,
  "TopProjects": [
    {"Name": "myapp", "Count": 89},
    {"Name": "infra", "Count": 38}
  ],
  "StaleMemoryCount": 5
}
```

---

## Data Management

### GET /export

Export all data as JSON.

```bash
curl http://localhost:7438/export > backup.json
curl "http://localhost:7438/export?project=myapp" > myapp.json
```

### POST /import

Import data from JSON.

```bash
curl -X POST http://localhost:7438/import \
  -H "Content-Type: application/json" \
  -d @backup.json
```

```json
{"status": "imported"}
```

---

## Skills

### GET /skills

```bash
curl http://localhost:7438/skills
```

```json
[
  {"name": "react-19", "description": "React 19 patterns", "category": "coding", "version": "1.0"},
  {"name": "sdd-init", "description": "Initialize SDD context", "category": "sdd", "version": "1.0"}
]
```

### GET /skills/{name}

```bash
curl http://localhost:7438/skills/react-19
```

```json
{"name": "react-19", "content": "---\nname: react-19\n...", "path": "/Users/.../.claude/skills/react-19/SKILL.md"}
```

### PUT /skills/{name}

```bash
curl -X PUT http://localhost:7438/skills/react-19 \
  -H "Content-Type: application/json" \
  -d '{"content": "---\nname: react-19\n---\n\nUpdated content..."}'
```

---

## Agents

### GET /agents

List all supported agents and their status.

```bash
curl http://localhost:7438/agents
```

```json
[
  {"name": "claude-code", "display_name": "Claude Code", "installed": true, "configured": true, "config_path": "~/.claude/mcp/mio.json"},
  {"name": "cursor", "display_name": "Cursor", "installed": true, "configured": false, "config_path": "~/.cursor/mcp.json"}
]
```

### POST /admin/setup?agent={name}

Setup Mio for a specific agent.

```bash
curl -X POST "http://localhost:7438/admin/setup?agent=cursor"
```

```json
{"status": "ok", "agent": "cursor", "output": "Setting up Mio for Cursor...\n  [ok] MCP config → ...", "message": "Setup completed for cursor."}
```

### POST /admin/uninstall?agent={name}

Remove Mio from a specific agent.

```bash
curl -X POST "http://localhost:7438/admin/uninstall?agent=cursor"
```

```json
{"status": "ok", "agent": "cursor", "output": "Uninstalling Mio from Cursor...\n  [ok] Removed MCP config", "message": "Uninstalled Mio from cursor."}
```
