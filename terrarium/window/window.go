// Package window is the window module of the Terrarium: a window with a
// pixel buffer the program draws on, and the input it gets.
//
// Ebitengine is used only to open the window, read its input and show the
// finished buffer; every pixel is drawn by the program, with the drawing
// functions every target shares (package canvas). Importing this package
// adds window and pen to terrarium.Install:
//
//	import _ "lizardscript/terrarium/window"
package window

import (
	"image/color"
	"lizardscript/evaluator"
	"lizardscript/object"
	"lizardscript/terrarium"
	"lizardscript/terrarium/internal/arg"
	"lizardscript/terrarium/internal/canvas"
	"lizardscript/terrarium/internal/pixels"
	"lizardscript/terrarium/pen"
	"lizardscript/token"
	"strings"
)

func init() {
	terrarium.Register(func(env *object.Environment) {
		env.Define("window", Module())
		env.Define("pen", pen.Module())
	})
}

// Kind is the kind of the natives that are windows.
const Kind = "window"

// maxSide is the biggest width or height of a window's buffer.
const maxSide = 8192

// black is the color of a new window.
var black = color.NRGBA{A: 255}

// Window is a window and the buffer it shows. Its size is the buffer's, in
// pixels; with a scale of 2 each one is shown as 2x2 on the screen.
type Window struct {
	buf       *pixels.Image
	title     string
	scale     int
	fps       int
	resizable bool
	frameless bool

	changed      bool // drawn on since it was last shown
	titleChanged bool // title changed while it runs
}

// Pixels and Touched make a window a drawing target.
func (w *Window) Pixels() *pixels.Image { return w.buf }
func (w *Window) Touched()              { w.changed = true }

// Describe is the size, for Inspect: <window 600x400>.
func (w *Window) Describe() string { return w.buf.Describe() }

// resize gives the window a buffer of the new size, keeping the pixels
// that still fit.
func (w *Window) resize(width, height int) {
	if bigger, err := w.buf.Resized(width, height); err == nil {
		w.buf = bigger
		w.changed = true
	}
}

// Module is the window module: open, run and title, and the shared drawing
// functions size, clear, fill, span, spans and blit.
func Module() *object.Hash {
	return arg.Module("window", canvas.Add(map[string]object.BuiltinFunction{
		"open":  open,
		"run":   run,
		"title": title,
	}))
}

// windowArg reads argument i as a window.
func windowArg(args []object.Object, i int) (*Window, *object.Error) {
	v, err := arg.Native(args, i, Kind)
	if err != nil {
		return nil, err
	}
	return v.(*Window), nil
}

// window.open(w, h, title, options = {}) makes a window of w by h pixels.
// It shows on the screen when window.run starts, so the program can draw
// its first picture before. The options are scale (1 to 8, default 1),
// resizable and frameless (default false) and fps, the frames a second
// (1 to 240, default 60).
func open(args ...object.Object) object.Object {
	if err := arg.Count(args, 3, 4); err != nil {
		return err
	}
	width, err := arg.IntIn(args, 0, "width", 1, maxSide)
	if err != nil {
		return err
	}
	height, err := arg.IntIn(args, 1, "height", 1, maxSide)
	if err != nil {
		return err
	}
	name, err := arg.String(args, 2, "title")
	if err != nil {
		return err
	}
	buf, berr := pixels.New(width, height, black)
	if berr != nil {
		return arg.Errorf("%v", berr)
	}
	w := &Window{buf: buf, title: name, scale: 1, fps: 60, changed: true}
	if len(args) == 4 {
		options, err := arg.Hash(args, 3, "options")
		if err != nil {
			return err
		}
		if err := w.setOptions(options); err != nil {
			return err
		}
	}
	return &object.Native{Kind: Kind, Value: w}
}

// optionNames are the keys window.open takes in its options.
var optionNames = []string{"fps", "frameless", "resizable", "scale"}

// setOptions reads the options of window.open.
func (w *Window) setOptions(options *object.Hash) *object.Error {
	var err *object.Error
	options.Each(func(pair object.HashPair) {
		if err != nil {
			return
		}
		value := []object.Object{pair.Value}
		switch name := pair.Key.Inspect(); name {
		case "scale":
			w.scale, err = arg.IntIn(value, 0, "scale", 1, 8)
		case "fps":
			w.fps, err = arg.IntIn(value, 0, "fps", 1, 240)
		case "resizable":
			w.resizable, err = boolOption(value, name)
		case "frameless":
			w.frameless, err = boolOption(value, name)
		default:
			err = arg.Errorf("unknown option %q (the options are %s)", name, strings.Join(optionNames, ", "))
		}
	})
	return err
}

