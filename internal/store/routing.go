package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// ModelRoutingConfig holds the model routing preferences
type ModelRoutingConfig struct {
	Version         int                `json:"version"`
	AvailableModels []string           `json:"available_models"`
	Preferences     RoutingPreferences `json:"preferences"`
}

type RoutingPreferences struct {
	DefaultPlanning  string `json:"default_planning"`
	DefaultExecution string `json:"default_execution"`
	CostMode         string `json:"cost_mode"`
}

var validRoutingModels = map[string]bool{
	"opus":   true,
	"sonnet": true,
	"haiku":  true,
}

var validCostModes = map[string]bool{
	"aggressive": true,
	"balanced":   true,
	"unlimited":  true,
}

// DefaultModelRoutingConfig returns sensible defaults for model routing.
func DefaultModelRoutingConfig() ModelRoutingConfig {
	return ModelRoutingConfig{
		Version:         1,
		AvailableModels: []string{"opus", "sonnet", "haiku"},
		Preferences: RoutingPreferences{
			DefaultPlanning:  "opus",
			DefaultExecution: "sonnet",
			CostMode:         "balanced",
		},
	}
}

// ensureKVMeta creates the kv_meta table if it does not exist.
func (s *Store) ensureKVMeta() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS kv_meta (key TEXT PRIMARY KEY, value TEXT)`)
	return err
}

// GetModelRouting reads the model routing config from kv_meta.
// Returns defaults if no config has been saved yet.
func (s *Store) GetModelRouting() (ModelRoutingConfig, error) {
	if err := s.ensureKVMeta(); err != nil {
		return DefaultModelRoutingConfig(), fmt.Errorf("ensure kv_meta: %w", err)
	}

	var raw string
	err := s.db.QueryRow(`SELECT value FROM kv_meta WHERE key = 'model_routing'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return DefaultModelRoutingConfig(), nil
	}
	if err != nil {
		return DefaultModelRoutingConfig(), fmt.Errorf("query model_routing: %w", err)
	}

	var cfg ModelRoutingConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DefaultModelRoutingConfig(), fmt.Errorf("unmarshal model_routing: %w", err)
	}
	return cfg, nil
}

// SetModelRouting validates and saves the model routing config to kv_meta.
func (s *Store) SetModelRouting(cfg ModelRoutingConfig) error {
	if err := ValidateModelRoutingConfig(cfg); err != nil {
		return err
	}

	if err := s.ensureKVMeta(); err != nil {
		return fmt.Errorf("ensure kv_meta: %w", err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal model_routing: %w", err)
	}

	_, err = s.db.Exec(`INSERT OR REPLACE INTO kv_meta (key, value) VALUES ('model_routing', ?)`, string(data))
	if err != nil {
		return fmt.Errorf("save model_routing: %w", err)
	}
	return nil
}

// ValidateModelRoutingConfig validates the model routing configuration.
func ValidateModelRoutingConfig(cfg ModelRoutingConfig) error {
	// Validate all available models
	for _, m := range cfg.AvailableModels {
		if !validRoutingModels[m] {
			return fmt.Errorf("invalid model %q: must be one of opus, sonnet, haiku", m)
		}
	}

	// Validate cost mode
	if !validCostModes[cfg.Preferences.CostMode] {
		return fmt.Errorf("invalid cost_mode %q: must be one of aggressive, balanced, unlimited", cfg.Preferences.CostMode)
	}

	// Build a set of available models for fast lookup
	available := make(map[string]bool, len(cfg.AvailableModels))
	for _, m := range cfg.AvailableModels {
		available[m] = true
	}

	// Validate default_planning is in available_models
	if cfg.Preferences.DefaultPlanning != "" && !available[cfg.Preferences.DefaultPlanning] {
		return fmt.Errorf("default_planning %q is not in available_models", cfg.Preferences.DefaultPlanning)
	}

	// Validate default_execution is in available_models
	if cfg.Preferences.DefaultExecution != "" && !available[cfg.Preferences.DefaultExecution] {
		return fmt.Errorf("default_execution %q is not in available_models", cfg.Preferences.DefaultExecution)
	}

	return nil
}
