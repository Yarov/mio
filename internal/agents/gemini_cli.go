package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register(&GeminiCLI{})
}

// GeminiCLI implements the Agent interface for Gemini CLI.
type GeminiCLI struct{}

func (g *GeminiCLI) Name() string        { return "gemini-cli" }
func (g *GeminiCLI) DisplayName() string { return "Gemini CLI" }

func (g *GeminiCLI) Detect() bool {
	return DetectByCommand("gemini") || DetectByDir(g.configDir())
}

func (g *GeminiCLI) configDir() string {
	return filepath.Join(HomeDir(), ".gemini")
}

func (g *GeminiCLI) mcpConfigPath() string {
	return filepath.Join(g.configDir(), "settings.json")
}

func (g *GeminiCLI) protocolPath() string {
	return filepath.Join(g.configDir(), "GEMINI.md")
}

func (g *GeminiCLI) skillsDir() string {
	return filepath.Join(g.configDir(), "skills")
}

func (g *GeminiCLI) Status() AgentStatus {
	return AgentStatus{
		Name:        g.Name(),
		DisplayName: g.DisplayName(),
		Installed:   g.Detect(),
		Configured:  HasMCPConfig(g.mcpConfigPath()),
		ConfigPath:  g.mcpConfigPath(),
	}
}

func (g *GeminiCLI) Setup(binPath string) error {
	// 1. MCP config
	if err := WriteMCPToSharedJSON(g.mcpConfigPath(), binPath); err != nil {
		return fmt.Errorf("write MCP config: %w", err)
	}
	PrintStep("ok", fmt.Sprintf("MCP config → %s", g.mcpConfigPath()))

	// 2. Skills
	srcDir := FindProjectFile(binPath, "skills")
	if srcDir != "" {
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			n, _ := CopySkills(srcDir, g.skillsDir())
			if n > 0 {
				PrintStep("ok", fmt.Sprintf("Installed %d skill files", n))
			} else {
				PrintStep("ok", "Skills already up to date")
			}
		}
	}

	// 3. Protocol
	protocolSrc := FindProjectFile(binPath, "protocols/gemini-cli.md")
	if protocolSrc != "" {
		data, err := os.ReadFile(protocolSrc)
		if err == nil {
			if err := InstallProtocol(g.protocolPath(), string(data)); err != nil {
				PrintStep("warn", fmt.Sprintf("Protocol: %v", err))
			} else {
				PrintStep("ok", fmt.Sprintf("Protocol → %s", g.protocolPath()))
			}
		}
	}

	return nil
}

func (g *GeminiCLI) Uninstall(_ bool) error {
	if err := RemoveMCPFromSharedJSON(g.mcpConfigPath()); err == nil {
		PrintStep("ok", "Removed MCP config")
	}
	if err := RemoveSkills(g.skillsDir()); err == nil {
		PrintStep("ok", "Removed skills")
	}
	if err := RemoveProtocol(g.protocolPath()); err == nil {
		PrintStep("ok", "Removed protocol")
	}
	return nil
}
