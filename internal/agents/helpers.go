package agents

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	mio "mio"
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

// MergeMioMCPEnv merges key/value pairs into the "env" block of the mio MCP server
// in a shared mcp.json (e.g. ~/.cursor/mcp.json). Preserves other keys on the server entry.
func MergeMioMCPEnv(configPath string, env map[string]string) error {
	if len(env) == 0 {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	cfg := make(map[string]interface{})
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	servers, ok := cfg["mcpServers"].(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := servers["mio"].(map[string]interface{})
	if !ok || raw == nil {
		return nil
	}
	var existing map[string]interface{}
	if cur, ok := raw["env"].(map[string]interface{}); ok && cur != nil {
		existing = cur
	} else {
		existing = make(map[string]interface{})
	}
	for k, v := range env {
		existing[k] = v
	}
	raw["env"] = existing
	servers["mio"] = raw
	cfg["mcpServers"] = servers
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append(out, '\n'), 0644)
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

// CursorMCPAutoAllowPattern is the Cursor permissions.json allowlist entry so MCP
// tools from the server named "mio" run without per-call approval prompts.
// See: https://cursor.com/docs/reference/permissions
const CursorMCPAutoAllowPattern = "mio:*"

// CursorDisableWorkspaceTrustKey disables trust prompts at workspace open.
// This is a convenience default for teams that want zero-friction agent startup.
const CursorDisableWorkspaceTrustKey = "security.workspace.trust.enabled"

// MergeCursorPermissionsAllowlist adds pattern to the mcpAllowlist array in permissionsPath
// (~/.cursor/permissions.json). Preserves other top-level keys and existing entries.
// Accepts JSON or JSONC (comments/trailing commas), then writes normalized JSON.
func MergeCursorPermissionsAllowlist(permissionsPath, pattern string) error {
	if pattern == "" {
		return nil
	}
	cfg := make(map[string]interface{})
	if data, err := os.ReadFile(permissionsPath); err == nil {
		parsed, err := parseJSONOrJSONC(data)
		if err != nil {
			return fmt.Errorf("parse permissions.json: %w", err)
		}
		cfg = parsed
	} else if !os.IsNotExist(err) {
		return err
	}

	raw, _ := cfg["mcpAllowlist"].([]interface{})
	list := make([]string, 0, len(raw)+1)
	seen := make(map[string]struct{})
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		list = append(list, s)
	}
	if _, has := seen[pattern]; !has {
		list = append(list, pattern)
	}

	outList := make([]interface{}, len(list))
	for i, s := range list {
		outList[i] = s
	}
	cfg["mcpAllowlist"] = outList

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(permissionsPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(permissionsPath, append(data, '\n'), 0644)
}

// RemoveCursorPermissionsAllowlistPattern removes pattern from mcpAllowlist and drops
// the key if empty. Deletes the file if no top-level keys remain.
func RemoveCursorPermissionsAllowlistPattern(permissionsPath, pattern string) error {
	data, err := os.ReadFile(permissionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cfg, err := parseJSONOrJSONC(data)
	if err != nil {
		return nil // leave non-JSON files untouched
	}
	raw, ok := cfg["mcpAllowlist"].([]interface{})
	if !ok {
		return nil
	}
	filtered := make([]interface{}, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok || s == pattern {
			continue
		}
		filtered = append(filtered, s)
	}
	if len(filtered) == 0 {
		delete(cfg, "mcpAllowlist")
	} else {
		cfg["mcpAllowlist"] = filtered
	}
	if len(cfg) == 0 {
		return os.Remove(permissionsPath)
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(permissionsPath, append(out, '\n'), 0644)
}

// MergeCursorSettingsBool sets/overwrites a bool setting in ~/.cursor/settings.json.
// Accepts JSON or JSONC input and writes normalized JSON.
func MergeCursorSettingsBool(settingsPath, dottedKey string, value bool) error {
	if dottedKey == "" {
		return nil
	}
	cfg := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		parsed, err := parseJSONOrJSONC(data)
		if err != nil {
			return fmt.Errorf("parse settings.json: %w", err)
		}
		cfg = parsed
	} else if !os.IsNotExist(err) {
		return err
	}

	cfg[dottedKey] = value
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0644)
}

func parseJSONOrJSONC(data []byte) (map[string]interface{}, error) {
	cfg := make(map[string]interface{})
	if err := json.Unmarshal(data, &cfg); err == nil {
		return cfg, nil
	}

	sanitized := stripJSONTrailingCommas(stripJSONComments(strings.TrimPrefix(string(data), "\ufeff")))
	if err := json.Unmarshal([]byte(sanitized), &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON/JSONC: %w", err)
	}
	return cfg, nil
}

func stripJSONComments(s string) string {
	const (
		modeNormal = iota
		modeString
		modeLineComment
		modeBlockComment
	)
	var b strings.Builder
	mode := modeNormal
	escape := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch mode {
		case modeNormal:
			if c == '"' {
				mode = modeString
				b.WriteByte(c)
				continue
			}
			if c == '/' && i+1 < len(s) {
				n := s[i+1]
				if n == '/' {
					mode = modeLineComment
					i++
					continue
				}
				if n == '*' {
					mode = modeBlockComment
					i++
					continue
				}
			}
			b.WriteByte(c)
		case modeString:
			b.WriteByte(c)
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				mode = modeNormal
			}
		case modeLineComment:
			if c == '\n' {
				mode = modeNormal
				b.WriteByte('\n')
			}
		case modeBlockComment:
			if c == '*' && i+1 < len(s) && s[i+1] == '/' {
				mode = modeNormal
				i++
			}
		}
	}
	return b.String()
}

