#!/usr/bin/env bash
# Mio statusline for Claude Code — retro terminal style
# Receives JSON via stdin with model, workspace, cost, context_window data

# Colors — neon retro palette
C='\033[0;36m'     # cyan
M='\033[0;35m'     # magenta
G='\033[0;32m'     # green
Y='\033[0;33m'     # yellow
R='\033[0;31m'     # red
W='\033[0;37m'     # white
D='\033[2m'        # dim
B='\033[1m'        # bold
N='\033[0m'        # reset

# Read JSON from stdin
input=$(cat)

parts=()

# ── Mio branding ──
parts+=("${B}${M}◆ MIO${N}")

# ── Model ──
MODEL=$(echo "$input" | jq -r '.model.display_name // "Claude"' 2>/dev/null)
if [ -n "$MODEL" ]; then
  parts+=("${D}${C}${MODEL}${N}")
fi

# ── Project ──
DIR=$(echo "$input" | jq -r '.workspace.current_dir // ""' 2>/dev/null)
if [ -n "$DIR" ]; then
  DIR_NAME=$(basename "$DIR")
  parts+=("${W}${DIR_NAME}${N}")
fi

# ── Git branch ──
if git rev-parse --git-dir &>/dev/null 2>&1; then
  branch=$(git branch --show-current 2>/dev/null)
  if [ -n "$branch" ]; then
    dirty=""
    if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
      dirty="${Y}*${N}"
    fi
    parts+=("${G}⎇ ${branch}${dirty}${N}")
  fi
fi

# ── Lines diff ──
ADDED=$(echo "$input" | jq -r '.cost.total_lines_added // 0' 2>/dev/null)
REMOVED=$(echo "$input" | jq -r '.cost.total_lines_removed // 0' 2>/dev/null)
if [ "$ADDED" -gt 0 ] 2>/dev/null || [ "$REMOVED" -gt 0 ] 2>/dev/null; then
  parts+=("${G}↑${ADDED}${N} ${R}↓${REMOVED}${N}")
fi

# ── Context window ──
CTX_SIZE=$(echo "$input" | jq -r '.context_window.context_window_size // 0' 2>/dev/null)
INPUT_TOKENS=$(echo "$input" | jq -r '.context_window.current_usage.input_tokens // 0' 2>/dev/null)
CACHE_CREATE=$(echo "$input" | jq -r '.context_window.current_usage.cache_creation_input_tokens // 0' 2>/dev/null)
CACHE_READ=$(echo "$input" | jq -r '.context_window.current_usage.cache_read_input_tokens // 0' 2>/dev/null)

if [ "$CTX_SIZE" -gt 0 ] 2>/dev/null; then
  TOTAL_USED=$((INPUT_TOKENS + CACHE_CREATE + CACHE_READ))
  CTX_PCT=$((TOTAL_USED * 100 / CTX_SIZE))
  [ "$CTX_PCT" -gt 100 ] && CTX_PCT=100

  BAR_W=10
  FILLED=$((CTX_PCT * BAR_W / 100))
  EMPTY=$((BAR_W - FILLED))

  BAR_COLOR=$C
  [ "$CTX_PCT" -ge 50 ] && BAR_COLOR=$Y
  [ "$CTX_PCT" -ge 80 ] && BAR_COLOR=$R

  BAR="${D}ctx${N} ${BAR_COLOR}▐"
  for ((i=0; i<FILLED; i++)); do BAR+="█"; done
  for ((i=0; i<EMPTY; i++)); do BAR+="░"; done
  BAR+="▌${N}"

  parts+=("${BAR} ${D}${CTX_PCT}%${N}")
fi

# ── Mio memory stats (cached 60s) ──
MIO_CACHE="/tmp/mio-statusline-cache"
refresh=true
if [ -f "$MIO_CACHE" ]; then
  age=$(( $(date +%s) - $(stat -f%m "$MIO_CACHE" 2>/dev/null || stat -c%Y "$MIO_CACHE" 2>/dev/null || echo 0) ))
  [ "$age" -lt 60 ] && refresh=false
fi

if $refresh; then
  stats=$(mio stats 2>/dev/null)
  count=$(echo "$stats" | jq -r '.TotalObservations // 0' 2>/dev/null || echo "0")
  sessions=$(echo "$stats" | jq -r '.TotalSessions // 0' 2>/dev/null || echo "0")
  echo "${count}:${sessions}" > "$MIO_CACHE"
fi

cached=$(cat "$MIO_CACHE" 2>/dev/null || echo "0:0")
mio_count=$(echo "$cached" | cut -d: -f1)
mio_sessions=$(echo "$cached" | cut -d: -f2)

if [ "$mio_count" -gt 0 ] 2>/dev/null; then
  parts+=("${M}⧫${N}${Y}${mio_count}${N}${D}mem${N} ${C}${mio_sessions}${N}${D}ses${N}")
fi

# ── Join with retro separator ──
output=""
for i in "${!parts[@]}"; do
  [ "$i" -gt 0 ] && output+=" ${D}░${N} "
  output+="${parts[$i]}"
done

echo -e "${output}\033[K"
