package terrarium

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"lizardscript/bundle"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestImg(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir())
	d := quote(dir)
	check(t, []testCase{
		{"new", `img.new(3, 2)`, "<image 3x2>", false},
		{"type", `type(img.new(1, 1))`, "NATIVE", false},
		{"size", `img.size(img.new(3, 2))`, "(3, 2)", false},
		{"new is transparent", `img.get(img.new(1, 1), 0, 0)`, "#00000000", false},
		{"new with a color", `img.get(img.new(2, 2, #ff0000), 1, 1)`, "#ff0000", false},
		{"set then get keeps alpha", `let m = img.new(2, 2); img.set(m, 1, 0, #12345678); img.get(m, 1, 0)`, "#12345678", false},
		{"pixels", `let m = img.new(2, 1); img.set(m, 1, 0, #ffffff); img.pixels(m)`, "[#00000000, #ffffff]", false},
		{"from pixels", `let m = img.from_pixels((2, 1), [#ff0000, #00ff00]); [img.size(m), img.get(m, 1, 0)]`, "[(2, 1), #00ff00]", false},
		{"identity", `let a = img.new(1, 1); [a == a, a == img.new(1, 1)]`, "[true, false]", false},
		{"save and load", `let m = img.new(2, 2, #336699); img.set(m, 0, 0, #ff000080); let p = fs.join(` + d + `, "a.png"); img.save(m, p); let b = img.load(p); [img.size(b), img.get(b, 0, 0), img.get(b, 1, 1)]`,
			"[(2, 2), #ff000080, #336699]", false},
		{"load shrinks", `let p = fs.join(` + d + `, "big.png"); img.save(img.new(40, 20, #ffffff), p); img.size(img.load(p, 10))`, "(10, 5)", false},
		{"load never enlarges", `let p = fs.join(` + d + `, "small.png"); img.save(img.new(4, 2), p); img.size(img.load(p, 10))`, "(4, 2)", false},

		{"get outside", `img.get(img.new(2, 2), 2, 0)`, "the pixel (2, 0) is outside the 2x2 image", true},
		{"set a non color", `img.set(img.new(1, 1), 0, 0, 5)`, "want a COLOR color, got INTEGER", true},
		{"not an image", `img.size(5)`, "want an image or a window as the target, got INTEGER", true},
		{"get not an image", `img.get("x", 0, 0)`, "want an image, got STRING", true},
		{"new negative", `img.new(-1, 2)`, "the width must be from 0 to 32768, got -1", true},
		{"new too big", `img.new(30000, 30000)`, "the image is too big", true},
		{"from pixels wrong count", `img.from_pixels((2, 2), [#ffffff])`, "a 2x2 image needs 4 colors, got 1", true},
		{"from pixels non color", `img.from_pixels((1, 1), [1])`, "pixel 0 must be a COLOR, got INTEGER", true},
		{"from pixels bad size", `img.from_pixels((1.5, 1), [#ffffff])`, "the size must be two whole numbers", true},
		{"load missing", `img.load(fs.join(` + d + `, "none.png"))`, "can't load", true},
		{"load not an image", `let p = fs.join(` + d + `, "t.png"); fs.write(p, "text"); img.load(p)`, "not a PNG, JPEG or GIF image", true},
		{"load bad side", `img.load("x.png", -1)`, "the max_side must be from 0 to 32768", true},
		{"save into a missing folder", `img.save(img.new(1, 1), fs.join(` + d + `, "no/x.png"))`, "can't save", true},
	})
}

func TestImgLoadKeepsAlphaWhenShrinking(t *testing.T) {
	// Half red, half transparent: the shrunk pixel is red at half alpha, not
	// a darker red.
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	path := filepath.Join(t.TempDir(), "half.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, src); err != nil {
		t.Fatal(err)
	}
	file.Close()
	got := run(t, `img.get(img.load(`+quote(filepath.ToSlash(path))+`, 1), 0, 0)`).Inspect()
	if got != "#ff000080" {
		t.Fatalf("got %s, want #ff000080", got)
	}
}