func stripJSONTrailingCommas(s string) string {
	var b strings.Builder
	inString := false
	escape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			b.WriteByte(c)
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			b.WriteByte(c)
			continue
		}

		if c == ',' {
			j := i + 1
			for j < len(s) {
				ws := s[j]
				if ws == ' ' || ws == '\t' || ws == '\n' || ws == '\r' {
					j++
					continue
				}
				break
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
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

// WriteIfChanged writes data to path only if the content has changed.
func WriteIfChanged(path string, data []byte) error {
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == string(data) {
			return nil
		}
	}
	return os.WriteFile(path, data, 0644)
}

// --- Embedded asset installation helpers ---
// These try embedded assets first (compiled binary), then fall back to
// filesystem lookup (development mode with FindProjectFile).

// InstallSkillsFromAssets installs skills, preferring embedded assets.
func InstallSkillsFromAssets(binPath, destDir string) (int, error) {
	// Primary: embedded assets
	if n, err := InstallEmbeddedDir("skills", destDir); err == nil {
		return n, nil
	}
	// Fallback: filesystem
	srcDir := FindProjectFile(binPath, "skills")
	if srcDir == "" {
		return 0, fmt.Errorf("skills not found")
	}
	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		return 0, fmt.Errorf("skills directory invalid")
	}
	return CopySkills(srcDir, destDir)
}

// InstallProtocolFromAssets installs a protocol file, preferring embedded assets.
// embeddedPath is like "protocols/cursor.md", destPath is the final destination.
func InstallProtocolFromAssets(binPath, embeddedPath, destPath string) error {
	// Primary: embedded assets
	if content, ok := ReadEmbeddedFile(embeddedPath); ok {
		return InstallProtocol(destPath, content)
	}
	// Fallback: filesystem
	srcPath := FindProjectFile(binPath, embeddedPath)
	if srcPath == "" {
		return fmt.Errorf("%s not found", embeddedPath)
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return InstallProtocol(destPath, string(data))
}

// InstallCursorMDCRuleFromAssets writes a Cursor-native rule file (.mdc): YAML frontmatter
// with alwaysApply, then the protocol body. Cursor applies .mdc rules more reliably than
// plain .md in ~/.cursor/rules (see Cursor community discussions on global rules).
func InstallCursorMDCRuleFromAssets(binPath, embeddedPath, destPath string) error {
	var body string
	if content, ok := ReadEmbeddedFile(embeddedPath); ok {
		body = content
	} else {
		srcPath := FindProjectFile(binPath, embeddedPath)
		if srcPath == "" {
			return fmt.Errorf("%s not found", embeddedPath)
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		body = string(data)
	}
	front := "---\ndescription: \"Mio — persistent memory via MCP (mem_save, mem_search, mem_context, session tools, SDD)\"\nalwaysApply: true\n---\n\n"
	out := front + body
	if existing, err := os.ReadFile(destPath); err == nil && string(existing) == out {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(destPath, []byte(out), 0644)
}

// ReadEmbeddedFile reads a file from the embedded assets.
// Returns the content and true if found, empty string and false otherwise.
func ReadEmbeddedFile(path string) (string, bool) {
	data, err := mio.Assets.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// InstallEmbeddedFile copies an embedded file to a destination path.
// Skips if content is identical. Sets the given file mode.
func InstallEmbeddedFile(embeddedPath, destPath string, mode os.FileMode) error {
	data, err := mio.Assets.ReadFile(embeddedPath)
	if err != nil {
		return fmt.Errorf("embedded file %s not found: %w", embeddedPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	if existing, err := os.ReadFile(destPath); err == nil {
		if string(existing) == string(data) {
			return nil // already identical
		}
	}

	return os.WriteFile(destPath, data, mode)
}

// InstallEmbeddedDir copies an embedded directory tree to a destination.
// Returns the number of files installed (skipping identical ones).
func InstallEmbeddedDir(embeddedDir, destDir string) (int, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, err
	}

	installed := 0
	err := fs.WalkDir(mio.Assets, embeddedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(embeddedDir, path)
		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		srcData, err := mio.Assets.ReadFile(path)
		if err != nil {
			return nil
		}

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

// PrintStep prints a formatted setup/uninstall step.
func PrintStep(status, msg string) {
	fmt.Printf("  [%s] %s\n", status, msg)
}
