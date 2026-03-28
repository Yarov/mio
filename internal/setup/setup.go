package setup

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MCP tools that Mio exposes
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

// Memory protocol instructions injected into CLAUDE.md
const memoryProtocol = `## Mio — Persistent Memory Protocol (ALWAYS ACTIVE)

Mio is an MCP server for persistent memory across sessions. This protocol is MANDATORY.

### PROACTIVE SAVE — do NOT wait for the user to ask

Call ` + "`mcp__mio__mem_save`" + ` IMMEDIATELY after ANY of these:

**After decisions or conventions:**
- Architecture or design decision made
- Convention documented or established
- Tool or library choice made with tradeoffs
- Workflow change agreed upon

**After completing work:**
- Bug fix completed (include root cause)
- Feature implemented with non-obvious approach
- Configuration change or environment setup done

**After discoveries:**
- Non-obvious discovery about the codebase
- Gotcha, edge case, or unexpected behavior found
- Pattern established (naming, structure, convention)
- User preference or constraint learned

**Self-check after EVERY task:**
> "Did I just make a decision, fix a bug, learn something non-obvious, or establish a convention? If yes, call mem_save NOW."

### SEARCH MEMORY — check before starting work

Call ` + "`mcp__mio__mem_search`" + ` or ` + "`mcp__mio__mem_context`" + ` when:
- User's FIRST message references a project or feature — search for prior work before responding
- Starting work on something that might have been done before
- User asks to recall anything ("remember", "what did we do", "acordate", "que hicimos")
- User mentions a topic you have no context on

### SESSION START

At the beginning of every session, call ` + "`mcp__mio__mem_context`" + ` to load recent memories and recover context from prior sessions.

### SESSION CLOSE — before saying "done" / "listo"

Call ` + "`mcp__mio__mem_session_end`" + ` with a summary structured as:

` + "```" + `
Goal: [what we were working on]
Accomplished: [completed items with key details]
Discoveries: [technical findings, gotchas, non-obvious learnings]
Next Steps: [what remains to be done]
Files: [key files modified]
` + "```" + `

This is NOT optional. If you skip this, the next session starts blind.

### Memory format

Structure content for mem_save as:
` + "```" + `
What: [what was done]
Why: [motivation/context]
Where: [files/modules affected]
Learned: [key takeaway]
` + "```" + `

### Observation types

Use the correct type: ` + "`bugfix`" + `, ` + "`decision`" + `, ` + "`architecture`" + `, ` + "`discovery`" + `, ` + "`pattern`" + `, ` + "`config`" + `, ` + "`preference`" + `, ` + "`learning`" + `, ` + "`summary`" + `

### Topic keys

For evolving topics, use ` + "`topic_key`" + ` so updates replace instead of duplicating. Use ` + "`mcp__mio__mem_suggest_topic_key`" + ` to generate one.

### Relations

When a new decision supersedes an old one, use ` + "`mcp__mio__mem_relate`" + ` with type "supersedes". When fixing a bug caused by a prior decision, use "caused_by".

### Mio Architect — Automatic SDD Pipeline (v2.0)

When the user describes a **significant change** (new feature, refactor, complex bugfix), activate the ` + "`mio-architect`" + ` skill automatically:

**Activate when:**
- User describes a new feature: "quiero agregar...", "add...", "necesito..."
- User requests a refactor: "refactoriza...", "mejora...", "cambia..."
- User has a complex bug touching multiple files
- User explicitly says: "architect", "sdd", "planea", "diseña"

**Do NOT activate for:**
- Simple fixes (typos, one-line changes, obvious bugs)
- Questions about code
- Running commands

**Pipeline:** The architect assesses scope (small/medium/large) and drives the SDD pipeline. Each phase is delegated to a sub-agent with fresh context via the Agent tool — the orchestrator NEVER does real work directly.

**Shortcuts:**
- ` + "`/sdd-ff {description}`" + ` — Fast-forward through planning phases (explore → propose → spec → design → tasks), stop before implementation. Best for medium-scope changes.
- ` + "`/sdd-continue`" + ` — Auto-detect pipeline state and execute the next phase. Works across sessions — recovers full state from Mio memory.
- ` + "`/mio-architect {description}`" + ` — Full pipeline with user approval at each gate. Best for large changes.

**Phases:** explore → propose → spec + design (parallel) → tasks → apply → verify → archive. Each phase saves artifacts to Mio memory for cross-session recovery.`

// Markers to identify the Mio section in CLAUDE.md
const (
	mioSectionStart = "## Mio — Persistent Memory Protocol (ALWAYS ACTIVE)"
	mioSectionEnd   = `Each phase saves artifacts to Mio memory for cross-session recovery.`
)

