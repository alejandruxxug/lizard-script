package build

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
)

// iconSet is the app icon the build was given: a square master, big enough
// to be shrunk to every size an icon needs, and optionally a hand-drawn
// small one for 16 pixels, where a shrunk master turns to mush.
type iconSet struct {
	master    *image.NRGBA
	masterPNG []byte // the file as it was given, used as is at its own size
	small     *image.NRGBA
	cache     map[int][]byte
}

// minIconSide is the smallest master: the biggest icon of Windows is 256.
const minIconSide = 256

// loadIcons reads the master and, when small is not "", the small icon.
func loadIcons(master, small string) (*iconSet, error) {
	data, err := os.ReadFile(master)
	if err != nil {
		return nil, fmt.Errorf("--icon: %w", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("--icon %s: not a PNG image: %v", master, err)
	}
	b := img.Bounds()
	if b.Dx() != b.Dy() || b.Dx() < minIconSide {
		return nil, fmt.Errorf("--icon %s: is %dx%d, it must be square and at least %dx%d", master, b.Dx(), b.Dy(), minIconSide, minIconSide)
	}
	set := &iconSet{master: toNRGBA(img), masterPNG: data, cache: map[int][]byte{}}
	if small == "" {
		return set, nil
	}
	file, err := os.Open(small)
	if err != nil {
		return nil, fmt.Errorf("--icon-small: %w", err)
	}
	defer file.Close()
	img, err = png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("--icon-small %s: not a PNG image: %v", small, err)
	}
	if b := img.Bounds(); b.Dx() != b.Dy() {
		return nil, fmt.Errorf("--icon-small %s: is %dx%d, it must be square", small, b.Dx(), b.Dy())
	}
	set.small = toNRGBA(img)
	return set, nil
}

// side is the side of the master in pixels.
func (s *iconSet) side() int { return s.master.Bounds().Dx() }

// png is the icon at size by size pixels, as a PNG file. Sizes the master
// can't fill (it is smaller) are an error; ask only for what has() says.
func (s *iconSet) png(size int) []byte {
	if data, ok := s.cache[size]; ok {
		return data
	}
	src := s.master
	if size <= 16 && s.small != nil {
		src = s.small
	}
	data := s.masterPNG
	if src != s.master || size != s.side() {
		var out bytes.Buffer
		png.Encode(&out, resize(src, size))
		data = out.Bytes()
	}
	s.cache[size] = data
	return data
}

// has reports whether an icon of size can be made: the master is at least
// that big.
func (s *iconSet) has(size int) bool { return size <= s.side() }

// toNRGBA copies an image into the plain 8-bit form the resizer reads.
func toNRGBA(img image.Image) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	return out
}

// weight is how much one source pixel counts to a destination pixel.
type weight struct {
	index int
	w     float64
}

// weights lists, for each of the dst pixels of a line, the src pixels it
// covers and the part of each one it covers.
func weights(src, dst int) [][]weight {
	scale := float64(src) / float64(dst)
	all := make([][]weight, dst)
	for i := range all {
		from, to := float64(i)*scale, float64(i+1)*scale
		for p := int(from); p < src && float64(p) < to; p++ {
			overlap := min(to, float64(p+1)) - max(from, float64(p))
			if overlap > 0 {
				all[i] = append(all[i], weight{p, overlap / scale})
			}
		}
	}
	return all
}

// resize shrinks (or stretches) src to a square of size pixels by averaging
// the pixels each new one covers, with the colors weighted by their alpha so
// a clear edge doesn't darken the picture.
func resize(src *image.NRGBA, size int) *image.NRGBA {
	b := src.Bounds()
	if b.Dx() == size && b.Dy() == size {
		return src
	}
	wx, wy := weights(b.Dx(), size), weights(b.Dy(), size)
	// The first pass goes across, to rows of premultiplied floats.
	mid := make([]float64, size*b.Dy()*4)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < size; x++ {
			var r, g, bl, a float64
			for _, c := range wx[x] {
				i := src.PixOffset(b.Min.X+c.index, b.Min.Y+y)
				pa := float64(src.Pix[i+3]) / 255
				r += float64(src.Pix[i]) * pa * c.w
				g += float64(src.Pix[i+1]) * pa * c.w
				bl += float64(src.Pix[i+2]) * pa * c.w
				a += pa * c.w
			}
			o := (y*size + x) * 4
			mid[o], mid[o+1], mid[o+2], mid[o+3] = r, g, bl, a
		}
	}
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, bl, a float64
			for _, c := range wy[y] {
				o := (c.index*size + x) * 4
				r += mid[o] * c.w
				g += mid[o+1] * c.w
				bl += mid[o+2] * c.w
				a += mid[o+3] * c.w
			}
			i := out.PixOffset(x, y)
			if a > 0 {
				out.Pix[i] = clamp(r / a)
				out.Pix[i+1] = clamp(g / a)
				out.Pix[i+2] = clamp(bl / a)
			}
			out.Pix[i+3] = clamp(a * 255)
		}
	}
	return out
}

// clamp rounds v to a byte.
func clamp(v float64) uint8 {
	return uint8(max(0, min(255, v+0.5)))
}
