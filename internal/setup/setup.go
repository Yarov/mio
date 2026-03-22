package setup

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
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

### Mio Architect — Automatic SDD Pipeline

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

The architect assesses scope (small/medium/large) and drives the SDD pipeline: explore → propose → spec → design → tasks → apply → verify → archive. Each phase saves artifacts to Mio memory for cross-session recovery.`

// Markers to identify the Mio section in CLAUDE.md
const (
	mioSectionStart = "## Mio — Persistent Memory Protocol (ALWAYS ACTIVE)"
	mioSectionEnd   = `Each phase saves artifacts to Mio memory for cross-session recovery.`
)

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

	// Read existing settings
	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse settings.json: %w", err)
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

	if added == 0 {
		fmt.Println("  [ok] All tools already in allowlist")
		return nil
	}

	permissions["allow"] = allowRaw
	settings["permissions"] = permissions

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(settingsPath, append(data, '\n'), 0644); err != nil {
		return err
	}

	fmt.Printf("  [ok] Added %d tools to allowlist\n", added)
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
