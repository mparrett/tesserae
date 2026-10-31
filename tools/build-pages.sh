#!/usr/bin/env bash
# build-pages.sh [outdir] — the GitHub Pages site (default _site): the browser
# build with sound (tools/build-web-onevm.sh), plus .nojekyll so Pages serves
# the files verbatim.
#
# Pages can't set COOP/COEP. lg -w already handles that: index.html registers
# the coi-serviceworker.js it ships beside it and reloads once, and the worker
# adds the headers, so the page ends up cross-origin isolated (the
# SharedArrayBuffer input ring and undertone's audio ring need it).
#
# Env: as build-web-onevm.sh (GO, TESS_STORAGE_ID).
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
out="${1:-$here/_site}"
# Clear only a previous site (it has .nojekyll) or an empty dir, never an
# arbitrary outdir someone passed.
if [[ -d "$out" && -n "$(ls -A "$out")" && ! -e "$out/.nojekyll" ]]; then
  echo "build-pages: $out isn't empty and isn't a previous site; refusing to clear it" >&2; exit 1
fi
rm -rf "$out"
"$here/tools/build-web-onevm.sh" "$out"
touch "$out/.nojekyll"
missing=0
for f in index.html coi-serviceworker.js undertone-page.js; do
  [[ -s "$out/$f" ]] || { echo "build-pages: missing $f" >&2; missing=1; }
done
exit "$missing"
