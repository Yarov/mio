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
	if err := WriteMCPToSharedJSON(c.mcpConfigPath(), binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", c.mcpConfigPath()))
	return nil
}

func (c *ContinueDev) Uninstall(_ bool) error {
	if err := RemoveMCPFromSharedJSON(c.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}
	return nil
}
