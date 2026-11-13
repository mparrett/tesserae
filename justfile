# Local stand-in for CI: GitHub Actions only builds and deploys the Pages site
# (.github/workflows/pages.yml), so `just ci` is the check to run before merging. The tests,
# sim and stock web build use tools/lg-pinned, a stock lg that tools/build-lg.sh builds from
# the let-go version audio-host/go.mod pins (gitignored). The Pages build compiles against the
# same go.mod, so the tests run the let-go that ships. LG=/path/to/lg runs another let-go.

default:
    @just --list

# Everything a merge should pass: tests, a simulator pass, the web build.
ci: lg test sim web

# Point tools/lg-pinned at audio-host/go.mod's let-go, building it on first use.
[private]
lg:
    @[ -n "${LG:-}" ] || tools/build-lg.sh

# The test suites (test/*.lg); args pass through to test/run.lg.
test *args: lg
    tools/test.sh {{args}}

# Every level for a few game-seconds under the autopilot. The tests don't
# load tools/sim.lg, so this is what catches it falling behind the game.
sim secs="5": lg
    "${LG:-./tools/lg-pinned}" -source-paths . tools/sim.lg --secs {{secs}}

# The browser build (compiles main.lg and every namespace it loads).
web out="dist/web": lg
    tools/build-web.sh {{out}}

# The GitHub Pages site (browser build with sound), smoke-tested as Pages serves
# it. undertone and let-go come from audio-host/go.mod; see build-web-onevm.sh.
pages out="_site":
    tools/build-pages.sh {{out}}
    uv run --with=playwright python tools/pages-smoke.py {{out}}
