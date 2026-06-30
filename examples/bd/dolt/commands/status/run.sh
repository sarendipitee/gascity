#!/bin/sh
# gc dolt status — Check if the Dolt server is running.
#
# Exits 0 if the server is reachable, 1 otherwise.
# Lightweight status probe for manual checks and scripts; the dolt-health order
# uses structured `gc dolt health --json | gc dolt health-check` diagnostics.
#
# Environment: GC_CITY_PATH
set -e

: "${GC_CITY_PATH:?GC_CITY_PATH must be set}"
PACK_DIR="${GC_PACK_DIR:-$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)}"
GC_BEADS_BD_SCRIPT="$GC_CITY_PATH/.gc/scripts/gc-beads-bd.sh"

if [ ! -x "$GC_BEADS_BD_SCRIPT" ]; then
  echo "gc dolt status: gc-beads-bd not found" >&2
  exit 1
fi

# First try the canonical runtime-env projection. When the published managed
# runtime state is stale after a supervisor restart, runtime.sh exits 78 before
# probe can report "not running". Fall back to the probe's own state/port
# discovery so status remains a liveness check instead of a resolution error.
if sh -c '. "$1/assets/scripts/runtime.sh"; "$2" probe >/dev/null 2>&1' \
  status-runtime "$PACK_DIR" "$GC_BEADS_BD_SCRIPT" >/dev/null 2>&1; then
  exit 0
fi

status=$?
if [ "$status" -eq 78 ]; then
  # probe exits 0 if running, 2 if not running.
  if GC_CITY_PATH="$GC_CITY_PATH" GC_PACK_DIR="$PACK_DIR" "$GC_BEADS_BD_SCRIPT" probe >/dev/null 2>&1; then
    exit 0
  fi
  exit 1
fi

exit 1
