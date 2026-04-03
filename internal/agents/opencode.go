package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register(&OpenCode{})
}

// OpenCode implements the Agent interface for OpenCode.
type OpenCode struct{}

func (o *OpenCode) Name() string        { return "opencode" }
func (o *OpenCode) DisplayName() string { return "OpenCode" }

func (o *OpenCode) Detect() bool {
	return DetectByCommand("opencode") || DetectByDir(o.configDir())
}

func (o *OpenCode) configDir() string {
	return filepath.Join(HomeDir(), ".config", "opencode")
}

func (o *OpenCode) configPath() string {
	return filepath.Join(o.configDir(), "opencode.json")
}

func (o *OpenCode) protocolPath() string {
	return filepath.Join(o.configDir(), "mio-protocol.md")
}

func (o *OpenCode) pluginsDir() string {
	return filepath.Join(o.configDir(), "plugins")
}

func (o *OpenCode) commandsDir() string {
	return filepath.Join(o.configDir(), "commands")
}

func (o *OpenCode) Status() AgentStatus {
	return AgentStatus{
		Name:        o.Name(),
		DisplayName: o.DisplayName(),
		Installed:   o.Detect(),
		Configured:  hasOpenCodeMCP(o.configPath()),
		ConfigPath:  o.configPath(),
	}
}

func (o *OpenCode) Setup(binPath string) error {
	// Read or create base config
	config := o.readConfig()

	// 1. MCP server entry
	mcp, _ := config["mcp"].(map[string]interface{})
	if mcp == nil {
		mcp = make(map[string]interface{})
	}
	mcp["mio"] = map[string]interface{}{
		"type":    "local",
		"command": []string{binPath, "mcp"},
		"enabled": true,
	}
	config["mcp"] = mcp
	PrintStep("ok", "MCP server configured")

	// 2. Auto-allow Mio tools (no prompts for memory operations)
	config = o.configurePermissions(config)

	// 3. Protocol file → instructions array
	if content, ok := ReadEmbeddedFile("protocols/opencode.md"); ok {
		if err := os.MkdirAll(filepath.Dir(o.protocolPath()), 0755); err != nil {
			PrintStep("warn", fmt.Sprintf("Protocol dir: %v", err))
		} else if err := WriteIfChanged(o.protocolPath(), []byte(content)); err != nil {
			PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
		} else {
			PrintStep("ok", fmt.Sprintf("Protocol → %s", o.protocolPath()))
		}
		config = addInstruction(config, o.protocolPath())
	} else {
		protocolSrc := FindProjectFile(binPath, "protocols/opencode.md")
		if protocolSrc != "" {
			data, err := os.ReadFile(protocolSrc)
			if err == nil {
				if err := os.MkdirAll(filepath.Dir(o.protocolPath()), 0755); err != nil {
					PrintStep("warn", fmt.Sprintf("Protocol dir: %v", err))
				} else if err := WriteIfChanged(o.protocolPath(), data); err != nil {
					PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
				} else {
					PrintStep("ok", fmt.Sprintf("Protocol → %s", o.protocolPath()))
				}
			}
			config = addInstruction(config, o.protocolPath())
		}
	}

	// 4. Plugins (backend + TUI)
	o.installPlugins(binPath, config)

	// 5. Custom commands (SDD pipeline mapped to OpenCode commands)
	o.installCommands(binPath)

	// 6. Write merged config
	if err := o.writeConfig(config); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("Config → %s", o.configPath()))

	return nil
}

