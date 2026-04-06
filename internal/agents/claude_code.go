package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	Register(&ClaudeCode{})
}

// MCP tools that Mio exposes (used for Claude Code allowlist)
var mioTools = []string{
	"mcp__mio__mem_save",
	"mcp__mio__mem_search",
	"mcp__mio__mem_update",
	"mcp__mio__mem_delete",
	"mcp__mio__mem_get_observation",
	"mcp__mio__mem_context",
	"mcp__mio__mem_timeline",
	"mcp__mio__mem_session_start",
	"mcp__mio__mem_session_end",
	"mcp__mio__mem_session_summary",
	"mcp__mio__mem_save_prompt",
	"mcp__mio__mem_relations",
	"mcp__mio__mem_relate",
	"mcp__mio__mem_suggest_topic_key",
	"mcp__mio__mem_stats",
	"mcp__mio__mem_surface",
	"mcp__mio__mem_cross_project",
	"mcp__mio__mem_consolidate",
	"mcp__mio__mem_gc",
	"mcp__mio__mem_summarize",
	"mcp__mio__mem_graph",
	"mcp__mio__mem_enhanced_search",
	"mcp__mio__mem_agent_knowledge",
}

// ClaudeCode implements the Agent interface for Claude Code.
type ClaudeCode struct{}

func (c *ClaudeCode) Name() string        { return "claude-code" }
func (c *ClaudeCode) DisplayName() string { return "Claude Code" }

func (c *ClaudeCode) Detect() bool {
	return DetectByCommand("claude") || DetectByDir(c.claudeDir())
}

func (c *ClaudeCode) Status() AgentStatus {
	configPath := filepath.Join(c.claudeDir(), "mcp", "mio.json")
	return AgentStatus{
		Name:        c.Name(),
		DisplayName: c.DisplayName(),
		Installed:   c.Detect(),
		Configured:  HasMCPConfig(configPath),
		ConfigPath:  configPath,
	}
}

func (c *ClaudeCode) claudeDir() string {
	return filepath.Join(HomeDir(), ".claude")
}

func (c *ClaudeCode) Setup(binPath string) error {
	dir := c.claudeDir()

	// 1. Write MCP config (own file)
	configPath := filepath.Join(dir, "mcp", "mio.json")
	if err := WriteMCPToOwnFile(configPath, binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", configPath))

	// 2. Update allowlist + statusline in settings.json
	if err := c.updateSettings(dir); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}

	// 3. Install protocol using markers
	protocolFile := filepath.Join(dir, "CLAUDE.md")
	protocolContent := c.loadProtocol(binPath)
	if err := c.installClaudeProtocol(protocolFile, protocolContent); err != nil {
		return fmt.Errorf("install protocol: %w", err)
	}
	PrintStep("ok", "Memory protocol → CLAUDE.md")

	// 4. Install statusline
	if err := c.installStatusline(binPath, dir); err != nil {
		PrintStep("warn", fmt.Sprintf("Statusline: %v", err))
	}

	// 5. Install output style
	if err := c.installOutputStyle(binPath, dir); err != nil {
		PrintStep("warn", fmt.Sprintf("Output style: %v", err))
	}

	// 6. Install skills
	if err := c.installSkills(binPath, dir); err != nil {
		PrintStep("warn", fmt.Sprintf("Skills: %v", err))
	}

	// 7. Install hooks
	if err := c.installHooks(binPath, dir); err != nil {
		PrintStep("warn", fmt.Sprintf("Hooks: %v", err))
	}

	// 8. Install launchd (shared with Cursor setup)
	if err := InstallLaunchd(binPath); err != nil {
		PrintStep("warn", fmt.Sprintf("Launchd: %v", err))
	}

	return nil
}

