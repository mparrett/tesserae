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

## 2026-10-09: v1 batch 1 (boss, glyph sprites, balance)

Two agents ran in parallel worktrees (`v1/boss`, `v1/glyphs`), with CPU-heavy steps through
`tools/heavy.sh`, while the main session did captures, the gallery and balance. The merge had
one conflict, in the ball draw call, and both sides were kept.

### 6.2 The Grout Core (boss agent)
- **decision: data-driven boss.** The Yars homage is five level keys (`:core :shield :neutral
  :destroyer :swirl`) on an ordinary paddle level, plus SHIELD and CORE tile types. Shield
  cells scroll along ring paths through `w/put!`/`w/clear!`, touching only cells whose
  occupancy changes.
- **decision: frames.** Core and shield are world tiles. The neutral zone is tinted background
  tiles, re-tinted a few at a time, plus sparkles. Destroyer and Swirl live in the screen
  frame, beside the trowel.
- **surprise: regen vs round trip.** A paddle round trip is about 2.3 s, so 3 s regen nearly
  cancelled erosion. Fixed with a 3×3 core, a non-mending inner arc, and a 4 s regen pause
  after a core hit. Autoplay wins in about 130 s.
- **decision: fairness.** Threat timers pause while a ball is stuck, a Swirl never warns during
  a stun, and the Destroyer can't stun during a Swirl.
- **surprise:** let-go `spit` has no `:append` (an upstream candidate).
- **deferral:** rotation on the boss, because the band would turn vertical; it needs a
  screen-frame overlay first.

### Shape-aware sprite glyphs (glyph agent)
- **decision:** the ball core is a sub-cell glyph sprite. Disc area coverage per 2×2 quadrant
  or 2×3 sextant subcell is thresholded at more than half, and the mask *is* the glyph index
  (a 64-entry table, no codebook). `--glyphs quadrant|sextant|off`, default quadrant; `off` is
  byte-identical to v0.
- **surprise:** quadrants barely help vertically. A cell is 1×2 square pixels, so a quadrant
  subcell is ½ px wide but a full pixel tall, and balls become crisp pills. Sextants look
  round but need a font or terminal that draws them.
- **surprise:** centre-point sampling made notched shapes. Area coverage (2 scanlines per
  subcell row) fixed it, at about 0.1–0.2 ms per ball.
- Glyph modes emit about 19% fewer cells, and balls stay sharp during coarse rotation frames.
- **fixed after merge** (flagged by the agent): coarse frames now clear light on the pixels
  they skip, and text cells diff on colour too, so fading score popups redraw.

### Balance (main session, `tools/sim.lg`)
- A headless simulator runs every level with the autopilot at a fixed 1/50 s step and reports
  clear time, lives lost and peak balls. A whole sweep takes about 6 min wall time under the
  heavy lock.
- **surprise:** the autopilot never drops a ball on paddle levels, but end-games slogged.
  Hunting the last tiles left 8–12 standing at 240 s.
- **decision: "the last tesserae sing."** At 8 or fewer goal tiles left, they pulse and gently
  pull the Spark (14 u/s²). It's thematic, and it fixes the slog.
- **decision: piece snap.** Striking a ghost lays its whole tetromino, Tetris-style. Blueprint
  needs about 4× fewer hits.
- **decision:** paddle-level spark speeds +20%. Marble falls cost 150 points instead of a life,
  following the labyrinth convention. Grout decay slowed from every 7 s to every 9 s.
- **surprise:** an autopilot that aims at the nearest front-row tile did *worse* (vertical
  ping-pong loops). Aiming at a random goal tile once per descent is better.
- Sweep after tuning (240 s cap): 1.1 201 s, 1.2 154 s, 2.1 63 s, 2.2 89 s, 5.1 135 s, 6.1 114 s.
  Blueprint and Grout Creep end just short (7 and 10 left). The tilt levels can't be judged
  with the naive tilt bot.

## 2026-10-09: v1 meta layer (progress, unlocks, medals, tally)

- **decision: let-go `storage`.** String key/values, one file per key under
  `~/.config/let-go/storage/<store>/`. Records are EDN (`pr-str`/`read-string`). `play.sh` pins
  `-storage-id tesserae`, because without the flag a `main.lg` run keys its store by directory
  name, and every worktree would get its own progress. Records are keyed by level id, not
  index, so inserting a level keeps old saves. Every call is guarded; a failure drops to
  in-memory progress.
- **decision: medals from par.** Gold at or under par, silver up to 1.5× par, bronze
  otherwise, with pars taken from the autopilot sweep. A clear without losing a spark or marble
  adds +500 and a ✦. Autoplay never saves.
- **surprise:** tally text drawn straight over tiles and fireworks was unreadable, because a
  glyph's bg is the pixel beneath it. The tally now sits on one dark `rect!` plate, and the
  fireworks keep to the edges.
- **decision:** `game.lg` gets `meta-*` hooks plus one section. The logic lives in
  `tesserae/meta.lg`, so it merges cleanly with level work and can be tested without a
  terminal.
- **opportunity:** let-go ships `clojure.test`. `./tools/test.sh` runs 143 assertions in about
  1 s (input parsing including kitty CSI-u, packing determinism, map validity, storage round
  trip).
- **main session:** the tilt autopilot now follows a BFS path around walls and pits with a
  velocity controller, so the sim can judge tilt levels (4.1 and 4.2 clear in about 38 s with
  perfect knowledge; expect 1.5–2 min for a human).

## 2026-10-09: 3.3 Mortar Well and ball materials (well agent)

