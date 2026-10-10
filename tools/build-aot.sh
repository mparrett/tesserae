#!/usr/bin/env bash
# Build an lg binary with TESSERAE's hot namespaces AOT-lowered to native Go.
#
#   tools/build-aot.sh [--letgo DIR] [--out PATH] [--lg HOST_LG] [--keep] [NS ...]
#
#   --letgo  let-go checkout/worktree the Go is generated into and built from
#            (default $TESSERAE_AOT_LETGO, else ../worktrees/let-go-tesserae-aot next to
#            this repo). Never the
#            canonical checkout: the script writes (then removes) files there.
#   --out    output binary (default $TESSERAE_AOT_OUT, else ../lg-bin/lg-tesserae-aot)
#   --lg     lg used to run the lowering (default: built from --letgo first, so
#            the lowerer and the runtime the Go compiles against always match)
#   AOT_SKIP=a,b  env: defn names to leave on the VM (e.g. AOT_SKIP=compose! on a
#            let-go older than #1051, where lowering it runs over an hour; let-go#1050)
#   --keep   leave the generated Go in the worktree (for reading it)
#   NS       namespaces to lower (default: tesserae.gfx tesserae.world)
#
# Mechanism (mirrors let-go benchmark/aot/build.lg): every defn/defn- of each
# namespace is lowered via ir.passes.pipeline/lower-ns-to-go-result into
# <letgo>/pkg/rt/core_go_lowered/tesserae/<ns>/; a wireup file blank-imports
# those packages under the lg_tesserae build tag, so their init() registers
# Go-native overrides that replace the bytecode vars when the game `require`s
# the namespace from source. Everything else (defs, macros, top-level forms)
# still loads from the game's .lg files at runtime, so the binary must be run
# against the SAME source it was built from: rebuild after editing gfx/world.
#
# Needs the let-go fixes on branch wt/tesserae-aot (typed trampoline results,
# guarded overrides for typed-param fns, cached var reads); on stock let-go
# gfx does not compile and most hot fns stay bytecode. "Native" in the verify
# step means the var holds the Go adapter; an adapter for a fn with float64 /
# int64 params still calls the bytecode fn when an argument has another type
# (e.g. an int passed to a ^double param), so keep hot call sites float.
#
# Re-runnable on whatever the current source is; generated files are removed
# from the worktree afterwards (unless --keep). Lowering runs typeinfer to a
# fixpoint and is slow: ~18 min on 4 vCPU, nearly all of it compose!
# (the rest of gfx + world lower in ~15 s).
set -euo pipefail

game="$(cd "$(dirname "$0")/.." && pwd)"
letgo="${TESSERAE_AOT_LETGO:-$game/../worktrees/let-go-tesserae-aot}"
out="${TESSERAE_AOT_OUT:-$game/../lg-bin/lg-tesserae-aot}"
host_lg=""
keep=0
nss=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --letgo) letgo="$2"; shift 2 ;;
    --out)   out="$2"; shift 2 ;;
    --lg)    host_lg="$2"; shift 2 ;;
    --keep)  keep=1; shift ;;
    -h|--help) sed -n '2,35p' "$0"; exit 0 ;;
    *) nss+=("$1"); shift ;;
  esac
done
[[ ${#nss[@]} -gt 0 ]] || nss=(tesserae.gfx tesserae.world)
letgo="$(cd "$letgo" && pwd)"
mkdir -p "$(dirname "$out")"
out="$(cd "$(dirname "$out")" && pwd)/$(basename "$out")"
canonical="${TESSERAE_LETGO_CANONICAL:-$game/../let-go}"
[[ ! -d "$canonical" || "$letgo" != "$(cd "$canonical" && pwd)" ]] || { echo "build-aot: refusing to write into the canonical let-go checkout" >&2; exit 2; }
[[ -f "$letgo/lg.go" ]] || { echo "build-aot: $letgo is not a let-go checkout" >&2; exit 2; }

gen_root="$letgo/pkg/rt/core_go_lowered"
gen_dir="$gen_root/tesserae"
wireup="$letgo/zz_lg_tesserae_gen.go"
log="$(mktemp -t build-aot.XXXXXX)"

cleanup() {
  if [[ $keep -eq 0 ]]; then
    rm -f "$wireup"
    rm -rf "$gen_dir"
    rmdir "$gen_root" 2>/dev/null || true
  fi
}
trap cleanup EXIT

if [[ -e "$gen_dir" || -e "$wireup" ]]; then
  echo "build-aot: removing stale generated files from a previous run"
  rm -rf "$gen_dir" "$wireup"
fi

if [[ -z "$host_lg" ]]; then
  host_lg="$letgo/build/lg-aot-host"
  echo "build-aot: building host lg from $letgo ..."
  ( cd "$letgo" && go build -o "$host_lg" . )
fi

echo "build-aot: lowering ${nss[*]} with $(readlink -f "$host_lg") (slow) ..."
( cd "$game" && "$host_lg" -source-paths "$game" "$game/tools/aot-lower.lg" "$gen_root" "${nss[@]}" ) \
  >"$log" 2>&1 || { grep -v '^WARNING' "$log" | tail -20; echo "build-aot: lowering failed (log $log)" >&2; exit 1; }
grep -E '^AOT-' "$log" | grep -v '^AOT-LOWERED' || true
lowered=$(grep -c '^AOT-LOWERED' "$log" || true)
fallback=$(grep -c '^AOT-FALLBACK' "$log" || true)
echo "build-aot: $lowered defn arities emitted as Go, $fallback fell back"

{
  echo "//go:build lg_tesserae"
  echo
  echo "package main"
  echo
  echo "import ("
  grep '^AOT-PKG' "$log" | awk '{print "\t_ \"github.com/nooga/let-go/pkg/rt/core_go_lowered/" $2 "\""}'
  echo ")"
} >"$wireup"

echo "build-aot: go build -tags lg_tesserae -o $out ..."
mkdir -p "$(dirname "$out")"
( cd "$letgo" && go build -tags lg_tesserae -o "$out" . )

# Verify: in the tagged binary, after `require`, which defn vars hold a
# native fn and which are still bytecode.
verify="$(mktemp --suffix=.lg -t build-aot-verify.XXXXXX)"
{
  for ns in "${nss[@]}"; do echo "(require '$ns)"; done
  echo "(doseq [[nsn names] ["
  grep -E '^AOT-(LOWERED|FALLBACK)' "$log" | awk '{print $2}' | awk -F/ '{print $1, $2}' | sort -u |
    awk '{a[$1]=a[$1] " \"" $2 "\""} END {for (n in a) print "  [\"" n "\" [" a[n] "]]"}'
  echo "  ]]"
  echo "  (doseq [n names]"
  echo "    (let [v @(find-var (symbol nsn n))]"
  echo "      (println (if (re-find #\"native-fn\" (str v)) \"NATIVE\" \"BYTECODE\") (str nsn \"/\" n)))))"
} >"$verify"
vout="$(cd "$game" && "$out" -source-paths "$game" "$verify" 2>&1)" || { echo "$vout"; echo "build-aot: verify run failed" >&2; exit 1; }
rm -f "$verify"
echo "$vout" | grep '^BYTECODE' || true
nat=$(echo "$vout" | grep -c '^NATIVE' || true)
byc=$(echo "$vout" | grep -c '^BYTECODE' || true)
echo "build-aot: verify: $nat defns dispatch native, $byc still bytecode"
echo "build-aot: OK -> $out  (lowering log: $log)"