func (c *ClaudeCode) Uninstall(purge bool) error {
	dir := c.claudeDir()

	// 1. Remove MCP config
	configPath := filepath.Join(dir, "mcp", "mio.json")
	if err := os.Remove(configPath); err == nil {
		PrintStep("ok", "Removed MCP config")
	} else if !os.IsNotExist(err) {
		PrintStep("warn", fmt.Sprintf("Could not remove MCP config: %v", err))
	}

	// 2. Remove tools from allowlist + statusline from settings
	c.removeFromSettings(dir)

	// 3. Remove protocol from CLAUDE.md
	protocolFile := filepath.Join(dir, "CLAUDE.md")
	c.removeClaudeProtocol(protocolFile)

	// 4. Remove statusline file
	if err := os.Remove(filepath.Join(dir, "statusline.sh")); err == nil {
		PrintStep("ok", "Removed statusline.sh")
	}

	// 5. Remove output style
	if err := os.Remove(filepath.Join(dir, "output-styles", "mio.md")); err == nil {
		PrintStep("ok", "Removed output style")
	}

	// 6. Remove skills
	skillsDir := filepath.Join(dir, "skills")
	if err := RemoveSkills(skillsDir); err == nil {
		PrintStep("ok", "Removed skills")
	}

	// 7. Remove hooks from settings.json
	c.removeHooksFromSettings(dir)

	// 8. Remove launchd only if no other agent still uses Mio
	if !otherAgentWantsLaunchd("claude-code") {
		UninstallLaunchd()
	}

	// 9. Purge data
	if purge {
		dataDir := filepath.Join(HomeDir(), ".mio")
		if info, err := os.Stat(dataDir); err == nil && info.IsDir() {
			if err := os.RemoveAll(dataDir); err == nil {
				PrintStep("ok", "Purged data directory (~/.mio)")
			}
		}
	} else {
		PrintStep("info", "Data directory (~/.mio) preserved. Use --purge to delete.")
	}

	return nil
}

// --- Claude Code specific helpers ---

func (c *ClaudeCode) loadProtocol(binPath string) string {
	// Primary: read from embedded assets (always available in compiled binary)
	if content, ok := ReadEmbeddedFile("protocols/claude-code.md"); ok {
		return content
	}
	// Fallback: try filesystem (development mode)
	src := FindProjectFile(binPath, "protocols/claude-code.md")
	if src != "" {
		if data, err := os.ReadFile(src); err == nil {
			return string(data)
		}
	}
	return "## Mio — Persistent Memory Protocol\n\nMio is an MCP server for persistent memory. Use mcp__mio__mem_save, mcp__mio__mem_search, mcp__mio__mem_context.\n"
}

