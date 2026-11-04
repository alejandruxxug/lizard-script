package window

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/pixels"
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestHeadless(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"events and set", `
let w = window.open(20, 10, "", {"headless": true, "scale": 2})
let log = []
window.inject(w, {"type": "down", "pos": (3, 4)})
window.inject(w, {"type": "move", "pos": (5.5, 4), "pen": {"pressure": 0.5, "source": "tablet"}})
window.inject(w, {"type": "key", "key": "A", "mods": {"cmd": true}})
let frames = 0
window.run(w, function(events) {
  frames += 1
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down") { push(log, [e.type, e.pos, e.button, e.pen.source]) }
    elif (e.type == "move") { push(log, [e.type, e.pen.pressure, e.samples[0].pos]) }
    elif (e.type == "key") { push(log, [e.type, e.key, e.mods.cmd]) }
    else { push(log, [e.type, e.size]) }
  }
  if (frames == 1) { window.set(w, {"size": (30, 12)}) }
  frames < 3
})
[frames, log, window.size(w)]`,
			`[3, [["down", (3, 4), "left", "mouse"], ["move", 0.5, (5, 4)], ["key", "A", true], ["resize", (30, 12)]], (30, 12)]`},
		{"surface events", `
let w = window.open(20, 10, "", {"headless": true, "scale": 2})
window.surface(w, "canvas", [5, 0, 10, 10], 1)
window.inject(w, {"type": "down", "pos": (7.5, 1)})
let got = null
window.run(w, function(events) {
  if (len(events) > 0) { got = [events[0].surface, events[0].surface_pos] }
  false
})
got`, `["canvas", (5, 2)]`},
		{"drop event", `
let w = window.open(20, 10, "", {"headless": true})
window.inject(w, {"type": "drop", "pos": (3, 4), "paths": ["/tmp/a.png", "/tmp/b.png"]})
let got = null
window.run(w, function(events) { got = [events[0].type, events[0].pos, events[0].paths]; false })
got`, `["drop", (3, 4), ["/tmp/a.png", "/tmp/b.png"]]`},
		{"on demand", `
let w = window.open(4, 4, "", {"headless": true, "on_demand": true})
let ticks = 0
window.run(w, function(events) {
  ticks += 1
  if (ticks < 3) { window.needs_paint(w) }
})
ticks`, "3"},
		{"runs again", `
let w = window.open(4, 4, "", {"headless": true})
let n = 0
window.run(w, function(events) { n += 1; false })
window.run(w, function(events) { n += 1; false })
n`, "2"},
		{"focus and leave", `
let w = window.open(4, 4, "", {"headless": true})
window.inject(w, {"type": "focus", "focused": false})
window.inject(w, {"type": "leave"})
let got = []
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) { push(got, events[i]) }
  false
})
got`, `[{"type": "focus", "focused": false}, {"type": "leave"}]`},
		{"chrome while it runs", `
let w = window.open(40, 20, "splash", {"headless": true, "frameless": true, "transparent": true, "floating": true, "scale": 3})
let sizes = []
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) { push(sizes, events[i].size) }
  if (len(sizes) == 0) {
    window.set(w, {"frameless": false, "floating": false, "scale": 2, "size": (64, 40), "min_size": (32, 20), "center": true, "resizable": true, "title": "editor"})
    return true
  }
  false
})
[sizes, window.size(w), img.size(window.screenshot(w))]`, `[[(64, 40)], (64, 40), (64, 40)]`},
		{"no smaller than min_size", `
let w = window.open(40, 20, "", {"headless": true, "min_size": (30, 30)})
window.set(w, {"size": (10, 10)})
window.size(w)`, "(30, 30)"},
	} {
		for _, engine := range engines {
			t.Run(tc.name+"/"+engine.name, func(t *testing.T) {
				if got := runOn(t, engine.run, tc.source, nil).Inspect(); got != tc.want {
					t.Fatalf("got %s\nwant %s", got, tc.want)
				}
			})
		}
	}
}

func TestHeadlessErrors(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"transparent later", `window.set(window.open(1, 1, ""), {"transparent": true})`, "transparent can only be chosen in window.open"},
		{"unknown set option", `window.set(window.open(1, 1, ""), {"fast": true})`, `unknown option "fast" (the options are center, floating`},
		{"bad size", `window.set(window.open(1, 1, ""), {"size": (0, 5)})`, "the size must be a point (w, h) of whole numbers from 1 to 8192, got (0, 5)"},
		{"inject a bad type", `window.inject(window.open(1, 1, ""), {"type": "wave"})`, `an event's type must be one of`},
		{"inject drop with no paths", `window.inject(window.open(1, 1, ""), {"type": "drop", "pos": (0, 0)})`, "a drop event needs a list of paths, got nothing"},
		{"inject drop with a number", `window.inject(window.open(1, 1, ""), {"type": "drop", "pos": (0, 0), "paths": [1]})`, "a drop event's paths are strings, got 1"},
		{"inject with no pos", `window.inject(window.open(1, 1, ""), {"type": "down"})`, "a down event needs a point pos, got nothing"},
		{"an error in tick", `window.run(window.open(1, 1, "", {"headless": true}), function(e) { 1 + "a" })`, "type mismatch: INTEGER + STRING"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := runProgram(t, tc.source, nil).(*object.Error)
			if !ok || !strings.Contains(got.Message, tc.want) {
				t.Fatalf("got %v, want an error with %q", got, tc.want)
			}
		})
	}
}

// A benchmark of composing a frame in which a stroke changed a surface:
// what a drawing program pays per frame on top of its own drawing.
func BenchmarkComposeStroke(b *testing.B) {
	buf, _ := pixels.New(640, 400, black)
	w := &Window{buf: buf, scale: 2, fps: 60}
	s := &Surface{win: w, name: "canvas", rect: image.Rect(40, 20, 600, 380), scale: 1}
	w.screen.surfaces = []*Surface{s}
	w.layout()
	w.compose()
	ink := color.NRGBA{40, 50, 80, 255}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		y := i % (s.buf.H - 8)
		for row := y; row < y+8; row++ {
			s.buf.Span(row, 100, 108, ink)
		}
		s.Touched()
		s.dirt.rects = append(s.dirt.rects, image.Rect(100, y, 108, y+8))
		w.compose()
	}
}
