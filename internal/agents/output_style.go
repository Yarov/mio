package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ApplyOutputStyleToggle enables or disables the Mio output style in Claude Code's settings.
// When enabled: writes the style file and sets outputStyle in settings.json.
// When disabled: removes the style file and removes outputStyle from settings.json.
func ApplyOutputStyleToggle(enabled bool) error {
	claudeDir := filepath.Join(HomeDir(), ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	stylePath := filepath.Join(claudeDir, "output-styles", "mio.md")

	if enabled {
		// 1. Write the style file from embedded assets
		if err := InstallEmbeddedFile("output-styles/mio.md", stylePath, 0644); err != nil {
			return fmt.Errorf("install output style file: %w", err)
		}

		// 2. Set outputStyle in settings.json
		return setClaudeSettingsKey(settingsPath, "outputStyle", "mio")
	}

	// Disabled: remove file and settings key
	os.Remove(stylePath)
	return removeClaudeSettingsKey(settingsPath, "outputStyle")
}

// setClaudeSettingsKey sets a key in Claude Code's settings.json.
func setClaudeSettingsKey(settingsPath, key string, value interface{}) error {
	settings := readJSONFile(settingsPath)
	settings[key] = value
	return writeJSONFile(settingsPath, settings)
}

// removeClaudeSettingsKey removes a key from Claude Code's settings.json.
func removeClaudeSettingsKey(settingsPath, key string) error {
	settings := readJSONFile(settingsPath)
	if _, ok := settings[key]; !ok {
		return nil
	}
	delete(settings, key)
	return writeJSONFile(settingsPath, settings)
}

func readJSONFile(path string) map[string]interface{} {
	settings := make(map[string]interface{})
	data, err := os.ReadFile(path)
	if err != nil {
		return settings
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		// Backup corrupted file before it gets overwritten
		_ = os.WriteFile(path+".bak", data, 0644)
		PrintStep("warn", fmt.Sprintf("%s was corrupted, backed up to %s.bak", path, path))
		return make(map[string]interface{})
	}
	return settings
}

func writeJSONFile(path string, data map[string]interface{}) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}