// installClaudeProtocol handles Claude Code's special protocol format.
// Claude Code uses its own markers (section start/end) instead of HTML markers,
// for backward compatibility with existing installations.
func (c *ClaudeCode) installClaudeProtocol(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	existing := ""
	if data, err := os.ReadFile(filePath); err == nil {
		existing = string(data)
	}

	// Use standard markers for new installations
	sectionStart := "## Mio — Persistent Memory Protocol"

	if strings.Contains(existing, markerBegin) {
		// Already using marker format — update via standard helper
		return InstallProtocol(filePath, content)
	}

	if strings.Contains(existing, sectionStart) {
		// Legacy format (no markers) — find and replace the whole section
		startIdx := strings.Index(existing, sectionStart)
		// The section goes to the end of file (or next top-level heading)
		endContent := existing[startIdx:]
		nextH2 := strings.Index(endContent[1:], "\n## ")
		if nextH2 >= 0 {
			endIdx := startIdx + 1 + nextH2
			newContent := existing[:startIdx] + markerBegin + "\n" + content + "\n" + markerEnd + existing[endIdx:]
			return os.WriteFile(filePath, []byte(newContent), 0644)
		}
		// Goes to end of file
		newContent := existing[:startIdx] + markerBegin + "\n" + content + "\n" + markerEnd + "\n"
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	// New installation
	return InstallProtocol(filePath, content)
}

// removeClaudeProtocol removes the protocol from CLAUDE.md.
func (c *ClaudeCode) removeClaudeProtocol(filePath string) {
	// Try standard markers first
	if HasProtocol(filePath) {
		if err := RemoveProtocol(filePath); err == nil {
			PrintStep("ok", "Removed Mio section from CLAUDE.md")
			return
		}
	}

	// Legacy format without markers
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	content := string(data)
	sectionStart := "## Mio — Persistent Memory Protocol"
	startIdx := strings.Index(content, sectionStart)
	if startIdx < 0 {
		return
	}

	// Find end: next top-level heading or EOF
	rest := content[startIdx+1:]
	nextH2 := strings.Index(rest, "\n## ")
	var newContent string
	if nextH2 >= 0 {
		endIdx := startIdx + 1 + nextH2
		newContent = content[:startIdx] + strings.TrimLeft(content[endIdx:], "\n")
	} else {
		newContent = strings.TrimRight(content[:startIdx], "\n") + "\n"
	}

	if strings.TrimSpace(newContent) == "" || strings.TrimSpace(newContent) == "# Global Instructions" {
		os.Remove(filePath)
		PrintStep("ok", "Removed CLAUDE.md (was Mio-only)")
		return
	}

	os.WriteFile(filePath, []byte(newContent), 0644)
	PrintStep("ok", "Removed Mio section from CLAUDE.md")
}

func (c *ClaudeCode) updateSettings(dir string) error {
	settingsPath := filepath.Join(dir, "settings.json")

	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			_ = os.WriteFile(settingsPath+".bak", data, 0644)
			PrintStep("warn", "settings.json was corrupted, backed up")
			settings = make(map[string]interface{})
		}
	}

	permissions, ok := settings["permissions"].(map[string]interface{})
	if !ok {
		permissions = make(map[string]interface{})
	}

	allowRaw, ok := permissions["allow"].([]interface{})
	if !ok {
		allowRaw = []interface{}{}
	}

	existing := make(map[string]bool)
	for _, v := range allowRaw {
		if s, ok := v.(string); ok {
			existing[s] = true
		}
	}

	added := 0
	for _, tool := range mioTools {
		if !existing[tool] {
			allowRaw = append(allowRaw, tool)
			added++
		}
	}

	if added > 0 {
		permissions["allow"] = allowRaw
		settings["permissions"] = permissions
		PrintStep("ok", fmt.Sprintf("Added %d tools to allowlist", added))
	} else {
		PrintStep("ok", "All tools already in allowlist")
	}

	// Configure statusline
	statuslinePath := filepath.Join(dir, "statusline.sh")
	statuslineConfig := map[string]interface{}{
		"type":    "command",
		"command": statuslinePath,
	}
	statuslineChanged := false
	currentSL, _ := settings["statusLine"].(map[string]interface{})
	currentCmd, _ := currentSL["command"].(string)
	if currentCmd != statuslinePath {
		settings["statusLine"] = statuslineConfig
		statuslineChanged = true
		PrintStep("ok", "Statusline configured in settings.json")
	}

	// Configure output style
	outputStyleChanged := false
	currentOS, _ := settings["outputStyle"].(string)
	if currentOS != "mio" {
		settings["outputStyle"] = "mio"
		outputStyleChanged = true
		PrintStep("ok", "Output style set to 'mio' in settings.json")
	}

	if added == 0 && !statuslineChanged && !outputStyleChanged {
		return nil
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(data, '\n'), 0644)
}

