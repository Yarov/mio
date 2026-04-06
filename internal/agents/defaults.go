package agents

// DefaultSetupAgent picks a sensible target when `mio setup` is run with no argument.
// Prefers Cursor when present, then Claude Code, then the first detected agent, then cursor.
func DefaultSetupAgent() string {
	installed := DetectInstalled()
	for _, a := range installed {
		if a.Name() == "cursor" {
			return "cursor"
		}
	}
	for _, a := range installed {
		if a.Name() == "claude-code" {
			return "claude-code"
		}
	}
	if len(installed) > 0 {
		return installed[0].Name()
	}
	return "cursor"
}

// DefaultUninstallAgent picks a target when `mio uninstall` is run with no argument.
// Prefers an agent that still has Mio configured; otherwise falls back to DefaultSetupAgent().
func DefaultUninstallAgent() string {
	for _, a := range All() {
		st := a.Status()
		if st.Configured {
			return st.Name
		}
	}
	return DefaultSetupAgent()
}
