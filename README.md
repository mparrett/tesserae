# TESSERAE

A lunch-break mosaic breaker for the terminal, written in [let-go](https://github.com/nooga/let-go).

> Above the world hangs the Mosaic: ten thousand living tesserae, each a shard of the
> sky's memory. Then the Grout came. You are the last Lapidary. You carry a Trowel and a
> Spark. Break what is dead. Set what is lost. Turn the sky.

![title](docs/captures/m1-title.png)

## Play

```sh
./play.sh                       # title screen; SPACE to begin, 1-0, - and = to jump to a level
./play.sh --level 3             # start at level 3 (with its story card)
./play.sh --level 3 --skip-card --autoplay --stats   # watch the autopilot, show fps/bytes
./play.sh --glyphs sextant      # ball silhouettes: quadrant (default) | sextant | off
./play.sh --all                 # unlock every level on the title screen (testing)
./play.sh --reset-progress      # forget saved scores, medals and unlocks
LG=/path/to/lg ./play.sh        # use another let-go binary
```

### Progress, unlocks and medals

Progress is saved with let-go's `storage` namespace: one file per key under
`~/.config/let-go/storage/<store>/` (`$XDG_CONFIG_HOME` if set). `play.sh` passes
`-storage-id tesserae`; set `TESS_STORAGE_ID` to use another store. Keys are `best`
(overall best score), `furthest` (the furthest unlocked level id) and `level/<id>`
(best level score, best clear time, best medal, flawless flag), stored as EDN. If
storage fails the game keeps running and the title panel says "progress not saved".
`--autoplay` runs never save.

On the title screen a level key starts only unlocked levels: level 1, plus every
level up to the one after your furthest clear. `--level N` always bypasses locks.
Each level has a par time. A clear at or under par earns gold (●), within 1.5× par
silver, otherwise bronze; a clear without losing a spark adds +500 and a ✦. The clear
tally shows time against par, the medal, sets laid, the bonuses and new bests.

## Editor

```sh
./play.sh --edit levels/custom/mine.edn    # open (or start) a map; no file = levels/custom/untitled-N.edn
./play.sh --map levels/custom/twin-gates.edn   # play a custom map straight away, exit after the clear
```

![editor](docs/captures/v2-editor.png)

The editor draws the map with the game's renderer and theme, so `?` and `,` regions show
packed into tetrominoes exactly as they will play (the file keeps the raw chars). A pulsing
outline marks the cursor; with mirror on, a dim one marks its mirror cell. The panel shows
the palette, the brush, the level settings and the keys.

| keys | |
|---|---|
| ← ↑ → ↓ / H J K L | move the cursor |
| SPACE / ENTER | paint (or finish a rectangle) |
| `. ? , # + * % @ x $ b ^ ~` | pick that brush (`b` = ball start `B`) |
| [ / ] | step through the whole palette |
| C / TAB | next piece colour (I O T S Z J L) / solid ↔ ghost (`?`↔`,`, `I`↔`i`) |
| E | pick the brush from the tile under the cursor |
| F | flood-fill the region under the cursor |
| R | rectangle: R marks a corner, move, R or SPACE fills |
| M | mirror painting across the vertical centre line |
| D | pen: paint while moving |
| U | undo (30 steps) |
| T / G | theme / goal (`:clear :build :lamps :heart`; lamps turns tilt on, heart turns ward on) |
| 1-6 | flags: rotate, twin, tilt, gravity, magnet, ward |
| - / = and 9 / 0 | Spark speed / par time |
| P | play-test; Q in the game comes back with the map unchanged |
| S / Q | save / quit (Q asks again when there are unsaved changes) |

Custom maps are EDN, one key per line and one map row per line:

```clojure
{:name "Twin Gates"
 :theme :teal            ; indigo teal umber felt violet crimson grout
 :goal :clear            ; clear build lamps heart
 :speed 28.8
 :par 120                ; seconds for gold
 :rotate 15.0            ; optional: :rotate s, :twin true :balls 2, :tilt true,
                         ;   :gravity 9.0, :magnet true, :ward true :boundary :open
 :map ["........................"
       ...                ; 24 rows of 24 legend chars (see tesserae/levels.lg)
       "........................"]}
```

Loading checks the size, the legend (the Grout Core's `=` and `C` are not allowed), the
theme, the goal and the types, and `--map`/`--edit` stop with a message on a bad file.
Custom runs never touch saved progress or medals: the tally shows the medal against the
map's par, nothing is stored. Play-test refuses a map with nothing to play for, or a tilt
or ward map without a `B` start. Two examples ship in `levels/custom/`, both made in the
editor.

## Browser build

TESSERAE also runs in a browser tab, built with let-go's WASM target (`lg -w`) and
shown in xterm.js.

```sh
./tools/serve-web.sh            # build dist/web, serve http://127.0.0.1:8360/
NO_BUILD=1 ./tools/serve-web.sh 8361   # serve the existing build on another port
./tools/build-web.sh [outdir]   # build only (default dist/web, about 1 min, behind tools/heavy.sh)
node tools/web-shot.mjs http://127.0.0.1:8360/ /tmp/shot 6   # headless check: screenshots + fps
```

![browser](docs/captures/v2-web-play.png)

- `dist/web/index.html` is self-contained (about 8.5 MB, the program wasm is inlined)
  apart from xterm.js, which loads from jsDelivr. The shell is `web/shell.html`, an
  `-w-shell` template that picks the font size so the grid reaches 124x50 (the doubled
  raster) when the window allows it, and uses xterm's WebGL renderer.
- The page must be served cross-origin isolated (COOP/COEP headers). Otherwise the first
  key read fails with "no SharedArrayBuffer". `tools/coi-serve.py` does this;
  `python -m http.server` does not.
- No command line in a browser, so play.sh's flags come from the URL:
  `?level=3&skip-card&autoplay&stats&glyphs=sextant&fps=30&all`. The shell adds
  `?grid=small`, which skips the doubled raster: a quarter of the pixels, roughly twice
  the frame rate.
- Progress isn't saved in the browser yet. let-go's browser storage needs
  `localStorage`, and the VM runs in a Web Worker, which has none, so the title panel
  says "progress not saved".
- Ctrl-C and Q on the title screen don't quit, because a tab has nothing to quit to.
  Resizing the window refits the grid, and the game re-lays out within half a second.

## Tests

```sh
./tools/test.sh                 # clojure.test suites in test/; exits 1 on any failure
```

They cover `tesserae.input/parse` (plain keys, CSI/SS3 arrows, kitty CSI-u
press/repeat/release, BEL resize, Ctrl-C, mouse maps), tetromino packing determinism,
level map validity (24×24, goal count > 0, a par per level), the meta layer (medals,
unlocks, a storage round trip in the throwaway store `tesserae-test`) and custom map files
(EDN round trip, validation, the shipped examples).

You need a truecolor terminal at least 76x24. 124x50 or larger doubles the raster. Kitty,
Ghostty, WezTerm, foot and recent iTerm2 send real key-release events, so held keys feel
exact there. Elsewhere a short-hold model rides the key repeat. Mouse movement steers the
trowel in any terminal that reports motion.

| keys | |
|---|---|
| ← → / A D / mouse | move the trowel |
| SPACE / click | launch; magnet pulse on Lodestone; level the table on Tilt |
| ← ↑ → ↓ | tilt the table (Tilt) / move your borrowed tile (Heart) |
| TAB | take another tile (Heart, costs 25) |
| P | pause, R restart level, Q back to the menu, Ctrl-C quit |
| ] / [ | debug: win the level / lose a spark |

## The twelve levels

| | level | what's new |
|---|---|---|
| I · The Chipping | 1.1 First Light | Bricks are packed tetrominoes; clearing a whole piece is a SET bonus |
| | 1.2 Mitosis | Cell tiles split the Spark; 2-hit hard tiles; a glass pickup (the Spark shatters into three on the trowel) |
| II · The Turning | 2.1 Quarter Turn | The whole field rotates 90° every 15 s while play continues; your trowel stays put |
| | 2.2 Twin Trowels | Top and bottom trowels move together, two open edges, two balls, rotation |
| III · The Setting | 3.1 Blueprint | **Reverse breakout**: ghost tiles set solid (gold-rimmed) when struck; build the crown |
| | 3.2 Grout Creep | Reverse mode where set tiles periodically crumble back to ghosts |
| | 3.3 Mortar Well | Breakout × Tetris: mortar tetrominoes fall down a steel well; strike a piece's side to shove it, from below to turn it; full courses clear. Lay five |
| IV · The Tilt | 4.1 Tilt Table | No paddle: tilt a labyrinth to roll a marble to every lamp; pits swallow it |
| | 4.2 Marble Run | Three marbles, breakable soft walls |
| V · Lodestone | 5.1 Lodestone | Gravity, magnetic tiles that bend the Spark, a magnet trowel (catch / aim / release), rubber, lead and ghost balls (ghost slips through three tiles) |
| VI · The Heart | 6.1 The Heart | The Spark starts walled in; no paddle; you steer one tile the game lends you, and it moves on every 8 s |
| | 6.2 The Grout Core | Finale, after Yars' Revenge: a pulsing core (6 hits) behind a scrolling, self-mending shield; a rainbow neutral zone that spins the Spark; a homing Destroyer that stuns the trowel; a Swirl to dodge every 12 s |

## How it renders

Every terminal cell is two square *pixels* (`▀` with fg = top, bg = bottom), so the arena is a
true square raster that can be sampled through any rotation. Tiles and balls live in a 48×48
world; the screen samples it through the current field angle. That is why the quarter turns
are real rotations and not swaps.

Per frame (`tesserae/gfx.lg`):

1. The static layer (tiles, background, pegboard) sits in a posterized world-pixel cache. Only
   tiles that change are re-sampled.
2. Balls, trails, particles and lodestone halos add light to a screen-space accumulation
   buffer and mark their cells dirty. The trowel writes a solid overlay with horizontal
   anti-aliasing. Ball cores are sub-cell *glyph sprites*: the disc's coverage of the cell's
   2×2 quadrant or 2×3 sextant subcells is thresholded, and the mask is the glyph index
   (fg = ball light, bg = the cell's darker pixel). Sextants (U+1FB00) need a font or
   terminal that draws them (Ghostty, kitty, WezTerm, foot); `--glyphs off` is the plain
   light-buffer ball.
3. `compose!` walks dirty rows and cells only, adds light, posterizes to 6 bits per channel
   before diffing, and compares against what the terminal already shows. It emits changed
   cells with SGR state tracking, contiguous-run cursor elision and cached SGR strings, all
   inside one DEC 2026 synchronized frame.

Steady play costs about 60–150 changed cells, 2–4 KB, per frame at 50 fps. A full-field redraw
(rotation, level start) is the expensive case; see `docs/PLAN.md` for the AOT path.

Physics runs at a fixed 240 Hz substep: axis-separated tile collisions, paddle bounces
computed in the screen frame (so the field can turn under a moving ball), screen-frame
gravity and tilt, and inverse-square lodestones.

## Layout

```
main.lg              arg parsing, entry
tesserae/gfx.lg      square-pixel raster, rotation, light buffer, diffed emitter
tesserae/world.lg    24x24 tile grid (flat arrays), tetromino packer, pixel sampler
tesserae/input.lg    kitty key protocol, repeat-hold fallback, SGR mouse motion
tesserae/levels.lg   story, chapters, maps (ASCII or generator fns), themes
tesserae/game.lg     rules, physics, modes, HUD, cards, clear tally, frame loop
tesserae/meta.lg     saved progress (let-go storage), unlocks, par medals
tesserae/custom.lg   custom map EDN format: validate, read, write, -> level
tesserae/editor.lg   the level editor (--edit), play-test through game.lg's custom hooks
levels/custom/       custom maps (EDN)
test/                clojure.test suites, run by tools/test.sh
tools/               capture (tmux -> PNG/GIF), level runner, wait-for
docs/PLAN.md         phases and what's next
docs/captures/       milestone screenshots
```
