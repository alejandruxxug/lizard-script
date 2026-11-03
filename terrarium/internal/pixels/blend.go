package pixels

import (
	"image"
	"math"
	"sort"
)

// Blending: a layer drawn on its own (PushLayer) is mixed onto what is
// below it with a blend mode and an opacity (PopLayer, Composite). This is
// the W3C compositing formula: where both are there, the source color
// becomes (1 - ab) * Cs + ab * B(Cb, Cs), and then goes over the backdrop.

// Mode mixes a backdrop channel cb with a source channel cs, both 0 to 1.
type Mode func(cb, cs float64) float64

// Modes are the blend modes by name.
var Modes = map[string]Mode{
	"normal":   func(cb, cs float64) float64 { return cs },
	"multiply": func(cb, cs float64) float64 { return cb * cs },
	"screen":   func(cb, cs float64) float64 { return cb + cs - cb*cs },
	"overlay": func(cb, cs float64) float64 {
		if cb <= 0.5 {
			return 2 * cb * cs
		}
		return 1 - 2*(1-cb)*(1-cs)
	},
	"darken":  math.Min,
	"lighten": math.Max,
	"add":     func(cb, cs float64) float64 { return math.Min(1, cb+cs) },
}

// ModeNames are the names of the blend modes, in alphabetical order.
func ModeNames() []string {
	names := make([]string, 0, len(Modes))
	for name := range Modes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Composite mixes src onto m with its top left at (x, y), with a blend mode
// and an opacity from 0 to 1. It is cut to m's clip.
func (m *Image) Composite(src *Image, x, y int, mode Mode, opacity float64) {
	opacity = math.Max(0, math.Min(1, opacity))
	area := src.Bounds().Add(image.Pt(x, y)).Intersect(m.Clip())
	for dy := area.Min.Y; dy < area.Max.Y; dy++ {
		for dx := area.Min.X; dx < area.Max.X; dx++ {
			s := src.Pix[((dy-y)*src.W+dx-x)*4:]
			if s[3] == 0 {
				continue
			}
			mix(m.Pix[(dy*m.W+dx)*4:], s, mode, opacity)
		}
	}
}

// mix puts the source pixel s onto the backdrop pixel d.
func mix(d, s []byte, mode Mode, opacity float64) {
	a := float64(s[3]) / 255 * opacity
	ab := float64(d[3]) / 255
	out := a + ab*(1-a)
	if out <= 0 {
		return
	}
	for i := 0; i < 3; i++ {
		cs := float64(s[i]) / 255
		cb := float64(d[i]) / 255
		cr := (1-ab)*cs + ab*mode(cb, cs)
		premultiplied := a*cr + (1-a)*cb*ab
		d[i] = byte(math.Round(math.Min(1, premultiplied/out) * 255))
	}
	d[3] = byte(math.Round(out * 255))
}

// PushLayer opens a new transparent layer the size of the image, with the
// clip of the one below. Drawing goes to it until PopLayer. Layers nest.
func (m *Image) PushLayer() {
	below := m.Top()
	layer := &Image{W: m.W, H: m.H, Pix: make([]byte, len(m.Pix))}
	if below.clipped {
		layer.SetClip(below.clip)
	}
	m.layers = append(m.layers, layer)
}

// PopLayer closes the newest layer and mixes it onto the one below with a
// blend mode and an opacity. It reports false when no layer is open.
func (m *Image) PopLayer(mode Mode, opacity float64) bool {
	n := len(m.layers)
	if n == 0 {
		return false
	}
	layer := m.layers[n-1]
	m.layers = m.layers[:n-1]
	m.Top().Composite(layer, 0, 0, mode, opacity)
	return true
}

// Layers is how many layers are open.
func (m *Image) Layers() int { return len(m.layers) }

// Top is where drawing goes: the newest open layer, or the image itself.
func (m *Image) Top() *Image {
	if n := len(m.layers); n > 0 {
		return m.layers[n-1]
	}
	return m
}