func (c *ClaudeCode) removeFromSettings(dir string) {
	settingsPath := filepath.Join(dir, "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return
	}

	settings := make(map[string]interface{})
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}

	changed := false

	if permissions, ok := settings["permissions"].(map[string]interface{}); ok {
		if allowRaw, ok := permissions["allow"].([]interface{}); ok {
			mioToolSet := make(map[string]bool)
			for _, t := range mioTools {
				mioToolSet[t] = true
			}

			var filtered []interface{}
			for _, v := range allowRaw {
				if s, ok := v.(string); ok && mioToolSet[s] {
					continue
				}
				filtered = append(filtered, v)
			}

			if len(filtered) != len(allowRaw) {
				permissions["allow"] = filtered
				settings["permissions"] = permissions
				changed = true
				PrintStep("ok", fmt.Sprintf("Removed %d tools from allowlist", len(allowRaw)-len(filtered)))
			}
		}
	}

	if sl, ok := settings["statusLine"].(map[string]interface{}); ok {
		if cmd, _ := sl["command"].(string); strings.Contains(cmd, "statusline.sh") {
			delete(settings, "statusLine")
			changed = true
			PrintStep("ok", "Removed statusline from settings.json")
		}
	}

	if os, ok := settings["outputStyle"].(string); ok && os == "mio" {
		delete(settings, "outputStyle")
		changed = true
		PrintStep("ok", "Removed output style from settings.json")
	}

	if changed {
		out, _ := json.MarshalIndent(settings, "", "  ")
		os.WriteFile(settingsPath, append(out, '\n'), 0644)
	}
}

func (c *ClaudeCode) installStatusline(binPath, dir string) error {
	destPath := filepath.Join(dir, "statusline.sh")

	// Primary: embedded assets
	if err := InstallEmbeddedFile("statusline.sh", destPath, 0755); err == nil {
		PrintStep("ok", fmt.Sprintf("Statusline → %s", destPath))
		return nil
	}

	// Fallback: filesystem (development mode)
	srcPath := FindProjectFile(binPath, "statusline.sh")
	if srcPath == "" {
		return fmt.Errorf("statusline.sh not found")
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	if existing, err := os.ReadFile(destPath); err == nil {
		if string(existing) == string(data) {
			PrintStep("ok", "Statusline already installed")
			return nil
		}
	}

	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return err
	}
	PrintStep("ok", fmt.Sprintf("Statusline → %s", destPath))
	return nil
}

