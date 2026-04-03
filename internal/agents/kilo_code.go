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

func (k *KiloCode) protocolPath() string {
	return filepath.Join(k.configDir(), "rules", "mio.md")
}

func (k *KiloCode) skillsDir() string {
	return filepath.Join(k.configDir(), "skills")
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
	// 1. MCP config
	if err := WriteMCPToSharedJSON(k.mcpConfigPath(), binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", k.mcpConfigPath()))

	// 2. Skills
	n, err := InstallSkillsFromAssets(binPath, k.skillsDir())
	if err != nil {
		PrintStep("warn", fmt.Sprintf("Skills: %v", err))
	} else if n > 0 {
		PrintStep("ok", fmt.Sprintf("Installed %d skill files", n))
	} else {
		PrintStep("ok", "Skills already up to date")
	}

	// 3. Protocol (rules directory)
	if err := InstallProtocolFromAssets(binPath, "protocols/kilo-code.md", k.protocolPath()); err != nil {
		PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
	} else {
		PrintStep("ok", fmt.Sprintf("Protocol → %s", k.protocolPath()))
	}

	return nil
}

func (k *KiloCode) Uninstall(_ bool) error {
	// 1. Remove MCP config
	if err := RemoveMCPFromSharedJSON(k.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}

	// 2. Remove skills
	if err := RemoveSkills(k.skillsDir()); err == nil {
		PrintStep("ok", "Removed skills")
	}

	// 3. Remove protocol
	if err := RemoveProtocol(k.protocolPath()); err == nil {
		PrintStep("ok", "Removed protocol")
	}

	return nil
}
