package pixels

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// corner is a Vertex in one color.
func corner(x, y, z float64, c color.NRGBA) Vertex { return Vertex{X: x, Y: y, Z: z, C: c} }

// mask draws m's pixels as text, one row per line: "#" opaque, "+" partly
// covered, "." untouched.
func mask(m *Image) string {
	var b strings.Builder
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			switch a := m.At(x, y).A; {
			case a == 255:
				b.WriteByte('#')
			case a > 0:
				b.WriteByte('+')
			default:
				b.WriteByte('.')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// rows joins golden rows into a mask.
func rows(lines ...string) string { return strings.Join(lines, "\n") + "\n" }

// The golden masks below are the pixels whose middle is inside each
// triangle, worked out by hand with the top-left rule.
func TestTriangleCoverage(t *testing.T) {
	white := color.NRGBA{255, 255, 255, 255}
	for _, tc := range []struct {
		name string
		draw func(m *Image)
		want string
	}{
		{"right triangle", func(m *Image) {
			m.Triangle([3]Vertex{corner(0, 0, 0, white), corner(6, 0, 0, white), corner(0, 6, 0, white)}, false)
		}, rows("#####.", "####..", "###...", "##....", "#.....", "......")},
		{"either turn", func(m *Image) {
			m.Triangle([3]Vertex{corner(0, 0, 0, white), corner(0, 6, 0, white), corner(6, 0, 0, white)}, false)
		}, rows("#####.", "####..", "###...", "##....", "#.....", "......")},
		{"pixel middles only", func(m *Image) {
			m.Triangle([3]Vertex{corner(1.6, 1.6, 0, white), corner(4.4, 1.6, 0, white), corner(1.6, 4.4, 0, white)}, false)
		}, rows("......", "......", "..#...", "......", "......", "......")},
		{"top and left edges are in, bottom and right out", func(m *Image) {
			m.Triangle([3]Vertex{corner(1.5, 1.5, 0, white), corner(4.5, 1.5, 0, white), corner(1.5, 4.5, 0, white)}, false)
		}, rows("......", ".###..", ".##...", ".#....", "......", "......")},
		{"no area", func(m *Image) {
			m.Triangle([3]Vertex{corner(0, 0, 0, white), corner(3, 3, 0, white), corner(6, 6, 0, white)}, false)
		}, rows("......", "......", "......", "......", "......", "......")},
		{"far outside is cut", func(m *Image) {
			m.Triangle([3]Vertex{corner(-1e5, -1e5, 0, white), corner(3e5, -1e5, 0, white), corner(-1e5, 3e5, 0, white)}, false)
		}, rows("######", "######", "######", "######", "######", "######")},
		{"clip", func(m *Image) {
			m.SetClip(image.Rect(2, 1, 4, 3))
			m.Triangle([3]Vertex{corner(0, 0, 0, white), corner(6, 0, 0, white), corner(0, 6, 0, white)}, false)
		}, rows("......", "..##..", "..#...", "......", "......", "......")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := New(6, 6, color.NRGBA{})
			tc.draw(m)
			if got := mask(m); got != tc.want {
				t.Errorf("got\n%swant\n%s", got, tc.want)
			}
		})
	}
}

// Two triangles that share an edge paint each pixel of it once: a square of
// half transparent triangles comes out even, with no darker or missing
// diagonal, at any angle.
func TestTriangleSharedEdge(t *testing.T) {
	half := color.NRGBA{255, 0, 0, 128}
	for _, corners := range [][4][2]float64{
		{{0, 0}, {8, 0}, {8, 8}, {0, 8}},
		{{-1.3, 0.2}, {9.1, -0.7}, {8.6, 9.4}, {-0.4, 8.8}},
		{{4, -2}, {11, 4}, {4, 11}, {-3, 4}},
	} {
		m, _ := New(8, 8, color.NRGBA{})
		v := func(i int) Vertex { return corner(corners[i][0], corners[i][1], 0, half) }
		m.Triangle([3]Vertex{v(0), v(1), v(2)}, false)
		m.Triangle([3]Vertex{v(0), v(2), v(3)}, false)
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				if a := m.At(x, y).A; a != 0 && a != 128 {
					t.Errorf("square %v: pixel (%d, %d) has alpha %d: painted twice", corners, x, y, a)
				}
			}
		}
	}
}

