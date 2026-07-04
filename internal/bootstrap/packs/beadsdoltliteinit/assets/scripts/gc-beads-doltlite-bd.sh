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

json_escape_value() {
    printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

find_backend_plugin_command() {
    if [ "${GC_DOLTLITE_BACKEND_PLUGIN:-1}" = "0" ] || [ "${GC_DOLTLITE_BACKEND_PLUGIN:-1}" = "false" ] || [ "${GC_DOLTLITE_BACKEND_PLUGIN:-1}" = "off" ]; then
        return 0
    fi
    if [ -n "${GC_DOLTLITE_BACKEND_PLUGIN_COMMAND:-}" ]; then
        if [ ! -x "$GC_DOLTLITE_BACKEND_PLUGIN_COMMAND" ]; then
            echo "gc-beads-doltlite-bd: configured DoltLite backend plugin is not executable: $GC_DOLTLITE_BACKEND_PLUGIN_COMMAND" >&2
            exit 1
        fi
        printf '%s\n' "$GC_DOLTLITE_BACKEND_PLUGIN_COMMAND"
        return 0
    fi

    city_path="${GC_CITY_PATH:-}"
    if [ -z "$city_path" ]; then
        return 0
    fi
    for candidate in \
        "$city_path/.gc/runtime/packs/beads-doltlite/bin/bd-backend-doltlite" \
        "$city_path/.gc/runtime/packs/dolt/bin/bd-backend-doltlite" \
        "$city_path/rigs/beads-backend-doltlite-plugin/bin/bd-backend-doltlite" \
        "$city_path/beads-backend-doltlite-plugin/bin/bd-backend-doltlite" \
        "$city_path/../beads-backend-doltlite-plugin/bin/bd-backend-doltlite"; do
        if [ -x "$candidate" ]; then
            printf '%s\n' "$candidate"
            return 0
        fi
    done
}

find_gascity_backend_plugin_command() {
    if [ "${GC_DOLTLITE_BACKEND_PLUGIN:-1}" = "0" ] || [ "${GC_DOLTLITE_BACKEND_PLUGIN:-1}" = "false" ] || [ "${GC_DOLTLITE_BACKEND_PLUGIN:-1}" = "off" ]; then
        return 0
    fi
    if [ -n "${GC_DOLTLITE_GASCITY_BACKEND_PLUGIN_COMMAND:-}" ]; then
        if [ ! -x "$GC_DOLTLITE_GASCITY_BACKEND_PLUGIN_COMMAND" ]; then
            echo "gc-beads-doltlite-bd: configured Gas City DoltLite backend plugin is not executable: $GC_DOLTLITE_GASCITY_BACKEND_PLUGIN_COMMAND" >&2
            exit 1
        fi
        printf '%s\n' "$GC_DOLTLITE_GASCITY_BACKEND_PLUGIN_COMMAND"
        return 0
    fi

    bd_plugin="$(find_backend_plugin_command)"
    if [ -n "$bd_plugin" ]; then
        sibling="$(dirname "$bd_plugin")/gc-doltlite-fastpath"
        if [ -x "$sibling" ]; then
            printf '%s\n' "$sibling"
            return 0
        fi
    fi

    city_path="${GC_CITY_PATH:-}"
    if [ -z "$city_path" ]; then
        return 0
    fi
    for candidate in \
        "$city_path/.gc/runtime/packs/beads-doltlite/bin/gc-doltlite-fastpath" \
        "$city_path/.gc/runtime/packs/dolt/bin/gc-doltlite-fastpath" \
        "$city_path/rigs/beads-backend-doltlite-plugin/bin/gc-doltlite-fastpath" \
        "$city_path/beads-backend-doltlite-plugin/bin/gc-doltlite-fastpath" \
        "$city_path/../beads-backend-doltlite-plugin/bin/gc-doltlite-fastpath"; do
        if [ -x "$candidate" ]; then
            printf '%s\n' "$candidate"
            return 0
        fi
    done
}

write_doltlite_metadata() {
    dir="$1"
    database="$2"
    metadata_path="$dir/.beads/metadata.json"
    plugin_command="$(find_backend_plugin_command)"
    gascity_plugin_command="$(find_gascity_backend_plugin_command)"
    plugin_trace="${GC_DOLTLITE_BACKEND_PLUGIN_TRACE:-${GC_CITY_PATH:-$dir}/.gc/backend-plugin-trace.jsonl}"
    gascity_plugin_trace="${GC_DOLTLITE_GASCITY_BACKEND_PLUGIN_TRACE:-${GC_CITY_PATH:-$dir}/.gc/gascity-backend-plugin-trace.jsonl}"
    ops_path="$dir/.gc/ops.sqlite"
    database_json="$(json_escape_value "$database")"
    plugin_command_json="$(json_escape_value "$plugin_command")"
    gascity_plugin_command_json="$(json_escape_value "$gascity_plugin_command")"
    plugin_trace_json="$(json_escape_value "$plugin_trace")"
    gascity_plugin_trace_json="$(json_escape_value "$gascity_plugin_trace")"
    ops_path_json="$(json_escape_value "$ops_path")"
    tmp="$metadata_path.tmp.$$"
    mkdir -p "$dir/.beads"
    {
        printf '{\n'
        printf '  "attached_databases": [\n'
        printf '    {\n'
        printf '      "alias": "ops",\n'
        printf '      "path": "%s"\n' "$ops_path_json"
        printf '    }\n'
        printf '  ],\n'
        printf '  "backend": "doltlite",\n'
        printf '  "database": "doltlite",\n'
        printf '  "dolt_database": "%s"' "$database_json"
        if [ -n "$plugin_command" ]; then
            printf ',\n'
            printf '  "backend_plugin_command": "%s",\n' "$plugin_command_json"
            printf '  "backend_plugin_args": ["--trace", "%s", "serve"]' "$plugin_trace_json"
        fi
        if [ -n "$gascity_plugin_command" ]; then
            printf ',\n'
            printf '  "gascity_backend_command": "%s",\n' "$gascity_plugin_command_json"
            printf '  "gascity_backend_args": ["--trace", "%s", "serve"]' "$gascity_plugin_trace_json"
        fi
        printf '\n'
        printf '}\n'
    } >"$tmp"
    chmod 600 "$tmp"
    mv "$tmp" "$metadata_path"
}

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
        write_doltlite_metadata "$dir" "$database"
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
