# Local stand-in for CI: GitHub Actions only builds and deploys the Pages site
# (.github/workflows/pages.yml), so `just ci` is the check to run before merging. The tests,
# sim and stock web build use tools/lg-pinned, which each checkout symlinks to its let-go
# binary (gitignored). Keep it at the let-go commit audio-host/go.mod pins, which is what the
# Pages build compiles against, so the tests run the let-go that ships.

default:
    @just --list

# Everything a merge should pass: tests, a simulator pass, the web build.
ci: lg test sim web

# Fail early, with a hint, when this checkout has no tools/lg-pinned.
[private]
lg:
    @test -x tools/lg-pinned || { echo "tools/lg-pinned is missing: symlink it to a let-go binary (ln -s /path/to/lg tools/lg-pinned)" >&2; exit 1; }

# The test suites (test/*.lg); args pass through to test/run.lg.
test *args: lg
    tools/test.sh {{args}}

# Every level for a few game-seconds under the autopilot. The tests don't
# load tools/sim.lg, so this is what catches it falling behind the game.
sim secs="5": lg
    ./tools/lg-pinned -source-paths . tools/sim.lg --secs {{secs}}

# The browser build (compiles main.lg and every namespace it loads).
web out="dist/web": lg
    tools/build-web.sh {{out}}

# The GitHub Pages site (browser build with sound), smoke-tested as Pages serves
# it. undertone and let-go come from audio-host/go.mod; see build-web-onevm.sh.
pages out="_site":
    tools/build-pages.sh {{out}}
    uv run --with=playwright python tools/pages-smoke.py {{out}}
