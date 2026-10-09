# TESSERAE — phases

Started 2026-10-09 on let-go upstream `a13e042`.

## v0: a playable slice across the whole brief (this phase)

Goal: every headline idea is playable in at least one level, inside a story frame, on a
renderer good enough to show off the rotation.

- [x] Square-pixel raster engine: light accumulation, overlay, text layer, rotation-aware
      sampling, posterize-before-diff, SGR-tracked diff emitter, synchronized frames
- [x] World-pixel colour cache, snapped rest angles (full redraws ~2× cheaper)
- [x] Input: kitty press/repeat/release, repeat-hold fallback, SGR any-motion mouse
- [x] Tetromino-packed brick walls (seeded), piece-joined bevels, SET bonus per piece
- [x] Multiple balls, mitosis tiles, hard tiles, combo multiplier
- [x] 90° field rotation during play (warning, eased 1.2 s turn, open edge follows the turn)
- [x] Twin trowels (two open edges)
- [x] Reverse mode: ghost blueprint tiles set on hit; Grout Creep decay variant
- [x] Tilt labyrinth: no paddle, gravity from tilt, pits, lamps, multiple marbles
- [x] Gravity, lodestone tiles, magnet trowel (catch / release), rubber and lead balls
- [x] The Heart: ball starts walled in, no paddle, a borrowed tile the game picks (TAB to choose another)
- [x] Story: prologue, chapter cards (typewriter), epilogue; title attract mode
- [x] Capture tooling: tmux → PNG/GIF, level runner, autopilot
- [ ] AOT-lowered `lg` for the raster hot paths (background spike; see below)

## v1: depth and feel

- **Yars-style boss** ("the Qotile of the Grout"): a core behind a scrolling, regenerating
  shield of cells, a shimmering neutral-zone band where the Spark's colour and spin
  scramble, and a rotation every few seconds.
- **Shape-aware sprite glyphs** (shapescii lineage): balls and particles drawn with quadrant
  or sextant glyphs chosen by a 2×3 coverage bitmask, giving four to six times the effective
  resolution on hard-edged sprites. A table lookup, no codebook; gate sextants on font
  support.
- More reverse-mode puzzles: blueprints with line clears (Tetris rows) and a falling
  "mortar" piece you steer into place.
- Ball materials: glass (shatters into three on the trowel), ghost (passes through one tile),
  sticky.
- Persistent best scores and an unlocked-level list via let-go `storage`.
- Difficulty pass: per-level par times, a 3-star medal, a lives tuning pass after
  playtesting.
- Sound (OSC bell patterns or the xsofy audio module in the browser build).

## v2: platforms and reach

- WASM/xterm.js build (`lg -w`) via the joint-xsofy shell, for a shareable URL. Needs
  `key-pending?` polling, which the loop already uses.
- A level editor that writes the ASCII map format.
- An Orrery mode in the spirit of Turing Tumble: marble logic gates built from tiles
  (switch, crossover, bit) with puzzles solved by tilting.

## Performance notes

- VM cost is about 0.3 µs per simple op. Steady frames: 1–4 ms compose for 60–150 dirty
  cells. Full-field compose (rotation transitions, level start) was 107 ms. With the
  world-pixel cache it is about 2× cheaper (measured relative to a same-run baseline on a
  loaded box; re-time on a quiet box).
- Next levers, in order: AOT-lowered gfx/world namespaces (let-go `benchmark/aot`
  pipeline), a coarser rotation raster (sample at k=1 and double), and emitting runs of
  equal bg cells as `ECH`.
- let-go gaps found (upstream candidates): `bit-or`/`bit-and` take exactly 2 args (Clojure is
  variadic); `Math/PI`, `Math/sin` and `Math/cos` are unresolved while `Math/sqrt` and
  `Math/abs` resolve (use `math/*`); `(str sb)` on a StringBuilder prints the object
  (use `.toString`); the SGR mouse decoder drops the motion bit (1003 motion arrives as
  `:button :none` presses).
