#!/usr/bin/env bash
# build-web-onevm.sh [outdir] — the browser build with sound in one VM: the
# game's own wasm links undertone (audio-host's module, through lg -w
# -w-module/-import), renders in the game's worker into a SharedArrayBuffer
# ring, and posts it to the page, where undertone's webaudio/page.js owns the
# AudioContext and the worklet. No second wasm and no EDN hop, unlike
# build-web-audio.sh's bridge. See docs/audio/one-vm-spike.md.
#
# page.js wraps window.Worker to hear the ring, so it is injected into <head>,
# ahead of the runtime's script.
#
# Built with TinyGo by default, the only build that didn't hitch in the
# 2026-10-18 phone listen; GO=1 builds with Go. The ring needs cross-origin
# isolation: serve with tools/coi-serve.py, or rely on the COI service worker
# lg -w ships (as on Pages).
#
# Env: UNDERTONE (undertone checkout for page.js; default ../undertone beside
#      the canonical checkout), GO, TESS_STORAGE_ID.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
host="$here/audio-host"
out="${1:-$here/dist/web-onevm}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"

# page.js must match the undertone version audio-host links, so take it from
# the module cache at that version rather than from a checkout.
ut_dir="$(cd "$host" && go list -m -f '{{.Dir}}' github.com/mparrett/undertone)"
[[ -s "$ut_dir/webaudio/page.js" ]] || { echo "build-web-onevm: no webaudio/page.js in $ut_dir" >&2; exit 1; }

tmp="$(mktemp -d "${TMPDIR:-/tmp}/tess-onevm.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT
sed 's|<head>|<head>\n  <script src="undertone-page.js"></script>|' "$here/web/shell.html" > "$tmp/shell.html"

echo "building audio-host lg ..."
(cd "$host" && "$here/tools/heavy.sh" go build -o lg .)

tiny=1; [[ -n "${GO:-}" ]] && tiny=
cd "$here"
t0=$(date +%s)
LETGO_USE_TINYGO="$tiny" "$here/tools/heavy.sh" "$host/lg" -w "$out" -w-shell "$tmp/shell.html" \
  -w-module "$host" -import github.com/mparrett/undertone/letgo \
  -storage-id "${TESS_STORAGE_ID:-tesserae}" -source-paths . main.lg
cp "$ut_dir/webaudio/page.js" "$out/undertone-page.js"
chmod u+w "$out/undertone-page.js"
echo "built $out in $(( $(date +%s) - t0 ))s ($([[ -n $tiny ]] && echo TinyGo || echo Go)):"
ls -la "$out" | sed 1d
