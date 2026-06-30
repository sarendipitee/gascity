#!/usr/bin/env bash
set -euo pipefail

version="${1:-${GC_DOLTLITE_VERSION:-0.11.23}}"
version="${version#v}"

case "$(uname -s)" in
  Linux) os_name=linux ;;
  *) echo "DoltLite-linked gc releases are currently built on Linux only" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch_name=x64 ;;
  *) echo "DoltLite-linked gc releases currently support linux/amd64 only" >&2; exit 1 ;;
esac

asset="doltlite-lib-${os_name}-${arch_name}-${version}.zip"
base_url="https://github.com/dolthub/doltlite/releases/download/v${version}"
work_dir="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/gascity-doltlite-release-lib/${version}"
zip_path="${work_dir}/${asset}"
lib_dir="${work_dir}/lib"

download_file() {
  local url="$1"
  local dest="$2"
  mkdir -p "$(dirname "$dest")"
  python3 - "$url" "$dest" <<'PY'
import os
import sys
import tempfile
import urllib.request

url, dest = sys.argv[1], sys.argv[2]
headers = {}
token = os.environ.get("GITHUB_TOKEN")
if token:
    headers["Authorization"] = f"Bearer {token}"
request = urllib.request.Request(url, headers=headers)
directory = os.path.dirname(dest) or "."
fd, tmp = tempfile.mkstemp(prefix=".download-", dir=directory)
os.close(fd)
try:
    with urllib.request.urlopen(request, timeout=120) as response:
        with open(tmp, "wb") as out:
            while True:
                chunk = response.read(1024 * 1024)
                if not chunk:
                    break
                out.write(chunk)
    os.replace(tmp, dest)
except Exception:
    try:
        os.unlink(tmp)
    except OSError:
        pass
    raise
PY
}

if [ ! -r "${lib_dir}/doltlite.h" ] || [ ! -r "${lib_dir}/libdoltlite.a" ]; then
  rm -rf "$lib_dir"
  download_file "${base_url}/${asset}" "$zip_path"
  python3 - "$zip_path" "$lib_dir" <<'PY'
import os
import shutil
import sys
import tempfile
import zipfile

zip_path, dest = sys.argv[1], sys.argv[2]
tmp = tempfile.mkdtemp(prefix="doltlite-lib-")
try:
    with zipfile.ZipFile(zip_path) as archive:
        archive.extractall(tmp)
    entries = [os.path.join(tmp, name) for name in os.listdir(tmp)]
    src = entries[0] if len(entries) == 1 and os.path.isdir(entries[0]) else tmp
    os.makedirs(os.path.dirname(dest), exist_ok=True)
    shutil.copytree(src, dest)
finally:
    shutil.rmtree(tmp, ignore_errors=True)
PY
fi

if [ ! -r "${lib_dir}/doltlite.h" ] || [ ! -r "${lib_dir}/libdoltlite.a" ]; then
  echo "DoltLite release library is missing doltlite.h or libdoltlite.a under ${lib_dir}" >&2
  exit 1
fi

export DOLTLITE_LIB="$lib_dir"
export CGO_CFLAGS="-I${lib_dir}${CGO_CFLAGS:+ ${CGO_CFLAGS}}"
export CGO_LDFLAGS="-L${lib_dir} ${lib_dir}/libdoltlite.a -lz -lpthread -lm${CGO_LDFLAGS:+ ${CGO_LDFLAGS}}"
export LD_LIBRARY_PATH="${lib_dir}${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}"

if [ -n "${GITHUB_ENV:-}" ]; then
  {
    echo "DOLTLITE_LIB=${DOLTLITE_LIB}"
    echo "CGO_CFLAGS=${CGO_CFLAGS}"
    echo "CGO_LDFLAGS=${CGO_LDFLAGS}"
    echo "LD_LIBRARY_PATH=${LD_LIBRARY_PATH}"
  } >> "$GITHUB_ENV"
fi

echo "prepared DoltLite release library: ${DOLTLITE_LIB}"
