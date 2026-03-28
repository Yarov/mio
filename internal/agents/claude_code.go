package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
}

const launchdLabel = "com.mio.server"

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

	// 7. Install launchd
	if err := c.installLaunchd(binPath); err != nil {
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

	// 7. Remove launchd
	c.uninstallLaunchd()

	// 8. Purge data
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
	// Try to load from protocols/ directory
	src := FindProjectFile(binPath, "protocols/claude-code.md")
	if src != "" {
		if data, err := os.ReadFile(src); err == nil {
			return string(data)
		}
	}
	// Fallback: embedded minimal protocol
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

	if added == 0 && !statuslineChanged {
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

	if changed {
		out, _ := json.MarshalIndent(settings, "", "  ")
		os.WriteFile(settingsPath, append(out, '\n'), 0644)
	}
}

func (c *ClaudeCode) installStatusline(binPath, dir string) error {
	srcPath := FindProjectFile(binPath, "statusline.sh")
	if srcPath == "" {
		return fmt.Errorf("statusline.sh not found")
	}

	destPath := filepath.Join(dir, "statusline.sh")

	if existing, err := os.ReadFile(destPath); err == nil {
		newData, err := os.ReadFile(srcPath)
		if err == nil && string(existing) == string(newData) {
			PrintStep("ok", "Statusline already installed")
			return nil
		}
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return err
	}

	PrintStep("ok", fmt.Sprintf("Statusline → %s", destPath))
	return nil
}

func (c *ClaudeCode) installOutputStyle(binPath, dir string) error {
	srcPath := FindProjectFile(binPath, "output-styles/mio.md")
	if srcPath == "" {
		PrintStep("skip", "Output style not found in project")
		return nil
	}

	stylesDir := filepath.Join(dir, "output-styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		return err
	}

	destPath := filepath.Join(stylesDir, "mio.md")

	if existing, err := os.ReadFile(destPath); err == nil {
		newData, _ := os.ReadFile(srcPath)
		if string(existing) == string(newData) {
			PrintStep("ok", "Output style already installed")
			return nil
		}
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}

	PrintStep("ok", fmt.Sprintf("Output style → %s", destPath))
	return nil
}

func (c *ClaudeCode) installSkills(binPath, dir string) error {
	srcDir := FindProjectFile(binPath, "skills")
	if srcDir == "" {
		PrintStep("skip", "No skills directory found")
		return nil
	}

	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	destDir := filepath.Join(dir, "skills")
	installed, err := CopySkills(srcDir, destDir)
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

func (c *ClaudeCode) installLaunchd(binPath string) error {
	home := HomeDir()
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return err
	}

	plistPath := filepath.Join(agentsDir, launchdLabel+".plist")
	logPath := filepath.Join(home, ".mio", "server.log")

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>server</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, launchdLabel, binPath, logPath, logPath)

	if existing, err := os.ReadFile(plistPath); err == nil {
		if string(existing) == plist {
			PrintStep("ok", "Launchd service already installed")
			return nil
		}
	}

	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}

	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	unloadCmd.Run()

	loadCmd := exec.Command("launchctl", "load", plistPath)
	if out, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %s (%w)", string(out), err)
	}

	PrintStep("ok", fmt.Sprintf("Launchd service → %s", plistPath))
	PrintStep("ok", "Dashboard will auto-start on login and stay alive")
	return nil
}

func (c *ClaudeCode) uninstallLaunchd() {
	home := HomeDir()
	plistPath := filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")

	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	unloadCmd.Run()

	if err := os.Remove(plistPath); err == nil {
		PrintStep("ok", "Removed launchd service")
	} else if !os.IsNotExist(err) {
		PrintStep("warn", fmt.Sprintf("Could not remove launchd plist: %v", err))
	}
}
