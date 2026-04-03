package agents

import (
	"fmt"
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
	// Cursor uses .cursorrules at project root, but for global we use ~/.cursor/rules/mio.md
	return filepath.Join(c.configDir(), "rules", "mio.md")
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
	PrintStep("ok", fmt.Sprintf("MCP config → %s", c.mcpConfigPath()))

	// 2. Skills
	n, err := InstallSkillsFromAssets(binPath, c.skillsDir())
	if err != nil {
		PrintStep("warn", fmt.Sprintf("Skills: %v", err))
	} else if n > 0 {
		PrintStep("ok", fmt.Sprintf("Installed %d skill files", n))
	} else {
		PrintStep("ok", "Skills already up to date")
	}

	// 3. Protocol (global rules)
	if err := InstallProtocolFromAssets(binPath, "protocols/cursor.md", c.protocolPath()); err != nil {
		PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
	} else {
		PrintStep("ok", fmt.Sprintf("Protocol → %s", c.protocolPath()))
	}

	return nil
}

func (c *Cursor) Uninstall(purge bool) error {
	// 1. Remove MCP entry
	if err := RemoveMCPFromSharedJSON(c.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}

	// 2. Remove skills
	if err := RemoveSkills(c.skillsDir()); err == nil {
		PrintStep("ok", "Removed skills")
	}

	// 3. Remove protocol
	if err := RemoveProtocol(c.protocolPath()); err == nil {
		PrintStep("ok", "Removed protocol")
	}

	return nil
}
