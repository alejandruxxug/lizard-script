package window

import (
	"bytes"
	"fmt"
	"image"

	_ "image/jpeg" // registers the JPEG decoder for image.Decode
	_ "image/png"  // registers the PNG decoder for image.Decode

	"github.com/alejandruxxug/lizard-script/tools/bundle"
)

// windowIcon is the picture a window shows in its title bar, the taskbar
// and the Dock: the file as it was read, and the same decoded.
type windowIcon struct {
	data  []byte
	image image.Image
}

// loadIcon reads the icon of path: a PNG or JPEG file on disk or in a built
// program (a bundle: path).
func loadIcon(path string) (*windowIcon, error) {
	data, err := bundle.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't load the icon %s: %v", path, err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("can't load the icon %s: not a PNG or JPEG image", path)
	}
	return &windowIcon{data: data, image: img}, nil
}
