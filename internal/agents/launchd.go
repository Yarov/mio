package agents

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const launchdLabel = "com.mio.server"

// InstallLaunchd registers a launchd user agent so the Mio HTTP dashboard
// starts at login and stays running (macOS only). Idempotent.
func InstallLaunchd(binPath string) error {
	home := HomeDir()
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return err
	}

	plistPath := filepath.Join(agentsDir, launchdLabel+".plist")
	logPath := filepath.Join(home, ".mio", "server.log")

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>server</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, launchdLabel, binPath, logPath, logPath)

	if existing, err := os.ReadFile(plistPath); err == nil {
		if string(existing) == plist {
			PrintStep("ok", "Launchd service already installed")
			return nil
		}
	}

	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}

	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	unloadCmd.Run()

	loadCmd := exec.Command("launchctl", "load", plistPath)
	if out, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %s (%w)", string(out), err)
	}

	PrintStep("ok", fmt.Sprintf("Launchd service → %s", plistPath))
	PrintStep("ok", "Dashboard will auto-start on login and stay alive")
	return nil
}

// UninstallLaunchd removes the Mio launchd user agent (macOS).
func UninstallLaunchd() {
	home := HomeDir()
	plistPath := filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")

	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	unloadCmd.Run()

	if err := os.Remove(plistPath); err == nil {
		PrintStep("ok", "Removed launchd service")
	} else if !os.IsNotExist(err) {
		PrintStep("warn", fmt.Sprintf("Could not remove launchd plist: %v", err))
	}
}

// otherAgentWantsLaunchd returns true if another agent that runs Mio is still
// configured so we should keep the shared dashboard launchd job.
func otherAgentWantsLaunchd(skipName string) bool {
	for _, a := range All() {
		if a.Name() == skipName {
			continue
		}
		if a.Status().Configured {
			return true
		}
	}
	return false
}