func (o *OpenCode) Uninstall(_ bool) error {
	config := o.readConfig()
	changed := false

	// 1. Remove MCP entry
	if mcp, ok := config["mcp"].(map[string]interface{}); ok {
		if _, exists := mcp["mio"]; exists {
			delete(mcp, "mio")
			config["mcp"] = mcp
			changed = true
			PrintStep("ok", "Removed MCP config")
		}
	}

	// 2. Remove permissions
	if o.removePermissions(config) {
		changed = true
	}

	// 3. Remove protocol from instructions
	if removeInstruction(config, o.protocolPath()) {
		changed = true
		PrintStep("ok", "Removed protocol from instructions")
	}

	// 4. Remove protocol file
	if err := os.Remove(o.protocolPath()); err == nil {
		PrintStep("ok", "Removed protocol file")
	}

	// 5. Remove plugins
	if err := os.Remove(filepath.Join(o.pluginsDir(), "mio.ts")); err == nil {
		PrintStep("ok", "Removed backend plugin")
	}
	tuiDir := filepath.Join(o.pluginsDir(), "opencode-mio")
	if err := os.RemoveAll(tuiDir); err == nil {
		PrintStep("ok", "Removed TUI plugin")
	}
	o.unregisterTuiPlugin("./plugins/opencode-mio")

	// 6. Remove custom commands
	o.removeCommands()

	// 7. Write config if changed
	if changed {
		if err := o.writeConfig(config); err != nil {
			PrintStep("warn", fmt.Sprintf("Write config: %v", err))
		}
	}

	return nil
}

// --- Config read/write ---

func (o *OpenCode) readConfig() map[string]interface{} {
	config := make(map[string]interface{})
	data, err := os.ReadFile(o.configPath())
	if err != nil {
		return config
	}
	if err := json.Unmarshal(data, &config); err != nil {
		_ = os.WriteFile(o.configPath()+".bak", data, 0644)
		return make(map[string]interface{})
	}
	return config
}

func (o *OpenCode) writeConfig(config map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(o.configPath()), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(o.configPath(), append(data, '\n'), 0644)
}

// --- Permissions ---

// configurePermissions adds auto-allow for all Mio MCP tools using wildcard.
func (o *OpenCode) configurePermissions(config map[string]interface{}) map[string]interface{} {
	perms, _ := config["permission"].(map[string]interface{})
	if perms == nil {
		perms = make(map[string]interface{})
	}

	// Wildcard: allow all mio MCP tools without prompting
	if current, _ := perms["mio_*"].(string); current != "allow" {
		perms["mio_*"] = "allow"
		config["permission"] = perms
		PrintStep("ok", "Auto-allow Mio tools (permission: mio_* → allow)")
	} else {
		PrintStep("ok", "Mio tool permissions already configured")
	}

	return config
}

// removePermissions removes the Mio permission entry.
func (o *OpenCode) removePermissions(config map[string]interface{}) bool {
	perms, ok := config["permission"].(map[string]interface{})
	if !ok {
		return false
	}

	if _, exists := perms["mio_*"]; !exists {
		return false
	}

	delete(perms, "mio_*")
	if len(perms) == 0 {
		delete(config, "permission")
	} else {
		config["permission"] = perms
	}
	PrintStep("ok", "Removed Mio tool permissions")
	return true
}

// --- Instructions ---

func addInstruction(config map[string]interface{}, path string) map[string]interface{} {
	var instructions []interface{}
	if existing, ok := config["instructions"].([]interface{}); ok {
		instructions = existing
	}

	for _, v := range instructions {
		if s, ok := v.(string); ok && s == path {
			return config
		}
	}

	instructions = append(instructions, path)
	config["instructions"] = instructions
	return config
}

func removeInstruction(config map[string]interface{}, path string) bool {
	instructions, ok := config["instructions"].([]interface{})
	if !ok {
		return false
	}

	filtered := make([]interface{}, 0, len(instructions))
	removed := false
	for _, v := range instructions {
		if s, ok := v.(string); ok && s == path {
			removed = true
			continue
		}
		filtered = append(filtered, v)
	}

	if removed {
		if len(filtered) == 0 {
			delete(config, "instructions")
		} else {
			config["instructions"] = filtered
		}
	}
	return removed
}

// --- Plugins ---

