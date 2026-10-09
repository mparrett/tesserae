# TESSERAE devlog

Newest entries at the bottom. Tags: **decision**, **surprise**, **roadblock**, **deferral**,
**opportunity**.

## 2026-10-09: concept → v0

**Brief.** A full-screen ASCII/Unicode breakout with the look of Tetris, breakout and Yars'
Revenge. It should feel smooth, sized for a lunch break, with levels and a story. Novel asks:
reverse breakout, 90° field rotation during play, multiple paddles and balls,
gravity/magnetism, a tilt-labyrinth marble board, bounce materials, mitosis, and a walled-in
ball where you control one tile.

- **decision: renderer.** Every cell is two square pixels (`▀`, fg = top, bg = bottom). This
  goes beyond shapescii's "glyphs as shapes": rendering a real raster makes rotation a
  *sampling* change, so the quarter turns are continuous rotations, not swaps. Shapescii's
  research (via a subagent) fed in: posterize before the diff, track SGR state across cursor
  jumps, sync frames with DEC 2026, turn autowrap off, use an honest frame limiter, and don't
  bother with temporal A/B glyph mixing.
- **decision: frames of reference.** Tiles and balls live in a 48×48 world that rotates; the
  trowel, gravity and tilt live in the screen frame. Paddle collisions map the ball into the
  screen frame and back, so the field can turn under a moving ball. The tilt labyrinth falls
  out of the same machinery, since tilt is just screen-frame gravity.
- **decision: input.** Terminals have no key-up events. The kitty keyboard protocol (flags
  1|2|8) gives true press/repeat/release where it's supported. Elsewhere a short hold model
  rides the auto-repeat: 150 ms for a fresh press, 110 ms per repeat. SGR any-motion mouse
  (1003) gives analog trowel control. let-go's decoder drops the motion bit, but motion still
  arrives as `:button :none` presses, so no let-go change was needed.
- **decision: repo.** Started as a new sibling repo `~/projects-new/tesserae`. Moved to a
  private GitHub remote (`mparrett/tesserae`) on request.
- **surprise: let-go gaps** (upstream candidates, logged in PLAN.md):
  - `bit-or` takes exactly 2 args.
  - `Math/PI`, `Math/sin` and `Math/cos` are unresolved, while `Math/sqrt` and `Math/abs`
    resolve.
  - `(str sb)` on a StringBuilder returns `#<java.lang.StringBuilder>`.
- **roadblock → fixed:** the first frame had black holes. The diff emitter skipped the CUP for
  index-contiguous cells across a row break, and with autowrap off those cells piled up
  past the arena edge.
- **roadblock → fixed:** the SET bonus never fired, because breaking a tile zeroed its piece id
  before the "piece complete" check read it.
- **decision: tetromino walls.** A seeded packer fills `?` regions with real tetrominoes, and
  the bevels are piece-joined (per-tile neighbour masks), so pieces read as Tetris pieces.
  Walls merge into continuous steel the same way.
- **decision: reverse mode.** Ghost tiles are solid, and hitting one sets it. Blueprints are
  one tile thick, because thick blueprints leave interior ghosts that can never be hit. The
  first design idea, setting a ghost when the ball *leaves* it, could trap the ball in its
  own chamber.

## Performance arc

| step | full compose @0.3 rad (quiet box) |
|---|---|
| first cut (closure per pixel, static sample per pixel) | 107 ms |
| world-pixel colour cache, snapped rest angles | 71 ms |
| int-slot loop state, cached CUP strings, `=` instead of `not=`/`zero?` | 54 ms |
| same, coarse 2×2 sampling during rotation | 36 ms |
| AOT `lg` (gfx + world lowered to Go), coarse | ~20 ms |

- **surprise:** coarse sampling alone gained only 1.6×. Per-cell diff bookkeeping, not sampling,
  dominates the VM cost.
- **decision: AOT stays optional.** A background agent built `tools/build-aot.sh` using
  let-go's `benchmark/aot` pipeline. All 54 gfx/world arities dispatch native, with identical
  frame hashes, at roughly 1.8–3× faster. It needed two let-go fixes and one perf change,
  committed on the let-go worktree branch `wt/tesserae-aot` and not pushed:
  - `a8c24ea`: unbox float-typed runtime-call results; give typed-param fns a native override
    with a per-call type guard that falls back to bytecode.
  - `f01a7cd`: cache global var reads in lowered code.

  The catch is that one rebuild takes about 18 minutes, nearly all of it typeinfer reaching a
  fixpoint on `compose!` (an upstream candidate). So `./play.sh` defaults to the VM, and
  `LG=~/projects-new/lg-bin/lg-tesserae-aot ./play.sh` uses the fast binary. Rebuild it after
  gfx/world changes, through `tools/heavy.sh`.
- **decision: semaphore.** `tools/heavy.sh` puts a machine-wide `flock` around CPU-heavy jobs
  (AOT builds, Go builds, long captures), so only one runs at a time on this 4-vCPU box, and
  at `nice 10`. Agents run in their own tesserae worktrees on branches, at most two at once,
  and I merge them.
- **surprise (from the agent):** `timeout ./play.sh` without `--foreground` stops lg as a
  background process group, so the pane stays blank. Use `timeout --foreground`, or tmux
  directly.

## Deferrals

- The Yars-style boss, shape-aware sprite glyphs, glass/ghost balls and saved scores move to v1
  (now in progress).
- The WASM build and level editor are v2.
