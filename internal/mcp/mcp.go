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

	// mem_update
	s.mcp.AddTool(
		mcp.NewTool("mem_update",
			mcp.WithDescription("Update an existing memory by ID"),
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
			mcp.WithString("project", mcp.Description("Project name")),
			mcp.WithString("directory", mcp.Description("Working directory")),
		),
		s.handleSessionStart,
	)

	// mem_session_end
	s.mcp.AddTool(
		mcp.NewTool("mem_session_end",
			mcp.WithDescription("End a session with optional summary"),
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
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Observation ID")),
		),
		s.handleRelations,
	)

	// mem_relate
	s.mcp.AddTool(
		mcp.NewTool("mem_relate",
			mcp.WithDescription("Create a relation between two memories"),
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
			mcp.WithString("title", mcp.Required(), mcp.Description("Observation title")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Observation content")),
		),
		s.handleSuggestTopicKey,
	)

	// mem_stats
	s.mcp.AddTool(
		mcp.NewTool("mem_stats",
			mcp.WithDescription("Get memory system statistics and metrics"),
		),
		s.handleStats,
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
	sb.WriteString(fmt.Sprintf("Found %d results:\n\n", len(results)))
	for _, r := range results {
		preview := truncate(r.Content, 300)
		sb.WriteString(fmt.Sprintf("**#%d** [%s] %s (score: %.2f)\n", r.ID, r.Type, r.Title, r.Score))
		sb.WriteString(fmt.Sprintf("  %s\n", preview))
		if r.TopicKey != nil {
			sb.WriteString(fmt.Sprintf("  topic: %s\n", *r.TopicKey))
		}
		sb.WriteString(fmt.Sprintf("  created: %s | accessed: %d times\n\n", r.CreatedAt, r.AccessCount))
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
	sb.WriteString(fmt.Sprintf("Recent context (%d observations):\n\n", len(obs)))
	for _, o := range obs {
		preview := truncate(o.Content, 200)
		sb.WriteString(fmt.Sprintf("**#%d** [%s] %s\n  %s\n  %s\n\n", o.ID, o.Type, o.Title, preview, o.CreatedAt))
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
	sb.WriteString("Timeline:\n\n")
	for _, e := range entries {
		marker := "  "
		if e.IsFocus {
			marker = "→ "
		}
		preview := truncate(e.Content, 150)
		sb.WriteString(fmt.Sprintf("%s**#%d** [%s] %s\n    %s\n    %s\n\n", marker, e.ID, e.Type, e.Title, preview, e.CreatedAt))
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
	sb.WriteString(fmt.Sprintf("Recent sessions (%d):\n\n", len(sessions)))
	for _, sess := range sessions {
		status := "active"
		if sess.EndedAt != nil {
			status = "ended"
		}
		sb.WriteString(fmt.Sprintf("**%s** [%s] project=%s, observations=%d, started=%s\n",
			sess.ID[:8], status, sess.Project, sess.ObservationCount, sess.StartedAt))
		if sess.Summary != nil {
			sb.WriteString(fmt.Sprintf("  summary: %s\n", truncate(*sess.Summary, 200)))
		}
		sb.WriteString("\n")
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
	sb.WriteString(fmt.Sprintf("Relations for #%d (%d found):\n\n", id, len(related)))

	relMap := map[int64]string{}
	for _, r := range rels {
		other := r.ToID
		if r.ToID == id {
			other = r.FromID
		}
		relMap[other] = r.Type
	}

	for _, o := range related {
		relType := relMap[o.ID]
		preview := truncate(o.Content, 150)
		sb.WriteString(fmt.Sprintf("  [%s] **#%d** [%s] %s\n    %s\n\n", relType, o.ID, o.Type, o.Title, preview))
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

	data, _ := json.MarshalIndent(metrics, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
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
