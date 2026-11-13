#!/usr/bin/env bash
# build-web-onevm.sh [outdir] — the browser build with sound in one VM: the
# game's own wasm links undertone (audio-host's module, through lg -w
# -w-module/-import), renders in the game's worker into a SharedArrayBuffer
# ring, and posts it to the page, where undertone's webaudio/page.js owns the
# AudioContext and the worklet. It replaced the two-VM bridge (a second wasm
# on the page) on 2026-10-31; see docs/audio/one-vm-spike.md.
#
# page.js wraps window.Worker to hear the ring, so it is injected into <head>,
# ahead of the runtime's script.
#
# Built with TinyGo by default, the only build that didn't hitch in the
# 2026-10-18 phone listen; GO=1 builds with Go. The ring needs cross-origin
# isolation: serve with tools/coi-serve.py, or rely on the COI service worker
# lg -w ships (as on Pages).
#
# SEAM=1 instead plays through let-go's own browser audio (lg -w -w-audio,
# let-go #1112): undertone streams into let-go's device, page.js is left
# out, and the queue is let-go's (120 ms in a worker); undertone/set-queue,
# and with it --queue, does nothing there. Opt-in until it has had a phone
# listen (undertone #86, 2026-11-12).
#
# undertone's version is audio-host/go.mod's; let-go's too, for the wasm.
# Env: GO, SEAM, TESS_STORAGE_ID.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
host="$here/audio-host"
out="${1:-$here/dist/web-onevm}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"

seam=; [[ "${SEAM:-}" == 1 ]] && seam=1
shell="$here/web/shell.html"
audio_flags=()
if [[ -n $seam ]]; then
  audio_flags=(-w-audio)
else
  # page.js must match the undertone version audio-host links, so take it
  # from the module cache at that version rather than from a checkout. go
  # list prints an empty Dir until the module is downloaded, as on a fresh
  # CI runner.
  (cd "$host" && go mod download github.com/mparrett/undertone)
  ut_dir="$(cd "$host" && go list -m -f '{{.Dir}}' github.com/mparrett/undertone)"
  [[ -s "$ut_dir/webaudio/page.js" ]] || { echo "build-web-onevm: no webaudio/page.js in $ut_dir" >&2; exit 1; }
  tmp="$(mktemp -d "${TMPDIR:-/tmp}/tess-onevm.XXXXXX")"
  trap 'rm -rf "$tmp"' EXIT
  sed 's|<head>|<head>\n  <script src="undertone-page.js"></script>|' "$here/web/shell.html" > "$tmp/shell.html"
  shell="$tmp/shell.html"
fi

echo "building audio-host lg ..."
(cd "$host" && "$here/tools/heavy.sh" go build -o lg .)

tiny=1; [[ -n "${GO:-}" ]] && tiny=
cd "$here"
t0=$(date +%s)
LETGO_USE_TINYGO="$tiny" "$here/tools/heavy.sh" "$host/lg" -w "$out" -w-shell "$shell" ${audio_flags[@]+"${audio_flags[@]}"} \
  -w-module "$host" -import github.com/mparrett/undertone/letgo \
  -storage-id "${TESS_STORAGE_ID:-tesserae}" -source-paths . main.lg
if [[ -z $seam ]]; then
  cp "$ut_dir/webaudio/page.js" "$out/undertone-page.js"
  chmod u+w "$out/undertone-page.js"
fi
echo "built $out in $(( $(date +%s) - t0 ))s ($([[ -n $tiny ]] && echo TinyGo || echo Go), $([[ -n $seam ]] && echo "let-go audio" || echo "undertone page.js")):"
ls -la "$out" | sed 1d
