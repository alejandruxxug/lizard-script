package pixels

import (
	"image"
	"image/color"
	"testing"
)

// The scenes below paint gradients and blend layers the way the reference
// implementation did, and compare every pixel with its golden one. This
// buffer keeps colors unpremultiplied and mixes in floats, so a pixel that
// isn't opaque can round one step apart; an opaque one must match exactly.

var (
	red   = color.NRGBA{255, 0, 0, 255}
	blue  = color.NRGBA{0, 0, 255, 255}
	green = color.NRGBA{0, 255, 0, 255}
	gray  = color.NRGBA{40, 40, 40, 255}
)

var linear = &Gradient{AX: 1.0 / 16, Stops: []Stop{{0, red}, {0.5, blue}, {0.5, green}, {1, color.NRGBA{255, 255, 255, 128}}}}
var radial = &Gradient{Radial: true, OX: 8, OY: 2.5, Scale: 1.0 / 8, Stops: []Stop{{0, color.NRGBA{255, 255, 0, 255}}, {1, color.NRGBA{0, 0, 128, 200}}}}

// premultiplied is pixel i of m as a screen takes it.
func premultiplied(m *Image, i int) [4]uint8 {
	dst := make([]byte, len(m.Pix))
	m.Premultiplied(dst)
	return [4]uint8{dst[i*4], dst[i*4+1], dst[i*4+2], dst[i*4+3]}
}

// matchGolden compares every pixel of m with want.
func matchGolden(t *testing.T, m *Image, want [][4]uint8) {
	t.Helper()
	if len(want) != m.W*m.H {
		t.Fatalf("%d golden pixels for a %dx%d image", len(want), m.W, m.H)
	}
	for i, w := range want {
		got := premultiplied(m, i)
		tolerance := 1
		if w[3] == 255 {
			tolerance = 0
		}
		for k := 0; k < 4; k++ {
			if d := int(got[k]) - int(w[k]); d > tolerance || d < -tolerance {
				t.Errorf("pixel (%d, %d) is %v, want %v", i%m.W, i/m.W, got, w)
				break
			}
		}
	}
}

func TestGradientGolden(t *testing.T) {
	m, _ := New(16, 4, gray)
	m.GradientSpan(0, 0.5, 15.25, linear, 0, 0)
	stepped := *linear
	stepped.Steps = 4
	m.GradientSpan(1, 0, 16, &stepped, 0, 0)
	m.GradientSpan(2, 1.3, 14.7, radial, 0, 0)
	m.GradientSpan(3, 2+3, 10+3, linear, 3, 1)
	matchGolden(t, m, goldenGradient)

	clear, _ := New(16, 1, color.NRGBA{})
	clear.GradientSpan(0, 0, 16, linear, 0, 0)
	matchGolden(t, clear, goldenGradientClear)
}

func TestBlendGolden(t *testing.T) {
	back := []color.NRGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {128, 128, 128, 255}, {255, 255, 255, 255}, {0, 0, 0, 255}, {200, 100, 50, 255}, {30, 60, 90, 255}}
	src := []color.NRGBA{{0, 0, 255, 255}, {255, 255, 0, 255}, {128, 128, 128, 255}, {255, 0, 0, 128}, {10, 200, 30, 255}, {255, 255, 255, 64}, {100, 150, 200, 255}, {0, 0, 0, 0}}
	for _, tc := range []struct {
		mode string
		// want is over an opaque backdrop, half over a half transparent one
		want, half [][4]uint8
	}{
		{mode: "normal", want: goldenBlend_normal, half: goldenBlend_normal_half},
		{mode: "multiply", want: goldenBlend_multiply, half: goldenBlend_multiply_half},
		{mode: "screen", want: goldenBlend_screen, half: goldenBlend_screen_half},
		{mode: "overlay", want: goldenBlend_overlay, half: goldenBlend_overlay_half},
		{mode: "darken", want: goldenBlend_darken, half: goldenBlend_darken_half},
		{mode: "lighten", want: goldenBlend_lighten, half: goldenBlend_lighten_half},
		{mode: "add", want: goldenBlend_add, half: goldenBlend_add_half},
	} {
		for _, half := range []bool{false, true} {
			name, want := tc.mode, tc.want
			if half {
				name, want = tc.mode+" over half", tc.half
			}
			t.Run(name, func(t *testing.T) {
				d, _ := New(8, 1, color.NRGBA{})
				s, _ := New(8, 1, color.NRGBA{})
				for i := range back {
					c := back[i]
					if half {
						c.A = 128
					}
					d.Span(0, float64(i), float64(i+1), c)
					s.Span(0, float64(i), float64(i+1), src[i])
				}
				d.Composite(s, 0, 0, Modes[tc.mode], 0.6)
				matchGolden(t, d, want)
			})
		}
	}
}

// A layer is a composite of what was drawn on it, onto what is below.
func TestLayers(t *testing.T) {
	m, _ := New(2, 1, color.NRGBA{128, 128, 128, 255})
	m.SetClip(image.Rect(1, 0, 2, 1))
	m.PushLayer()
	if m.Top() == m || m.Layers() != 1 {
		t.Fatalf("PushLayer opened no layer")
	}
	m.Pixels().Fill(0, 0, 2, 1, color.NRGBA{255, 255, 255, 255}) // the clip goes with the layer
	if !m.PopLayer(Modes["multiply"], 1) || m.Top() != m {
		t.Fatalf("PopLayer closed no layer")
	}
	if got := m.At(0, 0); got != (color.NRGBA{128, 128, 128, 255}) {
		t.Errorf("the clipped pixel is %v, want gray", got)
	}
	if got := m.At(1, 0); got != (color.NRGBA{128, 128, 128, 255}) {
		t.Errorf("gray multiplied by white is %v, want gray", got)
	}
	if m.PopLayer(Modes["normal"], 1) {
		t.Errorf("PopLayer with no layer open reported true")
	}
}

func TestClip(t *testing.T) {
	white := color.NRGBA{255, 255, 255, 255}
	for _, tc := range []struct {
		name string
		draw func(m *Image)
		want string // the alpha of each pixel of the row: x for painted
	}{
		{"span", func(m *Image) { m.Span(0, 0, 6, white) }, "..xx.."},
		{"fill", func(m *Image) { m.Fill(-5, 0, 50, 1, white) }, "..xx.."},
		{"clear", func(m *Image) { m.Clear(white) }, "..xx.."},
		{"clear a rect", func(m *Image) { m.ClearRect(image.Rect(3, 0, 9, 1), white) }, "...x.."},
		{"blit", func(m *Image) { src, _ := New(6, 1, white); m.Blit(src, 0, 0) }, "..xx.."},
		{"gradient", func(m *Image) { m.GradientSpan(0, 0, 6, &Gradient{Stops: []Stop{{0, white}}}, 0, 0) }, "..xx.."},
		{"unclip", func(m *Image) { m.Unclip(); m.Span(0, 0, 6, white) }, "xxxxxx"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := New(6, 1, color.NRGBA{})
			m.SetClip(image.Rect(2, 0, 4, 9))
			tc.draw(m)
			got := ""
			for x := 0; x < 6; x++ {
				if m.At(x, 0).A == 255 {
					got += "x"
				} else {
					got += "."
				}
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