// UninstallClaudeCode removes all Mio integrations from Claude Code.
// By default keeps ~/.mio/ data intact. Use purge=true to also delete data.
func UninstallClaudeCode(purge bool) error {
	fmt.Println("Uninstalling Mio from Claude Code...")

	dir, err := claudeDir()
	if err != nil {
		return err
	}

	// 1. Remove MCP config
	mcpConfig := filepath.Join(dir, "mcp", "mio.json")
	if err := os.Remove(mcpConfig); err == nil {
		fmt.Println("  [ok] Removed MCP config")
	} else if !os.IsNotExist(err) {
		fmt.Printf("  [warn] Could not remove MCP config: %v\n", err)
	} else {
		fmt.Println("  [skip] MCP config not found")
	}

	// 2. Remove tools from allowlist and statusline from settings.json
	if err := removeFromSettings(dir); err != nil {
		fmt.Printf("  [warn] Could not update settings.json: %v\n", err)
	}

	// 3. Remove Mio section from CLAUDE.md
	if err := removeMemoryProtocol(dir); err != nil {
		fmt.Printf("  [warn] Could not update CLAUDE.md: %v\n", err)
	}

	// 4. Remove statusline
	statusline := filepath.Join(dir, "statusline.sh")
	if err := os.Remove(statusline); err == nil {
		fmt.Println("  [ok] Removed statusline.sh")
	} else if !os.IsNotExist(err) {
		fmt.Printf("  [warn] Could not remove statusline: %v\n", err)
	}

	// 5. Remove output style
	outputStyle := filepath.Join(dir, "output-styles", "mio.md")
	if err := os.Remove(outputStyle); err == nil {
		fmt.Println("  [ok] Removed output style")
	} else if !os.IsNotExist(err) {
		fmt.Printf("  [warn] Could not remove output style: %v\n", err)
	}

	// 6. Remove skills
	skillsDir := filepath.Join(dir, "skills")
	if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
		if err := os.RemoveAll(skillsDir); err == nil {
			fmt.Println("  [ok] Removed all skills")
		} else {
			fmt.Printf("  [warn] Could not remove skills: %v\n", err)
		}
	}

	// 7. Remove launchd service
	uninstallLaunchd()

	// 8. Purge data if requested
	if purge {
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".mio")
		if info, err := os.Stat(dataDir); err == nil && info.IsDir() {
			if err := os.RemoveAll(dataDir); err == nil {
				fmt.Println("  [ok] Purged data directory (~/.mio)")
			} else {
				fmt.Printf("  [warn] Could not purge data: %v\n", err)
			}
		}
	} else {
		fmt.Println("  [info] Data directory (~/.mio) preserved. Use --purge to also delete data.")
	}

	fmt.Println("\nMio has been uninstalled. Restart Claude Code to complete.")
	return nil
}

// removeFromSettings removes Mio tools from allowlist and statusline from settings.json
func removeFromSettings(dir string) error {
	settingsPath := filepath.Join(dir, "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  [skip] settings.json not found")
			return nil
		}
		return err
	}

	settings := make(map[string]interface{})
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse settings.json: %w", err)
	}

	changed := false

	// Remove tools from allowlist
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
				fmt.Printf("  [ok] Removed %d tools from allowlist\n", len(allowRaw)-len(filtered))
			}
		}
	}

	// Remove statusline
	if sl, ok := settings["statusLine"].(map[string]interface{}); ok {
		if cmd, _ := sl["command"].(string); strings.Contains(cmd, "statusline.sh") {
			delete(settings, "statusLine")
			changed = true
			fmt.Println("  [ok] Removed statusline from settings.json")
		}
	}

	if !changed {
		fmt.Println("  [skip] No Mio entries in settings.json")
		return nil
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0644)
}

// removeMemoryProtocol removes the Mio section from CLAUDE.md
func removeMemoryProtocol(dir string) error {
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")

	data, err := os.ReadFile(claudeMDPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  [skip] CLAUDE.md not found")
			return nil
		}
		return err
	}

	content := string(data)
	startIdx := strings.Index(content, mioSectionStart)
	if startIdx < 0 {
		fmt.Println("  [skip] Mio section not found in CLAUDE.md")
		return nil
	}

	endIdx := strings.Index(content, mioSectionEnd)
	if endIdx < 0 {
		// Remove from start marker to end of file
		content = strings.TrimRight(content[:startIdx], "\n")
	} else {
		endIdx += len(mioSectionEnd)
		content = content[:startIdx] + strings.TrimLeft(content[endIdx:], "\n")
	}

	content = strings.TrimRight(content, "\n") + "\n"

	// If file is now empty (only had Mio section), remove it
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || trimmed == "# Global Instructions" {
		if err := os.Remove(claudeMDPath); err == nil {
			fmt.Println("  [ok] Removed CLAUDE.md (was Mio-only)")
			return nil
		}
	}

	if err := os.WriteFile(claudeMDPath, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Println("  [ok] Removed Mio section from CLAUDE.md")
	return nil
}