// A built program's own pictures load like files on disk.
func TestImgLoadFromBundle(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, src); err != nil {
		t.Fatal(err)
	}
	saved := bundle.Current
	defer func() { bundle.Current = saved }()
	bundle.Current = &bundle.Program{Main: "main.liz", Files: fstest.MapFS{
		"main.liz":    {Data: []byte("puts(1)")},
		"art/dot.png": {Data: data.Bytes()},
	}}
	check(t, []testCase{
		{"load", `let m = img.load("bundle:/art/dot.png"); [img.size(m), img.get(m, 1, 0)]`, "[(2, 1), #00ff00]", false},
		{"load missing", `img.load("bundle:/art/nope.png")`, "can't load bundle:/art/nope.png: file does not exist", true},
	})
}

// The drawing functions are the same for every target; the window tests
// check that a window takes them too.
func TestCanvas(t *testing.T) {
	check(t, []testCase{
		{"clear", `let m = img.new(2, 2); img.clear(m, #0000ff); img.pixels(m)`, "[#0000ff, #0000ff, #0000ff, #0000ff]", false},
		{"fill", `let m = img.new(3, 1); img.fill(m, [1, 0, 2, 1], #ff0000); img.pixels(m)`, "[#00000000, #ff0000, #ff0000]", false},
		{"fill mixes", `let m = img.new(1, 1, #000000); img.fill(m, [0, 0, 1, 1], #ffffff80); img.get(m, 0, 0)`, "#808080", false},
		{"fill is cut", `let m = img.new(2, 2); img.fill(m, [-5, -5, 100, 100], #ffffff); img.get(m, 1, 1)`, "#ffffff", false},
		{"span", `let m = img.new(4, 2); img.span(m, 1, 1, 3, #00ff00); [img.get(m, 0, 1), img.get(m, 2, 1), img.get(m, 3, 1), img.get(m, 2, 0)]`,
			"[#00000000, #00ff00, #00000000, #00000000]", false},
		{"span soft edge", `let m = img.new(2, 1, #000000); img.span(m, 0, 0, 1.5, #ffffff); img.pixels(m)`, "[#ffffff, #808080]", false},
		{"spans", `let m = img.new(2, 2); img.spans(m, [[0, 0, 1], [1, 1, 2]], #ffffff); img.pixels(m)`, "[#ffffff, #00000000, #00000000, #ffffff]", false},
		{"blit", `let m = img.new(3, 1); let s = img.new(1, 1, #ff0000); img.blit(m, s, (2, 0)); img.blit(m, s, (-1, 0)); img.pixels(m)`,
			"[#00000000, #00000000, #ff0000]", false},
		{"blit onto itself", `let m = img.new(2, 1); img.set(m, 0, 0, #ffffff); img.blit(m, m, (1, 0)); img.pixels(m)`, "[#ffffff, #ffffff]", false},

		{"bad rect", `img.fill(img.new(1, 1), [0, 0, 1], #ffffff)`, "want a rect [x, y, w, h] of numbers, got [0, 0, 1]", true},
		{"bad span", `img.spans(img.new(1, 1), [[0, 0, 1], [0, "a", 1]], #ffffff)`, "span 1 must be [y, x0, x1] of numbers", true},
		{"span count", `img.span(img.new(1, 1), 0, 0, #ffffff)`, "want 5, got 4", true},
		{"span bad x1", `img.span(img.new(1, 1), 0, 0, "1", #ffffff)`, "want a number x1, got STRING", true},
		{"blit bad source", `img.blit(img.new(1, 1), 1, (0, 0))`, "want an image or a window as the source, got INTEGER", true},
		{"blit bad pos", `img.blit(img.new(1, 1), img.new(1, 1), 0)`, "want a POINT pos, got INTEGER", true},
	})
}

