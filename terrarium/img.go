package terrarium

import (
	"bytes"
	"encoding/base64"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/canvas"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/pixels"
	"github.com/alejandruxxug/lizard-script/tools/bundle"
	"image"
	"image/color"
	_ "image/gif"  // registers the GIF decoder for image.Decode
	_ "image/jpeg" // registers the JPEG decoder
	_ "image/png"  // registers the PNG decoder
)

// img: images. An image is a NATIVE value of kind "image" that holds its
// pixels; the program reads and changes them one by one, or all at once as
// a flat list of colors. Colors are kept as they are set, alpha included.
//
// An image is a drawing target, like a window or a surface: img.size,
// clear, fill, span, spans, blit, clip, the gradient spans and the blend
// layers are the shared functions of package canvas. While a layer is open
// on an image, every function that takes it uses the layer.

// ImageKind is the kind of the natives that are images.
const ImageKind = "image"

// maxSide is the biggest width or height of an image.
const maxSide = 1 << 15

func imgModule() *object.Hash {
	return arg.Module("img", canvas.Add(map[string]object.BuiltinFunction{
		"load":        imgLoad,
		"save":        imgSave,
		"new":         imgNew,
		"get":         imgGet,
		"set":         imgSet,
		"pixels":      imgPixels,
		"from_pixels": imgFromPixels,
		"png_base64":  imgPNGBase64,
	}))
}

// NewImage wraps pixels as the lizard-script value of an image.
func NewImage(m *pixels.Image) *object.Native {
	return &object.Native{Kind: ImageKind, Value: m}
}

// imageArg reads argument i as an image.
func imageArg(args []object.Object, i int) (*pixels.Image, *object.Error) {
	v, err := arg.Native(args, i, ImageKind)
	if err != nil {
		return nil, err
	}
	return v.(*pixels.Image), nil
}

// img.load(path, max_side = 0) reads a PNG, JPEG or GIF file, also one of
// a built program's own (a bundle: path). With a max_side above 0, an image
// longer than that on its longer side is shrunk to fit (never enlarged).
func imgLoad(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	path, err := arg.String(args, 0, "path")
	if err != nil {
		return err
	}
	side := 0
	if len(args) == 2 {
		if side, err = arg.IntIn(args, 1, "max_side", 0, maxSide); err != nil {
			return err
		}
	}
	data, rerr := bundle.ReadFile(path)
	if rerr != nil {
		return systemError("load", path, rerr)
	}
	config, _, derr := image.DecodeConfig(bytes.NewReader(data))
	if derr != nil {
		return arg.Errorf("can't load %s: not a PNG, JPEG or GIF image", path)
	}
	if config.Width > maxSide || config.Height > maxSide {
		return arg.Errorf("can't load %s: %dx%d is bigger than %d on a side", path, config.Width, config.Height, maxSide)
	}
	decoded, _, derr := image.Decode(bytes.NewReader(data))
	if derr != nil {
		return arg.Errorf("can't load %s: %v", path, derr)
	}
	m, ferr := pixels.FromImage(decoded, side)
	if ferr != nil {
		return arg.Errorf("can't load %s: %v", path, ferr)
	}
	return NewImage(m)
}

// img.save(target, path, rect = all) writes an image, a window or a
// surface, or the rectangle rect of it, as a PNG file, the whole file at
// once like fs.write.
func imgSave(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 3); err != nil {
		return err
	}
	path, err := arg.String(args, 1, "path")
	if err != nil {
		return err
	}
	data, err := encodePNG(args, 2)
	if err != nil {
		return err
	}
	if werr := WriteFileAtomic(path, data); werr != nil {
		return systemError("save", path, werr)
	}
	return arg.Null
}

// img.png_base64(target, rect = all) is the PNG of a target, or of the
// rectangle rect of it, as base64 text: a picture another program can take
// in JSON.
func imgPNGBase64(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	data, err := encodePNG(args, 1)
	if err != nil {
		return err
	}
	return arg.Str(base64.StdEncoding.EncodeToString(data))
}

// encodePNG is the PNG of the target in argument 0, cut to the rectangle in
// argument rectAt when there is one.
func encodePNG(args []object.Object, rectAt int) ([]byte, *object.Error) {
	t, err := canvas.TargetArg(args, 0, "target")
	if err != nil {
		return nil, err
	}
	m := t.Pixels()
	if len(args) > rectAt {
		r, err := canvas.RectArg(args, rectAt)
		if err != nil {
			return nil, err
		}
		if m = m.Crop(r); m.W == 0 || m.H == 0 {
			return nil, arg.Errorf("the rect %s is outside the %dx%d target", args[rectAt].Inspect(), t.Pixels().W, t.Pixels().H)
		}
	}
	var data bytes.Buffer
	if eerr := m.EncodePNG(&data); eerr != nil {
		return nil, arg.Errorf("can't make the PNG: %v", eerr)
	}
	return data.Bytes(), nil
}

