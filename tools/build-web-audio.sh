#!/usr/bin/env bash
# Spike: build-web.sh plus sound. The game bundle is the stock one; on top it
# adds undertone's page bridge (a second wasm on the main thread, which plays
# the commands the game sends from its worker), Go's wasm_exec.js for it, and
# a loader script injected into the shell. Serve with:
#   NO_BUILD=1 tools/serve-web.sh   (after copying, or pass the outdir below)
#
# UNDERTONE is the undertone checkout holding cmd/undertone-bridge.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
UNDERTONE="${UNDERTONE:-$here/../let-go-undertone-tesserae-sfx}"
out="${1:-$here/dist/web}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"
shell="$(mktemp -t tess-shell).html"
trap 'rm -f "$shell"' EXIT
sed 's|</body>|  <script src="undertone-bridge.js"></script>\n</body>|' "$here/web/shell.html" > "$shell"
TESS_WEB_SHELL="$shell" "$here/tools/build-web.sh" "$out"
echo "building undertone-bridge.wasm ..."
(cd "$UNDERTONE" && GOOS=js GOARCH=wasm "$here/tools/heavy.sh" go build -o "$out/undertone-bridge.wasm" ./cmd/undertone-bridge)
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/undertone-wasm_exec.js"
cp "$here/web/undertone-bridge.js" "$out/"
ls -la "$out" | sed 1d
