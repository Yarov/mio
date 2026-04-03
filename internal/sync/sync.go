package sync

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mio/internal/config"
	"mio/internal/store"
)

type Manifest struct {
	Version int           `json:"version"`
	Chunks  []ChunkMeta   `json:"chunks"`
}

type ChunkMeta struct {
	ID            string `json:"id"`
	Creator       string `json:"creator"`
	Timestamp     string `json:"timestamp"`
	SessionCount  int    `json:"session_count"`
	MemoryCount   int    `json:"memory_count"`
	PromptCount   int    `json:"prompt_count"`
}

type Syncer struct {
	store     *store.Store
	cfg       *config.Config
	transport Transport
	manifest  *Manifest
}

func NewSyncer(s *store.Store, cfg *config.Config, transport ...Transport) (*Syncer, error) {
	var t Transport
	if len(transport) > 0 && transport[0] != nil {
		t = transport[0]
	} else {
		chunksDir := filepath.Join(cfg.DataDir, "chunks")
		ft, err := NewFileTransport(chunksDir)
		if err != nil {
			return nil, err
		}
		t = ft
	}

	syncer := &Syncer{
		store:     s,
		cfg:       cfg,
		transport: t,
	}

	if err := syncer.loadManifest(); err != nil {
		syncer.manifest = &Manifest{Version: 1}
	}

	return syncer, nil
}

func (s *Syncer) manifestPath() string {
	return filepath.Join(s.cfg.DataDir, "chunks", "manifest.json")
}

func (s *Syncer) loadManifest() error {
	data, err := os.ReadFile(s.manifestPath())
	if err != nil {
		return err
	}
	s.manifest = &Manifest{}
	return json.Unmarshal(data, s.manifest)
}

func (s *Syncer) saveManifest() error {
	data, err := json.MarshalIndent(s.manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.manifestPath(), data, 0644)
}

// Export creates a new chunk with all data since the last chunk
func (s *Syncer) Export(project string) (*ChunkMeta, error) {
	data, err := s.store.ExportAll(project)
	if err != nil {
		return nil, fmt.Errorf("export data: %w", err)
	}

	// Filter to only new data since last chunk
	var cutoff string
	if len(s.manifest.Chunks) > 0 {
		cutoff = s.manifest.Chunks[len(s.manifest.Chunks)-1].Timestamp
	}

	if cutoff != "" {
		data = filterNewData(data, cutoff)
	}

	if len(data.Sessions) == 0 && len(data.Observations) == 0 && len(data.Prompts) == 0 {
		return nil, fmt.Errorf("no new data to export")
	}

	// Serialize
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}

	// Generate chunk ID from content hash
	hash := sha256.Sum256(jsonData)
	chunkID := fmt.Sprintf("%x", hash[:4]) // 8 character hex

	// Check for duplicate chunk
	exists, _ := s.transport.Exists(chunkID)
	if exists {
		return nil, fmt.Errorf("chunk %s already exists (duplicate data)", chunkID)
	}

	// Write chunk
	if err := s.transport.Write(chunkID, jsonData); err != nil {
		return nil, fmt.Errorf("write chunk: %w", err)
	}

	// Update manifest
	hostname, _ := os.Hostname()
	meta := ChunkMeta{
		ID:           chunkID,
		Creator:      hostname,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		SessionCount: len(data.Sessions),
		MemoryCount:  len(data.Observations),
		PromptCount:  len(data.Prompts),
	}
	s.manifest.Chunks = append(s.manifest.Chunks, meta)

	if err := s.saveManifest(); err != nil {
		return nil, fmt.Errorf("save manifest: %w", err)
	}

	return &meta, nil
}

// Import reads all chunks not yet imported
func (s *Syncer) Import() (int, error) {
	chunkIDs, err := s.transport.List()
	if err != nil {
		return 0, fmt.Errorf("list chunks: %w", err)
	}

	known := make(map[string]bool)
	for _, c := range s.manifest.Chunks {
		known[c.ID] = true
	}

	imported := 0
	for _, id := range chunkIDs {
		if known[id] {
			continue
		}

		data, err := s.transport.Read(id)
		if err != nil {
			return imported, fmt.Errorf("read chunk %s: %w", id, err)
		}

		var exportData store.ExportData
		if err := json.Unmarshal(data, &exportData); err != nil {
			return imported, fmt.Errorf("unmarshal chunk %s: %w", id, err)
		}

		if err := s.store.ImportData(&exportData); err != nil {
			return imported, fmt.Errorf("import chunk %s: %w", id, err)
		}

		imported++
	}

	return imported, nil
}

// Status returns current sync state
func (s *Syncer) Status() map[string]interface{} {
	chunkIDs, _ := s.transport.List()

	known := make(map[string]bool)
	for _, c := range s.manifest.Chunks {
		known[c.ID] = true
	}

	pending := 0
	for _, id := range chunkIDs {
		if !known[id] {
			pending++
		}
	}

	return map[string]interface{}{
		"total_chunks":   len(chunkIDs),
		"known_chunks":   len(s.manifest.Chunks),
		"pending_import": pending,
		"last_sync":      lastSyncTime(s.manifest),
	}
}

func lastSyncTime(m *Manifest) string {
	if len(m.Chunks) == 0 {
		return "never"
	}
	return m.Chunks[len(m.Chunks)-1].Timestamp
}

func filterNewData(data *store.ExportData, cutoff string) *store.ExportData {
	filtered := &store.ExportData{}

	for _, s := range data.Sessions {
		if s.StartedAt > cutoff {
			filtered.Sessions = append(filtered.Sessions, s)
		}
	}
	for _, o := range data.Observations {
		if o.CreatedAt > cutoff {
			filtered.Observations = append(filtered.Observations, o)
		}
	}
	for _, p := range data.Prompts {
		if p.CreatedAt > cutoff {
			filtered.Prompts = append(filtered.Prompts, p)
		}
	}

	return filtered
}
