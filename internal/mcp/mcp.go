package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mio/internal/config"
	"mio/internal/store"
)

type Server struct {
	store  *store.Store
	cfg    *config.Config
	mcp    *server.MCPServer
}

func New(s *store.Store, cfg *config.Config) *Server {
	srv := &Server{store: s, cfg: cfg}
	srv.mcp = server.NewMCPServer(
		"mio",
		"0.1.0",
		server.WithToolCapabilities(true),
	)
	srv.registerTools()
	return srv
}

func (s *Server) ServeStdio() error {
	stdio := server.NewStdioServer(s.mcp)
	return stdio.Listen(context.Background(), os.Stdin, os.Stdout)
}

func (s *Server) registerTools() {
	// Tool loading strategy:
	// EAGER (always in context): mem_save, mem_search, mem_get_observation, mem_context, mem_session_summary
	//   → Core tools needed every session
	// DEFERRED (loaded on-demand via ToolSearch): mem_update, mem_delete, mem_timeline,
	//   mem_session_start, mem_session_end, mem_save_prompt, mem_relations, mem_relate,
	//   mem_suggest_topic_key, mem_stats
	//   → Rarely used tools, ~40% token reduction at session startup

	// --- Eager tools (always available) ---

	// mem_save
	s.mcp.AddTool(
		mcp.NewTool("mem_save",
			mcp.WithDescription("Save a memory/observation. Fields: title (action verb + what), type (bugfix|decision|architecture|discovery|pattern|config|preference|learning), content (structured What/Why/Where/Learned), session_id, project, scope (project|personal|global), topic_key (for evolving topics), importance (0.0-1.0)"),
			mcp.WithString("title", mcp.Required(), mcp.Description("Brief title: action verb + what")),
			mcp.WithString("type", mcp.Required(), mcp.Description("Type: bugfix, decision, architecture, discovery, pattern, config, preference, learning")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Structured content with What/Why/Where/Learned")),
			mcp.WithString("session_id", mcp.Description("Current session ID")),
			mcp.WithString("project", mcp.Description("Project name")),
			mcp.WithString("scope", mcp.Description("Scope: project, personal, or global")),
			mcp.WithString("topic_key", mcp.Description("Stable key for evolving topics (enables upsert)")),
			mcp.WithNumber("importance", mcp.Description("Importance 0.0-1.0, default 0.5")),
			mcp.WithString("agent", mcp.Description("Agent name that created this memory (e.g. claude-code, cursor)")),
		),
		s.handleSave,
	)

	// mem_search
	s.mcp.AddTool(
		mcp.NewTool("mem_search",
			mcp.WithDescription("Search memories using full-text search with temporal decay and importance weighting"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithString("project", mcp.Description("Filter by project")),
			mcp.WithString("type", mcp.Description("Filter by observation type")),
			mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
		),
		s.handleSearch,
	)

	// --- Deferred tools (loaded on-demand via ToolSearch to reduce session startup tokens) ---

	// mem_update
	s.mcp.AddTool(
		mcp.NewTool("mem_update",
			mcp.WithDescription("Update an existing memory by ID"),
			mcp.WithDeferLoading(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Observation ID")),
			mcp.WithString("title", mcp.Required(), mcp.Description("New title")),
			mcp.WithString("content", mcp.Required(), mcp.Description("New content")),
		),
		s.handleUpdate,
	)

	// mem_delete
	s.mcp.AddTool(
		mcp.NewTool("mem_delete",
			mcp.WithDescription("Delete a memory (soft delete by default)"),
			mcp.WithDeferLoading(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Observation ID")),
			mcp.WithBoolean("hard", mcp.Description("Hard delete (permanent)")),
		),
		s.handleDelete,
	)

	// mem_get_observation
	s.mcp.AddTool(
		mcp.NewTool("mem_get_observation",
			mcp.WithDescription("Get full content of a memory by ID"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Observation ID")),
		),
		s.handleGetObservation,
	)

	// mem_context
	s.mcp.AddTool(
		mcp.NewTool("mem_context",
			mcp.WithDescription("Get recent observations for context"),
			mcp.WithString("project", mcp.Description("Filter by project")),
			mcp.WithNumber("limit", mcp.Description("Max results")),
		),
		s.handleContext,
	)

	// mem_timeline
	s.mcp.AddTool(
		mcp.NewTool("mem_timeline",
			mcp.WithDescription("Get chronological timeline around an observation"),
			mcp.WithDeferLoading(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Focus observation ID")),
			mcp.WithNumber("before", mcp.Description("Entries before (default 5)")),
			mcp.WithNumber("after", mcp.Description("Entries after (default 5)")),
		),
		s.handleTimeline,
	)

	// mem_session_start
	s.mcp.AddTool(
		mcp.NewTool("mem_session_start",
			mcp.WithDescription("Register a new session"),
			mcp.WithDeferLoading(true),
			mcp.WithString("project", mcp.Description("Project name")),
			mcp.WithString("directory", mcp.Description("Working directory")),
		),
		s.handleSessionStart,
	)

	// mem_session_end
	s.mcp.AddTool(
		mcp.NewTool("mem_session_end",
			mcp.WithDescription("End a session with optional summary"),
			mcp.WithDeferLoading(true),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
			mcp.WithString("summary", mcp.Description("Session summary")),
		),
		s.handleSessionEnd,
	)

	// mem_session_summary
	s.mcp.AddTool(
		mcp.NewTool("mem_session_summary",
			mcp.WithDescription("Get recent sessions with observation counts"),
			mcp.WithString("project", mcp.Description("Filter by project")),
			mcp.WithNumber("limit", mcp.Description("Max results (default 10)")),
		),
		s.handleSessionSummary,
	)

	// mem_save_prompt
	s.mcp.AddTool(
		mcp.NewTool("mem_save_prompt",
			mcp.WithDescription("Archive a user prompt"),
			mcp.WithDeferLoading(true),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Prompt content")),
			mcp.WithString("project", mcp.Description("Project name")),
		),
		s.handleSavePrompt,
	)

	// mem_relations
	s.mcp.AddTool(
		mcp.NewTool("mem_relations",
			mcp.WithDescription("Get related observations for a memory"),
			mcp.WithDeferLoading(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Observation ID")),
		),
		s.handleRelations,
	)

	// mem_relate
	s.mcp.AddTool(
		mcp.NewTool("mem_relate",
			mcp.WithDescription("Create a relation between two memories"),
			mcp.WithDeferLoading(true),
			mcp.WithNumber("from_id", mcp.Required(), mcp.Description("Source observation ID")),
			mcp.WithNumber("to_id", mcp.Required(), mcp.Description("Target observation ID")),
			mcp.WithString("type", mcp.Required(), mcp.Description("Relation type: supersedes, relates_to, contradicts, builds_on, caused_by, resolved_by")),
			mcp.WithNumber("strength", mcp.Description("Relation strength 0.0-1.0 (default 1.0)")),
		),
		s.handleRelate,
	)

	// mem_suggest_topic_key
	s.mcp.AddTool(
		mcp.NewTool("mem_suggest_topic_key",
			mcp.WithDescription("Generate a stable topic key from title and content"),
			mcp.WithDeferLoading(true),
			mcp.WithString("title", mcp.Required(), mcp.Description("Observation title")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Observation content")),
		),
		s.handleSuggestTopicKey,
	)

	// mem_stats
	s.mcp.AddTool(
		mcp.NewTool("mem_stats",
			mcp.WithDescription("Get memory system statistics and metrics"),
			mcp.WithDeferLoading(true),
		),
		s.handleStats,
	)

	// --- Innovation tools ---

	// mem_surface (Proactive Memory Surfacing) — eager, auto-invoked by hooks
	s.mcp.AddTool(
		mcp.NewTool("mem_surface",
			mcp.WithDescription("Proactively surface relevant memories based on current context text. Returns top matches the agent should be reminded about. Project is auto-detected from working directory if not specified."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Current context/prompt text to find relevant memories for")),
			mcp.WithString("project", mcp.Description("Filter by project (auto-detected from cwd if empty)")),
			mcp.WithNumber("limit", mcp.Description("Max results (default 5)")),
		),
		s.handleSurface,
	)

	// mem_cross_project (Cross-Project Knowledge) — deferred
	s.mcp.AddTool(
		mcp.NewTool("mem_cross_project",
			mcp.WithDescription("Search memories across ALL projects, prioritizing global and personal scope. Use when knowledge from other projects may be relevant."),
			mcp.WithDeferLoading(true),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithNumber("limit", mcp.Description("Max results (default 10)")),
		),
		s.handleCrossProject,
	)

	// mem_consolidate (Memory Consolidation) — deferred admin tool
	s.mcp.AddTool(
		mcp.NewTool("mem_consolidate",
			mcp.WithDescription("Consolidate duplicate/overlapping memories by topic_key. Merges related observations into compact knowledge nuggets."),
			mcp.WithDeferLoading(true),
			mcp.WithString("project", mcp.Description("Project to consolidate (all if empty)")),
		),
		s.handleConsolidate,
	)

	// mem_gc (Garbage Collection) — deferred admin tool
	s.mcp.AddTool(
		mcp.NewTool("mem_gc",
			mcp.WithDescription("Run memory decay and garbage collection. Reduces importance of stale memories and archives dead ones."),
			mcp.WithDeferLoading(true),
			mcp.WithNumber("stale_days", mcp.Description("Days without access to consider stale (default 60)")),
			mcp.WithNumber("archive_threshold", mcp.Description("Importance below which memories get archived (default 0.1)")),
		),
		s.handleGC,
	)

	// mem_graph (Decision Graph) — deferred
	s.mcp.AddTool(
		mcp.NewTool("mem_graph",
			mcp.WithDescription("Build a decision graph from relations around a focal observation. Visualizes how decisions connect."),
			mcp.WithDeferLoading(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Focal observation ID")),
			mcp.WithNumber("depth", mcp.Description("Max traversal depth (default 3)")),
		),
		s.handleGraph,
	)

	// mem_enhanced_search (Enhanced Search) — deferred
	s.mcp.AddTool(
		mcp.NewTool("mem_enhanced_search",
			mcp.WithDescription("Enhanced search with scope-aware boosting, revision value, and agent diversity scoring"),
			mcp.WithDeferLoading(true),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithString("project", mcp.Description("Filter by project")),
			mcp.WithString("type", mcp.Description("Filter by type")),
			mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
		),
		s.handleEnhancedSearch,
	)

	// mem_agent_knowledge (Agent Collaboration) — deferred
	s.mcp.AddTool(
		mcp.NewTool("mem_agent_knowledge",
			mcp.WithDescription("Get what a specific agent has learned, or get contributions by all agents for a project"),
			mcp.WithDeferLoading(true),
			mcp.WithString("agent", mcp.Description("Agent name to query (empty = all agents summary)")),
			mcp.WithString("project", mcp.Description("Filter by project")),
			mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
		),
		s.handleAgentKnowledge,
	)
}

// --- Handlers ---

func (s *Server) handleSave(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	obs := &store.Observation{
		Title:      strArg(request, "title"),
		Type:       strArg(request, "type"),
		Content:    strArg(request, "content"),
		SessionID:  strArg(request, "session_id"),
		Scope:      strArg(request, "scope"),
		Importance: numArgDefault(request, "importance", 0.5),
		Agent:      strArg(request, "agent"),
	}

	if proj := strArg(request, "project"); proj != "" {
		obs.Project = &proj
	}
	if tk := strArg(request, "topic_key"); tk != "" {
		obs.TopicKey = &tk
	}
	if tn := strArg(request, "tool_name"); tn != "" {
		obs.ToolName = &tn
	}
	if obs.Scope == "" {
		obs.Scope = "project"
	}
	if obs.Agent == "" {
		obs.Agent = "claude-code"
	}

	id, err := s.store.Save(obs)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Saved observation #%d (sync_id: %s)", id, obs.SyncID)), nil
}

func (s *Server) handleSearch(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := strArg(request, "query")
	project := strArg(request, "project")
	obsType := strArg(request, "type")
	limit := intArg(request, "limit", 20)

	results, err := s.store.Search(query, project, obsType, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No results found."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Search Results\n\n> **%d** matches found\n\n---\n\n", len(results)))
	for i, r := range results {
		topic := ""
		if r.TopicKey != nil {
			topic = fmt.Sprintf(" `%s`", *r.TopicKey)
		}
		preview := truncate(r.Content, 300)
		sb.WriteString(fmt.Sprintf("### #%d — %s\n\n", r.ID, r.Title))
		sb.WriteString(fmt.Sprintf("| Type | Score | Accessed | Created |\n"))
		sb.WriteString(fmt.Sprintf("|------|-------|----------|---------|\n"))
		sb.WriteString(fmt.Sprintf("| `%s`%s | **%.2f** | %d times | %s |\n\n", r.Type, topic, r.Score, r.AccessCount, r.CreatedAt))
		sb.WriteString(fmt.Sprintf("> %s\n", preview))
		if i < len(results)-1 {
			sb.WriteString("\n---\n\n")
		} else {
			sb.WriteString("\n")
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleUpdate(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int64(intArg(request, "id", 0))
	title := strArg(request, "title")
	content := strArg(request, "content")

	if err := s.store.UpdateObservation(id, title, content); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Updated observation #%d", id)), nil
}

func (s *Server) handleDelete(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int64(intArg(request, "id", 0))
	hard := boolArg(request, "hard")

	var err error
	if hard {
		err = s.store.HardDelete(id)
	} else {
		err = s.store.SoftDelete(id)
	}
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	action := "soft-deleted"
	if hard {
		action = "permanently deleted"
	}
	return mcp.NewToolResultText(fmt.Sprintf("Observation #%d %s", id, action)), nil
}

func (s *Server) handleGetObservation(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int64(intArg(request, "id", 0))
	obs, err := s.store.GetObservation(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("observation #%d not found: %v", id, err)), nil
	}

	data, _ := json.MarshalIndent(obs, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleContext(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := strArg(request, "project")
	limit := intArg(request, "limit", 20)

	obs, err := s.store.RecentContext(project, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(obs) == 0 {
		return mcp.NewToolResultText("No recent context available."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Recent Context\n\n> **%d** observations\n\n---\n\n", len(obs)))
	sb.WriteString("| # | Type | Title | Date |\n")
	sb.WriteString("|---|------|-------|------|\n")
	for _, o := range obs {
		date := o.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		sb.WriteString(fmt.Sprintf("| **%d** | `%s` | %s | %s |\n", o.ID, o.Type, o.Title, date))
	}
	sb.WriteString("\n---\n\n")
	// Detail section with previews
	for _, o := range obs {
		preview := truncate(o.Content, 200)
		sb.WriteString(fmt.Sprintf("**#%d** — %s\n> %s\n\n", o.ID, o.Title, preview))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleTimeline(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int64(intArg(request, "id", 0))
	before := intArg(request, "before", 5)
	after := intArg(request, "after", 5)

	entries, err := s.store.Timeline(id, before, after)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Timeline\n\n---\n\n")
	for _, e := range entries {
		preview := truncate(e.Content, 150)
		date := e.CreatedAt
		if len(date) >= 16 {
			date = date[:16]
		}
		if e.IsFocus {
			sb.WriteString(fmt.Sprintf("### >>> #%d — %s\n\n", e.ID, e.Title))
			sb.WriteString(fmt.Sprintf("| Type | Date |\n|------|------|\n| `%s` | %s |\n\n", e.Type, date))
			sb.WriteString(fmt.Sprintf("> **%s**\n\n", preview))
		} else {
			sb.WriteString(fmt.Sprintf("**#%d** `%s` %s — _%s_\n> %s\n\n", e.ID, e.Type, e.Title, date, preview))
		}
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleSessionStart(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := uuid.New().String()
	project := strArg(request, "project")
	directory := strArg(request, "directory")

	if err := s.store.CreateSession(id, project, directory); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Session started: %s", id)), nil
}

func (s *Server) handleSessionEnd(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := strArg(request, "session_id")
	summary := strArg(request, "summary")

	var summaryPtr *string
	if summary != "" {
		summaryPtr = &summary
	}

	if err := s.store.EndSession(sessionID, summaryPtr); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Session ended: %s", sessionID)), nil
}

func (s *Server) handleSessionSummary(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := strArg(request, "project")
	limit := intArg(request, "limit", 10)

	sessions, err := s.store.RecentSessions(project, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(sessions) == 0 {
		return mcp.NewToolResultText("No sessions found."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Sessions\n\n> **%d** recent sessions\n\n---\n\n", len(sessions)))
	sb.WriteString("| Session | Status | Project | Memories | Started |\n")
	sb.WriteString("|---------|--------|---------|----------|---------|\n")
	for _, sess := range sessions {
		status := "active"
		if sess.EndedAt != nil {
			status = "ended"
		}
		started := sess.StartedAt
		if len(started) >= 10 {
			started = started[:10]
		}
		sb.WriteString(fmt.Sprintf("| `%s` | **%s** | %s | %d | %s |\n",
			sess.ID[:8], status, sess.Project, sess.ObservationCount, started))
	}
	sb.WriteString("\n---\n\n")
	// Summaries section
	hasSummaries := false
	for _, sess := range sessions {
		if sess.Summary != nil {
			if !hasSummaries {
				sb.WriteString("### Summaries\n\n")
				hasSummaries = true
			}
			sb.WriteString(fmt.Sprintf("**`%s`** (%s)\n> %s\n\n", sess.ID[:8], sess.Project, truncate(*sess.Summary, 200)))
		}
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleSavePrompt(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := strArg(request, "session_id")
	content := strArg(request, "content")
	project := strArg(request, "project")

	id, err := s.store.SavePrompt(sessionID, content, project)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Prompt saved #%d", id)), nil
}

func (s *Server) handleRelations(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int64(intArg(request, "id", 0))

	related, err := s.store.GetRelatedObservations(id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(related) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No related observations for #%d", id)), nil
	}

	rels, _ := s.store.GetRelations(id)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Relations for #%d\n\n> **%d** connected memories\n\n---\n\n", id, len(related)))

	relMap := map[int64]string{}
	for _, r := range rels {
		other := r.ToID
		if r.ToID == id {
			other = r.FromID
		}
		relMap[other] = r.Type
	}

	sb.WriteString("| Relation | # | Type | Title |\n")
	sb.WriteString("|----------|---|------|-------|\n")
	for _, o := range related {
		relType := relMap[o.ID]
		sb.WriteString(fmt.Sprintf("| `%s` | **%d** | `%s` | %s |\n", relType, o.ID, o.Type, o.Title))
	}
	sb.WriteString("\n---\n\n")
	for _, o := range related {
		preview := truncate(o.Content, 150)
		sb.WriteString(fmt.Sprintf("**#%d** — %s\n> %s\n\n", o.ID, o.Title, preview))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleRelate(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fromID := int64(intArg(request, "from_id", 0))
	toID := int64(intArg(request, "to_id", 0))
	relType := strArg(request, "type")
	strength := numArgDefault(request, "strength", 1.0)

	validRelTypes := map[string]bool{
		"supersedes": true, "relates_to": true, "contradicts": true,
		"builds_on": true, "caused_by": true, "resolved_by": true,
	}
	if !validRelTypes[relType] {
		return mcp.NewToolResultError(fmt.Sprintf("invalid relation type: %s", relType)), nil
	}

	if err := s.store.CreateRelation(fromID, toID, relType, strength); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Created relation: #%d -[%s]-> #%d (strength: %.1f)", fromID, relType, toID, strength)), nil
}

func (s *Server) handleSuggestTopicKey(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := strArg(request, "title")
	content := strArg(request, "content")

	if title == "" || content == "" {
		return mcp.NewToolResultError("both title and content are required"), nil
	}

	// Generate a human-readable topic key from the title
	words := strings.Fields(strings.ToLower(title))
	if len(words) > 5 {
		words = words[:5]
	}
	key := strings.Join(words, "-")
	// Remove non-alphanumeric except hyphens
	key = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, key)

	// Add a short hash suffix for uniqueness
	timestamp := time.Now().Format("0601")
	key = fmt.Sprintf("%s-%s", key, timestamp)

	return mcp.NewToolResultText(fmt.Sprintf("Suggested topic key: %s", key)), nil
}

func (s *Server) handleStats(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	metrics, err := s.store.GetMetrics()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Mio Stats\n\n---\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Observations | **%d** |\n", metrics.TotalObservations))
	sb.WriteString(fmt.Sprintf("| Sessions | **%d** |\n", metrics.TotalSessions))
	sb.WriteString(fmt.Sprintf("| Searches | **%d** |\n", metrics.TotalSearches))
	sb.WriteString(fmt.Sprintf("| Search Hit Rate | **%.1f%%** |\n", metrics.SearchHitRate*100))
	sb.WriteString(fmt.Sprintf("| Avg Search Latency | **%dms** |\n", metrics.AvgSearchLatencyMs))
	sb.WriteString(fmt.Sprintf("| Stale Memories | **%d** |\n", metrics.StaleMemoryCount))
	if len(metrics.TopProjects) > 0 {
		sb.WriteString("\n---\n\n### Top Projects\n\n")
		sb.WriteString("| Project | Memories |\n")
		sb.WriteString("|---------|----------|\n")
		for _, p := range metrics.TopProjects {
			sb.WriteString(fmt.Sprintf("| `%s` | %d |\n", p.Project, p.Count))
		}
	}
	sb.WriteString("\n")
	return mcp.NewToolResultText(sb.String()), nil
}

// --- Innovation handlers ---

func (s *Server) handleSurface(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text := strArg(request, "text")
	project := strArg(request, "project")
	limit := intArg(request, "limit", 5)

	// Auto-detect project from working directory if not specified
	if project == "" {
		if cwd, err := os.Getwd(); err == nil {
			project = inferProjectName(cwd)
		}
	}

	results, err := s.store.SurfaceRelevant(text, project, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No relevant memories to surface."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Surfaced Memories\n\n> **%d** relevant matches\n\n---\n\n", len(results)))
	for i, r := range results {
		preview := truncate(r.Content, 300)
		sb.WriteString(fmt.Sprintf("### #%d — %s\n\n", r.ID, r.Title))
		sb.WriteString("| Type | Relevance |")
		if r.Agent != "" {
			sb.WriteString(" Agent |")
		}
		sb.WriteString("\n|------|-----------|")
		if r.Agent != "" {
			sb.WriteString("-------|")
		}
		sb.WriteString(fmt.Sprintf("\n| `%s` | **%.2f** |", r.Type, r.Score))
		if r.Agent != "" {
			sb.WriteString(fmt.Sprintf(" `%s` |", r.Agent))
		}
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("> %s\n", preview))
		// Show connected memories (relations)
		if rels, err := s.store.GetRelations(r.ID); err == nil && len(rels) > 0 {
			sb.WriteString("\n**Related:** ")
			relStrs := make([]string, 0, len(rels))
			for _, rel := range rels {
				otherID := rel.ToID
				if otherID == r.ID {
					otherID = rel.FromID
				}
				relStrs = append(relStrs, fmt.Sprintf("`#%d` (%s)", otherID, rel.Type))
			}
			sb.WriteString(strings.Join(relStrs, " · "))
			sb.WriteString("\n")
		}
		if i < len(results)-1 {
			sb.WriteString("\n---\n\n")
		} else {
			sb.WriteString("\n")
		}
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleCrossProject(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := strArg(request, "query")
	limit := intArg(request, "limit", 10)

	results, err := s.store.CrossProjectSearch(query, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No cross-project results found."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Cross-Project Search\n\n> **%d** results across projects\n\n---\n\n", len(results)))
	sb.WriteString("| # | Project | Type | Score | Title |\n")
	sb.WriteString("|---|---------|------|-------|-------|\n")
	for _, r := range results {
		project := "—"
		if r.Project != nil {
			project = *r.Project
		}
		sb.WriteString(fmt.Sprintf("| **%d** | `%s` | `%s` | %.2f | %s |\n", r.ID, project, r.Type, r.Score, r.Title))
	}
	sb.WriteString("\n---\n\n")
	for _, r := range results {
		preview := truncate(r.Content, 250)
		project := "—"
		if r.Project != nil {
			project = *r.Project
		}
		sb.WriteString(fmt.Sprintf("**#%d** [%s] — %s\n> %s\n\n", r.ID, project, r.Title, preview))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleConsolidate(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := strArg(request, "project")

	count, err := s.store.Consolidate(project)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Consolidated %d memory groups", count)), nil
}

func (s *Server) handleGC(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	staleDays := intArg(request, "stale_days", 60)
	threshold := numArgDefault(request, "archive_threshold", 0.1)

	decayed, archived, err := s.store.DecayAndGC(staleDays, threshold)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("GC complete: %d memories decayed, %d archived", decayed, archived)), nil
}

func (s *Server) handleGraph(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := int64(intArg(request, "id", 0))
	depth := intArg(request, "depth", 3)

	graph, err := s.store.BuildDecisionGraph(id, depth)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data, _ := json.MarshalIndent(graph, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleEnhancedSearch(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := strArg(request, "query")
	project := strArg(request, "project")
	obsType := strArg(request, "type")
	limit := intArg(request, "limit", 20)

	results, err := s.store.EnhancedSearch(query, project, obsType, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No results found."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Enhanced Search\n\n> **%d** results (TF-IDF ranked)\n\n---\n\n", len(results)))
	for i, r := range results {
		preview := truncate(r.Content, 300)
		agent := ""
		if r.Agent != "" {
			agent = fmt.Sprintf(" | Agent: `%s`", r.Agent)
		}
		sb.WriteString(fmt.Sprintf("### #%d — %s\n\n", r.ID, r.Title))
		sb.WriteString("| Type | Score | Revisions | Scope |\n")
		sb.WriteString("|------|-------|-----------|-------|\n")
		sb.WriteString(fmt.Sprintf("| `%s` | **%.2f** | %d | `%s` |\n\n", r.Type, r.Score, r.RevisionCount, r.Scope))
		if agent != "" {
			sb.WriteString(fmt.Sprintf("> Agent: `%s`\n\n", r.Agent))
		}
		sb.WriteString(fmt.Sprintf("> %s\n", preview))
		if i < len(results)-1 {
			sb.WriteString("\n---\n\n")
		} else {
			sb.WriteString("\n")
		}
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleAgentKnowledge(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	agentName := strArg(request, "agent")
	project := strArg(request, "project")
	limit := intArg(request, "limit", 20)

	if agentName != "" {
		// Single agent knowledge
		results, err := s.store.AgentKnowledge(agentName, limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if len(results) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No knowledge found for agent '%s'", agentName)), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Agent Knowledge: `%s`\n\n> **%d** memories\n\n---\n\n", agentName, len(results)))
		sb.WriteString("| # | Project | Type | Title |\n")
		sb.WriteString("|---|---------|------|-------|\n")
		for _, o := range results {
			proj := "—"
			if o.Project != nil {
				proj = *o.Project
			}
			sb.WriteString(fmt.Sprintf("| **%d** | `%s` | `%s` | %s |\n", o.ID, proj, o.Type, o.Title))
		}
		sb.WriteString("\n---\n\n")
		for _, o := range results {
			preview := truncate(o.Content, 200)
			sb.WriteString(fmt.Sprintf("**#%d** — %s\n> %s\n\n", o.ID, o.Title, preview))
		}
		return mcp.NewToolResultText(sb.String()), nil
	}

	// All agents summary
	contributions, err := s.store.AgentContributions(project, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(contributions) == 0 {
		return mcp.NewToolResultText("No agent contributions found."), nil
	}

	var sb strings.Builder
	sb.WriteString("# Agent Contributions\n\n---\n\n")
	for agent, obs := range contributions {
		name := agent
		if name == "" {
			name = "(unknown)"
		}
		sb.WriteString(fmt.Sprintf("### `%s` — %d memories\n\n", name, len(obs)))
		sb.WriteString("| # | Type | Title |\n")
		sb.WriteString("|---|------|-------|\n")
		for _, o := range obs {
			sb.WriteString(fmt.Sprintf("| %d | `%s` | %s |\n", o.ID, o.Type, o.Title))
		}
		sb.WriteString("\n---\n\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

// --- Argument helpers ---

func strArg(req mcp.CallToolRequest, name string) string {
	args := req.GetArguments()
	if v, ok := args[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intArg(req mcp.CallToolRequest, name string, defaultVal int) int {
	args := req.GetArguments()
	if v, ok := args[name]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case string:
			// Handle string-typed numeric params (some MCP clients send numbers as strings)
			var parsed int
			if _, err := fmt.Sscanf(n, "%d", &parsed); err == nil {
				return parsed
			}
		}
	}
	return defaultVal
}

func numArgDefault(req mcp.CallToolRequest, name string, defaultVal float64) float64 {
	args := req.GetArguments()
	if v, ok := args[name]; ok {
		if n, ok := v.(float64); ok {
			return n
		}
	}
	return defaultVal
}

func boolArg(req mcp.CallToolRequest, name string) bool {
	args := req.GetArguments()
	if v, ok := args[name]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// inferProjectName extracts a project name from a directory path.
// Uses the last component of the path (e.g., "/Users/yarov/projects/mio" → "mio").
func inferProjectName(dir string) string {
	if dir == "" {
		return ""
	}
	// Remove trailing slash
	for len(dir) > 1 && dir[len(dir)-1] == '/' {
		dir = dir[:len(dir)-1]
	}
	// Get last path component
	lastSlash := strings.LastIndex(dir, "/")
	if lastSlash >= 0 && lastSlash < len(dir)-1 {
		return dir[lastSlash+1:]
	}
	return dir
}