- **decision: the falling piece is tiles.** The mortar tetromino is MORTAR tiles in the grid,
  moved with `w/clear!`/`w/put!`, so ball collisions, the render cache and bevel joins come
  free. An axis/velocity hint recorded in `axis-move` tells the hit which face was struck:
  a side hit shoves, a hit from below turns it (with kicks), a hit from above tamps it.
- **surprise: the Spark barely visits the well.** With the well high in the field, the bot's
  Spark hit a falling piece once in 120 s. Lowering the well (four open rows above its mouth)
  and aiming high up the outer wall raised that to 8 hits, all of which moved the piece.
- **decision: "the Mortar seeks its course".** Random entry points choked the well about every
  25 s, and greedy entry points won with no player input. Taking the best of 12 random entries
  (scored on full rows, holes and height) gives an autopilot clear in about 95–100 s at
  5 lines, and the Spark's shoves and turns correct it.
- **decision: forgiving choke.** Reaching the brim costs a spark and empties the well; laid
  courses still count.
- **decision:** a 7-bag randomizer, a NEXT preview in the panel, and `=` for the 12th level.
- **decision: materials.** Glass shatters into three sparks (±25°) on its next trowel bounce.
  Ghost breaks the next three breakables outright without bouncing (walls still bounce it).
- **merge (main):** the title panel came from meta (`meta-title-panel!`, generic over
  levels), and the HUD keeps both meta rows 11–12 and the well preview on rows 13–15.
  3.3 gets par 150. Tests: 150 assertions pass.
- **deferral:** speeding up the fall as you progress, queueing shoves that a ball blocks, a
  landing shadow for the piece.

## 2026-10-09: v2 browser build (web agent)

- **decision: shell template.** `lg -w -w-shell web/shell.html` is let-go's xterm shell plus a
  font size picked so the grid reaches 124×50 (the doubled raster) and the WebGL renderer.
  `?grid=small` skips doubling. Flags come from URL params via `js/url-param`, which is nil
  outside a browser, so `?level=3&skip-card&autoplay&stats` mirrors the play.sh options.
- **surprise:** `lg -w` evaluates main.lg while compiling, so the first build started the game
  and hung. The entry is now guarded with `(when-not *compiling-aot* ...)`, as xsofy does.
- **surprise:** an xterm.js resize sends the VM no signal, so under `*in-wasm*` input.lg polls
  `term/size`.
- **roadblock → deferral:** progress can't be saved in the browser. let-go's browser storage
  uses `localStorage`, but the VM runs in a Web Worker, which has none. The guard shows
  "progress not saved". Upstream candidate: a storage bridge from the page into the worker.
- **surprise: perf.** The Go-wasm VM is roughly 3–5× slower than native: 8–20 fps on the
  doubled raster (headless Chromium with software WebGL, loaded box), 25–40 fps with
  `?grid=small`. Native holds 50. Deferred: an adaptive grid, and AOT-lowered gfx in the wasm
  build.
- Ctrl-C and title Q don't quit in a tab. The kitty keyboard and mouse escapes are harmless in
  xterm.js.
- **main session, sim sweep after all merges (240 s cap):** 10 of 12 levels clear (1.1 202 s,
  1.2 126 s, 2.1 89 s, 2.2 112 s, 4.x 38 s, 5.1 136 s, 6.1 39 s, 6.2 139 s). 3.1, 3.2 and 3.3
  end short (14, 5 and 1 left); Blueprint varies a lot run to run with a bot that can't aim at
  thin line art.

## 2026-10-09: v2 level editor (editor agent)

- `./play.sh --edit [file]` opens a 24×24 editor drawn by the game's own renderer. `?` and `,`
  show packed: `load-map!` re-packs the whole field on each edit (about 16 ms), while the EDN
  keeps the raw chars. All arena cells are marked dirty rather than invalidated, so `compose!`
  still diffs.
- Brushes come from the levels.lg legend (boss chars left out). There's flood fill,
  rectangles, mirror painting, a pen, 30-step undo, and theme, goal, flag, speed and par
  settings. `P` play-tests in the same process, and Q returns to the editor.
- `--map file.edn` plays a custom map. game.lg gained a small custom section (`CUSTOM-LI`,
  `level-at`, `on-custom-exit`) and guards so custom runs never write progress or medals.
- **surprise:** the editor pushes kitty flags 1 (not 11) so `?` arrives as text rather than
  shift+`/`; play-test switches back to the game's flags. `edn/read-string` resolves but is
  nil (related to the open let-go #992 work).
- **surprise:** piping play.sh's stdout (e.g. through `tee`) makes `term/size` nil, which falls
  back to a 100×30 layout.
- Two example maps ship in `levels/custom/`, both made with the editor through tmux
  send-keys. Tests: 196 assertions.

## 2026-10-09: AOT rebuild roadblock

- **roadblock:** rebuilding the AOT binary after the glyph-sprite merge ran into trouble. The
  lowering of `compose!` ran 95 min (about 18 min before the merge) and was killed.
  typeinfer cost looks superlinear in function size or local count. Logged with repro commits
  in `docs/letgo-upstream-candidates.md` as a likely let-go bug.
- **decision:** `tools/aot-lower.lg` takes `AOT_SKIP=compose!`. The AOT binary lowers
  everything else in gfx and world (`sample`, `glow!`, `ball!`, `shape-disc!`, `rect!`,
  colour helpers), and `compose!` stays on the VM. That keeps most of the win (sampling and
  sprites were 2.5–3×) without the hour-long build.
