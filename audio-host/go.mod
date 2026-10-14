module tesserae/audiohost

go 1.27

// Spike: undertone comes from a local worktree with the arcade shots; drop
// the replace once they land upstream.
replace github.com/mparrett/let-go-undertone => ../../let-go-undertone-tesserae-sfx

require (
	github.com/mparrett/let-go-undertone v0.0.0-20261012230501-54dcaefe21f8
	github.com/nooga/let-go v1.13.1-0.20261009071543-a13e04279138
)

require (
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/ebitengine/gomobile v0.0.0-20260820040257-d11f821a26a6 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/oto/v3 v3.5.1 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/go-gl/gl v0.0.0-20260331235117-4566fea9a276 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20260823155953-d41da22a9587 // indirect
	github.com/hajimehoshi/ebiten/v2 v2.10.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/jfreymuth/pulse v0.1.3 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/zeebo/bencode v1.0.0 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/term v0.46.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
