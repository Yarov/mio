package agents

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	markerBegin = "<!-- BEGIN:mio -->"
	markerEnd   = "<!-- END:mio -->"
)

// --- Detection helpers ---

// DetectByDir returns true if the directory exists.
func DetectByDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// DetectByCommand returns true if the command exists in PATH.
func DetectByCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// HomeDir returns the user's home directory or panics.
func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return home
}

// --- MCP config helpers ---

// MIOEntry returns the standard MCP server entry for Mio.
func MIOEntry(binPath string) map[string]interface{} {
	return map[string]interface{}{
		"command": binPath,
		"args":    []string{"mcp"},
	}
}

// WriteMCPToOwnFile writes a standalone MCP config file (Claude Code style).
// The file contains: {"mcpServers": {"mio": {command, args}}}
func WriteMCPToOwnFile(configPath, binPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"mio": MIOEntry(binPath),
		},
	}

	// Check if already identical
	if existing, err := os.ReadFile(configPath); err == nil {
		newData, _ := json.MarshalIndent(config, "", "  ")
		if string(existing) == string(newData)+"\n" || string(existing) == string(newData) {
			return nil
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append(data, '\n'), 0644)
}

// WriteMCPToSharedJSON merges a "mio" entry into an existing shared JSON config.
// Handles formats like: {"mcpServers": {"mio": ...}} (Cursor, Gemini, Continue, etc.)
func WriteMCPToSharedJSON(configPath, binPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	config := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			// Backup corrupted file
			_ = os.WriteFile(configPath+".bak", data, 0644)
			config = make(map[string]interface{})
		}
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}

	servers["mio"] = MIOEntry(binPath)
	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append(data, '\n'), 0644)
}

// RemoveMCPFromOwnFile removes the standalone MCP config file.
func RemoveMCPFromOwnFile(configPath string) error {
	err := os.Remove(configPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// RemoveMCPFromSharedJSON removes the "mio" entry from a shared JSON config.
func RemoveMCPFromSharedJSON(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	config := make(map[string]interface{})
	if err := json.Unmarshal(data, &config); err != nil {
		return nil // don't touch corrupted files
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return nil
	}

	if _, exists := servers["mio"]; !exists {
		return nil
	}

	delete(servers, "mio")
	config["mcpServers"] = servers

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append(out, '\n'), 0644)
}

// HasMCPConfig returns true if the config file contains a "mio" MCP entry.
func HasMCPConfig(configPath string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}
	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return false
	}
	_, exists := servers["mio"]
	return exists
}

// --- Protocol file helpers ---

// InstallProtocol injects Mio protocol content into a file using markers.
// If markers already exist, replaces the content between them.
// If not, appends at the end.
func InstallProtocol(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	wrapped := markerBegin + "\n" + content + "\n" + markerEnd

	existing := ""
	if data, err := os.ReadFile(filePath); err == nil {
		existing = string(data)
	}

	if startIdx := strings.Index(existing, markerBegin); startIdx >= 0 {
		endIdx := strings.Index(existing, markerEnd)
		if endIdx >= 0 {
			endIdx += len(markerEnd)
			newContent := existing[:startIdx] + wrapped + existing[endIdx:]
			if newContent == existing {
				return nil // already up to date
			}
			return os.WriteFile(filePath, []byte(newContent), 0644)
		}
	}

	// Append
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	if existing != "" {
		existing += "\n"
	}
	existing += wrapped + "\n"
	return os.WriteFile(filePath, []byte(existing), 0644)
}

// RemoveProtocol removes the Mio section (between markers) from a file.
func RemoveProtocol(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content := string(data)
	startIdx := strings.Index(content, markerBegin)
	if startIdx < 0 {
		return nil // no markers found
	}

	endIdx := strings.Index(content, markerEnd)
	if endIdx < 0 {
		return nil
	}
	endIdx += len(markerEnd)

	// Remove markers and surrounding whitespace
	newContent := content[:startIdx] + strings.TrimLeft(content[endIdx:], "\n")
	newContent = strings.TrimRight(newContent, "\n") + "\n"

	// If file is now empty, remove it
	if strings.TrimSpace(newContent) == "" {
		return os.Remove(filePath)
	}

	return os.WriteFile(filePath, []byte(newContent), 0644)
}

// HasProtocol returns true if the file contains Mio markers.
func HasProtocol(filePath string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), markerBegin)
}

// --- Skills helpers ---

// CopySkills copies skill files from srcDir to destDir.
// Returns the number of files installed.
func CopySkills(srcDir, destDir string) (int, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, err
	}

	installed := 0
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(srcDir, path)
		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		srcData, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Skip if identical
		if existing, err := os.ReadFile(destPath); err == nil {
			if string(existing) == string(srcData) {
				return nil
			}
		}

		if err := os.WriteFile(destPath, srcData, 0644); err != nil {
			return nil
		}
		installed++
		return nil
	})

	return installed, err
}

// RemoveSkills removes the skills directory.
func RemoveSkills(skillsDir string) error {
	if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
		return os.RemoveAll(skillsDir)
	}
	return nil
}

// --- Binary helpers ---

// FindBinaryPath returns the resolved path of the current executable.
func FindBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}

// FindProjectFile looks for a file relative to the binary location.
// Walks up directories looking for the file in the mio project.
func FindProjectFile(binPath, relPath string) string {
	binDir := filepath.Dir(binPath)
	candidates := []string{
		filepath.Join(binDir, relPath),
		filepath.Join(binDir, "..", relPath),
		filepath.Join(binDir, "..", "..", relPath),
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, relPath),
			filepath.Join(cwd, "..", relPath),
		)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

// PrintStep prints a formatted setup/uninstall step.
func PrintStep(status, msg string) {
	fmt.Printf("  [%s] %s\n", status, msg)
}
