package sync

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Transport defines the interface for reading/writing sync chunks
type Transport interface {
	Write(chunkID string, data []byte) error
	Read(chunkID string) ([]byte, error)
	List() ([]string, error)
	Exists(chunkID string) (bool, error)
}

// FileTransport implements Transport for local filesystem
type FileTransport struct {
	dir string
}

func NewFileTransport(dir string) (*FileTransport, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create chunks dir: %w", err)
	}
	return &FileTransport{dir: dir}, nil
}

func (t *FileTransport) Write(chunkID string, data []byte) error {
	path := filepath.Join(t.dir, chunkID+".jsonl.gz")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create chunk file: %w", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	if _, err := gz.Write(data); err != nil {
		return fmt.Errorf("write compressed data: %w", err)
	}

	return nil
}

func (t *FileTransport) Read(chunkID string) ([]byte, error) {
	path := filepath.Join(t.dir, chunkID+".jsonl.gz")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open chunk file: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gz.Close()

	return io.ReadAll(gz)
}

func (t *FileTransport) List() ([]string, error) {
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".gz" {
			name := e.Name()
			// Remove .jsonl.gz extension
			id := name[:len(name)-len(".jsonl.gz")]
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (t *FileTransport) Exists(chunkID string) (bool, error) {
	path := filepath.Join(t.dir, chunkID+".jsonl.gz")
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
