#!/usr/bin/env bash
# Spike: build-web.sh plus sound. The game bundle is the stock one; on top it
# adds undertone's page bridge (a second wasm on the main thread, which plays
# the commands the game sends from its worker), Go's wasm_exec.js for it, and
# a loader script injected into the shell. Serve with:
#   NO_BUILD=1 tools/serve-web.sh   (after copying, or pass the outdir below)
#
# UNDERTONE is the undertone checkout holding cmd/undertone-bridge.
# ?audiolab on the page adds a latency panel (web/audiolab.js).
# The bridge is built with TinyGo: 6.4 MB against Go's 29 MB (TinyGo 0.43,
# 2026-10-25), and it played snappier on a phone. GO=1 builds it with Go. It needs
# -stack-size=1MB: at TinyGo's default the bridge overflows a goroutine stack
# once it's idle, which 0.43 reports as a stack overflow or just hangs the page.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
UNDERTONE="${UNDERTONE:-$here/../let-go-undertone-tesserae-sfx}"
out="${1:-$here/dist/web}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"
tmp="$(mktemp -d -t tess-shell)"
trap 'rm -rf "$tmp"' EXIT
shell="$tmp/shell.html"
sed 's|</body>|  <script src="audiolab.js"></script>\n  <script src="undertone-bridge.js"></script>\n</body>|' "$here/web/shell.html" > "$shell"
TESS_WEB_SHELL="$shell" "$here/tools/build-web.sh" "$out"
echo "building undertone-bridge.wasm ..."
if [ -z "${GO:-}" ]; then
  (cd "$UNDERTONE" && "$here/tools/heavy.sh" tinygo build -target wasm -no-debug -opt=z -stack-size=1MB \
     -o "$out/undertone-bridge.wasm" ./cmd/undertone-bridge)
  cp "$(tinygo env TINYGOROOT)/targets/wasm_exec.js" "$out/undertone-wasm_exec.js"
else
  (cd "$UNDERTONE" && GOOS=js GOARCH=wasm "$here/tools/heavy.sh" go build -o "$out/undertone-bridge.wasm" ./cmd/undertone-bridge)
  cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/undertone-wasm_exec.js"
fi
cp "$here/web/undertone-bridge.js" "$here/web/audiolab.js" "$out/"
ls -la "$out" | sed 1d
