package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"mio/internal/config"
	"mio/internal/store"
)

type HTTPServer struct {
	store *store.Store
	cfg   *config.Config
	mux   *http.ServeMux
}

func New(s *store.Store, cfg *config.Config) *HTTPServer {
	srv := &HTTPServer{store: s, cfg: cfg, mux: http.NewServeMux()}
	srv.registerRoutes()
	return srv
}

func (s *HTTPServer) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.HTTPPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	// Write PID file so other processes can check if we're alive
	pidPath := filepath.Join(s.cfg.DataDir, "server.pid")
	os.MkdirAll(s.cfg.DataDir, 0755)
	os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644)

	fmt.Fprintf(os.Stderr, "Mio HTTP server listening on %s (pid %d)\n", addr, os.Getpid())
	return http.Serve(ln, s.mux)
}

func (s *HTTPServer) registerRoutes() {
	s.mux.HandleFunc("GET /", s.handleDashboard)
	s.mux.HandleFunc("GET /skills", s.handleSkills)
	s.mux.HandleFunc("GET /skills/{name}", s.handleSkillGet)
	s.mux.HandleFunc("PUT /skills/{name}", s.handleSkillUpdate)
	s.mux.HandleFunc("POST /admin/setup", s.handleAdminSetup)
	s.mux.HandleFunc("POST /admin/uninstall", s.handleAdminUninstall)
	s.mux.HandleFunc("GET /agents", s.handleAgents)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("POST /observations", s.handleSave)
	s.mux.HandleFunc("GET /observations/{id}", s.handleGet)
	s.mux.HandleFunc("PUT /observations/{id}", s.handleUpdate)
	s.mux.HandleFunc("DELETE /observations/{id}", s.handleDelete)
	s.mux.HandleFunc("GET /search", s.handleSearch)
	s.mux.HandleFunc("GET /context", s.handleContext)
	s.mux.HandleFunc("GET /timeline/{id}", s.handleTimeline)
	s.mux.HandleFunc("GET /relations/{id}", s.handleRelations)
	s.mux.HandleFunc("POST /sessions", s.handleCreateSession)
	s.mux.HandleFunc("PUT /sessions/{id}/end", s.handleEndSession)
	s.mux.HandleFunc("GET /sessions", s.handleListSessions)
	s.mux.HandleFunc("GET /stats", s.handleStats)
	s.mux.HandleFunc("GET /export", s.handleExport)
	s.mux.HandleFunc("POST /import", s.handleImport)

	// Innovation endpoints
	s.mux.HandleFunc("GET /graph/{id}", s.handleGraph)
	s.mux.HandleFunc("POST /gc", s.handleGC)
	s.mux.HandleFunc("POST /consolidate", s.handleConsolidate)
	s.mux.HandleFunc("GET /cross-project", s.handleCrossProject)
	s.mux.HandleFunc("GET /surface", s.handleSurface)
	s.mux.HandleFunc("GET /agents/knowledge", s.handleAgentKnowledge)
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "0.1.0"})
}