func (o *OpenCode) installPlugins(binPath string, config map[string]interface{}) {
	destDir := o.pluginsDir()
	if err := os.MkdirAll(destDir, 0755); err != nil {
		PrintStep("warn", fmt.Sprintf("Plugins dir: %v", err))
		return
	}

	// 1. Backend plugin (session hooks)
	destPath := filepath.Join(destDir, "mio.ts")
	if err := InstallEmbeddedFile("plugins/opencode/mio.ts", destPath, 0644); err == nil {
		PrintStep("ok", fmt.Sprintf("Backend plugin → %s", destPath))
	} else {
		backendSrc := FindProjectFile(binPath, "plugins/opencode/mio.ts")
		if backendSrc != "" {
			data, err := os.ReadFile(backendSrc)
			if err == nil {
				if err := WriteIfChanged(destPath, data); err != nil {
					PrintStep("warn", fmt.Sprintf("Backend plugin: %v", err))
				} else {
					PrintStep("ok", fmt.Sprintf("Backend plugin → %s", destPath))
				}
			}
		}
	}

	// 2. TUI plugin (local package for status footer)
	tuiDestDir := filepath.Join(destDir, "opencode-mio")
	if n, err := InstallEmbeddedDir("plugins/opencode-mio", tuiDestDir); err == nil {
		if n > 0 {
			PrintStep("ok", fmt.Sprintf("TUI plugin → %s (%d files)", tuiDestDir, n))
		} else {
			PrintStep("ok", "TUI plugin already up to date")
		}
	} else {
		tuiSrcDir := FindProjectFile(binPath, "plugins/opencode-mio")
		if tuiSrcDir != "" {
			n, err := CopySkills(tuiSrcDir, tuiDestDir)
			if err != nil {
				PrintStep("warn", fmt.Sprintf("TUI plugin: %v", err))
			} else if n > 0 {
				PrintStep("ok", fmt.Sprintf("TUI plugin → %s (%d files)", tuiDestDir, n))
			} else {
				PrintStep("ok", "TUI plugin already up to date")
			}
		}
	}

	// 3. Register TUI plugin in tui.json
	if err := o.registerTuiPlugin("./plugins/opencode-mio"); err != nil {
		PrintStep("warn", fmt.Sprintf("tui.json: %v", err))
	} else {
		PrintStep("ok", "TUI plugin registered in tui.json")
	}
}

// addPluginRef adds a plugin reference to the "plugin" array in config.
func addPluginRef(config map[string]interface{}, ref string) {
	var plugins []interface{}
	if existing, ok := config["plugin"].([]interface{}); ok {
		plugins = existing
	}

	for _, p := range plugins {
		if s, ok := p.(string); ok && s == ref {
			return // already registered
		}
	}

	plugins = append(plugins, ref)
	config["plugin"] = plugins
}

// removePluginRef removes a plugin reference from the "plugin" array in config.
func removePluginRef(config map[string]interface{}, ref string) bool {
	plugins, ok := config["plugin"].([]interface{})
	if !ok {
		return false
	}

	filtered := make([]interface{}, 0, len(plugins))
	removed := false
	for _, p := range plugins {
		if s, ok := p.(string); ok && s == ref {
			removed = true
			continue
		}
		filtered = append(filtered, p)
	}

	if removed {
		if len(filtered) == 0 {
			delete(config, "plugin")
		} else {
			config["plugin"] = filtered
		}
	}
	return removed
}

// --- TUI plugin registration (tui.json) ---

func (o *OpenCode) registerTuiPlugin(ref string) error {
	tuiPath := filepath.Join(o.configDir(), "tui.json")

	config := make(map[string]interface{})
	if data, err := os.ReadFile(tuiPath); err == nil {
		json.Unmarshal(data, &config)
	}

	addPluginRef(config, ref)

	if _, ok := config["$schema"]; !ok {
		config["$schema"] = "https://opencode.ai/tui.json"
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tuiPath, append(data, '\n'), 0644)
}

func (o *OpenCode) unregisterTuiPlugin(ref string) {
	tuiPath := filepath.Join(o.configDir(), "tui.json")
	data, err := os.ReadFile(tuiPath)
	if err != nil {
		return
	}

	config := make(map[string]interface{})
	if err := json.Unmarshal(data, &config); err != nil {
		return
	}

	if removePluginRef(config, ref) {
		out, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(tuiPath, append(out, '\n'), 0644)
		PrintStep("ok", "Removed TUI plugin from tui.json")
	}
}

// --- Custom Commands (SDD pipeline + Mio status) ---

