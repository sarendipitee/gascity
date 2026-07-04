#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: install-doltlite-archive.sh VERSION [--cache]

Downloads DoltLite release archives, verifies GitHub release SHA-256 digests,
and installs doltlite, doltlite-remotesrv, headers, and libdoltlite.
Use VERSION=latest to resolve the latest release.
USAGE
}

version="${1:-}"
if [[ -z "$version" ]]; then
  usage
  exit 2
fi
shift || true

use_cache=false
while (($#)); do
  case "$1" in
    --cache) use_cache=true ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
  shift
done

case "$(uname -s)" in
  Darwin) os=osx ;;
  Linux) os=linux ;;
  *)
    echo "Unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64) arch=x64 ;;
  *)
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

target="${os}-${arch}"
case "$target" in
  linux-x64|linux-arm64|osx-arm64|osx-x64) ;;
  *)
    echo "No DoltLite release artifact for ${target}" >&2
    exit 1
    ;;
esac

github_api() {
  local url="$1"
  local auth_header=()
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    auth_header=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
  fi
  curl -fsSL --retry 5 --retry-delay 2 --retry-all-errors --retry-connrefused \
    "${auth_header[@]}" \
    -H "Accept: application/vnd.github+json" \
    "$url"
}

if [[ "$version" == "latest" ]]; then
  if ! command -v jq >/dev/null 2>&1; then
    echo "jq is required to resolve the latest DoltLite release" >&2
    exit 1
  fi
  version="$(github_api "https://api.github.com/repos/dolthub/doltlite/releases/latest" | jq -r .tag_name)"
fi

if [[ -z "$version" || "$version" == "null" ]]; then
  echo "Failed to resolve DoltLite version" >&2
  exit 1
fi

version_no_v="${version#v}"
base_url="https://github.com/dolthub/doltlite/releases/download/${version}"
tools_archive="doltlite-tools-${target}-${version_no_v}.zip"
lib_archive="doltlite-lib-${target}-${version_no_v}.zip"

github_release_asset_sha() {
  local asset="$1"
  if ! command -v jq >/dev/null 2>&1; then
    echo "jq is required to resolve GitHub release asset checksums" >&2
    exit 1
  fi
  github_api "https://api.github.com/repos/dolthub/doltlite/releases/tags/${version}" \
    | jq -r --arg asset "$asset" '.assets[] | select(.name == $asset) | .digest // empty' \
    | sed 's/^sha256://'
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d ' ' -f 1
  else
    shasum -a 256 "$1" | cut -d ' ' -f 1
  fi
}

download_and_verify() {
  local archive="$1"
  local dst="$2"
  local expected
  expected="$(github_release_asset_sha "$archive")"
  if [[ -z "$expected" ]]; then
    echo "No DoltLite checksum found for ${version}/${archive}" >&2
    exit 1
  fi
  curl -fsSL --retry 5 --retry-delay 2 --retry-all-errors --retry-connrefused \
    -o "$dst" "${base_url}/${archive}"
  local actual
  actual="$(sha256_file "$dst")"
  if [[ "$actual" != "$expected" ]]; then
    echo "DoltLite checksum mismatch for ${archive}" >&2
    echo "expected: $expected" >&2
    echo "actual:   $actual" >&2
    exit 1
  fi
}

install_file() {
  local mode="$1"
  local src="$2"
  local dst="$3"
  mkdir -p "$(dirname "$dst")"
  install -m "$mode" "$src" "$dst"
}

install_file_with_sudo_fallback() {
  local mode="$1"
  local src="$2"
  local dst="$3"
  local dst_dir
  dst_dir="$(dirname "$dst")"
  mkdir -p "$dst_dir"
  if [[ -w "$dst_dir" ]]; then
    install_file "$mode" "$src" "$dst"
  elif command -v sudo >/dev/null 2>&1; then
    sudo install -m "$mode" "$src" "$dst"
  else
    echo "Cannot write $dst and sudo is unavailable" >&2
    exit 1
  fi
}

if $use_cache; then
  prefix="${RUNNER_TOOL_CACHE:-$HOME/.local}/gascity-doltlite/${version}/${target}"
else
  install_dir="${DOLTLITE_INSTALL_DIR:-/usr/local/bin}"
  prefix="$(dirname "$install_dir")"
fi
bin_dir="${prefix}/bin"
include_dir="${prefix}/include"
lib_dir="${prefix}/lib"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

download_and_verify "$tools_archive" "${tmp}/${tools_archive}"
download_and_verify "$lib_archive" "${tmp}/${lib_archive}"
unzip -q -d "$tmp" "${tmp}/${tools_archive}"
unzip -q -d "$tmp" "${tmp}/${lib_archive}"

tools_dir="${tmp}/doltlite-tools-${target}-${version_no_v}"
lib_src_dir="${tmp}/doltlite-lib-${target}-${version_no_v}"

for bin in doltlite doltlite-remotesrv; do
  src="${tools_dir}/${bin}"
  [[ -f "$src" ]] || continue
  if $use_cache; then
    install_file 0755 "$src" "${bin_dir}/${bin}"
  else
    install_file_with_sudo_fallback 0755 "$src" "${bin_dir}/${bin}"
  fi
done

for hdr in sqlite3.h doltlite.h doltlite_remotesrv.h; do
  src="${lib_src_dir}/${hdr}"
  [[ -f "$src" ]] || continue
  if $use_cache; then
    install_file 0644 "$src" "${include_dir}/${hdr}"
  else
    install_file_with_sudo_fallback 0644 "$src" "${include_dir}/${hdr}"
  fi
done

for lib in libdoltlite.a libdoltlite.so libdoltlite.dylib; do
  src="${lib_src_dir}/${lib}"
  [[ -f "$src" ]] || continue
  mode=0644
  case "$lib" in
    *.so|*.dylib) mode=0755 ;;
  esac
  if $use_cache; then
    install_file "$mode" "$src" "${lib_dir}/${lib}"
  else
    install_file_with_sudo_fallback "$mode" "$src" "${lib_dir}/${lib}"
  fi
done

if $use_cache; then
  if [[ -n "${GITHUB_PATH:-}" ]]; then
    echo "$bin_dir" >> "$GITHUB_PATH"
  fi
  if [[ -n "${GITHUB_ENV:-}" ]]; then
    {
      echo "DOLTLITE_LIB=$lib_dir"
      echo "GC_DOLTLITE_LIB=$lib_dir"
    } >> "$GITHUB_ENV"
  fi
fi

if [[ -x "${bin_dir}/doltlite" ]]; then
  "${bin_dir}/doltlite" version
else
  echo "Installed libdoltlite ${version} under ${lib_dir}"
fi
