package pixels

import (
	"image"
	"image/color"
	"math"
)

// BlitScaled paints src over m at k times its size (k above 0), with its
// top left at (x, y), cut to the clip. src can be m itself.
//
// Each pixel of m gets the color of src under its center. Not smooth, that
// is the nearest pixel of src: sharp, the big square pixels of pixel art.
// Smooth, it mixes the four pixels of src around that point when
// enlarging, and averages every pixel of src the target pixel covers when
// shrinking, so a thumbnail keeps thin lines faintly instead of losing
// them. Colors mix with their alpha, so a transparent pixel doesn't
// darken its neighbors.
func (m *Image) BlitScaled(src *Image, x, y int, k float64, smooth bool) {
	if src.W == 0 || src.H == 0 || !(k > 0) {
		return
	}
	if src == m {
		src = &Image{W: m.W, H: m.H, Pix: append([]byte(nil), m.Pix...)}
	}
	// The scaled size, held below a size no int sum can overflow at.
	side := func(n int) int { return max(1, int(math.Min(math.Round(float64(n)*k), 1<<30))) }
	w, h := side(src.W), side(src.H)
	rx, ry := float64(src.W)/float64(w), float64(src.H)/float64(h)
	area := image.Rect(x, y, x+w, y+h).Intersect(m.Clip())
	for ty := area.Min.Y; ty < area.Max.Y; ty++ {
		row := m.Pix[ty*m.W*4:]
		for tx := area.Min.X; tx < area.Max.X; tx++ {
			var c color.NRGBA
			switch {
			case !smooth:
				c = src.At(min(int((float64(tx-x)+0.5)*rx), src.W-1), min(int((float64(ty-y)+0.5)*ry), src.H-1))
			case k >= 1:
				c = src.bilinear((float64(tx-x)+0.5)*rx-0.5, (float64(ty-y)+0.5)*ry-0.5)
			default:
				c = src.average(float64(tx-x)*rx, float64(ty-y)*ry, rx, ry)
			}
			if c.A != 0 {
				blend(row[tx*4:], c, 1)
			}
		}
	}
}

// tally adds up colors with weights, premultiplied by their alpha.
type tally struct{ r, g, b, a, weight float64 }

// add counts the pixel (x, y) of m with weight w.
func (s *tally) add(m *Image, x, y int, w float64) {
	p := m.Pix[(y*m.W+x)*4:]
	a := float64(p[3]) * w
	s.r, s.g, s.b, s.a = s.r+float64(p[0])*a, s.g+float64(p[1])*a, s.b+float64(p[2])*a, s.a+a
	s.weight += w
}

// color is the tally as a color: the average of what was added.
func (s *tally) color() color.NRGBA {
	if s.a <= 0 || s.weight <= 0 {
		return color.NRGBA{}
	}
	channel := func(v float64) uint8 { return uint8(math.Min(255, math.Round(v/s.a))) }
	return color.NRGBA{R: channel(s.r), G: channel(s.g), B: channel(s.b), A: uint8(math.Min(255, math.Round(s.a/s.weight)))}
}

// bilinear is the color at (fx, fy), in pixels from the center of the top
// left one: the four pixels around it weighed by how near they are. Past the
// edges the edge pixels go on.
func (m *Image) bilinear(fx, fy float64) color.NRGBA {
	x0, y0 := math.Floor(fx), math.Floor(fy)
	tx, ty := fx-x0, fy-y0
	clamp := func(v float64, n int) int { return min(max(int(v), 0), n-1) }
	xa, xb := clamp(x0, m.W), clamp(x0+1, m.W)
	ya, yb := clamp(y0, m.H), clamp(y0+1, m.H)
	var s tally
	s.add(m, xa, ya, (1-tx)*(1-ty))
	s.add(m, xb, ya, tx*(1-ty))
	s.add(m, xa, yb, (1-tx)*ty)
	s.add(m, xb, yb, tx*ty)
	return s.color()
}

// average is the color of the rectangle of w by h pixels at (fx, fy): every
// pixel it touches, weighed by how much of it is inside.
func (m *Image) average(fx, fy, w, h float64) color.NRGBA {
	var s tally
	for y := int(fy); y < m.H && float64(y) < fy+h; y++ {
		wy := math.Min(float64(y+1), fy+h) - math.Max(float64(y), fy)
		for x := int(fx); x < m.W && float64(x) < fx+w; x++ {
			wx := math.Min(float64(x+1), fx+w) - math.Max(float64(x), fx)
			s.add(m, x, y, wx*wy)
		}
	}
	return s.color()
}
