// pix-host is audio-host plus a window: the stock lg CLI with undertone's
// `audio` namespace and a `pix` namespace that shows frames in an Ebitengine
// window (see pix.go). Spike: run through tools/play-pix.sh.
//
// Ebitengine has to own the main thread, so lg runs on a goroutine and main
// waits: for lg to finish, or for it to open a window, which runs here until
// it closes.
package main

import (
	"os"

	"github.com/nooga/let-go/pkg/cli"

	"github.com/mparrett/undertone/letgo"
)

var version, commit = "dev", "d1cf427+undertone+pix"

func main() {
	_ = letgo.CurrentMusic() // the import registers the audio namespace
	installPix()
	done := make(chan int, 1)
	go func() {
		code := cli.Main(version, commit)
		quit.Store(true) // an open window ends, so main gets back to this select
		done <- code
	}()
	for {
		select {
		case code := <-done:
			writeStats()
			os.Exit(code)
		case w := <-openReq:
			runWindow(w)
		}
	}
}
