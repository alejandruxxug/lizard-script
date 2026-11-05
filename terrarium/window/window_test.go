package window

import (
	"bytes"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/engine/vm"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/terrarium"
	"github.com/alejandruxxug/lizard-script/terrarium/pen"
	"math"
	"strings"
	"testing"
)

// No test opens a window: run goes through fakeLoop instead, which plays
// the frames of events it is given.

func init() {
	evaluator.Output = &bytes.Buffer{}
}

// frames are the events fakeLoop plays, one list per frame.
var frames [][]Event

// fakeLoop plays frames and reports how many ran, through played.
var played int

func fakeLoop(w *Window, frame func(events []Event) bool) error {
	played = 0
	for _, events := range frames {
		played++
		if !frame(events) {
			return nil
		}
	}
	return nil
}

// engines are the two ways a program runs: the tree-walker and the VM.
var engines = []struct {
	name string
	run  func(program *ast.Program, env *object.Environment) object.Object
}{
	{"evaluator", func(p *ast.Program, env *object.Environment) object.Object { return evaluator.Eval(p, env) }},
	{"vm", vm.Run},
}

// runProgram runs source with the whole Terrarium, the window included,
// on the tree-walker.
func runProgram(t *testing.T, source string, events [][]Event) object.Object {
	t.Helper()
	return runOn(t, engines[0].run, source, events)
}

// runOn runs source like runProgram, on the given engine.
func runOn(t *testing.T, engine func(*ast.Program, *object.Environment) object.Object, source string, events [][]Event) object.Object {
	t.Helper()
	frames, ran, loop = events, false, fakeLoop
	defer func() { loop = ebitenLoop }()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := object.NewEnvironment()
	terrarium.Install(env)
	return engine(program, env)
}

func TestWindow(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"open", `window.open(60, 40, "liz")`, "<window 60x40>"},
		{"type", `type(window.open(1, 1, ""))`, "NATIVE"},
		{"starts black", `let w = window.open(2, 2, ""); img.blit(img.new(2, 2), w, (0, 0)); window.size(w)`, "(2, 2)"},
		{"draws like an image", `let w = window.open(3, 1, ""); window.clear(w, #ffffff); window.fill(w, [1, 0, 1, 1], #ff0000); window.span(w, 0, 2, 3, #00ff00); let m = img.new(3, 1); img.blit(m, w, (0, 0)); img.pixels(m)`,
			"[#ffffff, #ff0000, #00ff00]"},
		{"img functions take a window", `let w = window.open(2, 1, ""); img.spans(w, [[0, 0, 1]], #0000ff); img.size(w)`, "(2, 1)"},
		{"blit an image", `let w = window.open(2, 1, ""); window.blit(w, img.new(1, 1, #ff0000), (1, 0)); let m = img.new(2, 1); img.blit(m, w, (0, 0)); img.pixels(m)`,
			"[#000000, #ff0000]"},
		{"options", `window.open(10, 10, "", {"scale": 2, "resizable": true, "frameless": false, "fps": 30})`, "<window 10x10>"},
		{"title", `window.title(window.open(1, 1, "a"), "b")`, "null"},
		{"pen", `[keys(pen), type(pen.available())]`, `[["available", "curve"], "BOOLEAN"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := runProgram(t, tc.source, nil).Inspect(); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestWindowErrors(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"too small", `window.open(0, 10, "")`, "the width must be from 1 to 8192, got 0"},
		{"title a number", `window.open(1, 1, 5)`, "want a STRING title, got INTEGER"},
		{"options not a hash", `window.open(1, 1, "", 2)`, "want a HASH options, got INTEGER"},
		{"unknown option", `window.open(1, 1, "", {"size": 2})`, `unknown option "size" (the options are automation, floating, fps, frameless, headless, hover, icon, min_size, on_demand, resizable, scale, transparent)`},
		{"bad scale", `window.open(1, 1, "", {"scale": 9})`, "the scale must be from 1 to 8, got 9"},
		{"bad resizable", `window.open(1, 1, "", {"resizable": 1})`, "want a BOOLEAN resizable, got INTEGER"},
		{"run a non window", `window.run(img.new(1, 1), function(e) {})`, "window.run: want a window, got an image"},
		{"run a non function", `window.run(window.open(1, 1, ""), 5)`, "want a FUNCTION tick, got INTEGER"},
		{"draw on a number", `window.fill(1, [0, 0, 1, 1], #ffffff)`, "want an image or a window as the target, got INTEGER"},
		{"only one window", `let w = window.open(1, 1, ""); window.run(w, function(e) { false }); window.run(w, function(e) { false })`,
			"a program can show only one window"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runProgram(t, tc.source, nil)
			err, ok := got.(*object.Error)
			if !ok {
				t.Fatalf("got %s, want an error with %q", got.Inspect(), tc.want)
			}
			if !strings.Contains(err.Message, tc.want) {
				t.Fatalf("error %q doesn't contain %q", err.Message, tc.want)
			}
		})
	}
}

