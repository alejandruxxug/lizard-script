// Package pixels is the pixel buffer under the img and window modules. It
// only stores and mixes colors: what to draw is always decided by the
// lizard-script program.
package pixels

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

// MaxPixels limits the size of one image, so img.new(100000, 100000) is an
// error instead of using all the memory.
const MaxPixels = 1 << 26

// Image is a buffer of W by H pixels, row by row from the top left. Each
// pixel is four bytes, red, green, blue and alpha, not premultiplied, so a
// color set is the color read back.
type Image struct {
	W, H int
	Pix  []byte
}

// New makes an image filled with c. It fails when the size is negative or
// bigger than MaxPixels.
func New(w, h int, c color.NRGBA) (*Image, error) {
	if w < 0 || h < 0 {
		return nil, fmt.Errorf("the size can't be negative, got %dx%d", w, h)
	}
	if h != 0 && w > MaxPixels/h {
		return nil, fmt.Errorf("the image is too big: %dx%d is more than %d pixels", w, h, MaxPixels)
	}
	m := &Image{W: w, H: h, Pix: make([]byte, w*h*4)}
	if c != (color.NRGBA{}) {
		m.Clear(c)
	}
	return m, nil
}

// Describe is the size, for Inspect: 640x480.
func (m *Image) Describe() string { return fmt.Sprintf("%dx%d", m.W, m.H) }

// At is the color of the pixel (x, y), which must be in the image.
func (m *Image) At(x, y int) color.NRGBA {
	p := m.Pix[(y*m.W+x)*4:]
	return color.NRGBA{R: p[0], G: p[1], B: p[2], A: p[3]}
}

// Set replaces the pixel (x, y), which must be in the image, with c.
func (m *Image) Set(x, y int, c color.NRGBA) {
	p := m.Pix[(y*m.W+x)*4:]
	p[0], p[1], p[2], p[3] = c.R, c.G, c.B, c.A
}

// Clear replaces every pixel with c.
func (m *Image) Clear(c color.NRGBA) {
	for i := 0; i < len(m.Pix); i += 4 {
		m.Pix[i], m.Pix[i+1], m.Pix[i+2], m.Pix[i+3] = c.R, c.G, c.B, c.A
	}
}

// blend paints c over the pixel p with a coverage from 0 to 1, the way a
// color is painted over another (source over).
func blend(p []byte, c color.NRGBA, cover float64) {
	a := float64(c.A) / 255 * cover
	if a <= 0 {
		return
	}
	if a >= 1 {
		p[0], p[1], p[2], p[3] = c.R, c.G, c.B, 255
		return
	}
	below := float64(p[3]) / 255 * (1 - a)
	out := a + below
	mix := func(top, bottom byte) byte {
		return byte(math.Round((float64(top)*a + float64(bottom)*below) / out))
	}
	p[0], p[1], p[2] = mix(c.R, p[0]), mix(c.G, p[1]), mix(c.B, p[2])
	p[3] = byte(math.Round(out * 255))
}

// Span paints c over the pixels from x0 up to x1 of row y. A fractional end
// covers its pixel partly, so the edge is soft. What is outside is cut.
func (m *Image) Span(y int, x0, x1 float64, c color.NRGBA) {
	if y < 0 || y >= m.H || c.A == 0 {
		return
	}
	x0 = math.Max(x0, 0)
	x1 = math.Min(x1, float64(m.W))
	if !(x1 > x0) {
		return
	}
	first := int(math.Floor(x0))
	last := int(math.Ceil(x1)) - 1
	row := m.Pix[y*m.W*4:]
	cover := func(x int) float64 {
		return math.Min(float64(x+1), x1) - math.Max(float64(x), x0)
	}
	blend(row[first*4:], c, cover(first))
	if last == first {
		return
	}
	for x := first + 1; x < last; x++ {
		blend(row[x*4:], c, 1)
	}
	blend(row[last*4:], c, cover(last))
}

// Fill paints c over the rectangle at (x, y) of w by h pixels. Its rows are
// rounded to whole pixels; its sides can cover a pixel partly.
func (m *Image) Fill(x, y, w, h float64, c color.NRGBA) {
	for row := int(math.Round(y)); row < int(math.Round(y+h)); row++ {
		m.Span(row, x, x+w, c)
	}
}

