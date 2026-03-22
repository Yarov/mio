package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mio/internal/config"
	"mio/internal/store"
)

func testSyncer(t *testing.T) (*Syncer, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir:              dir,
		DBPath:               filepath.Join(dir, "test.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         0,
	}
	s, err := store.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	syncer, err := NewSyncer(s, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return syncer, s
}

func seedObservation(t *testing.T, s *store.Store, title string) {
	t.Helper()
	obs := &store.Observation{
		Title:      title,
		Type:       "discovery",
		Content:    "Test content for sync testing: " + title,
		Scope:      "project",
		Importance: 0.5,
	}
	if _, err := s.Save(obs); err != nil {
		t.Fatal(err)
	}
}

func TestExport_CreatesChunk(t *testing.T) {
	syncer, s := testSyncer(t)
	s.CreateSession("sync-sess", "proj", "/dir")
	seedObservation(t, s, "Export chunk test")

	meta, err := syncer.Export("")
	if err != nil {
		t.Fatal(err)
	}
	if meta.ID == "" {
		t.Error("chunk ID should not be empty")
	}
	if meta.MemoryCount != 1 {
		t.Errorf("MemoryCount = %d, want 1", meta.MemoryCount)
	}
	if meta.SessionCount != 1 {
		t.Errorf("SessionCount = %d, want 1", meta.SessionCount)
	}

	// Verify chunk file exists
	exists, _ := syncer.transport.Exists(meta.ID)
	if !exists {
		t.Error("chunk file should exist")
	}

	// Verify manifest updated
	if len(syncer.manifest.Chunks) != 1 {
		t.Errorf("manifest chunks = %d, want 1", len(syncer.manifest.Chunks))
	}
}

func TestExport_NoData(t *testing.T) {
	syncer, _ := testSyncer(t)

	_, err := syncer.Export("")
	if err == nil {
		t.Fatal("expected error for empty export")
	}
}

func TestExport_DuplicateChunkDetected(t *testing.T) {
	syncer, s := testSyncer(t)
	seedObservation(t, s, "Duplicate chunk test")

	_, err := syncer.Export("")
	if err != nil {
		t.Fatal(err)
	}

	// Reset manifest to trick it into trying again
	syncer.manifest.Chunks = nil

	_, err = syncer.Export("")
	if err == nil {
		t.Fatal("expected error for duplicate chunk")
	}
}

func TestImport_ReadsChunks(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir:              dir,
		DBPath:               filepath.Join(dir, "test.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         0,
	}
	s, _ := store.New(cfg)
	defer s.Close()

	syncer, _ := NewSyncer(s, cfg)

	// Manually write a chunk
	data := &store.ExportData{
		Sessions: []store.Session{{
			ID:        "import-sess",
			Project:   "proj",
			Directory: "/dir",
			StartedAt: time.Now().UTC().Format(time.RFC3339),
		}},
	}
	jsonData, _ := json.Marshal(data)
	syncer.transport.Write("manual-chunk", jsonData)

	count, err := syncer.Import()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("imported = %d, want 1", count)
	}
}

func TestExportImportCycle(t *testing.T) {
	// Source: create data and export
	dir1 := t.TempDir()
	cfg1 := &config.Config{
		DataDir:              dir1,
		DBPath:               filepath.Join(dir1, "source.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         0,
	}
	s1, _ := store.New(cfg1)
	defer s1.Close()

	s1.CreateSession("cycle-sess", "proj", "/dir")
	seedObservation(t, s1, "Cycle test observation")

	syncer1, _ := NewSyncer(s1, cfg1)
	meta, err := syncer1.Export("")
	if err != nil {
		t.Fatal(err)
	}

	// Destination: import the chunk
	dir2 := t.TempDir()
	cfg2 := &config.Config{
		DataDir:              dir2,
		DBPath:               filepath.Join(dir2, "dest.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         0,
	}
	s2, _ := store.New(cfg2)
	defer s2.Close()

	// Copy chunk file to destination chunks dir
	chunksDir2 := filepath.Join(dir2, "chunks")
	os.MkdirAll(chunksDir2, 0755)

	chunkData, _ := syncer1.transport.Read(meta.ID)
	ft2, _ := NewFileTransport(chunksDir2)
	ft2.Write(meta.ID, chunkData)

	syncer2, _ := NewSyncer(s2, cfg2)
	count, err := syncer2.Import()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("imported = %d, want 1", count)
	}

	// Verify data arrived
	obs, _ := s2.RecentContext("", 0)
	if len(obs) != 1 {
		t.Errorf("expected 1 observation in destination, got %d", len(obs))
	}
}

func TestStatus(t *testing.T) {
	syncer, s := testSyncer(t)
	seedObservation(t, s, "Status test observation content")
	syncer.Export("")

	status := syncer.Status()
	if status["total_chunks"].(int) < 1 {
		t.Error("total_chunks should be >= 1")
	}
	if status["pending_import"].(int) != 0 {
		t.Errorf("pending_import = %d, want 0", status["pending_import"])
	}
	if status["last_sync"] == "never" {
		t.Error("last_sync should not be 'never' after export")
	}
}

func TestFilterNewData(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-2 * time.Hour).Format(time.RFC3339)
	recent := now.Add(-30 * time.Minute).Format(time.RFC3339)
	cutoff := now.Add(-1 * time.Hour).Format(time.RFC3339)

	data := &store.ExportData{
		Sessions: []store.Session{
			{ID: "old", StartedAt: old},
			{ID: "new", StartedAt: recent},
		},
		Observations: []store.Observation{
			{Title: "old obs", CreatedAt: old},
			{Title: "new obs", CreatedAt: recent},
		},
		Prompts: []store.Prompt{
			{Content: "old prompt", CreatedAt: old},
			{Content: "new prompt", CreatedAt: recent},
		},
	}

	filtered := filterNewData(data, cutoff)

	if len(filtered.Sessions) != 1 || filtered.Sessions[0].ID != "new" {
		t.Errorf("filtered sessions = %v, want only 'new'", filtered.Sessions)
	}
	if len(filtered.Observations) != 1 || filtered.Observations[0].Title != "new obs" {
		t.Errorf("filtered observations = %v, want only 'new obs'", filtered.Observations)
	}
	if len(filtered.Prompts) != 1 || filtered.Prompts[0].Content != "new prompt" {
		t.Errorf("filtered prompts = %v, want only 'new prompt'", filtered.Prompts)
	}
}
