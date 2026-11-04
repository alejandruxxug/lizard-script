package window

import (
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/terrarium"
	"github.com/alejandruxxug/lizard-script/terrarium/pen"
	"image"
	"strings"
	"testing"
)

// addSurface gives w the surface name over rect, through window.surface.
func addSurface(t *testing.T, w *Window, name, rect string, scale int) *Surface {
	t.Helper()
	env := object.NewEnvironment()
	env.Define("w", &object.Native{Kind: Kind, Value: w})
	got := runIn(t, env, `window.surface(w, "`+name+`", `+rect+`, `+itoa(scale)+`)`)
	n, ok := got.(*object.Native)
	if !ok {
		t.Fatalf("window.surface gave %s", got.Inspect())
	}
	return n.Value.(*Surface)
}

func itoa(n int) string { return string(rune('0' + n)) }

// runIn runs source in env, with the whole Terrarium, on the tree-walker.
func runIn(t *testing.T, env *object.Environment, source string) object.Object {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	terrarium.Install(env)
	return evaluator.Eval(program, env)
}

func TestSurfaces(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"make", `let w = window.open(10, 10, "", {"scale": 2}); window.surface(w, "canvas", [2, 2, 4, 3])`, "<surface canvas 8x6>"},
		{"the same value", `let w = window.open(10, 10, ""); window.surface(w, "a", [0, 0, 2, 2]) == window.surface(w, "a", [1, 1, 2, 2])`, "true"},
		{"a scale of its own", `let w = window.open(10, 10, "", {"scale": 4}); window.size(window.surface(w, "a", [0, 0, 2, 3], 2))`, "(4, 6)"},
		{"resized keeps pixels", `let w = window.open(10, 10, ""); let s = window.surface(w, "a", [0, 0, 2, 1]); window.clear(s, #ff0000); window.surface(w, "a", [0, 0, 3, 1]); img.blit(img.new(1, 1), s, (0, 0)); let m = img.new(3, 1); img.blit(m, s, (0, 0)); img.pixels(m)`,
			"[#ff0000, #ff0000, #00000000]"},
		{"removed", `let w = window.open(10, 10, ""); window.surface(w, "a", [0, 0, 2, 2]); window.surface(w, "a", [0, 0, 0, 0])`, "null"},
		{"drawn like an image", `let w = window.open(4, 1, ""); let s = window.surface(w, "a", [0, 0, 4, 1]); window.fill(s, [1, 0, 2, 1], #00ff00); img.fill(s, [3, 0, 1, 1], #0000ff); let m = img.new(4, 1); img.blit(m, s, (0, 0)); img.pixels(m)`,
			"[#00000000, #00ff00, #00ff00, #0000ff]"},
		{"screenshot", `let w = window.open(2, 1, "", {"scale": 2}); window.clear(w, #ffffff); let s = window.surface(w, "a", [1, 0, 1, 1], 1); window.fill(s, [0, 0, 1, 2], #ff0000); let shot = window.screenshot(w); [img.size(shot), img.get(shot, 1, 0), img.get(shot, 2, 1), img.get(shot, 3, 1)]`,
			"[(4, 2), #ffffff, #ff0000, #ffffff]"},
		{"screenshot with no surfaces", `let w = window.open(3, 2, "", {"scale": 3}); img.size(window.screenshot(w))`, "(3, 2)"},
		{"full screenshot with no surfaces", `let w = window.open(2, 1, "", {"scale": 3}); window.fill(w, [1, 0, 1, 1], #ff0000); let shot = window.screenshot(w, {"full": true}); [img.size(shot), img.get(shot, 2, 2), img.get(shot, 3, 0), img.get(shot, 5, 2)]`,
			"[(6, 3), #000000, #ff0000, #ff0000]"},
		{"full screenshot with surfaces", `let w = window.open(2, 1, "", {"scale": 2}); let s = window.surface(w, "a", [1, 0, 1, 1], 2); window.fill(s, [0, 0, 1, 1], #00ff00); let a = window.screenshot(w); let b = window.screenshot(w, {"full": true}); [img.size(a), img.size(b), img.get(b, 1, 1), img.get(b, 2, 1), img.get(b, 3, 0)]`,
			"[(2, 1), (4, 2), #000000, #00ff00, #00ff00]"},
		{"full false", `let w = window.open(3, 2, "", {"scale": 3}); img.size(window.screenshot(w, {"full": false}))`, "(3, 2)"},
		{"dirty", `let w = window.open(4, 4, ""); window.dirty(w, [0, 0, 1, 1]); window.dirty(window.surface(w, "a", [0, 0, 1, 1]), [0, 0, 1, 1])`, "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := runProgram(t, tc.source, nil).Inspect(); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestSurfaceErrors(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"bad rect", `window.surface(window.open(1, 1, ""), "a", [0, 0])`, "want a rect [x, y, w, h] of numbers"},
		{"bad scale", `window.surface(window.open(1, 1, ""), "a", [0, 0, 1, 1], 9)`, "the scale must be from 1 to 8, got 9"},
		{"dirty an image", `window.dirty(img.new(1, 1), [0, 0, 1, 1])`, "want a window or a surface, got an image"},
		{"screenshot unknown option", `window.screenshot(window.open(1, 1, ""), {"all": true})`, `unknown option "all", the only option is full`},
		{"screenshot bad full", `window.screenshot(window.open(1, 1, ""), {"full": 1})`, "the option full must be true or false, got 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := runProgram(t, tc.source, nil).(*object.Error)
			if !ok || !strings.Contains(got.Message, tc.want) {
				t.Fatalf("got %v, want an error with %q", got, tc.want)
			}
		})
	}
}

