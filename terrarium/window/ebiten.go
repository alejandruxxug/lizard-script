package window

import (
	"lizardscript/terrarium/pen"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The Ebitengine side of the window. Only WritePixels shows pixels; none of
// Ebitengine's drawing, text or vector functions are used.

// ebitenLoop opens the window and runs frame once a frame. It must be
// called from the program's main goroutine, which is where a lizard-script
// program runs.
func ebitenLoop(w *Window, frame func(events []Event) bool) error {
	ebiten.SetWindowTitle(w.title)
	ebiten.SetWindowSize(w.buf.W*w.scale, w.buf.H*w.scale)
	ebiten.SetWindowDecorated(!w.frameless)
	if w.resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetTPS(w.fps)
	g := &game{w: w, frame: frame, lastX: -1, lastY: -1}
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		return err
	}
	return nil
}

// game is what Ebitengine runs.
type game struct {
	w     *Window
	frame func(events []Event) bool

	started        bool
	layoutW        int // the buffer size the window has room for
	layoutH        int
	shownW, shownH int    // the size of the screen last written
	premultiplied  []byte // the buffer as the screen takes it
	lastX, lastY   int
}

// Layout makes the screen the size of the buffer. A resizable window's
// buffer follows the window: its size divided by the scale.
func (g *game) Layout(outW, outH int) (int, int) {
	if !g.w.resizable {
		return g.w.buf.W, g.w.buf.H
	}
	g.layoutW, g.layoutH = max(1, outW/g.w.scale), max(1, outH/g.w.scale)
	return g.layoutW, g.layoutH
}

// Update runs one frame of the program.
func (g *game) Update() error {
	if !g.started {
		g.started = true
		pen.Start()
	}
	events := g.poll()
	if g.w.resizable && g.layoutW > 0 && (g.layoutW != g.w.buf.W || g.layoutH != g.w.buf.H) {
		g.w.resize(g.layoutW, g.layoutH)
		events = append(events, Event{Type: "resize", W: g.layoutW, H: g.layoutH})
	}
	closing := ebiten.IsWindowBeingClosed()
	if closing {
		events = append(events, Event{Type: "close"})
	}
	keepGoing := g.frame(events)
	if g.w.titleChanged {
		ebiten.SetWindowTitle(g.w.title)
		g.w.titleChanged = false
	}
	if !keepGoing || closing {
		return ebiten.Termination
	}
	return nil
}

// Draw shows the buffer when it changed, or when the screen is new.
func (g *game) Draw(screen *ebiten.Image) {
	b := screen.Bounds()
	buf := g.w.buf
	if b.Dx() != buf.W || b.Dy() != buf.H {
		return
	}
	if !g.w.changed && b.Dx() == g.shownW && b.Dy() == g.shownH {
		return
	}
	if len(g.premultiplied) != len(buf.Pix) {
		g.premultiplied = make([]byte, len(buf.Pix))
	}
	buf.Premultiplied(g.premultiplied)
	screen.WritePixels(g.premultiplied)
	g.w.changed = false
	g.shownW, g.shownH = b.Dx(), b.Dy()
}

// DrawFinalScreen scales the buffer to the window with sharp pixels, so a
// scale of 2 shows each pixel as a clean 2x2 square.
func (g *game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	screen.DrawImage(offscreen, &ebiten.DrawImageOptions{GeoM: geoM, Filter: ebiten.FilterNearest})
}

var buttons = []struct {
	button ebiten.MouseButton
	name   string
}{
	{ebiten.MouseButtonLeft, "left"},
	{ebiten.MouseButtonRight, "right"},
	{ebiten.MouseButtonMiddle, "middle"},
}

// poll reads the input of this frame as events.
func (g *game) poll() []Event {
	var events []Event
	x, y := ebiten.CursorPosition()
	mods := Mods{
		Shift: ebiten.IsKeyPressed(ebiten.KeyShift),
		Alt:   ebiten.IsKeyPressed(ebiten.KeyAlt),
		Cmd:   ebiten.IsKeyPressed(ebiten.KeyMeta),
		Ctrl:  ebiten.IsKeyPressed(ebiten.KeyControl),
	}
	native := pen.Drain()
	pointer := func(kind, button string, p Pen) {
		events = append(events, Event{Type: kind, X: x, Y: y, Button: button, Mods: mods, Pen: p})
	}

	held := false
	for _, b := range buttons {
		if inpututil.IsMouseButtonJustPressed(b.button) {
			pointer("down", b.name, penAt(native, pen.PhaseDown, true))
		}
		held = held || ebiten.IsMouseButtonPressed(b.button)
	}
	if x != g.lastX || y != g.lastY {
		pointer("move", "", penAt(native, pen.PhaseDrag, held))
	}
	g.lastX, g.lastY = x, y
	for _, b := range buttons {
		if inpututil.IsMouseButtonJustReleased(b.button) {
			pointer("up", b.name, penAt(native, pen.PhaseUp, false))
		}
	}
	if _, dy := ebiten.Wheel(); dy != 0 {
		events = append(events, Event{Type: "scroll", X: x, Y: y, DY: dy, Mods: mods})
	}

	for _, k := range inpututil.AppendPressedKeys(nil) {
		if g.repeats(k) {
			events = append(events, Event{Type: "key", Key: k.String(), Mods: mods})
		}
	}
	// The text as the system composed it (dead keys, option combinations).
	for _, r := range ebiten.AppendInputChars(nil) {
		if r >= 32 && r != 127 {
			events = append(events, Event{Type: "type", Text: string(r), Mods: mods})
		}
	}
	return events
}

// repeats reports whether a held key sends an event this frame: when it
// goes down, then after half a second about 15 times a second.
func (g *game) repeats(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	delay, every := max(1, g.w.fps/2), max(1, g.w.fps/15)
	return d == 1 || (d >= delay && (d-delay)%every == 0)
}

// penAt is the pen of a pointer event: the newest native sample of the
// phase when there is one, else the latest reading while a button is held,
// else a plain mouse.
func penAt(native []pen.Sample, phase string, held bool) Pen {
	latest := pen.Latest()
	for i := len(native) - 1; i >= 0; i-- {
		if s := native[i]; s.Phase == phase && s.Source != "mouse" {
			return newPen(s.Pressure, s.Tilt, s.Source, latest.Eraser)
		}
	}
	if held && latest.OK && latest.Source != "" && latest.Source != "mouse" {
		return newPen(latest.Pressure, latest.Tilt, latest.Source, latest.Eraser)
	}
	return newPen(1, [2]float64{}, "mouse", false)
}

// newPen is a pen with the device's response curve applied to the raw
// pressure.
func newPen(raw float64, tilt [2]float64, source string, eraser bool) Pen {
	return Pen{Pressure: pen.Shape(source, raw), Raw: raw, Tilt: tilt, Source: source, Eraser: eraser}
}
