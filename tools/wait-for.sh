#!/usr/bin/env bash
# wait-for.sh <tmux-target> <regex> [timeout-s] — poll the pane text until it matches
t="$1"; re="$2"; to="${3:-60}"; end=$((SECONDS+to))
while [ $SECONDS -lt $end ]; do
  tmux capture-pane -t "$t" -p | grep -qE "$re" && exit 0
  sleep 0.05
done
echo "timeout waiting for: $re" >&2; exit 1
