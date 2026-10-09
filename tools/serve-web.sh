#!/usr/bin/env bash
# serve-web.sh [port] — build the browser version (tools/build-web.sh) and serve
# dist/web on http://127.0.0.1:<port>/ (default 8360) with cross-origin
# isolation headers. Without COOP/COEP the page boots but dies at the first
# read-key ("no SharedArrayBuffer"), so never use `python -m http.server`.
#
#   NO_BUILD=1 tools/serve-web.sh     serve the existing dist/web
#   HOST=0.0.0.0 tools/serve-web.sh   reachable from the LAN
#
# URL flags stand in for play.sh's: ?level=3&skip-card&autoplay&stats&glyphs=sextant&fps=30
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
port="${1:-8360}"
[[ "${NO_BUILD:-}" == "1" ]] || "$here/tools/build-web.sh"
if ss -ltn "sport = :$port" 2>/dev/null | grep -q LISTEN; then
  echo "port $port is busy; pass another: tools/serve-web.sh <port>" >&2; exit 1
fi
echo "TESSERAE: http://127.0.0.1:$port/"
exec python3 "$here/tools/coi-serve.py" "$port" "$here/dist/web" "${HOST:-127.0.0.1}"
