#!/bin/bash
# Mio: UserPromptSubmit hook
# On first message: loads memory context for continuity
# After inactivity: gentle reminder to persist important context

STATE_FILE="${HOME}/.mio/.hook_state"
INACTIVITY_THRESHOLD="${MIO_INACTIVITY_THRESHOLD:-900}"  # 15 minutes in seconds

mkdir -p "$(dirname "$STATE_FILE")"

now=$(date +%s)

if [ ! -f "$STATE_FILE" ]; then
    tmp_state="${STATE_FILE}.tmp.$$"
    echo "$now" > "$tmp_state" && mv "$tmp_state" "$STATE_FILE"
    cat <<'PROMPT'
Session starting — load your memory context with mcp__mio__mem_context and surface relevant memories with mcp__mio__mem_surface before responding.
PROMPT
    exit 0
fi

last_active=$(cat "$STATE_FILE" 2>/dev/null)
# Validate it's a number
if ! [[ "$last_active" =~ ^[0-9]+$ ]]; then
    last_active=0
fi

elapsed=$((now - last_active))

tmp_state="${STATE_FILE}.tmp.$$"
echo "$now" > "$tmp_state" && mv "$tmp_state" "$STATE_FILE"

if [ "$elapsed" -gt "$INACTIVITY_THRESHOLD" ]; then
    echo "It's been a while since the last interaction. If you've made decisions or discoveries that aren't saved yet, now is a good time to persist them."
fi
