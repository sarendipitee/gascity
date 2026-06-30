#!/bin/sh
# gc-beads-doltlite-bd — minimal exec: beads provider for DoltLite bootstrap.
#
# This init-pack copy exists before the full external beads-doltlite pack is
# installed. It intentionally avoids managed Dolt server lifecycle behavior.

set -e

op="${1:-}"
if [ $# -gt 0 ]; then
    shift
fi

run_bd_doltlite() {
    dir="$1"
    shift
    (
        cd "$dir" || exit 1
        export BEADS_DIR="$dir/.beads"
        export BEADS_BACKEND="doltlite"
        export GC_BEADS_BACKEND="doltlite"
        unset GC_DOLT GC_DOLT_HOST GC_DOLT_PORT GC_DOLT_USER GC_DOLT_PASSWORD
        unset BEADS_DOLT_DATABASE BEADS_DOLT_PORT
        unset BEADS_DOLT_SERVER_DATABASE BEADS_DOLT_SERVER_HOST BEADS_DOLT_SERVER_MODE
        unset BEADS_DOLT_SERVER_PORT BEADS_DOLT_SERVER_USER BEADS_DOLT_PASSWORD
        export BEADS_DOLT_AUTO_START=0
        "${BD_BIN:-bd}" "$@"
    )
}

case "$op" in
    start|ensure-ready|health|recover|stop|shutdown|probe)
        exit 0
        ;;
    init)
        dir="${1:-}"
        prefix="${2:-}"
        database="${3:-}"
        if [ -z "$dir" ] || [ -z "$prefix" ]; then
            echo "gc-beads-doltlite-bd: init requires DIR and PREFIX" >&2
            exit 1
        fi
        if [ -z "$database" ]; then
            database="$prefix"
        fi
        mkdir -p "$dir/.beads/doltlite"
        run_bd_doltlite "$dir" init --backend doltlite --quiet -p "$prefix" --database "$database" --skip-hooks --skip-agents
        ;;
    create|get|update|close|reopen|list|ready|children|list-by-label|set-metadata|delete|dep-add|dep-remove|dep-list)
        dir="${GC_BEADS_SCOPE_ROOT:-${GC_CITY_PATH:-}}"
        if [ -z "$dir" ]; then
            echo "gc-beads-doltlite-bd: $op requires GC_CITY_PATH or GC_BEADS_SCOPE_ROOT" >&2
            exit 1
        fi
        run_bd_doltlite "$dir" "$op" "$@"
        ;;
    "")
        echo "gc-beads-doltlite-bd: missing operation" >&2
        exit 1
        ;;
    *)
        echo "gc-beads-doltlite-bd: unsupported operation: $op" >&2
        exit 1
        ;;
esac
