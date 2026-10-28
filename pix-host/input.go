package main

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/nooga/let-go/pkg/vm"
)

// Window input for lg, in input.lg's own terms: (pix/events) drains
// [key event] pairs gathered by Update since the last call, with real
// releases, so input.lg holds a key for exactly as long as it is down (as it
// does for kitty). Mouse motion over the arena arrives as [:mouse-wx x], x in
// world units (0..48), and a left click as [:click :press].

var (
	inMu     sync.Mutex
	inQueue  []vm.Value
	keys     []ebiten.Key
	lastCurX = -1
	lastCurY = -1

	// where Draw last put the arena, for mapping the cursor onto it
	viewMu              sync.Mutex
	viewX, viewY, viewK int
	viewW, viewH        int
)

var keyNames = func() map[ebiten.Key]string {
	m := map[ebiten.Key]string{
		ebiten.KeyArrowLeft: "left", ebiten.KeyArrowRight: "right",
		ebiten.KeyArrowUp: "up", ebiten.KeyArrowDown: "down",
		ebiten.KeySpace: "space", ebiten.KeyEnter: "enter", ebiten.KeyNumpadEnter: "enter",
		ebiten.KeyEscape: "esc", ebiten.KeyTab: "tab", ebiten.KeyBackspace: "backspace",
		ebiten.KeyBracketLeft: "[", ebiten.KeyBracketRight: "]", ebiten.KeyBackslash: "\\",
		ebiten.KeyMinus: "-", ebiten.KeyEqual: "=", ebiten.KeyComma: ",",
		ebiten.KeyPeriod: ".", ebiten.KeySlash: "/",
	}
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		m[k] = string(rune('a' + k - ebiten.KeyA))
	}
	for k := ebiten.KeyDigit0; k <= ebiten.KeyDigit9; k++ {
		m[k] = string(rune('0' + k - ebiten.KeyDigit0))
	}
	return m
}()

func pushEvent(k, ev vm.Value) {
	inQueue = append(inQueue, vm.NewArrayVector([]vm.Value{k, ev}))
}

// pollInput runs in Update, once a tick.
func pollInput() {
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	inMu.Lock()
	defer inMu.Unlock()
	injectInput()
	keys = inpututil.AppendJustPressedKeys(keys[:0])
	for _, k := range keys {
		name, ok := keyNames[k]
		if !ok {
			continue
		}
		if ctrl && name == "c" {
			name = "ctrl-c"
		}
		pushEvent(vm.Keyword(name), vm.Keyword("press"))
	}
	keys = inpututil.AppendJustReleasedKeys(keys[:0])
	for _, k := range keys {
		if name, ok := keyNames[k]; ok {
			pushEvent(vm.Keyword(name), vm.Keyword("release"))
			if name == "c" { // a held ctrl-c releases as c; release both
				pushEvent(vm.Keyword("ctrl-c"), vm.Keyword("release"))
			}
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		pushEvent(vm.Keyword("click"), vm.Keyword("press"))
	}
	cx, cy := ebiten.CursorPosition()
	if cx == lastCurX && cy == lastCurY {
		return
	}
	lastCurX, lastCurY = cx, cy
	viewMu.Lock()
	x0, y0, k, w, h := viewX, viewY, viewK, viewW, viewH
	viewMu.Unlock()
	if k == 0 || cy < y0 || cy >= y0+h*k {
		return
	}
	px := float64(cx-x0) / float64(k)
	wx := min(48, max(0, px*48/float64(w)))
	pushEvent(vm.Keyword("mouse-wx"), vm.MakeFloat(wx))
}

func setView(x, y, k, w, h int) {
	viewMu.Lock()
	viewX, viewY, viewK, viewW, viewH = x, y, k, w, h
	viewMu.Unlock()
}

func drainEvents([]vm.Value) (vm.Value, error) {
	inMu.Lock()
	defer inMu.Unlock()
	if len(inQueue) == 0 {
		return vm.NewArrayVector(nil), nil
	}
	out := vm.NewArrayVector(inQueue)
	inQueue = nil
	return out, nil
}

// PIX_INJECT="tick:key:event,..." (e.g. "90:space:press,200:left:press,260:left:release";
// "120:mouse-wx:30" moves the mouse to world x 30)
// queues those events at those Update ticks, as if typed into the window, so
// a headless check can drive the game without the desktop.
var injects = func() map[int][][2]string {
	m := map[int][][2]string{}
	for _, f := range strings.Split(os.Getenv("PIX_INJECT"), ",") {
		p := strings.Split(f, ":")
		if len(p) != 3 {
			continue
		}
		if n, err := strconv.Atoi(p[0]); err == nil {
			m[n] = append(m[n], [2]string{p[1], p[2]})
		}
	}
	return m
}()

var ticks int

func injectInput() {
	ticks++
	for _, e := range injects[ticks] {
		if x, err := strconv.ParseFloat(e[1], 64); err == nil {
			pushEvent(vm.Keyword(e[0]), vm.MakeFloat(x))
			continue
		}
		pushEvent(vm.Keyword(e[0]), vm.Keyword(e[1]))
	}
}
