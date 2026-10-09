#!/usr/bin/env bash
# run-level.sh <level 1-10> [extra args]  — autoplay a level in tmux session $TESS_SESSION (default tess)
here="$(cd "$(dirname "$0")/.." && pwd)"
s="${TESS_SESSION:-tess}"
tmux kill-session -t "$s" 2>/dev/null
tmux new-session -d -s "$s" -x 160 -y 52 "cd $here && ./play.sh --level $1 --skip-card --autoplay --stats ${*:2} 2>${TESS_ERR:-/dev/null}; sleep 60"
