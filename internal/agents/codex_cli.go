package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register(&CodexCLI{})
}

// CodexCLI implements the Agent interface for OpenAI Codex CLI.
type CodexCLI struct{}

func (c *CodexCLI) Name() string        { return "codex-cli" }
func (c *CodexCLI) DisplayName() string { return "Codex CLI" }

func (c *CodexCLI) Detect() bool {
	return DetectByCommand("codex") || DetectByDir(c.configDir())
}

func (c *CodexCLI) configDir() string {
	return filepath.Join(HomeDir(), ".codex")
}

func (c *CodexCLI) mcpConfigPath() string {
	return filepath.Join(c.configDir(), "config.json")
}

func (c *CodexCLI) protocolPath() string {
	return filepath.Join(c.configDir(), "agents.md")
}

func (c *CodexCLI) skillsDir() string {
	return filepath.Join(c.configDir(), "skills")
}

func (c *CodexCLI) Status() AgentStatus {
	return AgentStatus{
		Name:        c.Name(),
		DisplayName: c.DisplayName(),
		Installed:   c.Detect(),
		Configured:  HasMCPConfig(c.mcpConfigPath()),
		ConfigPath:  c.mcpConfigPath(),
	}
}

func (c *CodexCLI) Setup(binPath string) error {
	if err := WriteMCPToSharedJSON(c.mcpConfigPath(), binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", c.mcpConfigPath()))

	srcDir := FindProjectFile(binPath, "skills")
	if srcDir != "" {
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			n, _ := CopySkills(srcDir, c.skillsDir())
			if n > 0 {
				PrintStep("ok", fmt.Sprintf("Installed %d skill files", n))
			} else {
				PrintStep("ok", "Skills already up to date")
			}
		}
	}

	protocolSrc := FindProjectFile(binPath, "protocols/codex-cli.md")
	if protocolSrc != "" {
		data, err := os.ReadFile(protocolSrc)
		if err == nil {
			if err := InstallProtocol(c.protocolPath(), string(data)); err != nil {
				PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
			} else {
				PrintStep("ok", fmt.Sprintf("Protocol → %s", c.protocolPath()))
			}
		}
	}

	return nil
}

func (c *CodexCLI) Uninstall(_ bool) error {
	if err := RemoveMCPFromSharedJSON(c.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}
	if err := RemoveSkills(c.skillsDir()); err == nil {
		PrintStep("ok", "Removed skills")
	}
	if err := RemoveProtocol(c.protocolPath()); err == nil {
		PrintStep("ok", "Removed protocol")
	}
	return nil
}
