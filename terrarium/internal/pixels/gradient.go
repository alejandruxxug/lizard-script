package pixels

import (
	"image/color"
	"math"
)

// Gradients. The program works out what a gradient is (where it starts,
// which way it runs, its stops and its steps); this file only finds its
// color at each pixel of a span, which would be too slow one pixel at a
// time in lizard-script.

// Stop is a color at a place t, 0 to 1, along a gradient.
type Stop struct {
	T float64
	C color.NRGBA
}

// Gradient is a gradient ready to paint. A linear one projects a pixel onto
// its axis; a radial one measures the pixel's distance from its center.
type Gradient struct {
	Radial bool
	OX, OY float64 // linear: where t is 0; radial: the center
	AX, AY float64 // linear: the axis divided by its squared length
	Scale  float64 // radial: 1 / radius
	Stops  []Stop  // at least one, in order of T
	Steps  int     // below 2: smooth; n: n flat bands
}

// colorAt is the color at t, from 0 to 1.
func (g *Gradient) colorAt(t float64) color.NRGBA {
	st := g.Stops
	if t <= st[0].T {
		return st[0].C
	}
	j := 0
	for j < len(st) && st[j].T <= t {
		j++
	}
	if j == len(st) {
		return st[len(st)-1].C
	}
	a, b := st[j-1], st[j]
	f := (t - a.T) / (b.T - a.T)
	mix := func(p, q uint8) uint8 {
		return uint8(math.Round(float64(p) + (float64(q)-float64(p))*f))
	}
	return color.NRGBA{R: mix(a.C.R, b.C.R), G: mix(a.C.G, b.C.G), B: mix(a.C.B, b.C.B), A: mix(a.C.A, b.C.A)}
}

// At is the color at the point (x, y) of the gradient's own space.
func (g *Gradient) At(x, y float64) color.NRGBA {
	var t float64
	if g.Radial {
		t = math.Hypot(x-g.OX, y-g.OY) * g.Scale
	} else {
		t = (x-g.OX)*g.AX + (y-g.OY)*g.AY
	}
	t = math.Max(0, math.Min(1, t))
	if g.Steps >= 2 {
		band := math.Min(math.Floor(t*float64(g.Steps)), float64(g.Steps-1))
		t = band / float64(g.Steps-1)
	}
	return g.colorAt(t)
}

// GradientSpan paints row y from x0 up to x1 like Span, but each pixel takes
// its color from g at the pixel's middle minus (dx, dy): a shape moved by
// (dx, dy) keeps its gradient. The ends are soft the same way.
func (m *Image) GradientSpan(y int, x0, x1 float64, g *Gradient, dx, dy float64) {
	first, last, ok := m.cut(y, &x0, &x1)
	if !ok {
		return
	}
	row := m.Pix[y*m.W*4:]
	py := float64(y) + 0.5 - dy
	for x := first; x <= last; x++ {
		cover := math.Min(float64(x+1), x1) - math.Max(float64(x), x0)
		blend(row[x*4:], g.At(float64(x)+0.5-dx, py), cover)
	}
}
