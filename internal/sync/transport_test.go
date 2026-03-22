package sync

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func testTransport(t *testing.T) *FileTransport {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "chunks")
	ft, err := NewFileTransport(dir)
	if err != nil {
		t.Fatal(err)
	}
	return ft
}

func TestFileTransport_WriteRead(t *testing.T) {
	ft := testTransport(t)
	data := []byte(`{"hello":"world","count":42}`)

	if err := ft.Write("chunk-001", data); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	got, err := ft.Read("chunk-001")
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("Read() = %q, want %q", got, data)
	}
}

func TestFileTransport_GzipCompression(t *testing.T) {
	ft := testTransport(t)
	data := []byte("test data for compression")

	if err := ft.Write("gzip-test", data); err != nil {
		t.Fatal(err)
	}

	// Read raw file and verify it's valid gzip
	raw, err := os.ReadFile(filepath.Join(ft.dir, "gzip-test.jsonl.gz"))
	if err != nil {
		t.Fatal(err)
	}

	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("file is not valid gzip: %v", err)
	}
	gz.Close()
}

func TestFileTransport_List(t *testing.T) {
	ft := testTransport(t)

	for _, id := range []string{"aaa", "bbb", "ccc"} {
		if err := ft.Write(id, []byte("data")); err != nil {
			t.Fatal(err)
		}
	}

	ids, err := ft.List()
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(ids)
	if len(ids) != 3 || ids[0] != "aaa" || ids[1] != "bbb" || ids[2] != "ccc" {
		t.Errorf("List() = %v, want [aaa bbb ccc]", ids)
	}
}

func TestFileTransport_Exists(t *testing.T) {
	ft := testTransport(t)

	exists, err := ft.Exists("missing")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Exists() = true for missing chunk")
	}

	if err := ft.Write("present", []byte("data")); err != nil {
		t.Fatal(err)
	}

	exists, err = ft.Exists("present")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Exists() = false for present chunk")
	}
}

func TestFileTransport_ReadMissing(t *testing.T) {
	ft := testTransport(t)

	_, err := ft.Read("nonexistent")
	if err == nil {
		t.Error("Read() should return error for missing chunk")
	}
}
