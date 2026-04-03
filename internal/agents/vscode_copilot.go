package agents

import (
	"fmt"
	"path/filepath"
)

func init() {
	Register(&VSCodeCopilot{})
}

// VSCodeCopilot implements the Agent interface for VS Code GitHub Copilot.
type VSCodeCopilot struct{}

func (v *VSCodeCopilot) Name() string        { return "vscode-copilot" }
func (v *VSCodeCopilot) DisplayName() string { return "VS Code Copilot" }

func (v *VSCodeCopilot) Detect() bool {
	return DetectByCommand("code")
}

func (v *VSCodeCopilot) mcpConfigPath() string {
	// VS Code uses project-level .vscode/mcp.json, but for global setup
	// we use the user settings directory
	return filepath.Join(HomeDir(), ".vscode", "mcp.json")
}

func (v *VSCodeCopilot) protocolPath() string {
	return filepath.Join(HomeDir(), ".github", "copilot-instructions.md")
}

func (v *VSCodeCopilot) skillsDir() string {
	return filepath.Join(HomeDir(), ".copilot", "skills")
}

func (v *VSCodeCopilot) Status() AgentStatus {
	return AgentStatus{
		Name:        v.Name(),
		DisplayName: v.DisplayName(),
		Installed:   v.Detect(),
		Configured:  HasMCPConfig(v.mcpConfigPath()),
		ConfigPath:  v.mcpConfigPath(),
	}
}

func (v *VSCodeCopilot) Setup(binPath string) error {
	if err := WriteMCPToSharedJSON(v.mcpConfigPath(), binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", v.mcpConfigPath()))

	n, _ := InstallSkillsFromAssets(binPath, v.skillsDir())
	if n > 0 {
		PrintStep("ok", fmt.Sprintf("Installed %d skill files", n))
	} else {
		PrintStep("ok", "Skills already up to date")
	}

	if err := InstallProtocolFromAssets(binPath, "protocols/vscode-copilot.md", v.protocolPath()); err != nil {
		PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
	} else {
		PrintStep("ok", fmt.Sprintf("Protocol → %s", v.protocolPath()))
	}

	return nil
}

func (v *VSCodeCopilot) Uninstall(_ bool) error {
	if err := RemoveMCPFromSharedJSON(v.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}
	if err := RemoveSkills(v.skillsDir()); err == nil {
		PrintStep("ok", "Removed skills")
	}
	if err := RemoveProtocol(v.protocolPath()); err == nil {
		PrintStep("ok", "Removed protocol")
	}
	return nil
}
