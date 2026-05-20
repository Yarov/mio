package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// OutputStyleConfig holds the output style preferences.
type OutputStyleConfig struct {
	Version int  `json:"version"`
	Enabled bool `json:"enabled"`
}

// DefaultOutputStyleConfig returns sensible defaults (enabled).
func DefaultOutputStyleConfig() OutputStyleConfig {
	return OutputStyleConfig{
		Version: 1,
		Enabled: true,
	}
}

// GetOutputStyle reads the output style config from kv_meta.
// Returns defaults (enabled) if no config has been saved yet.
func (s *Store) GetOutputStyle() (OutputStyleConfig, error) {
	if err := s.ensureKVMeta(); err != nil {
		return DefaultOutputStyleConfig(), fmt.Errorf("ensure kv_meta: %w", err)
	}

	var raw string
	err := s.db.QueryRow(`SELECT value FROM kv_meta WHERE key = 'output_style'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return DefaultOutputStyleConfig(), nil
	}
	if err != nil {
		return DefaultOutputStyleConfig(), fmt.Errorf("query output_style: %w", err)
	}

	var cfg OutputStyleConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DefaultOutputStyleConfig(), fmt.Errorf("unmarshal output_style: %w", err)
	}
	return cfg, nil
}

// SetOutputStyle saves the output style config to kv_meta.
func (s *Store) SetOutputStyle(cfg OutputStyleConfig) error {
	if err := s.ensureKVMeta(); err != nil {
		return fmt.Errorf("ensure kv_meta: %w", err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal output_style: %w", err)
	}

	_, err = s.db.Exec(`INSERT OR REPLACE INTO kv_meta (key, value) VALUES ('output_style', ?)`, string(data))
	if err != nil {
		return fmt.Errorf("save output_style: %w", err)
	}
	return nil
}
