# let-go upstream candidates found while building TESSERAE

These are local notes only. Nothing here has been filed or posted. Before anything goes
upstream it goes through the joint-xsofy `upstream-outbound` pipeline: search for existing
issues, run the review rubric, draft in `docs-xsofy/outbound/`, then a human pass.

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

These are on local branch `wt/tesserae-aot` in `~/projects-new/worktrees/let-go-tesserae-aot`.
They are not pushed and have no upstream PR.

- `a8c24ea fix(ir)`, two fixes:
  1. A float-typed result of a runtime call (e.g. `(int (math/floor x))`) was assigned
     straight into a `float64` and generated invalid Go. Even `(defn h [x] (math/floor x))`
     failed to compile.
  2. Functions with typed (int64/float64/string) params were generated but never registered
     as overrides, so they silently stayed on bytecode. They now get a guarded override that
     checks argument types per call and falls back to the bytecode fn on a mismatch
     (`rt.InvokeGoOverrideFallback`).
- `f01a7cd perf(ir)`: lowered code caches global var lookups (`rt.CachedVar`) instead of doing
  a namespace map lookup on every read.
- **Observation, not fixed:** typeinfer takes about 18 minutes to reach a fixpoint on one large
  fn (`tesserae.gfx/compose!`). Everything else in gfx and world lowers in about 15 s. Turning
  off inlining didn't help. Worth a repro for the typeinfer census work (#1040/#1048).
- **Observation:** the lowerer guesses `:int` for params used in arithmetic, so float params
  need `^double` hints. Without them, `sample`'s params became int64.
- Inside lowered code, `.append` on a StringBuilder goes through reflection on every call, and
  `volatile!`, `vswap!`, `not=`, `zero?` and `odd?` go through runtime var calls.
  Typed-array `aget` results are untyped, so arithmetic on them stays boxed.

Measured on the game's full-frame compose (quiet box): VM 71 ms → AOT 32 ms (rotated),
`sample` 2.9×, `glow!` 2.7×, idle compose 3×. Frame bytes were identical (hash-checked at four
angles).
