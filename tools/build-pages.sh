#!/usr/bin/env bash
# build-pages.sh [outdir] — the GitHub Pages site (default _site): the browser
# build with sound (tools/build-web-audio.sh), plus .nojekyll so Pages serves
# the files verbatim.
#
# Pages can't set COOP/COEP. lg -w already handles that: index.html registers
# the coi-serviceworker.js it ships beside it and reloads once, and the worker
# adds the headers, so the page ends up cross-origin isolated (the
# SharedArrayBuffer input ring and undertone's 60 ms audio path need it).
#
# Env: as build-web-audio.sh (UNDERTONE, GO) and build-web.sh (LG, LETGO_SRC).
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
out="${1:-$here/_site}"
rm -rf "$out"
"$here/tools/build-web-audio.sh" "$out"
touch "$out/.nojekyll"
missing=0
for f in .nojekyll index.html coi-serviceworker.js undertone-bridge.wasm undertone-bridge.js undertone-wasm_exec.js; do
  [[ -s "$out/$f" || "$f" == .nojekyll ]] || { echo "build-pages: missing $f" >&2; missing=1; }
done
exit "$missing"
