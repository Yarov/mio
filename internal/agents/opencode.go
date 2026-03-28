package agents

import (
	"fmt"
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

func (o *OpenCode) mcpConfigPath() string {
	return filepath.Join(o.configDir(), "opencode.json")
}

func (o *OpenCode) Status() AgentStatus {
	return AgentStatus{
		Name:        o.Name(),
		DisplayName: o.DisplayName(),
		Installed:   o.Detect(),
		Configured:  HasMCPConfig(o.mcpConfigPath()),
		ConfigPath:  o.mcpConfigPath(),
	}
}

func (o *OpenCode) Setup(binPath string) error {
	if err := WriteMCPToSharedJSON(o.mcpConfigPath(), binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", o.mcpConfigPath()))
	return nil
}

func (o *OpenCode) Uninstall(_ bool) error {
	if err := RemoveMCPFromSharedJSON(o.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}
	return nil
}
