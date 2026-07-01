#!/usr/bin/env bash
# End-to-end smoke test for a clean DoltLite-backed `gc init`.
#
# This intentionally exercises the real fresh-init path:
#   - remote pack resolution
#   - released bd/gc DoltLite artifact install
#   - supervisor registration/start
#   - DB-backed beads runtime config
#   - bd create/delete against the new DoltLite store
#
# By default the throwaway city is unregistered and deleted at exit.
# Set KEEP_GC_INIT_SMOKE=1 to leave it behind for inspection.
set -euo pipefail

log() {
  printf '[gc-init-smoke] %s\n' "$*"
}

fail() {
  printf '[gc-init-smoke] FAIL: %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

validate_current_bd_wrapper() {
  local bd_path="$1"
  local lib_dir
  [ -f "$bd_path" ] || return 0
  lib_dir="$(python3 - "$bd_path" <<'PY'
import re
import sys

path = sys.argv[1]
try:
    text = open(path, "r", encoding="utf-8").read(4096)
except UnicodeDecodeError:
    raise SystemExit(0)
match = re.search(r"^lib_dir='([^']+)'", text, re.MULTILINE)
if match:
    print(match.group(1))
PY
)"
  [ -z "$lib_dir" ] || [ -d "$lib_dir" ] || fail "current bd wrapper points at missing libdoltlite dir: $lib_dir"
}

json_field() {
  local file="$1"
  local field="$2"
  python3 - "$file" "$field" <<'PY'
import json
import sys

path, field = sys.argv[1], sys.argv[2]
with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)
value = data
for part in field.split("."):
    value = value[part]
print(value)
PY
}

latest_doltlite_gc_release() {
  python3 - <<'PY'
import json
import re
import urllib.request

url = "https://api.github.com/repos/duncan4123/gascity/releases"
pattern = re.compile(r"^v\d+\.\d+\.\d+-doltlite\.workflow\.\d+$")
with urllib.request.urlopen(url, timeout=120) as response:
    releases = json.load(response)
candidates = [
    release for release in releases
    if not release.get("draft") and pattern.match(release.get("tag_name", ""))
]
if not candidates:
    raise SystemExit("no matching DoltLite gc workflow releases found")
candidates.sort(key=lambda release: release.get("published_at") or release.get("created_at") or "", reverse=True)
print(candidates[0]["tag_name"])
PY
}

pack_lock_commit() {
  local lock_file="$1"
  local source="$2"
  python3 - "$lock_file" "$source" <<'PY'
import sys
import tomllib

lock_file, source = sys.argv[1], sys.argv[2]
with open(lock_file, "rb") as f:
    data = tomllib.load(f)
print(data["packs"][source]["commit"])
PY
}

bd_sql_config_value() {
  local key="$1"
  bd sql --json "select \`key\`, value from config where \`key\` = '$key'" |
    python3 -c '
import json
import sys

key = sys.argv[1]
rows = json.load(sys.stdin)
for row in rows:
    if row.get("key") == key:
        print(row.get("value", ""))
        raise SystemExit(0)
raise SystemExit(1)
' "$key"
}

snapshot_file() {
  local source="$1"
  local dest="$2"
  if [ -n "$source" ] && [ -e "$source" ]; then
    cp -a "$source" "$dest"
  fi
}

restore_file() {
  local snapshot="$1"
  local dest="$2"
  local tmp
  if [ -e "$snapshot" ]; then
    mkdir -p "$(dirname "$dest")"
    tmp="$(dirname "$dest")/.restore-$(basename "$dest").$$"
    rm -f "$tmp"
    cp -a "$snapshot" "$tmp"
    mv -f "$tmp" "$dest"
  fi
}

cleanup() {
  local code=$?
  if [ "${HAD_GLOBAL_BEADS_ROLE:-0}" = "1" ]; then
    git config --global beads.role "$ORIGINAL_GLOBAL_BEADS_ROLE" >/dev/null 2>&1 || true
  else
    git config --global --unset beads.role >/dev/null 2>&1 || true
  fi
  if [ -n "${SNAPSHOT_DIR:-}" ] && [ -d "$SNAPSHOT_DIR" ]; then
    restore_file "$SNAPSHOT_DIR/gc" "$GC_BIN" || log "warning: failed to restore $GC_BIN"
    restore_file "$SNAPSHOT_DIR/bd" "$BD_BIN" || log "warning: failed to restore $BD_BIN"
    rm -rf "$SNAPSHOT_DIR"
  fi
  if [ "${KEEP_GC_INIT_SMOKE:-0}" = "1" ]; then
    log "keeping city for inspection: $CITY_DIR"
    exit "$code"
  fi
  if [ -n "${CITY_DIR:-}" ] && [ -d "$CITY_DIR" ]; then
    gc unregister "$CITY_DIR" >/dev/null 2>&1 || true
    chmod -R u+w "$CITY_DIR/.cache" "$CITY_DIR/.gc/runtime/packs" >/dev/null 2>&1 || true
    rm -rf "$CITY_DIR"
  fi
  exit "$code"
}

require_cmd gc
require_cmd bd
require_cmd git
require_cmd python3

ROOT="${GC_INIT_SMOKE_ROOT:-/data/projects}"
mkdir -p "$ROOT"
CITY_NAME="${GC_INIT_SMOKE_NAME:-gcinit-smoke-$(date -u +%Y%m%dT%H%M%SZ)-$$}"
CITY_DIR="$ROOT/$CITY_NAME"
trap cleanup EXIT

[ ! -e "$CITY_DIR" ] || fail "test city already exists: $CITY_DIR"

