#!/usr/bin/env bash
# run-level.sh <level 1-10> [extra args]  — autoplay a level in tmux session "tess"
here="$(cd "$(dirname "$0")/.." && pwd)"
tmux kill-session -t tess 2>/dev/null
tmux new-session -d -s tess -x 160 -y 52 "cd $here && ./play.sh --level $1 --skip-card --autoplay --stats ${*:2} 2>${TESS_ERR:-/dev/null}; sleep 60"
