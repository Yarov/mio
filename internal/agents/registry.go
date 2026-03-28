package agents

import "fmt"

var registry []Agent

// Register adds an agent to the global registry. Called from each agent's init().
func Register(a Agent) {
	registry = append(registry, a)
}

// Get returns an agent by name, or an error if not found.
func Get(name string) (Agent, error) {
	for _, a := range registry {
		if a.Name() == name {
			return a, nil
		}
	}
	return nil, fmt.Errorf("unknown agent: %s", name)
}

// All returns every registered agent.
func All() []Agent {
	return registry
}

// DetectInstalled returns agents that are installed on this system.
func DetectInstalled() []Agent {
	var found []Agent
	for _, a := range registry {
		if a.Detect() {
			found = append(found, a)
		}
	}
	return found
}

// Names returns the names of all registered agents.
func Names() []string {
	names := make([]string, len(registry))
	for i, a := range registry {
		names[i] = a.Name()
	}
	return names
}

// StatusAll returns the status of every registered agent.
func StatusAll() []AgentStatus {
	statuses := make([]AgentStatus, len(registry))
	for i, a := range registry {
		statuses[i] = a.Status()
	}
	return statuses
}
