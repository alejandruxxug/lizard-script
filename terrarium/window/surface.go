package window

import (
	"image"
	"image/color"
	"lizardscript/object"
	"lizardscript/terrarium"
	"lizardscript/terrarium/internal/arg"
	"lizardscript/terrarium/internal/canvas"
	"lizardscript/terrarium/internal/pixels"
	"math"
	"time"
)

// Surfaces. A window shows its own buffer, at its scale, and over it the
// surfaces it has, in the order they were made: each one a buffer of its
// own shown in a rectangle of the window, at a scale of its own. A window
// at scale 2 with a surface at scale 1 shows the surface at the screen's
// full resolution: sharp drawing in the middle of a big-pixel interface.
//
// The frame the screen gets is composed from them, at the finest scale
// they share, and only where something changed: each changed rectangle on
// its own, so two small changes far apart don't cost the window between.

// SurfaceKind is the kind of the natives that are surfaces.
const SurfaceKind = "surface"

// Surface is a buffer shown in a rectangle of its window.
type Surface struct {
	win    *Window
	name   string
	native *object.Native // the value the program has, the same every time
	buf    *pixels.Image
	rect   image.Rectangle // where it shows, in the window's buffer pixels
	scale  int             // screen pixels for one of its pixels
	dirt   dirt
}

// Base, Pixels and Touched make a surface a drawing target.
func (s *Surface) Base() *pixels.Image   { return s.buf }
func (s *Surface) Pixels() *pixels.Image { return s.buf.Top() }
func (s *Surface) Touched()              { s.dirt.touched, s.win.drew = true, true }

// Describe is the name and size, for Inspect: <surface canvas 800x600>.
func (s *Surface) Describe() string { return s.name + " " + s.buf.Describe() }

// dirt is what changed in a buffer since the frame was last composed:
// nothing, the whole buffer, or the rectangles the program declared with
// window.dirty.
type dirt struct {
	touched bool
	rects   []image.Rectangle // declared; none means the whole buffer
}

// regions is what changed, in the buffer's pixels of size full.
func (d dirt) regions(full image.Rectangle) []image.Rectangle {
	if !d.touched {
		return nil
	}
	if len(d.rects) == 0 {
		return []image.Rectangle{full}
	}
	return d.rects
}

// screen is how a window's frame is made from its buffer and surfaces.
type screen struct {
	surfaces []*Surface
	unit     int           // screen pixels for one pixel of the frame
	frame    *pixels.Image // the composed frame; nil with no surfaces
	all      bool          // compose the whole frame next time
	base     dirt          // the window buffer's
}

// layout works out the unit and the size of every surface's buffer again,
// after the window's size or scale changed, or a surface came or went.
func (w *Window) layout() {
	sc := &w.screen
	sc.unit = w.scale
	for _, s := range sc.surfaces {
		sc.unit = gcd(sc.unit, s.scale)
		s.fit()
	}
	sc.all = true
	if len(sc.surfaces) == 0 {
		sc.frame = nil
		return
	}
	fw, fh := w.frameSize()
	if sc.frame == nil || sc.frame.W != fw || sc.frame.H != fh {
		sc.frame, _ = pixels.New(fw, fh, black)
	}
}

// frameSize is the size of the composed frame: the window's buffer, in
// pixels of the unit. With no surfaces it is the buffer's size.
func (w *Window) frameSize() (int, int) {
	return w.buf.W * w.scale / w.screen.unit, w.buf.H * w.scale / w.screen.unit
}

// screenSize is the size of the window on the screen, in screen pixels.
func (w *Window) screenSize() (int, int) { return w.buf.W * w.scale, w.buf.H * w.scale }

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// fit gives the surface a buffer that covers its rectangle at its scale,
// keeping the pixels that still fit.
func (s *Surface) fit() {
	w := s.win
	bw := (s.rect.Dx()*w.scale + s.scale - 1) / s.scale
	bh := (s.rect.Dy()*w.scale + s.scale - 1) / s.scale
	if s.buf != nil && s.buf.W == bw && s.buf.H == bh {
		return
	}
	if s.buf == nil {
		s.buf, _ = pixels.New(bw, bh, transparent)
	} else if bigger, err := s.buf.Resized(bw, bh); err == nil {
		s.buf = bigger
	}
	s.dirt.touched = true
}

// transparent is the color of a new surface.
var transparent = color.NRGBA{}

