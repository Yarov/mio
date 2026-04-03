import { execSync } from "child_process"

interface MioStats {
  memories: number
  sessions: number
  online: boolean
}

function fetchStats(): MioStats {
  try {
    const raw = execSync("mio stats 2>/dev/null", {
      encoding: "utf-8",
      timeout: 3000,
    })
    const data = JSON.parse(raw)
    return {
      memories: data.TotalObservations ?? 0,
      sessions: data.TotalSessions ?? 0,
      online: true,
    }
  } catch {
    return { memories: 0, sessions: 0, online: false }
  }
}

export default {
  id: "opencode-mio",
  tui: async (api: any) => {
    // Show Mio status toast on startup
    const stats = fetchStats()
    if (stats.online) {
      api.ui.toast({
        message: `◆ MIO connected — ${stats.memories} memories, ${stats.sessions} sessions`,
        variant: "success",
        duration: 4000,
      })
    }

    // Register command accessible via ctrl+p
    api.command.register(() => [
      {
        title: "◆ Mio Status",
        value: "mio.status",
        description: "Show memory statistics",
        category: "Mio",
        onSelect() {
          const s = fetchStats()
          api.ui.toast({
            message: s.online
              ? `◆ MIO — ${s.memories} memories, ${s.sessions} sessions`
              : "◇ MIO — offline",
            variant: s.online ? "info" : "warning",
          })
          api.ui.dialog.clear()
        },
      },
      {
        title: "◆ Mio Search Memory",
        value: "mio.search",
        description: "Search through stored memories",
        category: "Mio",
        slash: {
          name: "mio-search",
        },
        onSelect() {
          api.ui.dialog.clear()
          // Navigate to prompt with search instruction
          api.command.trigger("prompt.focus")
        },
      },
    ])
  },
}
