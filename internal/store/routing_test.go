package store

import (
	"testing"
)

func TestDefaultModelRoutingConfig(t *testing.T) {
	cfg := DefaultModelRoutingConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}

	wantModels := []string{"opus", "sonnet", "haiku"}
	if len(cfg.AvailableModels) != len(wantModels) {
		t.Fatalf("expected %d available models, got %d", len(wantModels), len(cfg.AvailableModels))
	}
	for i, m := range wantModels {
		if cfg.AvailableModels[i] != m {
			t.Errorf("available_models[%d]: expected %q, got %q", i, m, cfg.AvailableModels[i])
		}
	}

	if cfg.Preferences.DefaultPlanning != "opus" {
		t.Errorf("expected default_planning=opus, got %q", cfg.Preferences.DefaultPlanning)
	}
	if cfg.Preferences.DefaultExecution != "sonnet" {
		t.Errorf("expected default_execution=sonnet, got %q", cfg.Preferences.DefaultExecution)
	}
	if cfg.Preferences.CostMode != "balanced" {
		t.Errorf("expected cost_mode=balanced, got %q", cfg.Preferences.CostMode)
	}
}

func TestGetModelRouting_NoConfig(t *testing.T) {
	s := testStore(t)

	cfg, err := s.GetModelRouting()
	if err != nil {
		t.Fatalf("GetModelRouting() unexpected error: %v", err)
	}

	defaults := DefaultModelRoutingConfig()
	if cfg.Version != defaults.Version {
		t.Errorf("expected version %d, got %d", defaults.Version, cfg.Version)
	}
	if cfg.Preferences.CostMode != defaults.Preferences.CostMode {
		t.Errorf("expected cost_mode %q, got %q", defaults.Preferences.CostMode, cfg.Preferences.CostMode)
	}
}

func TestSetAndGetModelRouting(t *testing.T) {
	s := testStore(t)

	want := ModelRoutingConfig{
		Version:         1,
		AvailableModels: []string{"sonnet", "haiku"},
		Preferences: RoutingPreferences{
			DefaultPlanning:  "sonnet",
			DefaultExecution: "haiku",
			CostMode:         "aggressive",
		},
	}

	if err := s.SetModelRouting(want); err != nil {
		t.Fatalf("SetModelRouting() error: %v", err)
	}

	got, err := s.GetModelRouting()
	if err != nil {
		t.Fatalf("GetModelRouting() error: %v", err)
	}

	if got.Version != want.Version {
		t.Errorf("version: got %d, want %d", got.Version, want.Version)
	}
	if got.Preferences.DefaultPlanning != want.Preferences.DefaultPlanning {
		t.Errorf("default_planning: got %q, want %q", got.Preferences.DefaultPlanning, want.Preferences.DefaultPlanning)
	}
	if got.Preferences.DefaultExecution != want.Preferences.DefaultExecution {
		t.Errorf("default_execution: got %q, want %q", got.Preferences.DefaultExecution, want.Preferences.DefaultExecution)
	}
	if got.Preferences.CostMode != want.Preferences.CostMode {
		t.Errorf("cost_mode: got %q, want %q", got.Preferences.CostMode, want.Preferences.CostMode)
	}
	if len(got.AvailableModels) != len(want.AvailableModels) {
		t.Errorf("available_models count: got %d, want %d", len(got.AvailableModels), len(want.AvailableModels))
	}
}

func TestSetModelRouting_InvalidModel(t *testing.T) {
	s := testStore(t)

	cfg := ModelRoutingConfig{
		Version:         1,
		AvailableModels: []string{"opus", "gpt-4"},
		Preferences: RoutingPreferences{
			DefaultPlanning:  "opus",
			DefaultExecution: "opus",
			CostMode:         "balanced",
		},
	}

	err := s.SetModelRouting(cfg)
	if err == nil {
		t.Fatal("expected error for invalid model name, got nil")
	}
}

func TestSetModelRouting_InvalidCostMode(t *testing.T) {
	s := testStore(t)

	cfg := ModelRoutingConfig{
		Version:         1,
		AvailableModels: []string{"opus", "sonnet"},
		Preferences: RoutingPreferences{
			DefaultPlanning:  "opus",
			DefaultExecution: "sonnet",
			CostMode:         "cheap",
		},
	}

	err := s.SetModelRouting(cfg)
	if err == nil {
		t.Fatal("expected error for invalid cost_mode, got nil")
	}
}

func TestSetModelRouting_DefaultNotInAvailable(t *testing.T) {
	s := testStore(t)

	cfg := ModelRoutingConfig{
		Version:         1,
		AvailableModels: []string{"sonnet", "haiku"},
		Preferences: RoutingPreferences{
			DefaultPlanning:  "opus", // opus not in available_models
			DefaultExecution: "sonnet",
			CostMode:         "balanced",
		},
	}

	err := s.SetModelRouting(cfg)
	if err == nil {
		t.Fatal("expected error when default_planning not in available_models, got nil")
	}
}

func TestValidateModelRoutingConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ModelRoutingConfig
		wantErr bool
	}{
		{
			name:    "valid defaults",
			cfg:     DefaultModelRoutingConfig(),
			wantErr: false,
		},
		{
			name: "all valid models",
			cfg: ModelRoutingConfig{
				AvailableModels: []string{"opus", "sonnet", "haiku"},
				Preferences: RoutingPreferences{
					DefaultPlanning:  "opus",
					DefaultExecution: "haiku",
					CostMode:         "unlimited",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid model in available list",
			cfg: ModelRoutingConfig{
				AvailableModels: []string{"opus", "claude-3"},
				Preferences: RoutingPreferences{
					DefaultPlanning:  "opus",
					DefaultExecution: "opus",
					CostMode:         "balanced",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid cost mode",
			cfg: ModelRoutingConfig{
				AvailableModels: []string{"opus"},
				Preferences: RoutingPreferences{
					DefaultPlanning:  "opus",
					DefaultExecution: "opus",
					CostMode:         "economy",
				},
			},
			wantErr: true,
		},
		{
			name: "default_planning not in available",
			cfg: ModelRoutingConfig{
				AvailableModels: []string{"haiku"},
				Preferences: RoutingPreferences{
					DefaultPlanning:  "opus",
					DefaultExecution: "haiku",
					CostMode:         "balanced",
				},
			},
			wantErr: true,
		},
		{
			name: "default_execution not in available",
			cfg: ModelRoutingConfig{
				AvailableModels: []string{"opus"},
				Preferences: RoutingPreferences{
					DefaultPlanning:  "opus",
					DefaultExecution: "sonnet",
					CostMode:         "balanced",
				},
			},
			wantErr: true,
		},
		{
			name: "aggressive cost mode valid",
			cfg: ModelRoutingConfig{
				AvailableModels: []string{"haiku"},
				Preferences: RoutingPreferences{
					DefaultPlanning:  "haiku",
					DefaultExecution: "haiku",
					CostMode:         "aggressive",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelRoutingConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelRoutingConfig() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