func (c *ClaudeCode) installOutputStyle(binPath, dir string) error {
	destPath := filepath.Join(dir, "output-styles", "mio.md")

	// Primary: embedded assets
	if err := InstallEmbeddedFile("output-styles/mio.md", destPath, 0644); err == nil {
		PrintStep("ok", fmt.Sprintf("Output style → %s", destPath))
		return nil
	}

	// Fallback: filesystem (development mode)
	srcPath := FindProjectFile(binPath, "output-styles/mio.md")
	if srcPath == "" {
		PrintStep("skip", "Output style not found")
		return nil
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	if existing, err := os.ReadFile(destPath); err == nil {
		if string(existing) == string(data) {
			PrintStep("ok", "Output style already installed")
			return nil
		}
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}
	PrintStep("ok", fmt.Sprintf("Output style → %s", destPath))
	return nil
}

func (c *ClaudeCode) installSkills(binPath, dir string) error {
	destDir := filepath.Join(dir, "skills")

	// Primary: embedded assets
	installed, err := InstallEmbeddedDir("skills", destDir)
	if err == nil && installed >= 0 {
		if installed == 0 {
			PrintStep("ok", "Skills already up to date")
		} else {
			PrintStep("ok", fmt.Sprintf("Installed %d skill files", installed))
		}
		return nil
	}

	// Fallback: filesystem (development mode)
	srcDir := FindProjectFile(binPath, "skills")
	if srcDir == "" {
		PrintStep("skip", "No skills directory found")
		return nil
	}

	info, statErr := os.Stat(srcDir)
	if statErr != nil || !info.IsDir() {
		return nil
	}

	installed, err = CopySkills(srcDir, destDir)
	if err != nil {
		return err
	}

	if installed == 0 {
		PrintStep("ok", "Skills already up to date")
	} else {
		PrintStep("ok", fmt.Sprintf("Installed %d skill files", installed))
	}
	return nil
}

func (c *ClaudeCode) installHooks(binPath, dir string) error {
	// Copy hook scripts to ~/.mio/hooks/
	mioHooksDir := filepath.Join(HomeDir(), ".mio", "hooks")
	if err := os.MkdirAll(mioHooksDir, 0755); err != nil {
		return err
	}

	hookFiles := []string{"user-prompt-submit.sh"}
	installed := 0
	for _, hookFile := range hookFiles {
		destPath := filepath.Join(mioHooksDir, hookFile)

		// Primary: embedded assets
		if err := InstallEmbeddedFile("hooks/"+hookFile, destPath, 0755); err == nil {
			installed++
			continue
		}

		// Fallback: filesystem (development mode)
		srcDir := FindProjectFile(binPath, "hooks")
		if srcDir == "" {
			continue
		}
		srcPath := filepath.Join(srcDir, hookFile)
		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}

		if existing, err := os.ReadFile(destPath); err == nil && string(existing) == string(srcData) {
			continue
		}

		if err := os.WriteFile(destPath, srcData, 0755); err != nil {
			return err
		}
		installed++
	}

	if installed > 0 {
		PrintStep("ok", fmt.Sprintf("Installed %d hook scripts → %s", installed, mioHooksDir))
	} else {
		PrintStep("ok", "Hooks already up to date")
	}

	// Configure hooks in settings.json
	settingsPath := filepath.Join(dir, "settings.json")
	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		json.Unmarshal(data, &settings)
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	hookPath := filepath.Join(mioHooksDir, "user-prompt-submit.sh")

	// Claude Code hook format: {"matcher": "", "hooks": [{"type": "command", "command": "..."}]}
	mioHookEntry := map[string]interface{}{
		"matcher": "",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": hookPath,
			},
		},
	}

	// Check if already configured
	if existing, ok := hooks["UserPromptSubmit"].([]interface{}); ok {
		for _, h := range existing {
			if hm, ok := h.(map[string]interface{}); ok {
				if hooksArr, ok := hm["hooks"].([]interface{}); ok {
					for _, hk := range hooksArr {
						if hkm, ok := hk.(map[string]interface{}); ok {
							if cmd, _ := hkm["command"].(string); cmd == hookPath {
								return nil // Already configured
							}
						}
					}
				}
			}
		}
		// Append to existing hooks
		hooks["UserPromptSubmit"] = append(existing, mioHookEntry)
	} else {
		hooks["UserPromptSubmit"] = []interface{}{mioHookEntry}
	}

	settings["hooks"] = hooks
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, append(data, '\n'), 0644); err != nil {
		return err
	}
	PrintStep("ok", "UserPromptSubmit hook configured in settings.json")
	return nil
}

func (c *ClaudeCode) removeHooksFromSettings(dir string) {
	settingsPath := filepath.Join(dir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return
	}

	settings := make(map[string]interface{})
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return
	}

	mioHooksDir := filepath.Join(HomeDir(), ".mio", "hooks")
	changed := false

	if upsHooks, ok := hooks["UserPromptSubmit"].([]interface{}); ok {
		var filtered []interface{}
		for _, h := range upsHooks {
			if hm, ok := h.(map[string]interface{}); ok {
				isMio := false
				// New format: check inside hooks array
				if hooksArr, ok := hm["hooks"].([]interface{}); ok {
					for _, hk := range hooksArr {
						if hkm, ok := hk.(map[string]interface{}); ok {
							if cmd, _ := hkm["command"].(string); strings.HasPrefix(cmd, mioHooksDir) {
								isMio = true
							}
						}
					}
				}
				// Legacy format: check command directly
				if cmd, _ := hm["command"].(string); strings.HasPrefix(cmd, mioHooksDir) {
					isMio = true
				}
				if isMio {
					changed = true
					continue
				}
			}
			filtered = append(filtered, h)
		}
		if len(filtered) == 0 {
			delete(hooks, "UserPromptSubmit")
		} else {
			hooks["UserPromptSubmit"] = filtered
		}
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}

	if changed {
		out, _ := json.MarshalIndent(settings, "", "  ")
		os.WriteFile(settingsPath, append(out, '\n'), 0644)
		PrintStep("ok", "Removed hooks from settings.json")
	}
}