// sddCommands maps skill names to their OpenCode command definitions.
var sddCommands = map[string]struct {
	description string
	template    string
}{
	"sdd-init": {
		description: "Initialize SDD context in this project",
		template:    "Initialize Spec-Driven Development for this project. Detect the stack, create persistence directories, and bootstrap the SDD pipeline.",
	},
	"sdd-explore": {
		description: "Explore and investigate before committing to a change",
		template:    "Explore the codebase to understand the area related to: $ARGUMENTS. Investigate architecture, patterns, and constraints before proposing changes.",
	},
	"sdd-propose": {
		description: "Create a change proposal with intent and scope",
		template:    "Create a change proposal for: $ARGUMENTS. Define intent, scope, approach, and affected areas.",
	},
	"sdd-spec": {
		description: "Write specifications with Given/When/Then scenarios",
		template:    "Write specifications for the current change. Include requirements and Given/When/Then acceptance scenarios.",
	},
	"sdd-design": {
		description: "Create technical design with architecture decisions",
		template:    "Create technical design for the current change. Document architecture decisions, component interactions, and implementation approach.",
	},
	"sdd-tasks": {
		description: "Break down change into implementation task checklist",
		template:    "Break down the current change into an ordered implementation task checklist organized by phases.",
	},
	"sdd-apply": {
		description: "Implement tasks from the change",
		template:    "Implement tasks from the current change. Follow the specs and design. Write actual code.",
	},
	"sdd-verify": {
		description: "Validate implementation matches specs and design",
		template:    "Verify the current implementation against specs, design, and tasks. Run the quality gate.",
	},
	"sdd-archive": {
		description: "Archive completed change and close the SDD cycle",
		template:    "Archive the completed change. Merge delta specs and close the SDD cycle.",
	},
	"sdd-continue": {
		description: "Auto-detect SDD state and execute next phase",
		template:    "Detect the current SDD pipeline state and execute the next phase automatically. Recover state from Mio memory if needed.",
	},
	"sdd-ff": {
		description: "Fast-forward through SDD planning phases",
		template:    "Fast-forward through SDD planning phases (explore → propose → spec → design → tasks) for: $ARGUMENTS. Stop before implementation so I can review.",
	},
	"mio-architect": {
		description: "Full SDD pipeline with approval gates",
		template:    "Activate the Mio Architect for: $ARGUMENTS. Drive the full SDD pipeline with user approval at each gate.",
	},
	"mio-status": {
		description: "Show Mio memory status and statistics",
		template:    "Call the mio.mem_stats MCP tool to get current memory statistics. Display total memories, sessions, types breakdown, and recent activity. Then call mio.mem_timeline with limit 5 to show the 5 most recent memories as a compact table.",
	},
}

func (o *OpenCode) installCommands(binPath string) {
	cmdDir := o.commandsDir()
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		PrintStep("warn", fmt.Sprintf("Commands dir: %v", err))
		return
	}

	installed := 0
	for name, cmd := range sddCommands {
		content := fmt.Sprintf("---\ndescription: %s\n---\n\n%s\n", cmd.description, cmd.template)
		destPath := filepath.Join(cmdDir, name+".md")
		if err := WriteIfChanged(destPath, []byte(content)); err != nil {
			PrintStep("warn", fmt.Sprintf("Command %s: %v", name, err))
		} else {
			installed++
		}
	}

	PrintStep("ok", fmt.Sprintf("Installed %d custom commands → %s", installed, cmdDir))
}

func (o *OpenCode) removeCommands() {
	removed := 0
	for name := range sddCommands {
		cmdPath := filepath.Join(o.commandsDir(), name+".md")
		if err := os.Remove(cmdPath); err == nil {
			removed++
		}
	}
	if removed > 0 {
		PrintStep("ok", fmt.Sprintf("Removed %d custom commands", removed))
	}
}

// --- Status check ---

func hasOpenCodeMCP(configPath string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}
	mcp, ok := config["mcp"].(map[string]interface{})
	if !ok {
		return false
	}
	_, exists := mcp["mio"]
	return exists
}
