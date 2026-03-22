#!/usr/bin/env bash
# Mio statusline for Claude Code
# Receives JSON via stdin with model, workspace, cost, context_window data

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

# Read JSON from stdin
input=$(cat)

parts=()

# Model
MODEL=$(echo "$input" | jq -r '.model.display_name // "Claude"' 2>/dev/null)
if [ -n "$MODEL" ]; then
  parts+=("${BOLD}${CYAN}${MODEL}${NC}")
fi

# Directory
DIR=$(echo "$input" | jq -r '.workspace.current_dir // ""' 2>/dev/null)
if [ -n "$DIR" ]; then
  DIR_NAME=$(basename "$DIR")
  parts+=("${DIM}${DIR_NAME}${NC}")
fi

# Git branch
if git rev-parse --git-dir &>/dev/null 2>&1; then
  branch=$(git branch --show-current 2>/dev/null)
  if [ -n "$branch" ]; then
    dirty=""
    if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
      dirty="*"
    fi
    parts+=("${GREEN}${branch}${dirty}${NC}")
  fi
fi

# Lines added/removed
ADDED=$(echo "$input" | jq -r '.cost.total_lines_added // 0' 2>/dev/null)
REMOVED=$(echo "$input" | jq -r '.cost.total_lines_removed // 0' 2>/dev/null)
if [ "$ADDED" -gt 0 ] 2>/dev/null || [ "$REMOVED" -gt 0 ] 2>/dev/null; then
  parts+=("${GREEN}+${ADDED}${NC} ${RED}-${REMOVED}${NC}")
fi

# Context window
CTX_SIZE=$(echo "$input" | jq -r '.context_window.context_window_size // 0' 2>/dev/null)
INPUT_TOKENS=$(echo "$input" | jq -r '.context_window.current_usage.input_tokens // 0' 2>/dev/null)
CACHE_CREATE=$(echo "$input" | jq -r '.context_window.current_usage.cache_creation_input_tokens // 0' 2>/dev/null)
CACHE_READ=$(echo "$input" | jq -r '.context_window.current_usage.cache_read_input_tokens // 0' 2>/dev/null)

if [ "$CTX_SIZE" -gt 0 ] 2>/dev/null; then
  TOTAL_USED=$((INPUT_TOKENS + CACHE_CREATE + CACHE_READ))
  CTX_PCT=$((TOTAL_USED * 100 / CTX_SIZE))
  [ "$CTX_PCT" -gt 100 ] && CTX_PCT=100

  BAR_W=8
  FILLED=$((CTX_PCT * BAR_W / 100))
  EMPTY=$((BAR_W - FILLED))

  BAR_COLOR=$GREEN
  [ "$CTX_PCT" -ge 50 ] && BAR_COLOR=$YELLOW
  [ "$CTX_PCT" -ge 80 ] && BAR_COLOR=$RED

  BAR="${BAR_COLOR}["
  for ((i=0; i<FILLED; i++)); do BAR+="="; done
  for ((i=0; i<EMPTY; i++)); do BAR+="."; done
  BAR+="]${NC}"

  parts+=("${BAR} ${DIM}${CTX_PCT}%${NC}")
fi

# Mio memory count (cached 60s)
MIO_CACHE="/tmp/mio-statusline-cache"
refresh=true
if [ -f "$MIO_CACHE" ]; then
  age=$(( $(date +%s) - $(stat -f%m "$MIO_CACHE" 2>/dev/null || stat -c%Y "$MIO_CACHE" 2>/dev/null || echo 0) ))
  [ "$age" -lt 60 ] && refresh=false
fi

if $refresh; then
  count=$(mio stats 2>/dev/null | jq -r '.TotalObservations // 0' 2>/dev/null || echo "0")
  echo "$count" > "$MIO_CACHE"
fi

mio_count=$(cat "$MIO_CACHE" 2>/dev/null || echo "0")
if [ "$mio_count" -gt 0 ] 2>/dev/null; then
  parts+=("${YELLOW}mio:${mio_count}${NC}")
fi

# Join
output=""
for i in "${!parts[@]}"; do
  [ "$i" -gt 0 ] && output+=" ${DIM}│${NC} "
  output+="${parts[$i]}"
done

echo -e "${output}\033[K"
