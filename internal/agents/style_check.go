package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// IsOutputStyleEnabled checks if the output style is currently active
// by looking at Claude Code's settings.json.
// Returns true (default) if no settings file exists or outputStyle is "mio".
// Returns false only if settings.json exists but has no outputStyle key
// (meaning it was explicitly disabled via the toggle).
func IsOutputStyleEnabled() bool {
	settingsPath := filepath.Join(HomeDir(), ".claude", "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return true // no settings file, default to enabled
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return true
	}

	// If outputStyle key doesn't exist, default to enabled (same as no file)
	val, exists := settings["outputStyle"]
	if !exists {
		return true
	}

	// If it exists and equals "mio", it's enabled
	if s, ok := val.(string); ok && s == "mio" {
		return true
	}

	// If it's set to something else, don't override
	return false
}
