#!/usr/bin/env bash
# build-web.sh [outdir] — build the browser version of TESSERAE with let-go's
# WASM target into a self-contained directory (default dist/web): index.html
# with the program wasm inlined (gzip+base64) and a COI service worker.
#
# The page uses web/shell.html (an -w-shell template: xterm.js from jsDelivr,
# a font size picked for a 124x50 grid, WebGL renderer). Serving it needs
# cross-origin isolation (COOP/COEP); see tools/serve-web.sh.
#
# lg -w compiles main.lg and its tesserae/* namespaces to bytecode, then runs
# `go build` (GOOS=js GOARCH=wasm) against let-go's source at the pinned
# commit, fetched through the Go module proxy (or LETGO_SRC=<checkout>).
# It's CPU-heavy, so it runs behind tools/heavy.sh.
#
# Env: LG (default tools/lg-pinned), LETGO_SRC (optional local let-go checkout),
#      TESS_STORAGE_ID (localStorage store id, default tesserae).
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
LG="${LG:-$here/tools/lg-pinned}"
out="${1:-$here/dist/web}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"
cd "$here"
t0=$(date +%s)
"$here/tools/heavy.sh" "$LG" -w "$out" -w-shell "$here/web/shell.html" \
  -storage-id "${TESS_STORAGE_ID:-tesserae}" -source-paths . main.lg
echo "built $out in $(( $(date +%s) - t0 ))s:"
ls -la "$out" | sed 1d