// The nearer triangle wins wherever both are, whichever is drawn first; a
// cleared depth buffer lets the next one through again.
func TestTriangleDepth(t *testing.T) {
	red, blue := color.NRGBA{255, 0, 0, 255}, color.NRGBA{0, 0, 255, 255}
	square := func(m *Image, z float64, c color.NRGBA, depth bool) {
		m.Triangle([3]Vertex{corner(0, 0, z, c), corner(4, 0, z, c), corner(4, 4, z, c)}, depth)
		m.Triangle([3]Vertex{corner(0, 0, z, c), corner(4, 4, z, c), corner(0, 4, z, c)}, depth)
	}
	// A slope from z 0 at the left to 1 at the right, through a flat 0.5.
	slope := func(m *Image, c color.NRGBA) {
		m.Triangle([3]Vertex{corner(0, 0, 0, c), corner(4, 0, 1, c), corner(4, 4, 1, c)}, true)
		m.Triangle([3]Vertex{corner(0, 0, 0, c), corner(4, 4, 1, c), corner(0, 4, 0, c)}, true)
	}
	for _, tc := range []struct {
		name string
		draw func(m *Image)
		want color.NRGBA // at (1, 1); (3, 1) is the other
		far  color.NRGBA
	}{
		{"near first", func(m *Image) { square(m, 0.2, red, true); square(m, 0.6, blue, true) }, red, red},
		{"near last", func(m *Image) { square(m, 0.6, blue, true); square(m, 0.2, red, true) }, red, red},
		{"no depth paints over", func(m *Image) { square(m, 0.2, red, true); square(m, 0.6, blue, false) }, blue, blue},
		{"cleared", func(m *Image) { square(m, 0.2, red, true); m.ClearDepth(); square(m, 0.6, blue, true) }, blue, blue},
		{"crossing", func(m *Image) { square(m, 0.5, red, true); slope(m, blue) }, blue, red},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := New(4, 4, color.NRGBA{})
			tc.draw(m)
			if got := m.At(1, 1); got != tc.want {
				t.Errorf("(1, 1) is %v, want %v", got, tc.want)
			}
			if got := m.At(3, 1); got != tc.far {
				t.Errorf("(3, 1) is %v, want %v", got, tc.far)
			}
		})
	}
}

// Colors are mixed from the corners by how near each pixel's middle is to
// them; the golden values are the weights worked out by hand.
func TestTriangleColors(t *testing.T) {
	m, _ := New(4, 4, color.NRGBA{})
	m.Triangle([3]Vertex{
		corner(0, 0, 0, color.NRGBA{255, 0, 0, 255}),
		corner(4, 0, 0, color.NRGBA{0, 255, 0, 255}),
		corner(0, 4, 0, color.NRGBA{0, 0, 255, 255}),
	}, false)
	for _, tc := range []struct {
		x, y int
		want color.NRGBA
	}{
		// The middle of (0, 0) is 1/8 of the way to green and to blue; the
		// middle of (1, 1) is 3/8 of the way to each.
		{0, 0, color.NRGBA{191, 32, 32, 255}},
		{2, 0, color.NRGBA{64, 159, 32, 255}},
		{0, 2, color.NRGBA{64, 32, 159, 255}},
		{1, 1, color.NRGBA{64, 96, 96, 255}},
		{2, 2, color.NRGBA{}}, // on the far edge: a bottom right one, out
	} {
		if got := m.At(tc.x, tc.y); got != tc.want {
			t.Errorf("(%d, %d) is %v, want %v", tc.x, tc.y, got, tc.want)
		}
	}

	// A color with alpha is painted over what is there.
	under, _ := New(1, 1, color.NRGBA{0, 0, 255, 255})
	c := color.NRGBA{255, 0, 0, 128}
	under.Triangle([3]Vertex{corner(0, 0, 0, c), corner(2, 0, 0, c), corner(0, 2, 0, c)}, true)
	if got := under.At(0, 0); got != (color.NRGBA{128, 0, 127, 255}) {
		t.Errorf("half red over blue is %v", got)
	}
}