func TestRunCallsTickEveryFrame(t *testing.T) {
	events := [][]Event{
		nil,
		{{Type: "down", X: 3, Y: 4, Button: "left", Pen: Pen{Pressure: 0.5, Source: "tablet"}}},
		{{Type: "key", Key: "Escape"}},
		{{Type: "move"}},
	}
	source := `
let w = window.open(10, 10, "")
let seen = []
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    push(seen, e.type)
    if (e.type == "down") {
      push(seen, [e.pos, e.button, e.pen.pressure, e.pen.source])
    }
    if (e.type == "key" && e.key == "Escape") {
      return false
    }
  }
  push(seen, "frame")
})
seen`
	want := `["frame", "down", [(3, 4), "left", 0.5, "tablet"], "frame", "key"]`
	for _, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			if got := runOn(t, engine.run, source, events).Inspect(); got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
			if played != 3 {
				t.Fatalf("played %d frames, want 3 (the run stops at false)", played)
			}
		})
	}
}

// An error in tick reads like an error in list.sort's comparison: the
// place of the call to window.run, then the tick's own error.
func TestRunStopsOnAnError(t *testing.T) {
	source := "window.run(window.open(1, 1, \"\"), function(events) {\n  1 + \"a\"\n})"
	want := "[line 1, column 11] runtime error: window.run: [line 2, column 5] runtime error: type mismatch: INTEGER + STRING"
	for _, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			got := runOn(t, engine.run, source, [][]Event{nil, nil})
			err, ok := got.(*object.Error)
			if !ok || err.Message != want {
				t.Fatalf("got %s, want %s", got.Inspect(), want)
			}
			if played != 1 {
				t.Fatalf("played %d frames after the error, want 1", played)
			}
		})
	}
}

func TestRunTakesEveryKindOfFunction(t *testing.T) {
	source := `
let frames = 0
function tick(events, limit = 2) {
  frames += 1
  frames < limit
}
window.run(window.open(1, 1, ""), tick)
frames`
	for _, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			if got := runOn(t, engine.run, source, [][]Event{nil, nil, nil}).Inspect(); got != "2" {
				t.Fatalf("got %s frames, want 2", got)
			}
		})
	}
}

func TestEventHash(t *testing.T) {
	mods := Mods{Shift: true}
	for _, tc := range []struct {
		event Event
		want  string
	}{
		{Event{Type: "down", X: 1, Y: 2, Button: "right", Mods: mods, Pen: Pen{Pressure: 1, Source: "mouse"}},
			`{"type": "down", "pos": (1, 2), "button": "right", "mods": {"shift": true, "alt": false, "cmd": false, "ctrl": false}, "pen": {"pressure": 1.0, "raw": 0.0, "source": "mouse", "tilt": (0.0, 0.0), "eraser": false}}`},
		{Event{Type: "scroll", X: 5, Y: 6, DY: -1.5},
			`{"type": "scroll", "pos": (5, 6), "dy": -1.5, "dx": 0.0, "mods": {"shift": false, "alt": false, "cmd": false, "ctrl": false}}`},
		{Event{Type: "key", Key: "A"}, `{"type": "key", "key": "A", "mods": {"shift": false, "alt": false, "cmd": false, "ctrl": false}}`},
		{Event{Type: "keyup", Key: "Space"}, `{"type": "keyup", "key": "Space", "mods": {"shift": false, "alt": false, "cmd": false, "ctrl": false}}`},
		{Event{Type: "pinch", X: 3, Y: 4, Scale: 1.25},
			`{"type": "pinch", "pos": (3, 4), "scale": 1.25, "mods": {"shift": false, "alt": false, "cmd": false, "ctrl": false}}`},
		{Event{Type: "type", Text: "é"}, `{"type": "type", "text": "é", "mods": {"shift": false, "alt": false, "cmd": false, "ctrl": false}}`},
		{Event{Type: "resize", W: 30, H: 20}, `{"type": "resize", "size": (30, 20)}`},
		{Event{Type: "close"}, `{"type": "close"}`},
	} {
		if got := tc.event.hash().Inspect(); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.event.Type, got, tc.want)
		}
	}
}

