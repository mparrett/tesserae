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
- [x] AOT-lowered `lg` for the raster hot paths (optional binary, `tools/build-aot.sh`)

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
  (switch, crossover, bit) with puzzles solved by tilting. **Done (v2/orrery):** bonus
  chapter VII · THE ORRERY (7.1 Escapement, 7.2 Gearwork, 7.3 Counting House), opened by
  clearing 6.2. Next: more parts (an interrupt that stops the hopper, a trigger that
  releases the next marble), a puzzle that needs both lanes and a gear, editor support.

## Performance notes

- VM cost is about 0.3 µs per simple op. Steady frames: 1–4 ms compose for 60–150 dirty
  cells. Full-field compose, rotated: 107 → 71 → 54 ms (VM), 36 ms coarse, ~20 ms coarse
  on the AOT binary. The DEVLOG has the full arc.
- Remaining levers: emitting runs of equal-bg cells as `ECH`, and rebuilding the AOT binary
  after gfx changes.
- let-go gaps found: see `docs/letgo-upstream-candidates.md`.