// onScreen is where the surface shows, in screen pixels.
func (s *Surface) onScreen() image.Rectangle {
	k := s.win.scale
	return image.Rect(s.rect.Min.X*k, s.rect.Min.Y*k, s.rect.Max.X*k, s.rect.Max.Y*k)
}

// spotAt is the surface under the screen point (sx, sy), the newest one on
// top, and the point in its pixels.
func (w *Window) spotAt(sx, sy float64) Spot {
	surfaces := w.screen.surfaces
	for i := len(surfaces) - 1; i >= 0; i-- {
		s := surfaces[i]
		r := s.onScreen()
		if sx >= float64(r.Min.X) && sy >= float64(r.Min.Y) && sx < float64(r.Max.X) && sy < float64(r.Max.Y) {
			k := float64(s.scale)
			return Spot{Name: s.name, X: int(math.Floor((sx - float64(r.Min.X)) / k)), Y: int(math.Floor((sy - float64(r.Min.Y)) / k))}
		}
	}
	return Spot{}
}

// maxSurfaces is how many surfaces a window can have.
const maxSurfaces = 64

// window.surface(w, name, rect, scale = 1) is the window's surface called
// name, shown in rect, [x, y, w, h] in the window's buffer pixels, with
// each of its pixels scale by scale screen pixels. It is made the first
// time, and moved or resized after: the same value every time. An empty
// rect removes it and gives null.
func surfaceFunc(args ...object.Object) object.Object {
	if err := arg.Count(args, 3, 4); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	name, err := arg.String(args, 1, "name")
	if err != nil {
		return err
	}
	r, err := canvas.RectArg(args, 2)
	if err != nil {
		return err
	}
	scale := 1
	if len(args) == 4 {
		if scale, err = arg.IntIn(args, 3, "scale", 1, 8); err != nil {
			return err
		}
	}
	s, i := w.surface(name)
	if r.Empty() {
		if s != nil {
			w.screen.surfaces = append(w.screen.surfaces[:i], w.screen.surfaces[i+1:]...)
			w.layout()
		}
		return arg.Null
	}
	if r.Dx() > maxSide || r.Dy() > maxSide {
		return arg.Errorf("the rect %s is bigger than %d on a side", args[2].Inspect(), maxSide)
	}
	if s == nil {
		if len(w.screen.surfaces) >= maxSurfaces {
			return arg.Errorf("a window can have at most %d surfaces", maxSurfaces)
		}
		s = &Surface{win: w, name: name}
		s.native = &object.Native{Kind: SurfaceKind, Value: s}
		w.screen.surfaces = append(w.screen.surfaces, s)
	}
	if s.rect != r || s.scale != scale || s.buf == nil {
		s.rect, s.scale = r, scale
		w.layout()
	}
	return s.native
}

// surface is the window's surface called name and its place, or nil.
func (w *Window) surface(name string) (*Surface, int) {
	for i, s := range w.screen.surfaces {
		if s.name == name {
			return s, i
		}
	}
	return nil, -1
}

// window.dirty(target, rect) says which part of a window or one of its
// surfaces changed this frame. Without it, a target drawn on counts as
// changed everywhere; with it, only the rectangles declared are shown
// again, so a program that redraws a small part saves the rest.
func dirty(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 2); err != nil {
		return err
	}
	var d *dirt
	if n, ok := args[0].(*object.Native); ok {
		switch v := n.Value.(type) {
		case *Window:
			d = &v.screen.base
		case *Surface:
			d = &v.dirt
		}
	}
	if d == nil {
		return arg.Errorf("want a window or a surface, got %s", arg.Name(args[0]))
	}
	r, err := canvas.RectArg(args, 1)
	if err != nil {
		return err
	}
	d.touched = true
	d.rects = append(d.rects, r)
	return arg.Null
}

// compose brings the frame up to date and gives the rectangles of it that
// changed, in frame pixels. With no surfaces there is no frame: the buffer
// is shown as it is.
func (w *Window) compose() []image.Rectangle {
	sc := &w.screen
	if sc.frame == nil {
		return nil
	}
	full := sc.frame.Bounds()
	var regions []image.Rectangle
	if sc.all {
		regions = []image.Rectangle{full}
	} else {
		k := w.scale
		for _, r := range sc.base.regions(w.buf.Bounds()) {
			regions = append(regions, w.toFrame(r, k, image.Point{}))
		}
		for _, s := range sc.surfaces {
			origin := s.onScreen().Min
			for _, r := range s.dirt.regions(s.buf.Bounds()) {
				regions = append(regions, w.toFrame(r, s.scale, origin))
			}
		}
	}
	sc.all = false
	sc.base = dirt{}
	for _, s := range sc.surfaces {
		s.dirt = dirt{}
	}
	start := time.Now()
	var changed []image.Rectangle
	for _, r := range disjoint(regions) {
		if r = r.Intersect(full); !r.Empty() {
			w.composeInto(sc.frame, r)
			changed = append(changed, r)
		}
	}
	if len(changed) > 0 {
		w.stats.composeMS = float64(time.Since(start).Microseconds()) / 1000
	}
	return changed
}

