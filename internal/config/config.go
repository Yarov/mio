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

	// Sync transport: "file" (default), "git", "s3"
	SyncTransport   string
	SyncGitRemote   string
	SyncGitBranch   string
	SyncS3Endpoint  string
	SyncS3Bucket    string
	SyncS3AccessKey string
	SyncS3SecretKey string
	SyncS3Region    string
}

func Default() *Config {
	dataDir := defaultDataDir()
	cfg := &Config{
		DataDir:              dataDir,
		DBPath:               filepath.Join(dataDir, "mio.db"),
		MaxObservationLength: 50000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
		HTTPPort:             7438,
		SyncTransport:        "file",
		SyncGitBranch:        "main",
		SyncS3Region:         "us-east-1",
	}
	cfg.applySyncEnv()
	return cfg
}

func (c *Config) applySyncEnv() {
	if v := os.Getenv("MIO_SYNC_TRANSPORT"); v != "" {
		c.SyncTransport = v
	}
	if v := os.Getenv("MIO_SYNC_GIT_REMOTE"); v != "" {
		c.SyncGitRemote = v
	}
	if v := os.Getenv("MIO_SYNC_GIT_BRANCH"); v != "" {
		c.SyncGitBranch = v
	}
	if v := os.Getenv("MIO_SYNC_S3_ENDPOINT"); v != "" {
		c.SyncS3Endpoint = v
	}
	if v := os.Getenv("MIO_SYNC_S3_BUCKET"); v != "" {
		c.SyncS3Bucket = v
	}
	if v := os.Getenv("MIO_SYNC_S3_ACCESS_KEY"); v != "" {
		c.SyncS3AccessKey = v
	}
	if v := os.Getenv("MIO_SYNC_S3_SECRET_KEY"); v != "" {
		c.SyncS3SecretKey = v
	}
	if v := os.Getenv("MIO_SYNC_S3_REGION"); v != "" {
		c.SyncS3Region = v
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
