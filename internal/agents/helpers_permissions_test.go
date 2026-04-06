package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeCursorPermissionsAllowlist(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "permissions.json")

	if err := MergeCursorPermissionsAllowlist(p, CursorMCPAutoAllowPattern); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	list, _ := cfg["mcpAllowlist"].([]interface{})
	if len(list) != 1 || list[0] != CursorMCPAutoAllowPattern {
		t.Fatalf("got %v", list)
	}

	// Idempotent
	if err := MergeCursorPermissionsAllowlist(p, CursorMCPAutoAllowPattern); err != nil {
		t.Fatal(err)
	}
	raw2, _ := os.ReadFile(p)
	if string(raw) != string(raw2) {
		t.Fatalf("second merge changed file")
	}
}

func TestMergeCursorPermissionsAllowlistPreservesKeys(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "permissions.json")
	initial := []byte(`{
  "mcpAllowlist": ["filesystem:*"],
  "other": true
}
`)
	if err := os.WriteFile(p, initial, 0644); err != nil {
		t.Fatal(err)
	}
	if err := MergeCursorPermissionsAllowlist(p, CursorMCPAutoAllowPattern); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["other"] != true {
		t.Fatalf("lost other key")
	}
	list, _ := cfg["mcpAllowlist"].([]interface{})
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %v", list)
	}
}

func TestRemoveCursorPermissionsAllowlistPattern(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "permissions.json")
	if err := MergeCursorPermissionsAllowlist(p, CursorMCPAutoAllowPattern); err != nil {
		t.Fatal(err)
	}
	if err := RemoveCursorPermissionsAllowlistPattern(p, CursorMCPAutoAllowPattern); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, err=%v", err)
	}
}

func TestMergeCursorPermissionsAllowlist_JSONC(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "permissions.json")
	initial := []byte(`{
  // existing allowlist
  "mcpAllowlist": [
    "filesystem:*",
  ],
  "other": true, /* keep this key */
}
`)
	if err := os.WriteFile(p, initial, 0644); err != nil {
		t.Fatal(err)
	}
	if err := MergeCursorPermissionsAllowlist(p, CursorMCPAutoAllowPattern); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("output must be valid JSON: %v", err)
	}
	if cfg["other"] != true {
		t.Fatalf("lost other key")
	}
	list, _ := cfg["mcpAllowlist"].([]interface{})
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %v", list)
	}
}

func TestRemoveCursorPermissionsAllowlistPattern_JSONC(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "permissions.json")
	initial := []byte(`{
  "mcpAllowlist": [
    "mio:*", // remove me
  ],
}
`)
	if err := os.WriteFile(p, initial, 0644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveCursorPermissionsAllowlistPattern(p, CursorMCPAutoAllowPattern); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, err=%v", err)
	}
}

func TestMergeCursorSettingsBool(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	initial := []byte(`{
  "editor.formatOnSave": true
}
`)
	if err := os.WriteFile(p, initial, 0644); err != nil {
		t.Fatal(err)
	}
	if err := MergeCursorSettingsBool(p, CursorDisableWorkspaceTrustKey, false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["editor.formatOnSave"] != true {
		t.Fatalf("lost existing key")
	}
	if v, ok := cfg[CursorDisableWorkspaceTrustKey].(bool); !ok || v {
		t.Fatalf("expected %s=false, got %v", CursorDisableWorkspaceTrustKey, cfg[CursorDisableWorkspaceTrustKey])
	}
}

func TestMergeCursorSettingsBool_JSONC(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	initial := []byte(`{
  // comment
  "editor.formatOnSave": true,
}
`)
	if err := os.WriteFile(p, initial, 0644); err != nil {
		t.Fatal(err)
	}
	if err := MergeCursorSettingsBool(p, CursorDisableWorkspaceTrustKey, false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	if v, ok := cfg[CursorDisableWorkspaceTrustKey].(bool); !ok || v {
		t.Fatalf("expected %s=false, got %v", CursorDisableWorkspaceTrustKey, cfg[CursorDisableWorkspaceTrustKey])
	}
}