// Blit paints src over m with its top left at (x, y). src can be m itself.
func (m *Image) Blit(src *Image, x, y int) {
	if src == m {
		src = &Image{W: m.W, H: m.H, Pix: append([]byte(nil), m.Pix...)}
	}
	for sy := max(0, -y); sy < src.H && sy+y < m.H; sy++ {
		for sx := max(0, -x); sx < src.W && sx+x < m.W; sx++ {
			s := src.Pix[(sy*src.W+sx)*4:]
			if s[3] == 0 {
				continue
			}
			blend(m.Pix[((sy+y)*m.W+sx+x)*4:], color.NRGBA{R: s[0], G: s[1], B: s[2], A: s[3]}, 1)
		}
	}
}

// Resized is a new image of w by h with the pixels of m that fit, and
// transparent pixels where m doesn't reach.
func (m *Image) Resized(w, h int) (*Image, error) {
	out, err := New(w, h, color.NRGBA{})
	if err != nil {
		return nil, err
	}
	for y := 0; y < min(h, m.H); y++ {
		copy(out.Pix[y*w*4:y*w*4+min(w, m.W)*4], m.Pix[y*m.W*4:])
	}
	return out, nil
}

// Premultiplied writes the pixels into dst (len W*H*4) with each color
// multiplied by its alpha, which is what a screen takes.
func (m *Image) Premultiplied(dst []byte) {
	for i := 0; i < len(m.Pix); i += 4 {
		a := m.Pix[i+3]
		switch a {
		case 255:
			copy(dst[i:i+4], m.Pix[i:i+4])
		case 0:
			dst[i], dst[i+1], dst[i+2], dst[i+3] = 0, 0, 0, 0
		default:
			for k := 0; k < 3; k++ {
				dst[i+k] = byte((int(m.Pix[i+k])*int(a) + 127) / 255)
			}
			dst[i+3] = a
		}
	}
}

// NRGBA is the image as a Go image. It shares the pixels.
func (m *Image) NRGBA() *image.NRGBA {
	return &image.NRGBA{Pix: m.Pix, Stride: m.W * 4, Rect: image.Rect(0, 0, m.W, m.H)}
}

// EncodePNG writes the image as a PNG.
func (m *Image) EncodePNG(w io.Writer) error {
	return png.Encode(w, m.NRGBA())
}

// FromImage turns a decoded image into an Image no more than maxSide pixels
// on its longer side (0 keeps its size). A bigger image is shrunk by
// averaging the pixels that fall in each new one; a smaller one is never
// enlarged.
func FromImage(src image.Image, maxSide int) (*Image, error) {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	w, h := sw, sh
	if longer := max(sw, sh); maxSide > 0 && longer > maxSide {
		w = max(1, (sw*maxSide+longer/2)/longer)
		h = max(1, (sh*maxSide+longer/2)/longer)
	}
	out, err := New(w, h, color.NRGBA{})
	if err != nil {
		return nil, err
	}
	for y := 0; y < h; y++ {
		y0, y1 := y*sh/h, max(y*sh/h+1, ((y+1)*sh+h-1)/h)
		for x := 0; x < w; x++ {
			x0, x1 := x*sw/w, max(x*sw/w+1, ((x+1)*sw+w-1)/w)
			// The colors are added premultiplied, so a transparent pixel
			// doesn't darken its neighbors, and divided by the alpha after.
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1 && sy < sh; sy++ {
				for sx := x0; sx < x1 && sx < sw; sx++ {
					cr, cg, cb, ca := src.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
					r, g, bl, a = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca)
					n++
				}
			}
			avg := func(sum uint64) uint64 { return (sum + n/2) / n }
			out.Set(x, y, unpremultiply(avg(r), avg(g), avg(bl), avg(a)))
		}
	}
	return out, nil
}

// unpremultiply turns 16-bit premultiplied channels into a color.
func unpremultiply(r, g, b, a uint64) color.NRGBA {
	if a == 0 {
		return color.NRGBA{}
	}
	channel := func(v uint64) uint8 { return uint8(min(255, (v*255+a/2)/a)) }
	return color.NRGBA{R: channel(r), G: channel(g), B: channel(b), A: uint8((a*255 + 32767) / 65535)}
}

// Target is something a program draws on: an image, or the buffer of a
// window. Drawing functions get its pixels and then say they touched them,
// so a window knows to show them again.
type Target interface {
	Pixels() *Image
	Touched()
}

// Pixels makes an Image a Target of its own.
func (m *Image) Pixels() *Image { return m }

// Touched does nothing for an image: nothing shows it.
func (m *Image) Touched() {}
