# let-go upstream candidates found while building TESSERAE

Working notes on let-go gaps hit while building the game. Filed so far: the AOT lowering
blowup as nooga/let-go#1050, with the fix in nooga/let-go#1051. The rest are unfiled
candidates.

Verified against `lg` built from nooga/let-go `a13e042` (2026-10-09).

## Clojure-compat gaps (small)

| form | let-go a13e042 | Clojure |
|---|---|---|
| `(bit-or 1 2 4)`, likewise `bit-and`, `bit-xor` | `bit-or expects 2 args` | variadic |
| `Math/sin`, `Math/cos`, `Math/PI` | `Can't resolve` | `java.lang.Math` statics |
| `Math/sqrt`, `Math/abs` | work | |
| `(str sb)` on a `StringBuilder` | `"#<java.lang.StringBuilder>"` | the contents |
| `(spit f s :append true)` | `wrong number of arguments 4` | appends |

Workarounds in the game: nested `bit-or`, `math/*` everywhere, `.toString`, and
`(binding [*out* *err*] …)` instead of append-to-file.

## Terminal

- **SGR mouse motion.** `decodeSGRMouse` ignores the motion bit (32), so with 1003
  any-motion enabled by hand, motion arrives as `{:action :press :button :none}`. That's
  usable, but you can't tell motion from a press. A small upstream change would decode bit
  32 as `:action :move` and add an opt-in `enable-mouse` mode for 1002/1003. The comment in
  `term.go` notes these modes as out of scope.
- `term/write` and `term/flush` worked well for one-string-per-frame output with DEC 2026
  sync. No issue there.

## AOT lowering (`benchmark/aot` pipeline applied to a real app)

These are on a local let-go branch (`wt/tesserae-aot`), not pushed and with no upstream PR
of their own.

- `a8c24ea fix(ir)`, two fixes (**part 1 is superseded by upstream #1044**, which fixes the
  same `trampoline-call-stmts` site with a checked assert and also covers the guarded
  direct-call fallback; drop it on rebase. Part 2 and f01a7cd have no upstream counterpart):
  1. A float-typed result of a runtime call (e.g. `(int (math/floor x))`) was assigned
     straight into a `float64` and generated invalid Go. Even `(defn h [x] (math/floor x))`
     failed to compile.
  2. Functions with typed (int64/float64/string) params were generated but never registered
     as overrides, so they silently stayed on bytecode. They now get a guarded override that
     checks argument types per call and falls back to the bytecode fn on a mismatch
     (`rt.InvokeGoOverrideFallback`).
- `f01a7cd perf(ir)`: lowered code caches global var lookups (`rt.CachedVar`) instead of doing
  a namespace map lookup on every read.
- **ROOT-CAUSED (2026-10-09), filed as #1050, fix in #1051: exponential `closure-info*` walk
  in Go emission, a regression from #767.** The 18 min → over 95 min `compose!` lowering isn't typeinfer. Typeinfer is
  healthy: about 2.1 enqueues per instruction (the #558 baseline), the same work counts on
  base, tip and tip+#1038/#1039, and 8–17 s total even on `compose!`. The time is in
  `ir.lower-go/closure-info*` (lower_go.lg:758 at a13e042, identical at tip 49858bd).
  - **How:** the depth-first walk over block-param sources (`unanimous-closure`, line 721)
    doesn't cache any answer that depended on a node still on the walk's stack (line 820,
    added by #767). Inside a loop nearly every walk reaches the header, so nothing in the
    body is cached, and each `when` join doubles the paths: O(2^K). The walk runs for every
    value, closures or not (`closure-value?`, `prepare-lower-body-metadata!`). licm amplifies
    it by threading hoisted constants through every block.
  - **Minimal repro:** `(defn f [a n] (loop [i 0] (when (< i n) (when (= 0 (aget a i))
    (aset a i 0)) ... K whens ... (recur (inc i)))) nil)`. K=6 takes 1.8 s, K=10 12.9 s,
    K=12 49.8 s; the same whens without the loop are linear.
  - **Bisect:** e344ff2 (before #767) gives 1.7 / 2.1 / 2.8 s at K=8/10/12; 0003a0b (#767)
    gives 2.5 / 7.5 / 27.9 s. The emitted Go is identical.
  - **Not fixed** by tip (#1040, #1044) or the open #1038/#1039.
  - **Fix direction (sketched and measured):** replace the per-query walk with one fixpoint
    over the block-param graph per function. A param starts pending, can become one closure,
    then nil. With this, gfx@7cc3cd9 including `compose!` lowers in about 67 s, with
    byte-identical Go wherever stock finishes. Not yet checked against let-go's #766/#767
    regression tests. Further ideas: skip the walk for params typeinfer proved scalar, and
    stop licm threading rematerialisable constants through every block.
  - **Write-up:** the repro scripts were kept locally; #1050 carries a standalone repro and
    #1051 the fix with tests.
  - Minor, uncaused: typeinfer time per drain rises about 1.8× over a 13× size range (a small
    #558-class effect, about 7 s of the hour).
- **Observation:** the lowerer guesses `:int` for params used in arithmetic, so float params
  need `^double` hints. Without them, `sample`'s params became int64.
- Inside lowered code, `.append` on a StringBuilder goes through reflection on every call, and
  `volatile!`, `vswap!`, `not=`, `zero?` and `odd?` go through runtime var calls.
  Typed-array `aget` results are untyped, so arithmetic on them stays boxed.

Measured on the game's full-frame compose (quiet box): VM 71 ms → AOT 32 ms (rotated),
`sample` 2.9×, `glow!` 2.7×, idle compose 3×. Frame bytes were identical (hash-checked at four
angles).

## WASM (`lg -w`)

- **Storage in the browser.** `storage/*` uses `localStorage`, but the VM runs in a Web Worker,
  where `localStorage` doesn't exist, so every call throws. Possible fixes: a page↔worker
  storage bridge (postMessage), or IndexedDB, which workers do have.
- An xterm.js resize doesn't reach the VM (no SIGWINCH equivalent), so apps have to poll
  `term/size`.
- `lg -w` evaluates the entry file while compiling, so a top-level `(main)` call runs at build
  time unless it's guarded by `*compiling-aot*`. This is documented in usage.md, but it's
  surprising for a game loop.
