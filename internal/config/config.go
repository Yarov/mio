package config

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	DataDir              string
	DBPath               string
	MaxObservationLength int
	MaxContextResults    int
	MaxSearchResults     int
	DedupeWindow         time.Duration
	HTTPPort             int
	EnableVectorSearch   bool
	EmbeddingModel       string
	EmbeddingEndpoint    string
}

func Default() *Config {
	dataDir := defaultDataDir()
	return &Config{
		DataDir:              dataDir,
		DBPath:               filepath.Join(dataDir, "mio.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
		HTTPPort:             7438,
		EnableVectorSearch:   false,
		EmbeddingModel:       "local",
		EmbeddingEndpoint:    "",
	}
}

func defaultDataDir() string {
	if dir := os.Getenv("MIO_DATA_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mio"
	}
	return filepath.Join(home, ".mio")
}

func (c *Config) EnsureDataDir() error {
	return os.MkdirAll(c.DataDir, 0755)
}
