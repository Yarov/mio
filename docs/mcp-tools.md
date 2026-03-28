# MCP Tools Reference

Mio exposes 16 tools via the Model Context Protocol. Agents call these automatically based on the memory protocol instructions.

## mem_save

Save a new memory/observation.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `title` | string | Yes | Brief title with action verb (3-200 chars) |
| `type` | string | Yes | One of: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary` |
| `content` | string | Yes | Structured content (10-50,000 chars) |
| `session_id` | string | No | Current session ID |
| `project` | string | No | Project name |
| `scope` | string | No | `project` (default), `personal`, or `global` |
| `topic_key` | string | No | Stable key for evolving topics (enables upsert) |
| `importance` | float | No | 0.0-1.0, default 0.5 |

**Returns**: `{id, sync_id}`

**Deduplication**: If `topic_key` matches an existing memory, it updates in place. If content hash matches within 15 minutes, it's skipped.

---

## mem_search

Full-text search with temporal decay and importance weighting.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query |
| `project` | string | No | Filter by project |
| `type` | string | No | Filter by observation type |
| `limit` | int | No | Max results (default 20) |

**Returns**: Array of observations with `Score` field, sorted by composite score (FTS rank x importance x recency).

---

## mem_update

Update an existing memory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | int | Yes | Observation ID |
| `title` | string | Yes | New title |
| `content` | string | Yes | New content |

**Returns**: `{status: "updated"}`

---

## mem_delete

Delete a memory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | int | Yes | Observation ID |
| `hard` | bool | No | Permanent delete (default: soft delete) |

**Returns**: `{status: "deleted"}`

---

## mem_get_observation

Get full content of a memory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | int | Yes | Observation ID |

**Returns**: Full observation object.

---

## mem_context

Get recent observations for context loading. Typically called at session start.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project` | string | No | Filter by project |
| `limit` | int | No | Max results (default 20) |

**Returns**: Array of recent observations, most recent first.

---

## mem_timeline

Get chronological context around a specific memory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | int | Yes | Focus observation ID |
| `before` | int | No | Entries before (default 5) |
| `after` | int | No | Entries after (default 5) |

**Returns**: Array of observations with `IsFocus: true` on the target.

---

## mem_session_start

Register a new session.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project` | string | No | Project name |
| `directory` | string | No | Working directory |

**Returns**: `{session_id}`

---

## mem_session_end

End a session with summary.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session_id` | string | Yes | Session ID |
| `summary` | string | No | Session summary (Goal, Accomplished, Discoveries, Next Steps, Files) |

**Returns**: `{status: "ended"}`

---

## mem_session_summary

Get recent sessions with metadata.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project` | string | No | Filter by project |
| `limit` | int | No | Max results (default 10) |

**Returns**: Array of sessions with observation counts.

---

## mem_save_prompt

Archive a user prompt for later reference.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session_id` | string | Yes | Session ID |
| `content` | string | Yes | Prompt content |
| `project` | string | No | Project name |

**Returns**: `{id}`

---

## mem_relations

Get related observations for a memory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | int | Yes | Observation ID |

**Returns**: Array of related observations with relation type and strength.

---

## mem_relate

Create a relation between two memories.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `from_id` | int | Yes | Source observation ID |
| `to_id` | int | Yes | Target observation ID |
| `type` | string | Yes | `supersedes`, `relates_to`, `contradicts`, `builds_on`, `caused_by`, `resolved_by` |
| `strength` | float | No | 0.0-1.0 (default 1.0) |

**Returns**: `{status: "related"}`

---

## mem_suggest_topic_key

Generate a stable topic key for evolving topics.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `title` | string | Yes | Observation title |
| `content` | string | Yes | Observation content |

**Returns**: `{topic_key}` — a normalized, stable key derived from the content.

---

## mem_stats

Get memory system statistics.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| *(none)* | | | |

**Returns**: `{TotalObservations, TotalSessions, TotalSearches, SearchHitRate, TopProjects, StaleMemoryCount}`
