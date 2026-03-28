package agents

import (
	"fmt"
	"path/filepath"
)

func init() {
	Register(&KiloCode{})
}

// KiloCode implements the Agent interface for Kilo Code.
type KiloCode struct{}

func (k *KiloCode) Name() string        { return "kilo-code" }
func (k *KiloCode) DisplayName() string { return "Kilo Code" }

func (k *KiloCode) Detect() bool {
	return DetectByDir(k.configDir())
}

func (k *KiloCode) configDir() string {
	return filepath.Join(HomeDir(), ".kilocode")
}

func (k *KiloCode) mcpConfigPath() string {
	return filepath.Join(k.configDir(), "mcp.json")
}

func (k *KiloCode) Status() AgentStatus {
	return AgentStatus{
		Name:        k.Name(),
		DisplayName: k.DisplayName(),
		Installed:   k.Detect(),
		Configured:  HasMCPConfig(k.mcpConfigPath()),
		ConfigPath:  k.mcpConfigPath(),
	}
}

func (k *KiloCode) Setup(binPath string) error {
	if err := WriteMCPToSharedJSON(k.mcpConfigPath(), binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", k.mcpConfigPath()))
	return nil
}

func (k *KiloCode) Uninstall(_ bool) error {
	if err := RemoveMCPFromSharedJSON(k.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}
	return nil
}