// SetupClaudeCode configures Mio as an MCP server for Claude Code.
func SetupClaudeCode() error {
	fmt.Println("Setting up Mio for Claude Code...")

	// 1. Find binary path
	binPath, err := findBinaryPath()
	if err != nil {
		return fmt.Errorf("cannot find mio binary: %w\nRun 'make install' first", err)
	}
	fmt.Printf("  [ok] Found mio at %s\n", binPath)

	// 2. Write MCP config
	if err := writeMCPConfig(binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}

	// 3. Update allowlist
	if err := updateAllowlist(); err != nil {
		return fmt.Errorf("update allowlist: %w", err)
	}

	// 4. Install memory protocol instructions
	if err := installMemoryProtocol(); err != nil {
		return fmt.Errorf("install memory protocol: %w", err)
	}

	// 5. Install statusline
	if err := installStatusline(binPath); err != nil {
		fmt.Printf("  [warn] Could not install statusline: %v\n", err)
	}

	// 6. Install output style
	if err := installOutputStyle(); err != nil {
		fmt.Printf("  [warn] Could not install output style: %v\n", err)
	}

	// 7. Install skills
	if err := installSkills(binPath); err != nil {
		fmt.Printf("  [warn] Could not install skills: %v\n", err)
	}

	// 8. Install launchd service for persistent HTTP dashboard
	if err := installLaunchd(binPath); err != nil {
		fmt.Printf("  [warn] Could not install launchd service: %v\n", err)
	}

	fmt.Println("\nMio is ready! Restart Claude Code to activate.")
	return nil
}

func findBinaryPath() (string, error) {
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

func claudeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

// writeMCPConfig creates ~/.claude/mcp/mio.json
func writeMCPConfig(binPath string) error {
	dir, err := claudeDir()
	if err != nil {
		return err
	}

	mcpDir := filepath.Join(dir, "mcp")
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return fmt.Errorf("create mcp dir: %w", err)
	}

	configPath := filepath.Join(mcpDir, "mio.json")

	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"mio": map[string]interface{}{
				"command": binPath,
				"args":    []string{"mcp"},
			},
		},
	}

	// Check if already configured with same content
	if existing, err := os.ReadFile(configPath); err == nil {
		newData, _ := json.MarshalIndent(config, "", "  ")
		if string(existing) == string(newData)+"\n" || string(existing) == string(newData) {
			fmt.Println("  [ok] MCP config already configured")
			return nil
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, append(data, '\n'), 0644); err != nil {
		return err
	}

	fmt.Printf("  [ok] MCP config written to %s\n", configPath)
	return nil
}

// updateAllowlist adds Mio tools to ~/.claude/settings.json permissions.allow
func updateAllowlist() error {
	dir, err := claudeDir()
	if err != nil {
		return err
	}

	settingsPath := filepath.Join(dir, "settings.json")

	// Read existing settings (with recovery from corruption)
	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			// Backup corrupted file and start fresh
			backupPath := settingsPath + ".bak"
			_ = os.WriteFile(backupPath, data, 0644)
			fmt.Printf("  [warn] settings.json was corrupted, backed up to %s\n", backupPath)
			settings = make(map[string]interface{})
		}
	}

	// Get or create permissions.allow
	permissions, ok := settings["permissions"].(map[string]interface{})
	if !ok {
		permissions = make(map[string]interface{})
	}

	allowRaw, ok := permissions["allow"].([]interface{})
	if !ok {
		allowRaw = []interface{}{}
	}

	// Build lookup of existing entries
	existing := make(map[string]bool)
	for _, v := range allowRaw {
		if s, ok := v.(string); ok {
			existing[s] = true
		}
	}

	// Add missing tools
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
		fmt.Printf("  [ok] Added %d tools to allowlist\n", added)
	} else {
		fmt.Println("  [ok] All tools already in allowlist")
	}

	// Configure statusline as object
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
		fmt.Println("  [ok] Statusline configured in settings.json")
	}

	if added == 0 && !statuslineChanged {
		return nil
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(settingsPath, append(data, '\n'), 0644); err != nil {
		return err
	}

	return nil
}