func (s *HTTPServer) handleSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title      string  `json:"title"`
		Type       string  `json:"type"`
		Content    string  `json:"content"`
		SessionID  string  `json:"session_id"`
		Project    string  `json:"project"`
		Scope      string  `json:"scope"`
		TopicKey   string  `json:"topic_key"`
		Importance float64 `json:"importance"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	obs := &store.Observation{
		Title:      req.Title,
		Type:       req.Type,
		Content:    req.Content,
		SessionID:  req.SessionID,
		Scope:      req.Scope,
		Importance: req.Importance,
	}
	if req.Project != "" {
		obs.Project = &req.Project
	}
	if req.TopicKey != "" {
		obs.TopicKey = &req.TopicKey
	}

	id, err := s.store.Save(obs)
	if err != nil {
		status := http.StatusInternalServerError
		if _, ok := err.(*store.ValidationError); ok {
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id, "sync_id": obs.SyncID})
}

func (s *HTTPServer) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	obs, err := s.store.GetObservation(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, http.StatusOK, obs)
}

func (s *HTTPServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.UpdateObservation(id, req.Title, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *HTTPServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	hard := r.URL.Query().Get("hard") == "true"
	if hard {
		err = s.store.HardDelete(id)
	} else {
		err = s.store.SoftDelete(id)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *HTTPServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	project := r.URL.Query().Get("project")
	obsType := r.URL.Query().Get("type")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	results, err := s.store.Search(query, project, obsType, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *HTTPServer) handleContext(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	obs, err := s.store.RecentContext(project, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, obs)
}

func (s *HTTPServer) handleTimeline(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	before, _ := strconv.Atoi(r.URL.Query().Get("before"))
	after, _ := strconv.Atoi(r.URL.Query().Get("after"))
	if before <= 0 {
		before = 5
	}
	if after <= 0 {
		after = 5
	}

	entries, err := s.store.Timeline(id, before, after)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

func (s *HTTPServer) handleRelations(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	related, err := s.store.GetRelatedObservations(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, related)
}

func (s *HTTPServer) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Project   string `json:"project"`
		Directory string `json:"directory"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id := fmt.Sprintf("%08x", strHash(req.Project+req.Directory+fmt.Sprint(time.Now().UnixNano())))
	if err := s.store.CreateSession(id, req.Project, req.Directory); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"session_id": id})
}

func (s *HTTPServer) handleEndSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Summary string `json:"summary"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var summaryPtr *string
	if req.Summary != "" {
		summaryPtr = &req.Summary
	}

	if err := s.store.EndSession(id, summaryPtr); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ended"})
}

func (s *HTTPServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	sessions, err := s.store.RecentSessions(project, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, sessions)
}

func (s *HTTPServer) handleStats(w http.ResponseWriter, _ *http.Request) {
	metrics, err := s.store.GetMetrics()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, metrics)
}

func (s *HTTPServer) handleExport(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	data, err := s.store.ExportAll(project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (s *HTTPServer) handleImport(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 50*1024*1024)) // 50MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var data store.ExportData
	if err := json.Unmarshal(body, &data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if err := s.store.ImportData(&data); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "imported"})
}

// IsRunning checks if a Mio HTTP server is already responding on the given port.
func IsRunning(port int) bool {
	url := fmt.Sprintf("http://localhost:%d/health", port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func strHash(s string) uint32 {
	var h uint32
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

// --- Innovation HTTP handlers ---

func (s *HTTPServer) handleGraph(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	depthStr := r.URL.Query().Get("depth")
	depth := 3
	if depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil {
			depth = d
		}
	}

	graph, err := s.store.BuildDecisionGraph(id, depth)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *HTTPServer) handleGC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StaleDays        int     `json:"stale_days"`
		ArchiveThreshold float64 `json:"archive_threshold"`
	}
	if err := readJSON(r, &req); err != nil {
		req.StaleDays = 60
		req.ArchiveThreshold = 0.1
	}
	if req.StaleDays == 0 {
		req.StaleDays = 60
	}
	if req.ArchiveThreshold == 0 {
		req.ArchiveThreshold = 0.1
	}

	decayed, archived, err := s.store.DecayAndGC(req.StaleDays, req.ArchiveThreshold)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"decayed":  decayed,
		"archived": archived,
	})
}

func (s *HTTPServer) handleConsolidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Project string `json:"project"`
	}
	readJSON(r, &req)

	count, err := s.store.Consolidate(req.Project)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"consolidated": count})
}

func (s *HTTPServer) handleCrossProject(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query required"})
		return
	}
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}

	results, err := s.store.CrossProjectSearch(query, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *HTTPServer) handleSurface(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	project := r.URL.Query().Get("project")
	if text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text required"})
		return
	}
	limit := 3
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}

	results, err := s.store.SurfaceRelevant(text, project, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *HTTPServer) handleAgentKnowledge(w http.ResponseWriter, r *http.Request) {
	agent := r.URL.Query().Get("agent")
	project := r.URL.Query().Get("project")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}

	if agent != "" {
		obs, err := s.store.AgentKnowledge(agent, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, obs)
		return
	}

	contribs, err := s.store.AgentContributions(project, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, contribs)
}