func TestResizeKeepsPixels(t *testing.T) {
	got := runProgram(t, `
let w = window.open(2, 1, "")
window.clear(w, #ffffff)
w`, nil)
	win := got.(*object.Native).Value.(*Window)
	win.changed = false
	win.resize(3, 2)
	if win.buf.W != 3 || win.buf.H != 2 || !win.changed {
		t.Fatalf("resize gave %dx%d, changed %v", win.buf.W, win.buf.H, win.changed)
	}
	if c := win.buf.At(1, 0); c.R != 255 || c.A != 255 {
		t.Errorf("the old pixel is %v, want white", c)
	}
	if c := win.buf.At(2, 1); c.A != 0 {
		t.Errorf("the new pixel is %v, want transparent", c)
	}
}

func TestPenAt(t *testing.T) {
	// The default trackpad curve is {0.8, 1, 1.5}: 0.7 is below it, 0.9 is
	// halfway, and halfway to the power of 1.5 is 0.3536.
	native := []pen.Sample{
		{Phase: "drag", Pressure: 0.9, Source: "trackpad"},
		{Phase: "drag", Pressure: 0.7, Source: "trackpad"},
	}
	if p := penAt(native, "drag", true); p.Raw != 0.7 || p.Pressure != 0 || p.Source != "trackpad" {
		t.Errorf("drag pen = %+v, want the newest sample, curved", p)
	}
	if p := penAt(native[:1], "drag", true); p.Raw != 0.9 || math.Abs(p.Pressure-0.3536) > 1e-4 {
		t.Errorf("drag pen = %+v, want raw 0.9 and pressure 0.3536", p)
	}
	if p := penAt(nil, "down", false); p.Pressure != 1 || p.Raw != 1 || p.Source != "mouse" {
		t.Errorf("no samples gave %+v, want a mouse", p)
	}
}

// TestPathsOfNoDrop checks the frames where nothing was dropped: Ebitengine
// gives a nil file system then, and reading it must not panic.
func TestPathsOfNoDrop(t *testing.T) {
	if got := pathsOf(nil); got != nil {
		t.Fatalf("pathsOf(nil) = %v, want nil", got)
	}
}

func TestCursor(t *testing.T) {
	for _, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			got := runOn(t, engine.run, `let w = window.open(4, 4, "", {"headless": true})
window.cursor(w, "grab")
w`, nil)
			if c := got.(*object.Native).Value.(*Window).cursor; c != "grab" {
				t.Errorf("cursor = %q, want grab", c)
			}
			for name := range cursorShapes {
				if !contains(cursorNames, name) {
					t.Errorf("%s is shaped but not accepted", name)
				}
			}
			if len(cursorShapes) != len(cursorNames) {
				t.Errorf("%d shapes for %d names", len(cursorShapes), len(cursorNames))
			}
		})
	}
	for _, src := range []string{`window.cursor(window.open(4, 4, "", {"headless": true}), "hand")`, `window.cursor(1, "text")`} {
		if _, ok := runProgram(t, src, nil).(*object.Error); !ok {
			t.Errorf("%s: want an error", src)
		}
	}
}

func TestPinchSum(t *testing.T) {
	drainPinch()
	if _, ok := drainPinch(); ok {
		t.Fatal("a pinch with no gesture")
	}
	recordPinch(0.5)
	recordPinch(-0.5)
	if s, ok := drainPinch(); !ok || s != 0.75 {
		t.Errorf("pinch = %v %v, want 0.75", s, ok)
	}
}
