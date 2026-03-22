package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.DataDir == "" {
		t.Fatal("DataDir should not be empty")
	}
	if cfg.DBPath == "" {
		t.Fatal("DBPath should not be empty")
	}
	if cfg.MaxObservationLength != 50000 {
		t.Errorf("MaxObservationLength = %d, want 50000", cfg.MaxObservationLength)
	}
	if cfg.MaxContextResults != 20 {
		t.Errorf("MaxContextResults = %d, want 20", cfg.MaxContextResults)
	}
	if cfg.MaxSearchResults != 20 {
		t.Errorf("MaxSearchResults = %d, want 20", cfg.MaxSearchResults)
	}
	if cfg.HTTPPort != 7438 {
		t.Errorf("HTTPPort = %d, want 7438", cfg.HTTPPort)
	}
	if cfg.EnableVectorSearch {
		t.Error("EnableVectorSearch should be false by default")
	}
	if cfg.DedupeWindow <= 0 {
		t.Error("DedupeWindow should be positive")
	}
}

func TestDefaultDataDir_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MIO_DATA_DIR", dir)

	got := defaultDataDir()
	if got != dir {
		t.Errorf("defaultDataDir() = %q, want %q", got, dir)
	}
}

func TestDefaultDataDir_FallbackToHome(t *testing.T) {
	t.Setenv("MIO_DATA_DIR", "")

	got := defaultDataDir()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}
	want := filepath.Join(home, ".mio")
	if got != want {
		t.Errorf("defaultDataDir() = %q, want %q", got, want)
	}
}

func TestEnsureDataDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "nested")
	cfg := &Config{DataDir: dir}

	if err := cfg.EnsureDataDir(); err != nil {
		t.Fatalf("EnsureDataDir() error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("path is not a directory")
	}
}
