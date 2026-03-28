package agents

// Agent defines the interface for setting up Mio with a specific AI agent.
type Agent interface {
	Name() string        // e.g. "claude-code", "cursor"
	DisplayName() string // e.g. "Claude Code", "Cursor"
	Detect() bool        // Returns true if the agent is installed on this system
	Setup(binPath string) error
	Uninstall(purge bool) error
	Status() AgentStatus
}

// AgentStatus reports whether an agent is installed and configured.
type AgentStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Installed   bool   `json:"installed"`   // The agent exists on the system
	Configured  bool   `json:"configured"`  // Mio is configured for this agent
	ConfigPath  string `json:"config_path"` // Where the MCP config lives
}

// SetupResult captures the outcome of setting up one agent.
type SetupResult struct {
	Agent string `json:"agent"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}