// toFrame turns r, in pixels of scale screen pixels at origin (in screen
// pixels), into frame pixels, rounded out.
func (w *Window) toFrame(r image.Rectangle, scale int, origin image.Point) image.Rectangle {
	u := w.screen.unit
	down := func(v int) int { return int(math.Floor(float64(v) / float64(u))) }
	up := func(v int) int { return int(math.Ceil(float64(v) / float64(u))) }
	return image.Rect(down(r.Min.X*scale+origin.X), down(r.Min.Y*scale+origin.Y),
		up(r.Max.X*scale+origin.X), up(r.Max.Y*scale+origin.Y))
}

// disjoint joins the rectangles that overlap until none do, so no pixel is
// composed twice.
func disjoint(rs []image.Rectangle) []image.Rectangle {
	var out []image.Rectangle
	for _, r := range rs {
		if r.Empty() {
			continue
		}
		for joined := true; joined; {
			joined = false
			for i := 0; i < len(out); i++ {
				if out[i].Overlaps(r) {
					r = r.Union(out[i])
					out = append(out[:i], out[i+1:]...)
					joined = true
					break
				}
			}
		}
		out = append(out, r)
	}
	return out
}

// composeInto makes the pixels of region r of the frame dst: the window's
// buffer scaled up, then every surface over it, in order.
func (w *Window) composeInto(dst *pixels.Image, r image.Rectangle) {
	u, k := w.screen.unit, w.scale
	base := w.buf
	for y := r.Min.Y; y < r.Max.Y; y++ {
		by := min(y*u/k, base.H-1)
		row := dst.Pix[y*dst.W*4+r.Min.X*4 : y*dst.W*4+r.Max.X*4]
		if y > r.Min.Y && by == min((y-1)*u/k, base.H-1) {
			// The same buffer row as the frame row above: copy that one.
			copy(row, dst.Pix[(y-1)*dst.W*4+r.Min.X*4:])
			continue
		}
		src := base.Pix[by*base.W*4:]
		for x := r.Min.X; x < r.Max.X; x++ {
			o, d := min(x*u/k, base.W-1)*4, (x-r.Min.X)*4
			row[d], row[d+1], row[d+2], row[d+3] = src[o], src[o+1], src[o+2], src[o+3]
		}
	}
	for _, s := range w.screen.surfaces {
		shown := s.onScreen()
		area := image.Rect(shown.Min.X/u, shown.Min.Y/u, shown.Max.X/u, shown.Max.Y/u).Intersect(r)
		for y := area.Min.Y; y < area.Max.Y; y++ {
			sy := (y*u - shown.Min.Y) / s.scale
			if sy >= s.buf.H {
				break
			}
			row := dst.Pix[y*dst.W*4:]
			for x := area.Min.X; x < area.Max.X; x++ {
				sx := (x*u - shown.Min.X) / s.scale
				if sx >= s.buf.W {
					break
				}
				pixels.Over(row[x*4:], s.buf.Pix[(sy*s.buf.W+sx)*4:])
			}
		}
	}
}

// shot is the window as it shows on the screen: its buffer with the
// surfaces over it, in frame pixels.
func (w *Window) shot() *pixels.Image {
	if len(w.screen.surfaces) == 0 {
		return w.buf.Crop(w.buf.Bounds())
	}
	fw, fh := w.frameSize()
	out, _ := pixels.New(fw, fh, black)
	w.composeInto(out, out.Bounds())
	return out
}

// window.screenshot(w) is an image of the window as it shows: its buffer
// with the surfaces over it, at the finest scale they share (the buffer's
// own size when it has no surfaces).
func screenshot(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	return terrarium.NewImage(w.shot())
}

// describe is a surface for automation's Dump.
func (s *Surface) describe() map[string]any {
	return map[string]any{
		"name":  s.name,
		"rect":  []int{s.rect.Min.X, s.rect.Min.Y, s.rect.Dx(), s.rect.Dy()},
		"scale": s.scale,
		"size":  []int{s.buf.W, s.buf.H},
	}
}
