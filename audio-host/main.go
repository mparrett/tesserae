// audio-host is the stock lg CLI with undertone's `audio` namespace linked in,
// so tesserae can run unchanged through play.sh (LG=audio-host/lg) and gain
// sound. Everything else, flags included, is let-go's own cli.Main.
package main

import (
	"os"

	"github.com/nooga/let-go/pkg/cli"

	"github.com/mparrett/undertone/letgo"
)

var version, commit = "dev", "9981aa0+undertone"

func main() {
	_ = letgo.CurrentMusic() // the import registers the audio namespace
	os.Exit(cli.Main(version, commit))
}
