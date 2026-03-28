# Plan: Soporte Multi-Agente para Mio

**Estado:** Pendiente
**Fecha:** 2026-03-22
**Prioridad:** Alta
**Archivo en repo:** `docs/plans/multi-agent-support.md`

## Context

Mio está hardcoded a Claude Code. El MCP server es estándar y funciona con cualquier agente, pero el setup solo conoce `~/.claude`. Queremos que `mio setup cursor` o `mio setup gemini-cli` funcione.

**Referencia:** agent-teams-lite ya resuelve esto con un bash script que detecta agentes instalados y copia skills + protocol file al directorio correcto de cada uno. Su enfoque es: **cada agente tiene un directorio de skills y un archivo de instrucciones (protocol file), nada más.** No usan MCP registration porque no son un MCP server — nosotros sí lo somos, eso nos da ventaja.

## Cómo lo hace agent-teams-lite

```bash
# Detección: check_agent "cursor" "cursor" → command -v cursor
# Skills: copiar a ~/.cursor/skills/, ~/.gemini/skills/, etc.
# Protocol: append orchestrator a .cursorrules, GEMINI.md, agents.md, etc.
# Markers: <!-- BEGIN:agent-teams-lite --> para idempotencia
```

Cada agente tiene un directorio en `examples/` con su protocol file:
- `examples/claude-code/CLAUDE.md` → `~/.claude/CLAUDE.md`
- `examples/cursor/.cursorrules` → `.cursorrules`
- `examples/gemini-cli/GEMINI.md` → `~/.gemini/GEMINI.md`
- `examples/codex/agents.md` → `~/.codex/agents.md`
- `examples/vscode/.instructions.md` → VS Code user prompts

## Nuestro enfoque (Mio = MCP server + skills + protocol)

Mio tiene una ventaja que ATL no tiene: **somos un MCP server**. Esto significa que además de skills y protocol, necesitamos registrar el server MCP en el config del agente. Pero la estructura base es la misma.

### Qué necesita cada agente

| Agente | MCP Config | Skills Dir | Protocol File | Extras |
|--------|-----------|------------|---------------|--------|
| **Claude Code** | `~/.claude/mcp/mio.json` (propio) | `~/.claude/skills/` | `~/.claude/CLAUDE.md` | allowlist, statusline, output-style |
| **Cursor** | `~/.cursor/mcp.json` (shared) | `~/.cursor/skills/` | `.cursorrules` (project) | — |
| **Gemini CLI** | `~/.gemini/settings.json` (shared) | `~/.gemini/skills/` | `~/.gemini/GEMINI.md` | — |
| **Codex CLI** | `~/.codex/config.toml` (TOML) | `~/.codex/skills/` | `~/.codex/agents.md` | — |
| **VS Code Copilot** | `.vscode/mcp.json` (project) | `~/.copilot/skills/` | `.instructions.md` (user prompts) | — |
| **OpenCode** | `~/.config/opencode/opencode.json` | `~/.config/opencode/skills/` | — | — |
| **Continue.dev** | `~/.continue/mcp.json` (shared) | — | — | — |
| **Kilo Code** | `.kilocode/mcp.json` (project) | — | — | — |

## Plan de implementación

### 1. Crear `internal/agents/` con interface y registro

**`internal/agents/agent.go`** (~50 LOC)
```go
type Agent interface {
    Name() string                    // "claude-code", "cursor"
    DisplayName() string             // "Claude Code", "Cursor"
    Detect() bool                    // command -v o directorio existe
    Setup(binPath string) error      // Registrar MCP + skills + protocol
    Uninstall() error                // Limpiar todo
    Status() AgentStatus
}

type AgentStatus struct {
    Installed  bool   // El agente existe en el sistema
    Configured bool   // Mio está configurado para este agente
    ConfigPath string // Dónde está el MCP config
}
```

**`internal/agents/registry.go`** (~40 LOC)
- `Register(Agent)`, `Get(name)`, `All()`, `DetectInstalled()`
- Cada agente se registra en su `init()`

### 2. Crear helpers compartidos

**`internal/agents/helpers.go`** (~100 LOC)

Funciones comunes usadas por la mayoría de agentes:
- `writeMCPToSharedJSON(configPath, binPath)` — lee JSON existente, merge `mcpServers.mio`, escribe. Usado por Cursor, Gemini, Continue, Kilo, etc.
- `removeMCPFromSharedJSON(configPath)` — remueve entry `mio` del JSON
- `detectByDir(path)` — verifica si directorio existe
- `detectByCommand(cmd)` — verifica si comando existe en PATH
- `copySkills(srcDir, destDir)` — copia directorio de skills (reutiliza lógica de setup.go)
- `installProtocolMarkers(filePath, content, marker)` — inyecta contenido con markers `<!-- BEGIN:mio -->` / `<!-- END:mio -->` (patrón de ATL)
- `removeProtocolMarkers(filePath, marker)` — remueve sección entre markers
- `mioMCPEntry(binPath)` — retorna `{"command": binPath, "args": ["mcp"]}`

### 3. Crear protocol templates

**`protocols/`** — directorio nuevo con el protocol file para cada agente

