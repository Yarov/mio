package setup

import (
	"fmt"

	"mio/internal/agents"
)

// Setup configures Mio for a specific agent.
func Setup(agentName string) error {
	agent, err := agents.Get(agentName)
	if err != nil {
		return err
	}

	binPath, err := agents.FindBinaryPath()
	if err != nil {
		return fmt.Errorf("cannot find mio binary: %w\nRun 'make install' first", err)
	}

	fmt.Printf("Setting up Mio for %s...\n", agent.DisplayName())
	fmt.Printf("  [ok] Found mio at %s\n", binPath)

	if err := agent.Setup(binPath); err != nil {
		return err
	}

	fmt.Printf("\nMio is ready for %s! Restart the agent to activate.\n", agent.DisplayName())
	return nil
}

// SetupAll configures Mio for all detected agents.
func SetupAll() []agents.SetupResult {
	installed := agents.DetectInstalled()
	if len(installed) == 0 {
		fmt.Println("No supported agents detected.")
		return nil
	}

	binPath, err := agents.FindBinaryPath()
	if err != nil {
		return []agents.SetupResult{{Agent: "all", Error: err.Error()}}
	}

	var results []agents.SetupResult
	for _, agent := range installed {
		fmt.Printf("\n── %s ──\n", agent.DisplayName())
		if err := agent.Setup(binPath); err != nil {
			fmt.Printf("  [error] %v\n", err)
			results = append(results, agents.SetupResult{Agent: agent.Name(), Error: err.Error()})
		} else {
			results = append(results, agents.SetupResult{Agent: agent.Name(), OK: true})
		}
	}

	fmt.Println("\nDone! Restart your agents to activate Mio.")
	return results
}

// Uninstall removes Mio from a specific agent.
func Uninstall(agentName string, purge bool) error {
	agent, err := agents.Get(agentName)
	if err != nil {
		return err
	}

	fmt.Printf("Uninstalling Mio from %s...\n", agent.DisplayName())

	if err := agent.Uninstall(purge); err != nil {
		return err
	}

	fmt.Printf("\nMio has been uninstalled from %s. Restart the agent to complete.\n", agent.DisplayName())
	return nil
}

// UninstallAll removes Mio from all configured agents.
func UninstallAll(purge bool) {
	for _, agent := range agents.All() {
		status := agent.Status()
		if !status.Configured {
			continue
		}
		fmt.Printf("\n── %s ──\n", agent.DisplayName())
		if err := agent.Uninstall(purge); err != nil {
			fmt.Printf("  [error] %v\n", err)
		}
	}
	fmt.Println("\nDone.")
}

// ListAgents prints the status of all known agents.
func ListAgents() {
	fmt.Println("Supported agents:")
	fmt.Println()
	for _, status := range agents.StatusAll() {
		installed := "not found"
		if status.Installed {
			installed = "installed"
		}
		configured := ""
		if status.Configured {
			configured = " ✓ Mio configured"
		}
		fmt.Printf("  %-18s %s%s\n", status.DisplayName, installed, configured)
	}
	fmt.Println()
}

// SetupClaudeCode is a backward-compatible alias for Setup("claude-code").
func SetupClaudeCode() error {
	return Setup("claude-code")
}

// UninstallClaudeCode is a backward-compatible alias for Uninstall("claude-code", purge).
func UninstallClaudeCode(purge bool) error {
	return Uninstall("claude-code", purge)
}