// The pixel-by-pixel functions: clip, gradients, layers, composite, and
// the PNG of a target.
func TestPaint(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir())
	d := quote(dir)
	grad := `{"kind": "linear", "origin": (0, 0), "axis": (0.25, 0), "stops": [[0, #000000], [1, #ffffff]]}`
	check(t, []testCase{
		{"clip", `let m = img.new(4, 1); img.clip(m, [1, 0, 2, 1]); img.fill(m, [0, 0, 4, 1], #ffffff); img.pixels(m)`,
			"[#00000000, #ffffff, #ffffff, #00000000]", false},
		{"unclip", `let m = img.new(2, 1); img.clip(m, [0, 0, 1, 1]); img.unclip(m); img.clear(m, #ff0000); img.pixels(m)`, "[#ff0000, #ff0000]", false},
		{"clear a rect", `let m = img.new(3, 1, #ffffff); img.clear(m, #00000000, [1, 0, 1, 1]); img.pixels(m)`, "[#ffffff, #00000000, #ffffff]", false},
		{"spans offset", `let m = img.new(3, 2); img.spans(m, [[0, 0, 1]], #ffffff, (2, 1)); img.get(m, 2, 1)`, "#ffffff", false},
		{"span gradient", `let m = img.new(4, 1); img.span_gradient(m, 0, 0, 4, ` + grad + `); img.pixels(m)`,
			"[#202020, #606060, #9f9f9f, #dfdfdf]", false},
		{"spans gradient keeps its place", `let m = img.new(4, 2); img.spans_gradient(m, [[0, 0, 2]], ` + grad + `, (2, 1)); [img.get(m, 2, 1), img.get(m, 3, 1)]`,
			"[#202020, #606060]", false},
		{"steps", `let m = img.new(4, 1); img.span_gradient(m, 0, 0, 4, {"kind": "linear", "origin": (0, 0), "axis": (0.25, 0), "stops": [[0, #000000], [1, #ffffff]], "steps": 2}); img.pixels(m)`,
			"[#000000, #000000, #ffffff, #ffffff]", false},
		{"radial", `let m = img.new(3, 1); img.span_gradient(m, 0, 0, 3, {"kind": "radial", "origin": (1.5, 0.5), "scale": 1, "stops": [[0, #ff0000], [1, #0000ff]]}); img.pixels(m)`,
			"[#0000ff, #ff0000, #0000ff]", false},
		{"layer", `let m = img.new(1, 1, #808080); img.push_layer(m); img.clear(m, #ffffff); let inside = img.get(m, 0, 0); img.pop_layer(m, "multiply", 1); [inside, img.get(m, 0, 0)]`,
			"[#ffffff, #808080]", false},
		{"layer opacity", `let m = img.new(1, 1, #000000); img.push_layer(m); img.clear(m, #ffffff); img.pop_layer(m, "normal", 0.5); img.get(m, 0, 0)`, "#808080", false},
		{"layers nest", `let m = img.new(1, 1, #000000); img.push_layer(m); img.push_layer(m); img.clear(m, #ff0000); img.pop_layer(m); img.pop_layer(m, "screen"); img.get(m, 0, 0)`, "#ff0000", false},
		{"composite", `let m = img.new(2, 1, #808080); img.composite(m, img.new(1, 1, #ffffff), "add", 1, (1, 0)); img.pixels(m)`, "[#808080, #ffffff]", false},
		{"png base64", `let s = img.png_base64(img.new(1, 1, #ff0000)); [type(s), s[0] + s[1] + s[2] + s[3]]`, `["STRING", "iVBO"]`, false},
		{"save a rect", `let p = fs.join(` + d + `, "part.png"); let m = img.new(4, 4); img.set(m, 2, 1, #00ff00); img.save(m, p, [2, 1, 2, 2]); let b = img.load(p); [img.size(b), img.get(b, 0, 0)]`,
			"[(2, 2), #00ff00]", false},

		{"bad gradient kind", `img.span_gradient(img.new(1, 1), 0, 0, 1, {"kind": "conic"})`, `bad gradient: "kind" must be "linear" or "radial"`, true},
		{"gradient no axis", `img.span_gradient(img.new(1, 1), 0, 0, 1, {"kind": "linear", "origin": (0, 0)})`, `a linear gradient needs a point "axis"`, true},
		{"gradient bad stop", `img.span_gradient(img.new(1, 1), 0, 0, 1, {"kind": "radial", "origin": (0, 0), "scale": 1, "stops": [[0, 1]]})`, "stop 0 must be [t, color], got [0, 1]", true},
		{"pop with no layer", `img.pop_layer(img.new(1, 1))`, "no layer to pop: call push_layer first", true},
		{"unknown mode", `let m = img.new(1, 1); img.push_layer(m); img.pop_layer(m, "burn")`, `unknown blend mode "burn" (the modes are add, darken, lighten, multiply, normal, overlay, screen)`, true},
		{"bad opacity", `img.composite(img.new(1, 1), img.new(1, 1), "normal", 2)`, "the opacity must be from 0 to 1, got 2", true},
		{"rect outside", `img.png_base64(img.new(1, 1), [5, 5, 1, 1])`, "the rect [5, 5, 1, 1] is outside the 1x1 target", true},
	})
}
