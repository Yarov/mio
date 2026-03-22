package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	fmt.Printf("Mio HTTP server listening on %s\n", addr)
	return http.ListenAndServe(addr, s.mux)
}

func (s *HTTPServer) registerRoutes() {
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

// Ensure strings package is used
var _ = strings.TrimSpace
