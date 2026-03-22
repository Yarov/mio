package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"mio/internal/config"
	"mio/internal/store"
)

func testServer(t *testing.T) *HTTPServer {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir:              dir,
		DBPath:               filepath.Join(dir, "test.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
		HTTPPort:             0,
	}
	s, err := store.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return New(s, cfg)
}

func doRequest(srv *HTTPServer, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON response: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

// --- Tests ---

func TestHealth(t *testing.T) {
	srv := testServer(t)
	w := doRequest(srv, "GET", "/health", nil)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	result := parseJSON(t, w)
	if result["status"] != "ok" {
		t.Errorf("status = %v, want 'ok'", result["status"])
	}
}

func TestSaveAndGet(t *testing.T) {
	srv := testServer(t)

	// Save
	body := map[string]interface{}{
		"title":      "Test save via HTTP",
		"type":       "discovery",
		"content":    "Content for HTTP save test observation",
		"importance": 0.7,
	}
	w := doRequest(srv, "POST", "/observations", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("save status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	result := parseJSON(t, w)
	id := result["id"].(float64)
	if id <= 0 {
		t.Fatal("expected positive ID")
	}

	// Get
	w = doRequest(srv, "GET", fmt.Sprintf("/observations/%d", int(id)), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", w.Code, http.StatusOK)
	}

	got := parseJSON(t, w)
	if got["Title"] != "Test save via HTTP" {
		t.Errorf("Title = %v, want %q", got["Title"], "Test save via HTTP")
	}
}

func TestSave_ValidationError(t *testing.T) {
	srv := testServer(t)
	body := map[string]interface{}{
		"title":   "ab",
		"type":    "invalid",
		"content": "short",
	}
	w := doRequest(srv, "POST", "/observations", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdate(t *testing.T) {
	srv := testServer(t)

	// Create
	w := doRequest(srv, "POST", "/observations", map[string]interface{}{
		"title": "Before update", "type": "discovery", "content": "Original content for update test",
	})
	id := int(parseJSON(t, w)["id"].(float64))

	// Update
	w = doRequest(srv, "PUT", fmt.Sprintf("/observations/%d", id), map[string]interface{}{
		"title": "After update", "content": "Updated content for the HTTP test",
	})
	if w.Code != http.StatusOK {
		t.Errorf("update status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify
	w = doRequest(srv, "GET", fmt.Sprintf("/observations/%d", id), nil)
	got := parseJSON(t, w)
	if got["Title"] != "After update" {
		t.Errorf("Title = %v, want %q", got["Title"], "After update")
	}
}

func TestDelete_Soft(t *testing.T) {
	srv := testServer(t)

	w := doRequest(srv, "POST", "/observations", map[string]interface{}{
		"title": "To delete soft", "type": "discovery", "content": "Content for soft delete HTTP test",
	})
	id := int(parseJSON(t, w)["id"].(float64))

	w = doRequest(srv, "DELETE", fmt.Sprintf("/observations/%d", id), nil)
	if w.Code != http.StatusOK {
		t.Errorf("delete status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDelete_Hard(t *testing.T) {
	srv := testServer(t)

	w := doRequest(srv, "POST", "/observations", map[string]interface{}{
		"title": "To delete hard", "type": "discovery", "content": "Content for hard delete HTTP test",
	})
	id := int(parseJSON(t, w)["id"].(float64))

	w = doRequest(srv, "DELETE", fmt.Sprintf("/observations/%d?hard=true", id), nil)
	if w.Code != http.StatusOK {
		t.Errorf("delete status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify gone
	w = doRequest(srv, "GET", fmt.Sprintf("/observations/%d", id), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("get after hard delete: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSearch(t *testing.T) {
	srv := testServer(t)

	doRequest(srv, "POST", "/observations", map[string]interface{}{
		"title": "Kubernetes deploy", "type": "config", "content": "Configured Kubernetes deployment manifests for production",
	})

	w := doRequest(srv, "GET", "/search?q=Kubernetes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", w.Code, http.StatusOK)
	}

	var results []interface{}
	json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) == 0 {
		t.Error("expected search results")
	}
}

func TestSearch_MissingQuery(t *testing.T) {
	srv := testServer(t)
	w := doRequest(srv, "GET", "/search", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContext(t *testing.T) {
	srv := testServer(t)

	doRequest(srv, "POST", "/observations", map[string]interface{}{
		"title": "Context test entry", "type": "learning", "content": "Content for context endpoint test",
	})

	w := doRequest(srv, "GET", "/context", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var results []interface{}
	json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) == 0 {
		t.Error("expected context results")
	}
}

func TestTimeline(t *testing.T) {
	srv := testServer(t)

	w := doRequest(srv, "POST", "/observations", map[string]interface{}{
		"title": "Timeline HTTP test", "type": "discovery", "content": "Content for timeline endpoint testing",
	})
	id := int(parseJSON(t, w)["id"].(float64))

	w = doRequest(srv, "GET", fmt.Sprintf("/timeline/%d", id), nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestRelations(t *testing.T) {
	srv := testServer(t)

	w := doRequest(srv, "POST", "/observations", map[string]interface{}{
		"title": "Relations test A", "type": "discovery", "content": "First observation for relations endpoint",
	})
	id := int(parseJSON(t, w)["id"].(float64))

	w = doRequest(srv, "GET", fmt.Sprintf("/relations/%d", id), nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSessions_CreateAndEnd(t *testing.T) {
	srv := testServer(t)

	// Create
	w := doRequest(srv, "POST", "/sessions", map[string]interface{}{
		"project": "test-proj", "directory": "/tmp/test",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create session status = %d, want %d", w.Code, http.StatusCreated)
	}

	result := parseJSON(t, w)
	sessionID := result["session_id"].(string)
	if sessionID == "" {
		t.Fatal("expected session_id")
	}

	// End
	w = doRequest(srv, "PUT", fmt.Sprintf("/sessions/%s/end", sessionID), map[string]interface{}{
		"summary": "Completed testing",
	})
	if w.Code != http.StatusOK {
		t.Errorf("end session status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListSessions(t *testing.T) {
	srv := testServer(t)

	doRequest(srv, "POST", "/sessions", map[string]interface{}{
		"project": "proj", "directory": "/dir",
	})

	w := doRequest(srv, "GET", "/sessions", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestStats(t *testing.T) {
	srv := testServer(t)
	w := doRequest(srv, "GET", "/stats", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	result := parseJSON(t, w)
	if _, ok := result["TotalObservations"]; !ok {
		t.Error("expected TotalObservations in stats")
	}
}

func TestExportImport(t *testing.T) {
	srv := testServer(t)

	doRequest(srv, "POST", "/observations", map[string]interface{}{
		"title": "Export via HTTP", "type": "discovery", "content": "Content for HTTP export and import test",
	})

	// Export
	w := doRequest(srv, "GET", "/export", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("export status = %d, want %d", w.Code, http.StatusOK)
	}
	exportData := w.Body.Bytes()

	// Import to a new server
	srv2 := testServer(t)
	req := httptest.NewRequest("POST", "/import", bytes.NewBuffer(exportData))
	req.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv2.mux.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Errorf("import status = %d, want %d, body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestInvalidID(t *testing.T) {
	srv := testServer(t)
	w := doRequest(srv, "GET", "/observations/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetObservation_NotFound(t *testing.T) {
	srv := testServer(t)
	w := doRequest(srv, "GET", "/observations/99999", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