| Archivo | Destino | Formato |
|---------|---------|---------|
| `protocols/claude-code.md` | `~/.claude/CLAUDE.md` | Ya existe como `memoryProtocol` const |
| `protocols/cursor.md` | `.cursorrules` (proyecto) | Adaptado: sin refs a Agent tool, sin /skill syntax |
| `protocols/gemini-cli.md` | `~/.gemini/GEMINI.md` | Adaptado para Gemini |
| `protocols/codex-cli.md` | `~/.codex/agents.md` | Adaptado para Codex |
| `protocols/vscode-copilot.md` | `.instructions.md` | Adaptado para Copilot |

**Diferencia clave vs ATL:** Nuestros protocols también instruyen al agente a usar las MCP tools de Mio (`mcp__mio__mem_save`, etc.), no solo las skills.

### 4. Implementar agentes

**Orden de prioridad** (de más a menos complejo):

| # | Archivo | Agente | Qué hace Setup() |
|---|---------|--------|-------------------|
| 1 | `claude_code.go` | Claude Code | MCP config (propio) + allowlist + protocol + skills + statusline + output-style (refactor de setup.go actual) |
| 2 | `cursor.go` | Cursor | MCP config (shared JSON) + skills + protocol (.cursorrules) |
| 3 | `gemini_cli.go` | Gemini CLI | MCP config (shared JSON) + skills + protocol (GEMINI.md) |
| 4 | `codex_cli.go` | Codex CLI | MCP config (TOML) + skills + protocol (agents.md) |
| 5 | `vscode_copilot.go` | VS Code Copilot | MCP config (project JSON) + skills + protocol (.instructions.md) |
| 6 | `opencode.go` | OpenCode | MCP config (shared JSON) + skills |
| 7 | `continue_dev.go` | Continue.dev | MCP config (shared JSON) solamente |
| 8 | `kilo_code.go` | Kilo Code | MCP config (project JSON) solamente |

### 5. Refactorizar `internal/setup/setup.go`

De ~750 LOC a ~120 LOC. Se queda como thin dispatcher:

```go
func Setup(agentName string) error          // Configura un agente específico
func SetupAll() []SetupResult               // Configura todos los detectados
func SetupDetected() []SetupResult          // Auto-detect + setup
func Uninstall(agentName string, purge bool) // Uninstall específico
func ListAgents() []AgentStatus             // Status de todos
```

Mantiene: `findBinaryPath()`, `findProjectFile()`

### 6. Actualizar `cmd/mio/main.go`

```bash
mio setup                     # Auto-detect (si solo hay 1 → ese; si hay varios → lista)
mio setup claude-code         # Específico
mio setup --all               # Todos los detectados
mio setup --list              # Mostrar agentes y estado
mio uninstall                 # Claude Code (backward compat)
mio uninstall cursor          # Específico
mio uninstall --all           # Todos los configurados
mio uninstall --purge         # + borrar datos
```

### 7. Actualizar dashboard

- `scanSkills()` → buscar en todos los agentes con SkillsProvider
- Nueva sección "Agents" en admin: tabla con estado de cada agente, botones setup/uninstall per-agent
- `POST /admin/setup?agent=cursor`

### Archivos a crear/modificar

| Archivo | Acción | LOC est. |
|---------|--------|----------|
| `internal/agents/agent.go` | **NUEVO** | ~50 |
| `internal/agents/registry.go` | **NUEVO** | ~40 |
| `internal/agents/helpers.go` | **NUEVO** | ~120 |
| `internal/agents/claude_code.go` | **NUEVO** (extract setup.go) | ~500 |
| `internal/agents/cursor.go` | **NUEVO** | ~80 |
| `internal/agents/gemini_cli.go` | **NUEVO** | ~70 |
| `internal/agents/codex_cli.go` | **NUEVO** | ~90 |
| `internal/agents/vscode_copilot.go` | **NUEVO** | ~80 |
| `internal/agents/opencode.go` | **NUEVO** | ~60 |
| `internal/agents/continue_dev.go` | **NUEVO** | ~50 |
| `internal/agents/kilo_code.go` | **NUEVO** | ~50 |
| `protocols/claude-code.md` | **NUEVO** (extract const) | ~100 |
| `protocols/cursor.md` | **NUEVO** | ~80 |
| `protocols/gemini-cli.md` | **NUEVO** | ~80 |
| `protocols/codex-cli.md` | **NUEVO** | ~80 |
| `protocols/vscode-copilot.md` | **NUEVO** | ~80 |
| `internal/setup/setup.go` | **REFACTOR** (750→120) | -630 |
| `cmd/mio/main.go` | **MODIFICAR** | ~60 cambios |
| `internal/server/dashboard.go` | **MODIFICAR** | ~40 cambios |
| `internal/server/templates/dashboard.html` | **MODIFICAR** | ~60 cambios |
| `README.md` | **MODIFICAR** | Actualizar |

**Total: ~1600 LOC nuevos, ~630 eliminados del refactor**

## Verificación

1. `make build && make test` — compila, 70 tests pasan
2. `mio setup --list` — muestra 8 agentes, detecta cuáles están instalados
3. `mio setup` — backward compat, configura Claude Code idéntico a antes
4. `mio setup cursor` — crea `~/.cursor/mcp.json`, copia skills, inyecta protocol
5. `mio uninstall cursor` — limpia todo de Cursor
6. `mio setup --all` — configura todos los detectados
7. Dashboard → Admin → muestra agentes con estado
8. Idempotencia: correr `mio setup cursor` dos veces no duplica nada
9. `mio uninstall --all` + verificar que no quedan archivos huérfanos