// img.new(w, h, color = #00000000) is a new image filled with color.
func imgNew(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 3); err != nil {
		return err
	}
	w, err := arg.IntIn(args, 0, "width", 0, maxSide)
	if err != nil {
		return err
	}
	h, err := arg.IntIn(args, 1, "height", 0, maxSide)
	if err != nil {
		return err
	}
	fill := color.NRGBA{}
	if len(args) == 3 {
		if fill, err = arg.Color(args, 2, "color"); err != nil {
			return err
		}
	}
	m, nerr := pixels.New(w, h, fill)
	if nerr != nil {
		return arg.Errorf("%v", nerr)
	}
	return NewImage(m)
}

// pixelArgs reads (image, x, y) and checks that the pixel is in the image.
func pixelArgs(args []object.Object) (*pixels.Image, int, int, *object.Error) {
	base, err := imageArg(args, 0)
	if err != nil {
		return nil, 0, 0, err
	}
	m := base.Pixels()
	x, err := arg.Int(args, 1, "x")
	if err != nil {
		return nil, 0, 0, err
	}
	y, err := arg.Int(args, 2, "y")
	if err != nil {
		return nil, 0, 0, err
	}
	if x < 0 || y < 0 || x >= int64(m.W) || y >= int64(m.H) {
		return nil, 0, 0, arg.Errorf("the pixel (%d, %d) is outside the %dx%d image", x, y, m.W, m.H)
	}
	return m, int(x), int(y), nil
}

// img.get(image, x, y) is the color of a pixel.
func imgGet(args ...object.Object) object.Object {
	if err := arg.Count(args, 3, 3); err != nil {
		return err
	}
	m, x, y, err := pixelArgs(args)
	if err != nil {
		return err
	}
	return arg.ColorOf(m.At(x, y))
}

// img.set(image, x, y, color) replaces a pixel with color (it doesn't mix
// it with the one below).
func imgSet(args ...object.Object) object.Object {
	if err := arg.Count(args, 4, 4); err != nil {
		return err
	}
	m, x, y, err := pixelArgs(args)
	if err != nil {
		return err
	}
	c, err := arg.Color(args, 3, "color")
	if err != nil {
		return err
	}
	m.Set(x, y, c)
	return arg.Null
}

// img.pixels(image) is every color of the image in one flat list, row by
// row from the top left: the pixel (x, y) is at y * width + x.
func imgPixels(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	base, err := imageArg(args, 0)
	if err != nil {
		return err
	}
	m := base.Pixels()
	out := &object.Array{Elements: make([]object.Object, m.W*m.H)}
	for i := range out.Elements {
		p := m.Pix[i*4:]
		out.Elements[i] = &object.Color{R: p[0], G: p[1], B: p[2], A: p[3]}
	}
	return out
}

// img.from_pixels(size, pixels) is a new image of size (w, h) made from a
// flat list of w * h colors, in the order img.pixels gives them.
func imgFromPixels(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 2); err != nil {
		return err
	}
	fw, fh, err := arg.Point(args, 0, "size")
	if err != nil {
		return err
	}
	w, h := int(fw), int(fh)
	if float64(w) != fw || float64(h) != fh || w < 0 || h < 0 || w > maxSide || h > maxSide {
		return arg.Errorf("the size must be two whole numbers from 0 to %d, got %s", maxSide, args[0].Inspect())
	}
	list, err := arg.Array(args, 1, "list of colors")
	if err != nil {
		return err
	}
	if len(list.Elements) != w*h {
		return arg.Errorf("a %dx%d image needs %d colors, got %d", w, h, w*h, len(list.Elements))
	}
	m, nerr := pixels.New(w, h, color.NRGBA{})
	if nerr != nil {
		return arg.Errorf("%v", nerr)
	}
	for i, e := range list.Elements {
		c, ok := e.(*object.Color)
		if !ok {
			return arg.Errorf("pixel %d must be a COLOR, got %s", i, arg.Name(e))
		}
		p := m.Pix[i*4:]
		p[0], p[1], p[2], p[3] = c.R, c.G, c.B, c.A
	}
	return NewImage(m)
}
