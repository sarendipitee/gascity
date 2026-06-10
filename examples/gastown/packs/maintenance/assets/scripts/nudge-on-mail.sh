#!/usr/bin/env bash
# nudge-on-mail — wake a session the moment a mail bead (type=message) is
# delivered to it. Mail delivery does not set gc.routed_to, so nudge-on-route
# never fires for mail. Without this, the recipient only sees new mail when
# a human types or a scheduled turn cycle fires — too slow for semi-autonomous
# agents like the mayor that should act on escalations promptly.
#
# Subscribes to bead.created events; whenever a bead with issue_type=message
# arrives it nudges the assignee session. Idempotent: a given bead_id is nudged
# at most once. Dedup state lives in $GC_PACK_STATE_DIR/nudge-on-mail-state.json.
#
# Runs as an exec order (no LLM, no agent, no wisp).
set -euo pipefail

__SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
. "$__SCRIPT_DIR/_bd_trace.sh" "nudge-on-mail"

if ! command -v jq >/dev/null 2>&1; then
    echo "nudge-on-mail: jq is required but not found in PATH" >&2
    exit 1
fi

CITY="${GC_CITY:-.}"
LOOKBACK="${GC_NUDGE_ON_MAIL_LOOKBACK:-2m}"
RETENTION="${GC_NUDGE_ON_MAIL_RETENTION:-1h}"
NUDGE_MESSAGE="${GC_NUDGE_ON_MAIL_MESSAGE:-You have new mail — run gc hook to process it}"

PACK_STATE_DIR="${GC_PACK_STATE_DIR:-${GC_CITY_RUNTIME_DIR:-$CITY/.gc/runtime}/packs/maintenance}"
STATE_FILE="$PACK_STATE_DIR/nudge-on-mail-state.json"
mkdir -p "$PACK_STATE_DIR"

duration_to_seconds() {
    case "$1" in
        *h) echo $(( ${1%h} * 3600 )) ;;
        *m) echo $(( ${1%m} * 60 )) ;;
        *s) echo "${1%s}" ;;
        *)  echo "$1" ;;
    esac
}

# Pull recent bead.created events. Best-effort: failure must not crash the
# controller's order loop.
EVENTS="$(gc events --type bead.created --since "$LOOKBACK" 2>/dev/null)" || exit 0
[ -n "$EVENTS" ] || exit 0

# Extract unique "<bead_id>\t<assignee>" pairs for mail beads only.
# Skip beads whose assignee is "human" — no session to nudge.
PAIRS="$(printf '%s\n' "$EVENTS" \
    | jq -r 'select(.payload.bead.issue_type == "message"
                    and (.payload.bead.assignee // "") != ""
                    and (.payload.bead.assignee // "") != "human")
             | [.payload.bead.id, .payload.bead.assignee] | @tsv' 2>/dev/null \
    | sort -u)" || PAIRS=""
[ -n "$PAIRS" ] || exit 0

STATE="$(cat "$STATE_FILE" 2>/dev/null || true)"
echo "$STATE" | jq -e 'type == "object"' >/dev/null 2>&1 || STATE='{}'

NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
NUDGED=0
while IFS="$(printf '\t')" read -r bead_id assignee; do
    [ -n "$bead_id" ] || continue
    [ -n "$assignee" ] || continue
    key="$bead_id"
    if echo "$STATE" | jq -e --arg k "$key" 'has($k)' >/dev/null 2>&1; then
        STATE="$(echo "$STATE" | jq --arg k "$key" --arg now "$NOW" '.[$k] = $now')"
        continue
    fi
    if gc session nudge "$assignee" "$NUDGE_MESSAGE" >/dev/null 2>&1; then
        STATE="$(echo "$STATE" | jq --arg k "$key" --arg now "$NOW" '.[$k] = $now')"
        NUDGED=$((NUDGED + 1))
    fi
done <<EOF
$PAIRS
EOF

# Prune entries older than RETENTION so the state file stays bounded.
RETENTION_S="$(duration_to_seconds "$RETENTION")"
STATE="$(echo "$STATE" | jq --argjson keep "$RETENTION_S" \
    'with_entries(select((now - (.value | fromdateiso8601)) <= $keep))')" || true

TMP="$(mktemp "$PACK_STATE_DIR/.nudge-on-mail-state.XXXXXX")"
printf '%s\n' "$STATE" > "$TMP"
mv -f "$TMP" "$STATE_FILE"

if [ "$NUDGED" -gt 0 ]; then
    echo "nudge-on-mail: nudged $NUDGED session(s) for new mail"
fi
