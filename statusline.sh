#!/usr/bin/env bash
# Mio statusline for Claude Code
# Shows: model | directory | git branch | lines +/- | context usage | mio memories

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
DIM='\033[2m'
RESET='\033[0m'

parts=()

# Model name
if [ -n "$CLAUDE_MODEL" ]; then
  model=$(echo "$CLAUDE_MODEL" | sed 's/claude-//' | cut -c1-12)
  parts+=("${CYAN}${model}${RESET}")
fi

# Current directory (short)
dir=$(basename "$PWD")
parts+=("${DIM}${dir}${RESET}")

# Git branch
if git rev-parse --git-dir &>/dev/null 2>&1; then
  branch=$(git branch --show-current 2>/dev/null)
  if [ -n "$branch" ]; then
    # Changed files count
    changed=$(git diff --shortstat 2>/dev/null | grep -oP '\d+ file' | grep -oP '\d+' || echo "0")
    additions=$(git diff --numstat 2>/dev/null | awk '{s+=$1} END {print s+0}')
    deletions=$(git diff --numstat 2>/dev/null | awk '{s+=$2} END {print s+0}')

    git_info="${GREEN}${branch}${RESET}"
    if [ "$additions" -gt 0 ] || [ "$deletions" -gt 0 ]; then
      git_info="${git_info} ${GREEN}+${additions}${RESET}${RED}-${deletions}${RESET}"
    fi
    parts+=("$git_info")
  fi
fi

# Context window usage
if [ -n "$CLAUDE_CONTEXT_WINDOW" ] && [ -n "$CLAUDE_CONTEXT_TOKENS_USED" ]; then
  pct=$((CLAUDE_CONTEXT_TOKENS_USED * 100 / CLAUDE_CONTEXT_WINDOW))
  bar_len=10
  filled=$((pct * bar_len / 100))
  empty=$((bar_len - filled))

  bar=""
  for ((i=0; i<filled; i++)); do bar+="█"; done
  for ((i=0; i<empty; i++)); do bar+="░"; done

  color=$GREEN
  if [ "$pct" -gt 75 ]; then color=$RED
  elif [ "$pct" -gt 50 ]; then color=$YELLOW
  fi

  parts+=("${color}${bar} ${pct}%${RESET}")
fi

# Mio memory count (cached for 60s)
MIO_CACHE="/tmp/mio-statusline-cache"
MIO_CACHE_TTL=60

refresh_mio=true
if [ -f "$MIO_CACHE" ]; then
  cache_age=$(( $(date +%s) - $(stat -f%m "$MIO_CACHE" 2>/dev/null || stat -c%Y "$MIO_CACHE" 2>/dev/null || echo 0) ))
  if [ "$cache_age" -lt "$MIO_CACHE_TTL" ]; then
    refresh_mio=false
  fi
fi

if $refresh_mio; then
  if command -v mio &>/dev/null; then
    count=$(mio stats 2>/dev/null | grep -o '"TotalObservations":[0-9]*' | grep -o '[0-9]*' || echo "0")
    echo "$count" > "$MIO_CACHE"
  else
    echo "?" > "$MIO_CACHE"
  fi
fi

mio_count=$(cat "$MIO_CACHE" 2>/dev/null || echo "?")
if [ "$mio_count" != "?" ] && [ "$mio_count" -gt 0 ] 2>/dev/null; then
  parts+=("${YELLOW}mio:${mio_count}${RESET}")
fi

# Join with separator
IFS='|'
output=""
for i in "${!parts[@]}"; do
  if [ "$i" -gt 0 ]; then
    output+=" ${DIM}│${RESET} "
  fi
  output+="${parts[$i]}"
done

echo -e "$output"