// boolOption reads an option that is true or false.
func boolOption(value []object.Object, name string) (bool, *object.Error) {
	b, ok := value[0].(*object.Boolean)
	if !ok {
		return false, arg.Errorf("want a BOOLEAN %s, got %s", name, arg.Name(value[0]))
	}
	return b.Value, nil
}

// window.title(w, text) changes the window's title.
func title(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	if w.title, err = arg.String(args, 1, "title"); err != nil {
		return err
	}
	w.titleChanged = true
	return arg.Null
}

// loop shows w and calls frame once a frame with the events since the
// last one, until frame returns false or the window closes. Tests replace
// it with one that opens no window.
var loop = ebitenLoop

// ran is set once a window has run: Ebitengine can open only one window in
// the life of a program.
var ran bool

// window.run(w, tick) shows the window and calls tick(events) every frame,
// with the list of events since the last frame (often empty), until tick
// returns false or the window is closed. Closing it sends one last "close"
// event first. An error in tick stops the window and is run's error.
func run(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	tick := args[1]
	if tick.Type() != object.FUNCTION_OBJ && tick.Type() != object.BUILTIN_OBJ {
		return arg.Errorf("want a FUNCTION tick, got %s", arg.Name(tick))
	}
	if ran {
		return arg.Errorf("a program can show only one window")
	}
	ran = true

	var failed *object.Error
	loopErr := loop(w, func(events []Event) bool {
		result := call(tick, events)
		if e, ok := result.(*object.Error); ok {
			failed = e
			return false
		}
		return result != arg.False
	})
	if failed != nil {
		return failed
	}
	if loopErr != nil {
		return arg.Errorf("the window failed: %v", loopErr)
	}
	return arg.Null
}

// call runs tick with the events as a list of hashes, the way list.sort
// calls its comparison: with the checks of a call written in the program,
// for a function of either engine.
func call(tick object.Object, events []Event) object.Object {
	list := &object.Array{Elements: make([]object.Object, len(events))}
	for i, e := range events {
		list.Elements[i] = e.hash()
	}
	return evaluator.ApplyFunction(token.Token{}, tick, []object.Object{list})
}

// Mods are the modifier keys held during an event.
type Mods struct{ Shift, Alt, Cmd, Ctrl bool }

// Pen is what the pointer knows besides its position.
type Pen struct {
	Pressure float64    // 0 to 1; 1 for a mouse
	Tilt     [2]float64 // tablet tilt, -1 to 1 on each axis
	Source   string     // "mouse", "trackpad" or "tablet"
	Eraser   bool       // the eraser end of a tablet pen
}

// Event is one input event. X and Y are buffer pixels.
type Event struct {
	Type   string // down, up, move, scroll, key, type, resize or close
	X, Y   int    // the pointer, for down, up, move and scroll
	Button string // down and up: "left", "right" or "middle"
	Key    string // key: the key's name, like "A", "Escape" or "ArrowLeft"
	Text   string // type: the character typed
	DY     float64
	W, H   int // resize: the new size
	Mods   Mods
	Pen    Pen // down, up and move
}

// hash is the event as the program sees it. Only the keys that mean
// something for its type are there.
func (e Event) hash() *object.Hash {
	h := object.NewHash()
	arg.Set(h, "type", arg.Str(e.Type))
	switch e.Type {
	case "down", "up", "move", "scroll":
		arg.Set(h, "pos", arg.PointOf(e.X, e.Y))
	}
	switch e.Type {
	case "down", "up":
		arg.Set(h, "button", arg.Str(e.Button))
	case "key":
		arg.Set(h, "key", arg.Str(e.Key))
	case "type":
		arg.Set(h, "text", arg.Str(e.Text))
	case "scroll":
		arg.Set(h, "dy", arg.Float(e.DY))
	case "resize":
		arg.Set(h, "size", arg.PointOf(e.W, e.H))
	}
	if e.Type != "resize" && e.Type != "close" {
		arg.Set(h, "mods", modsHash(e.Mods))
	}
	switch e.Type {
	case "down", "up", "move":
		arg.Set(h, "pen", penHash(e.Pen))
	}
	return h
}

func modsHash(m Mods) *object.Hash {
	h := object.NewHash()
	arg.Set(h, "shift", arg.Bool(m.Shift))
	arg.Set(h, "alt", arg.Bool(m.Alt))
	arg.Set(h, "cmd", arg.Bool(m.Cmd))
	arg.Set(h, "ctrl", arg.Bool(m.Ctrl))
	return h
}

func penHash(p Pen) *object.Hash {
	h := object.NewHash()
	arg.Set(h, "pressure", arg.Float(p.Pressure))
	arg.Set(h, "source", arg.Str(p.Source))
	arg.Set(h, "tilt", &object.Point{Coords: []object.Object{arg.Float(p.Tilt[0]), arg.Float(p.Tilt[1])}})
	arg.Set(h, "eraser", arg.Bool(p.Eraser))
	return h
}