// installMemoryProtocol writes the memory protocol to ~/.claude/CLAUDE.md
func installMemoryProtocol() error {
	dir, err := claudeDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	claudeMDPath := filepath.Join(dir, "CLAUDE.md")
	content := ""

	// Read existing CLAUDE.md if it exists
	if data, err := os.ReadFile(claudeMDPath); err == nil {
		content = string(data)
	}

	// Check if Mio section already exists
	if strings.Contains(content, mioSectionStart) {
		// Find the section and replace it
		startIdx := strings.Index(content, mioSectionStart)
		endIdx := strings.Index(content, mioSectionEnd)

		if startIdx >= 0 && endIdx >= 0 {
			endIdx += len(mioSectionEnd)
			newContent := content[:startIdx] + memoryProtocol + content[endIdx:]
			if newContent == content {
				fmt.Println("  [ok] Memory protocol already up to date")
				return nil
			}
			content = newContent
		} else {
			// End marker not found — replace from start marker to end of file
			// (or to next top-level heading)
			content = content[:startIdx] + memoryProtocol + "\n"
		}
	} else {
		// Append to existing content
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		content += memoryProtocol + "\n"
	}

	if err := os.WriteFile(claudeMDPath, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Printf("  [ok] Memory protocol installed in %s\n", claudeMDPath)
	return nil
}

// installStatusline copies statusline.sh to ~/.claude/
func installStatusline(binPath string) error {
	// Find statusline.sh relative to the binary's source project
	srcPath := findProjectFile(binPath, "statusline.sh")
	if srcPath == "" {
		return fmt.Errorf("statusline.sh not found")
	}

	dir, err := claudeDir()
	if err != nil {
		return err
	}

	destPath := filepath.Join(dir, "statusline.sh")

	// Check if already installed and identical
	if existing, err := os.ReadFile(destPath); err == nil {
		newData, err := os.ReadFile(srcPath)
		if err == nil && string(existing) == string(newData) {
			fmt.Println("  [ok] Statusline already installed")
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

	fmt.Printf("  [ok] Statusline installed in %s\n", destPath)
	return nil
}

// installOutputStyle copies the output style to ~/.claude/output-styles/
func installOutputStyle() error {
	dir, err := claudeDir()
	if err != nil {
		return err
	}

	stylesDir := filepath.Join(dir, "output-styles")
	if err := os.MkdirAll(stylesDir, 0755); err != nil {
		return err
	}

	destPath := filepath.Join(stylesDir, "mio.md")

	// Find source from project
	exe, _ := os.Executable()
	srcPath := findProjectFile(exe, "output-styles/mio.md")
	if srcPath == "" {
		fmt.Println("  [skip] Output style not found in project")
		return nil
	}

	if existing, err := os.ReadFile(destPath); err == nil {
		newData, _ := os.ReadFile(srcPath)
		if string(existing) == string(newData) {
			fmt.Println("  [ok] Output style already installed")
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

	fmt.Printf("  [ok] Output style installed in %s\n", destPath)
	return nil
}

// installSkills copies skill files from the project's skills/ directory to ~/.claude/skills/
func installSkills(binPath string) error {
	srcDir := findProjectFile(binPath, "skills")
	if srcDir == "" {
		fmt.Println("  [skip] No skills directory found in project")
		return nil
	}

	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	dir, err := claudeDir()
	if err != nil {
		return err
	}

	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	installed := 0
	err = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(srcDir, path)
		destPath := filepath.Join(skillsDir, relPath)

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

	if installed == 0 {
		fmt.Println("  [ok] Skills already up to date")
	} else {
		fmt.Printf("  [ok] Installed %d skill files\n", installed)
	}
	return err
}

const launchdLabel = "com.mio.server"

// installLaunchd creates a launchd plist so the HTTP dashboard starts on login and stays alive.
func installLaunchd(binPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

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

	// Check if already installed and identical
	if existing, err := os.ReadFile(plistPath); err == nil {
		if string(existing) == plist {
			fmt.Println("  [ok] Launchd service already installed")
			return nil
		}
	}

	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}

	// Unload old version if any, then load new one
	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	unloadCmd.Run() // ignore error — may not be loaded

	loadCmd := exec.Command("launchctl", "load", plistPath)
	if out, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %s (%w)", string(out), err)
	}

	fmt.Printf("  [ok] Launchd service installed (%s)\n", plistPath)
	fmt.Println("  [ok] Dashboard will auto-start on login and stay alive")
	return nil
}

// uninstallLaunchd removes the launchd plist.
func uninstallLaunchd() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")

	// Unload first
	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	unloadCmd.Run()

	if err := os.Remove(plistPath); err == nil {
		fmt.Println("  [ok] Removed launchd service")
	} else if !os.IsNotExist(err) {
		fmt.Printf("  [warn] Could not remove launchd plist: %v\n", err)
	}
}

// findProjectFile looks for a file relative to the binary location
// Walks up directories looking for the file in the mio project
func findProjectFile(binPath, relPath string) string {
	// Try relative to binary
	binDir := filepath.Dir(binPath)
	candidates := []string{
		filepath.Join(binDir, relPath),
		filepath.Join(binDir, "..", relPath),
		filepath.Join(binDir, "..", "..", relPath),
	}

	// Also try current working directory
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
