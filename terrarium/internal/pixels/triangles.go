package pixels

import (
	"image/color"
	"math"
)

// Triangles with a depth buffer: the one step of 3D drawing that is done
// pixel by pixel. The program works out where each corner lands on the
// screen, how far away it is and what color it has (the camera, the
// lights, the clipping); this file only fills the pixels between the
// corners, keeps the nearest one, and mixes the colors of the corners.

// Vertex is a corner of a triangle on the screen: X and Y in pixels from
// the top left, Z its depth (smaller is nearer), C its color.
type Vertex struct {
	X, Y, Z float64
	C       color.NRGBA
}

// subpixel is how many steps a pixel is cut into when the corners are
// snapped: the edges are then exact integers, so two triangles that share
// an edge never both paint, or both miss, a pixel of it.
const subpixel = 256

// farthest is how far a corner can be from the image, in pixels, before it
// is pulled in: it keeps the edge products inside an int64.
const farthest = 1 << 21

// ClearDepth makes every pixel of the depth buffer infinitely far, so the
// next triangles are drawn whatever was drawn before.
func (m *Image) ClearDepth() {
	if len(m.depth) != m.W*m.H {
		m.depth = make([]float32, m.W*m.H)
	}
	inf := float32(math.Inf(1))
	for i := range m.depth {
		m.depth[i] = inf
	}
}

// Triangle paints the triangle of the corners v over m, cut to the clip.
// A pixel is painted when its middle is inside, or on an edge that is a top
// or a left one (so a shared edge is painted once). Its depth and color are
// mixed from the corners' by how close the middle is to each. With depth,
// a pixel is only painted when it is nearer than what the depth buffer
// holds there, and then the buffer keeps its depth; without it the
// triangle is painted over whatever is there.
func (m *Image) Triangle(v [3]Vertex, depth bool) {
	if depth && len(m.depth) != m.W*m.H {
		m.ClearDepth()
	}
	var x, y [3]int64
	for i := range v {
		x[i], y[i] = snap(v[i].X), snap(v[i].Y)
	}
	area := edge(x[0], y[0], x[1], y[1], x[2], y[2])
	if area == 0 {
		return
	}
	if area < 0 {
		// Turn it around, so the inside is where every edge is positive.
		v[1], v[2] = v[2], v[1]
		x[1], x[2], y[1], y[2] = x[2], x[1], y[2], y[1]
		area = -area
	}

	clip := m.Clip()
	left := max(clip.Min.X, int(floorDiv(min(x[0], x[1], x[2]), subpixel)))
	right := min(clip.Max.X, int(floorDiv(max(x[0], x[1], x[2]), subpixel))+1)
	top := max(clip.Min.Y, int(floorDiv(min(y[0], y[1], y[2]), subpixel)))
	bottom := min(clip.Max.Y, int(floorDiv(max(y[0], y[1], y[2]), subpixel))+1)
	if left >= right || top >= bottom {
		return
	}

	// The edge across from each corner, its step to the next pixel to the
	// right and down, and whether a middle that falls exactly on it counts.
	var e0, stepX, stepY [3]int64
	var tie [3]bool
	px, py := int64(left)*subpixel+subpixel/2, int64(top)*subpixel+subpixel/2
	for i := range v {
		a, b := (i+1)%3, (i+2)%3
		e0[i] = edge(x[a], y[a], x[b], y[b], px, py)
		stepX[i] = -(y[b] - y[a]) * subpixel
		stepY[i] = (x[b] - x[a]) * subpixel
		tie[i] = topLeft(x[b]-x[a], y[b]-y[a])
	}

	whole := float64(area)
	for row := top; row < bottom; row++ {
		e := e0
		line := m.Pix[row*m.W*4:]
		for col := left; col < right; col++ {
			if inside(e, tie) {
				w0, w1, w2 := float64(e[0])/whole, float64(e[1])/whole, float64(e[2])/whole
				z := w0*v[0].Z + w1*v[1].Z + w2*v[2].Z
				if !depth || m.nearer(row*m.W+col, z) {
					paint(line[col*4:], mixCorners(v, w0, w1, w2))
				}
			}
			for i := range e {
				e[i] += stepX[i]
			}
		}
		for i := range e0 {
			e0[i] += stepY[i]
		}
	}
}

// snap turns a coordinate in pixels into subpixel steps, pulled in to
// farthest pixels from the image.
func snap(f float64) int64 {
	f = math.Max(-farthest, math.Min(farthest, f))
	if math.IsNaN(f) {
		return 0
	}
	return int64(math.Round(f * subpixel))
}

// floorDiv is a / b rounded down, for a b above 0.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// edge is twice the signed area of the triangle a, b, p: above 0 when p is
// on the inside of the edge from a to b (in a triangle turned so its area
// is above 0).
func edge(ax, ay, bx, by, px, py int64) int64 {
	return (bx-ax)*(py-ay) - (by-ay)*(px-ax)
}

// topLeft says whether an edge going (dx, dy) is a top edge (flat, with the
// inside below it) or a left one (the inside to its right).
func topLeft(dx, dy int64) bool {
	return dy < 0 || (dy == 0 && dx > 0)
}

// inside says whether a pixel's middle, with these edge values, is inside
// the triangle.
func inside(e [3]int64, tie [3]bool) bool {
	for i := range e {
		if e[i] < 0 || (e[i] == 0 && !tie[i]) {
			return false
		}
	}
	return true
}

// nearer tests the depth z at pixel i against the depth buffer, and keeps
// it when it is nearer.
func (m *Image) nearer(i int, z float64) bool {
	z32 := float32(z)
	if !(z32 < m.depth[i]) {
		return false
	}
	m.depth[i] = z32
	return true
}

// mixCorners is the color of the corners weighted by w0, w1 and w2, which
// add up to 1.
func mixCorners(v [3]Vertex, w0, w1, w2 float64) color.NRGBA {
	a, b, c := v[0].C, v[1].C, v[2].C
	if a == b && b == c {
		return a
	}
	channel := func(p, q, r uint8) uint8 {
		return uint8(math.Max(0, math.Min(255, math.Round(w0*float64(p)+w1*float64(q)+w2*float64(r)))))
	}
	return color.NRGBA{R: channel(a.R, b.R, c.R), G: channel(a.G, b.G, c.G), B: channel(a.B, b.B, c.B), A: channel(a.A, b.A, c.A)}
}

// paint puts c over the pixel p.
func paint(p []byte, c color.NRGBA) {
	if c.A == 255 {
		p[0], p[1], p[2], p[3] = c.R, c.G, c.B, 255
		return
	}
	blend(p, c, 1)
}