GC_BIN="$(command -v gc)"
BD_BIN="$(command -v bd)"
validate_current_bd_wrapper "$BD_BIN"
SNAPSHOT_DIR="$(mktemp -d "${TMPDIR:-/tmp}/gc-init-smoke-entrypoints.XXXXXX")"
snapshot_file "$GC_BIN" "$SNAPSHOT_DIR/gc"
snapshot_file "$BD_BIN" "$SNAPSHOT_DIR/bd"
log "gc: $GC_BIN ($("$GC_BIN" version))"
log "bd: $BD_BIN ($("$BD_BIN" version))"
log "city: $CITY_DIR"

expected_gc_release="$(latest_doltlite_gc_release)"
expected_pack_commit="$(git ls-remote https://github.com/duncan4123/gascity-packs refs/heads/main | awk '{print $1}')"
[ -n "$expected_pack_commit" ] || fail "could not resolve gascity-packs main"
log "expected latest gc release: $expected_gc_release"
log "expected gascity-packs main: $expected_pack_commit"

HAD_GLOBAL_BEADS_ROLE=0
ORIGINAL_GLOBAL_BEADS_ROLE=""
if ORIGINAL_GLOBAL_BEADS_ROLE="$(git config --global --get beads.role 2>/dev/null)"; then
  HAD_GLOBAL_BEADS_ROLE=1
fi
log "temporarily clearing global beads.role to verify installer seeds it"
git config --global --unset beads.role >/dev/null 2>&1 || true

log "running clean gc init"
env \
  -u GASCITY_SRC \
  -u GC_GASCITY_SRC \
  -u BD_SRC \
  -u BEADS_DOLTLITE_SRC \
  -u GC_BEADS_DOLTLITE_SRC \
  -u DOLTLITE_LIB \
  -u GC_DOLTLITE_LIB \
  -u GC_DOLTLITE_GC_RELEASE_VERSION \
  -u GC_DOLTLITE_GC_RELEASE_BASE \
  -u GC_DOLTLITE_BD_RELEASE_VERSION \
  -u GC_DOLTLITE_BD_RELEASE_BASE \
  GC_DOLTLITE_GO_CACHE_ROOT="${GC_DOLTLITE_SMOKE_GO_CACHE_ROOT:-$HOME/.cache/gc-init-smoke-go}" \
  gc init \
    --template gascity \
    --default-provider codex \
    --beads-backend doltlite \
    --skip-provider-readiness \
    --yes \
    "$CITY_DIR"

log "checking generated config"
grep -q 'backend = "doltlite"' "$CITY_DIR/city.toml" || fail "city.toml does not use DoltLite backend"
grep -q '\[imports.beads-doltlite\]' "$CITY_DIR/pack.toml" || fail "pack.toml missing beads-doltlite import"
grep -q 'version = "ref:main"' "$CITY_DIR/pack.toml" || fail "beads-doltlite import is not ref:main"

actual_pack_commit="$(pack_lock_commit "$CITY_DIR/packs.lock" "https://github.com/duncan4123/gascity-packs/tree/main/beads-doltlite")"
[ "$actual_pack_commit" = "$expected_pack_commit" ] ||
  fail "beads-doltlite pack commit = $actual_pack_commit, want $expected_pack_commit"

last_gc="$CITY_DIR/.gc/runtime/packs/beads-doltlite/last-build-gc.json"
last_bd="$CITY_DIR/.gc/runtime/packs/beads-doltlite/last-build-bd.json"
[ -s "$last_gc" ] || fail "missing gc build details: $last_gc"
[ -s "$last_bd" ] || fail "missing bd build details: $last_bd"

actual_gc_release="$(json_field "$last_gc" version)"
[ "$actual_gc_release" = "$expected_gc_release" ] ||
  fail "installed gc release = $actual_gc_release, want $expected_gc_release"

actual_gc_tags="$(json_field "$last_gc" tags)"
case "$actual_gc_tags" in
  *gascity_doltlite_lib*libsqlite3*|*libsqlite3*gascity_doltlite_lib*) ;;
  *) fail "gc build tags do not include gascity_doltlite_lib and libsqlite3: $actual_gc_tags" ;;
esac

actual_bd_source="$(json_field "$last_bd" source)"
case "$actual_bd_source" in
  release:*) ;;
  *) fail "bd was not installed from a release: $actual_bd_source" ;;
esac

seeded_role="$(git config --global --get beads.role 2>/dev/null || true)"
[ "$seeded_role" = "maintainer" ] ||
  fail "installer did not seed git config --global beads.role maintainer; got ${seeded_role:-<unset>}"

log "checking DoltLite beads DB config"
(
  cd "$CITY_DIR"
  prefix="$(bd_sql_config_value issue_prefix 2>/dev/null | tr -d '[:space:]' || true)"
  [ -n "$prefix" ] || fail "DB config issue_prefix returned empty"
  custom_types="$(bd_sql_config_value types.custom 2>/dev/null || true)"
  case "$custom_types" in
    *molecule*convoy*session*) ;;
    *) fail "DB config types.custom missing expected Gas City types: $custom_types" ;;
  esac

  bead_json="$(bd create "clean DoltLite gc init smoke test" --json)"
  bead_id="$(printf '%s\n' "$bead_json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
  [ -n "$bead_id" ] || fail "bd create did not return an id"
  bd delete "$bead_id" --force >/dev/null
)

log "checking supervisor-managed city status"
gc status "$CITY_DIR" >/dev/null

log "PASS: clean DoltLite gc init completed successfully"
