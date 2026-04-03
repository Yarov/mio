package agents

import (
	"fmt"
	"path/filepath"
)

func init() {
	Register(&ContinueDev{})
}

// ContinueDev implements the Agent interface for Continue.dev.
type ContinueDev struct{}

func (c *ContinueDev) Name() string        { return "continue-dev" }
func (c *ContinueDev) DisplayName() string { return "Continue.dev" }

func (c *ContinueDev) Detect() bool {
	return DetectByDir(c.configDir())
}

func (c *ContinueDev) configDir() string {
	return filepath.Join(HomeDir(), ".continue")
}

func (c *ContinueDev) mcpConfigPath() string {
	return filepath.Join(c.configDir(), "mcp.json")
}

func (c *ContinueDev) protocolPath() string {
	return filepath.Join(c.configDir(), "rules", "mio.md")
}

func (c *ContinueDev) skillsDir() string {
	return filepath.Join(c.configDir(), "skills")
}

func (c *ContinueDev) Status() AgentStatus {
	return AgentStatus{
		Name:        c.Name(),
		DisplayName: c.DisplayName(),
		Installed:   c.Detect(),
		Configured:  HasMCPConfig(c.mcpConfigPath()),
		ConfigPath:  c.mcpConfigPath(),
	}
}

func (c *ContinueDev) Setup(binPath string) error {
	// 1. MCP config
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

	// 3. Protocol (rules directory)
	if err := InstallProtocolFromAssets(binPath, "protocols/continue-dev.md", c.protocolPath()); err != nil {
		PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
	} else {
		PrintStep("ok", fmt.Sprintf("Protocol → %s", c.protocolPath()))
	}

	return nil
}

func (c *ContinueDev) Uninstall(_ bool) error {
	// 1. Remove MCP config
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
