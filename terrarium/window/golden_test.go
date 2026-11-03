package window

import (
	"image/color"
	"lizardscript/object"
	"testing"
)

// goldenCompose is what the reference implementation of composing (a
// premultiplied, 8-bit frame) gives for the scene of TestComposeGolden:
// a 4x3 buffer at scale 2 with a surface at scale 1 over (1, 1, 2, 1).
var goldenCompose = [][4]uint8{{0, 0, 50, 255}, {0, 0, 50, 255}, {60, 0, 50, 255}, {60, 0, 50, 255}, {120, 0, 50, 255}, {120, 0, 50, 255}, {180, 0, 50, 255}, {180, 0, 50, 255}, {0, 0, 50, 255}, {0, 0, 50, 255}, {60, 0, 50, 255}, {60, 0, 50, 255}, {120, 0, 50, 255}, {120, 0, 50, 255}, {180, 0, 50, 255}, {180, 0, 50, 255}, {0, 100, 50, 255}, {0, 100, 50, 255}, {255, 0, 0, 255}, {30, 178, 25, 255}, {120, 100, 50, 255}, {0, 0, 255, 255}, {180, 100, 50, 255}, {180, 100, 50, 255}, {0, 100, 50, 255}, {0, 100, 50, 255}, {213, 222, 211, 255}, {10, 20, 30, 255}, {120, 100, 50, 255}, {116, 100, 60, 255}, {180, 100, 50, 255}, {180, 100, 50, 255}, {0, 200, 50, 255}, {0, 200, 50, 255}, {60, 200, 50, 255}, {60, 200, 50, 255}, {120, 200, 50, 255}, {120, 200, 50, 255}, {180, 200, 50, 255}, {180, 200, 50, 255}, {0, 200, 50, 255}, {0, 200, 50, 255}, {60, 200, 50, 255}, {60, 200, 50, 255}, {120, 200, 50, 255}, {120, 200, 50, 255}, {180, 200, 50, 255}, {180, 200, 50, 255}}

func TestComposeGolden(t *testing.T) {
	got := runProgram(t, `window.open(4, 3, "", {"scale": 2})`, nil)
	w := got.(*object.Native).Value.(*Window)
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			w.buf.Set(x, y, color.NRGBA{uint8(x * 60), uint8(y * 100), 50, 255})
		}
	}
	s := addSurface(t, w, "canvas", "[1, 1, 2, 1]", 1)
	if s.buf.W != 4 || s.buf.H != 2 {
		t.Fatalf("the surface is %dx%d, want 4x2 (twice its rect)", s.buf.W, s.buf.H)
	}
	canvas := []color.NRGBA{{255, 0, 0, 255}, {0, 255, 0, 128}, {}, {0, 0, 255, 255}, {255, 255, 255, 200}, {10, 20, 30, 255}, {}, {100, 100, 100, 50}}
	for i, c := range canvas {
		s.buf.Span(i/4, float64(i%4), float64(i%4+1), c)
	}
	regions := w.compose()
	if len(regions) != 1 || regions[0] != w.screen.frame.Bounds() {
		t.Fatalf("the first compose made %v, want the whole frame", regions)
	}
	frame := w.screen.frame
	premultiplied := make([]byte, len(frame.Pix))
	frame.Premultiplied(premultiplied)
	for i, want := range goldenCompose {
		got := [4]uint8(premultiplied[i*4 : i*4+4])
		if got != want {
			t.Errorf("pixel (%d, %d) is %v, want %v", i%frame.W, i/frame.W, got, want)
		}
	}
	shot := w.shot()
	if shot.W != 8 || shot.H != 6 || string(shot.Pix) != string(frame.Pix) {
		t.Errorf("the screenshot is not the composed frame")
	}
}