// Only what changed is composed again, each changed rectangle on its own.
func TestComposeOnlyWhatChanged(t *testing.T) {
	w := runProgram(t, `window.open(100, 100, "", {"scale": 2})`, nil).(*object.Native).Value.(*Window)
	s := addSurface(t, w, "canvas", "[10, 10, 50, 50]", 1)
	w.compose()
	if got := w.compose(); len(got) != 0 {
		t.Fatalf("nothing changed but %v was composed", got)
	}

	s.Touched()
	if got := w.compose(); len(got) != 1 || got[0] != image.Rect(20, 20, 120, 120) {
		t.Fatalf("a touched surface composed %v, want its whole place (20,20)-(120,120)", got)
	}

	s.Touched()
	s.dirt.rects = []image.Rectangle{image.Rect(0, 0, 2, 2), image.Rect(90, 90, 100, 100)}
	w.Touched()
	w.screen.base.rects = []image.Rectangle{image.Rect(0, 0, 1, 1), image.Rect(0, 0, 2, 1)}
	want := []image.Rectangle{image.Rect(0, 0, 4, 2), image.Rect(20, 20, 22, 22), image.Rect(110, 110, 120, 120)}
	got := w.compose()
	if len(got) != len(want) {
		t.Fatalf("composed %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("region %d is %v, want %v", i, got[i], want[i])
		}
	}
}

func TestSpots(t *testing.T) {
	w := runProgram(t, `window.open(100, 100, "", {"scale": 2})`, nil).(*object.Native).Value.(*Window)
	addSurface(t, w, "back", "[0, 0, 50, 50]", 2)
	addSurface(t, w, "canvas", "[10, 10, 20, 20]", 1)
	for _, tc := range []struct {
		x, y float64 // screen pixels
		want Spot
	}{
		{5, 5, Spot{"back", 2, 2}},
		{20, 20, Spot{"canvas", 0, 0}}, // the newest is on top
		{59.5, 25, Spot{"canvas", 39, 5}},
		{60, 25, Spot{"back", 30, 12}},
		{150, 150, Spot{}},
	} {
		if got := w.spotAt(tc.x, tc.y); got != tc.want {
			t.Errorf("(%v, %v) is %+v, want %+v", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestDragSamples(t *testing.T) {
	w := runProgram(t, `window.open(100, 100, "", {"scale": 2})`, nil).(*object.Native).Value.(*Window)
	addSurface(t, w, "canvas", "[10, 10, 20, 20]", 1)
	native := []pen.Sample{
		{Phase: "down", X: 1, Y: 1},
		{Phase: "drag", X: 21, Y: 23, Pressure: 1, Source: "tablet", T: 1.5},
		{Phase: "drag", X: 30, Y: 30, Pressure: 0.5, Source: "tablet", T: 1.6},
	}
	e := w.at(Event{Type: "move"}, 31, 31)
	got := w.dragSamples(native, e)
	want := []Sample{
		{X: 10, Y: 11, Spot: Spot{"canvas", 1, 3}, Pressure: 1, Raw: 1, T: 1.5},
		{X: 15, Y: 15, Spot: Spot{"canvas", 11, 11}, Pressure: 0.5, Raw: 0.5, T: 1.6},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sample %d is %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPresence(t *testing.T) {
	var p presence
	for _, tc := range []struct {
		focused bool
		x, y    int
		want    string
	}{
		{true, 5, 5, ""},
		{true, -1, 5, "leave"},
		{true, 5, 5, "enter"},
		{false, 5, 5, "focus false, leave"},
		{false, 6, 6, ""},
		{true, 5, 5, "focus true, enter"},
		{true, 10, 5, "leave"},
	} {
		var got []string
		for _, e := range p.update(tc.focused, tc.x, tc.y, 10, 10) {
			if e.Type == "focus" {
				got = append(got, "focus "+map[bool]string{true: "true", false: "false"}[e.Focused])
			} else {
				got = append(got, e.Type)
			}
		}
		if strings.Join(got, ", ") != tc.want {
			t.Errorf("focused %v at (%d, %d) gave %v, want %q", tc.focused, tc.x, tc.y, got, tc.want)
		}
	}
}

func TestNewEventHashes(t *testing.T) {
	for _, tc := range []struct {
		event Event
		want  string
	}{
		{Event{Type: "focus", Focused: true}, `{"type": "focus", "focused": true}`},
		{Event{Type: "leave"}, `{"type": "leave"}`},
		{Event{Type: "move", X: 1, Y: 2, Spot: Spot{"canvas", 3, 4}, Pen: mouse, T: 0.5},
			`{"type": "move", "pos": (1, 2), "surface": "canvas", "surface_pos": (3, 4), "mods": {"shift": false, "alt": false, "cmd": false, "ctrl": false}, "pen": {"pressure": 1.0, "raw": 1.0, "source": "mouse", "tilt": (0.0, 0.0), "eraser": false}, "samples": [{"pos": (1, 2), "surface": "canvas", "surface_pos": (3, 4), "pressure": 1.0, "raw": 1.0, "t": 0.5}]}`},
	} {
		if got := tc.event.hash().Inspect(); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.event.Type, got, tc.want)
		}
	}
}
