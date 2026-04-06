package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register(&Cursor{})
}

// Cursor implements the Agent interface for Cursor IDE.
type Cursor struct{}

func (c *Cursor) Name() string        { return "cursor" }
func (c *Cursor) DisplayName() string { return "Cursor" }

func (c *Cursor) Detect() bool {
	return DetectByCommand("cursor") || DetectByDir(c.configDir())
}

func (c *Cursor) configDir() string {
	return filepath.Join(HomeDir(), ".cursor")
}

func (c *Cursor) mcpConfigPath() string {
	return filepath.Join(c.configDir(), "mcp.json")
}

func (c *Cursor) protocolPath() string {
	// Global Cursor rules installed by mio setup.
	return filepath.Join(c.configDir(), "rules", "mio.md")
}

func (c *Cursor) protocolMDCPath() string {
	// Native Cursor rule format (alwaysApply) — same content as mio.md, better pickup by the IDE.
	return filepath.Join(c.configDir(), "rules", "mio.mdc")
}

func (c *Cursor) skillsDir() string {
	return filepath.Join(c.configDir(), "skills")
}

func (c *Cursor) Status() AgentStatus {
	return AgentStatus{
		Name:        c.Name(),
		DisplayName: c.DisplayName(),
		Installed:   c.Detect(),
		Configured:  HasMCPConfig(c.mcpConfigPath()),
		ConfigPath:  c.mcpConfigPath(),
	}
}

func (c *Cursor) Setup(binPath string) error {
	// 1. MCP config (shared JSON)
	if err := WriteMCPToSharedJSON(c.mcpConfigPath(), binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	// Label memories when the model omits the optional "agent" argument (stdio MCP).
	if err := MergeMioMCPEnv(c.mcpConfigPath(), map[string]string{"MIO_DEFAULT_AGENT": "cursor"}); err != nil {
		return fmt.Errorf("merge MCP env: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", c.mcpConfigPath()))

	// 1b. Cursor MCP tool prompts: merge ~/.cursor/permissions.json (mcpAllowlist)
	permPath := filepath.Join(c.configDir(), "permissions.json")
	if err := MergeCursorPermissionsAllowlist(permPath, CursorMCPAutoAllowPattern); err != nil {
		PrintStep("warn", fmt.Sprintf("permissions.json: %v", err))
	} else {
		PrintStep("ok", fmt.Sprintf("MCP allowlist → %s (%s)", permPath, CursorMCPAutoAllowPattern))
	}

	// 1c. Disable Workspace Trust prompts for zero-friction startup in Cursor Agent.
	settingsPath := filepath.Join(c.configDir(), "settings.json")
	if err := MergeCursorSettingsBool(settingsPath, CursorDisableWorkspaceTrustKey, false); err != nil {
		PrintStep("warn", fmt.Sprintf("settings.json: %v", err))
	} else {
		PrintStep("ok", fmt.Sprintf("Cursor setting → %s (%s=false)", settingsPath, CursorDisableWorkspaceTrustKey))
	}

	// 2. Skills
	n, err := InstallSkillsFromAssets(binPath, c.skillsDir())
	if err != nil {
		PrintStep("warn", fmt.Sprintf("Skills: %v", err))
	} else if n > 0 {
		PrintStep("ok", fmt.Sprintf("Installed %d skill files", n))
	} else {
		PrintStep("ok", "Skills already up to date")
	}

	// 3. Protocol (global rules): legacy mio.md + mio.mdc for Cursor rule engine
	if err := InstallProtocolFromAssets(binPath, "protocols/cursor.md", c.protocolPath()); err != nil {
		PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
	} else {
		PrintStep("ok", fmt.Sprintf("Protocol → %s", c.protocolPath()))
	}
	if err := InstallCursorMDCRuleFromAssets(binPath, "protocols/cursor.md", c.protocolMDCPath()); err != nil {
		PrintStep("warn", fmt.Sprintf("Cursor .mdc rule: %v", err))
	} else {
		PrintStep("ok", fmt.Sprintf("Cursor rule → %s", c.protocolMDCPath()))
	}

	// 5. Dashboard at login (macOS launchd — shared with Claude Code setup)
	if err := InstallLaunchd(binPath); err != nil {
		PrintStep("warn", fmt.Sprintf("Launchd: %v", err))
	}

	return nil
}

func (c *Cursor) Uninstall(purge bool) error {
	// 1. Remove MCP entry
	if err := RemoveMCPFromSharedJSON(c.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}

	if err := RemoveCursorPermissionsAllowlistPattern(filepath.Join(c.configDir(), "permissions.json"), CursorMCPAutoAllowPattern); err != nil {
		PrintStep("warn", fmt.Sprintf("permissions.json: %v", err))
	} else {
		PrintStep("ok", "Removed Mio MCP allowlist entry from permissions.json (if present)")
	}

	// 2. Remove skills
	if err := RemoveSkills(c.skillsDir()); err == nil {
		PrintStep("ok", "Removed skills")
	}

	// 3. Remove protocol
	if err := RemoveProtocol(c.protocolPath()); err == nil {
		PrintStep("ok", "Removed protocol")
	}
	if err := os.Remove(c.protocolMDCPath()); err == nil {
		PrintStep("ok", "Removed Cursor .mdc rule")
	} else if !os.IsNotExist(err) {
		PrintStep("warn", fmt.Sprintf("Could not remove %s: %v", c.protocolMDCPath(), err))
	}

	// 4. Remove launchd only if no other agent still uses Mio
	if !otherAgentWantsLaunchd("cursor") {
		UninstallLaunchd()
	}

	return nil
}
