import type { Plugin } from "opencode/plugin"
import { execSync } from "child_process"

// Mio backend plugin for OpenCode — session lifecycle hooks
export default ((ctx) => {
  return {
    "session.created": async () => {
      // Warm up Mio server on session start (MCP will handle the actual connection)
      const mioPath = getMioPath()
      if (!mioPath) return
      try {
        execSync(`${mioPath} stats`, { timeout: 3000, stdio: "ignore" })
      } catch {
        // Server not running — MCP stdio will start it
      }
    },
  }
}) satisfies Plugin

function getMioPath(): string | null {
  try {
    return execSync("which mio", { encoding: "utf-8", timeout: 2000 }).trim()
  } catch {
    return null
  }
}
