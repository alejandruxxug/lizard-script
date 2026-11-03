package main

import (
	"fmt"
	"os"
	"strings"
)

// patch is the one change the copy has. Ebitengine creates its Metal
// device in an init function, on the main thread, so every program that
// links it pays for the device (about 17 ms of a 26 ms start on an M4), with
// a window or without. On macOS NewGraphics runs on the main thread too
// (Ebitengine's own comment there says so), so the device is created there
// the first time; on iOS, where it doesn't, init still creates it.
var patch = struct {
	file  string
	edits []edit
}{
	file: "internal/graphicsdriver/metal/graphics_darwin.go",
	edits: []edit{
		{"\t\"slices\"\n\t\"unsafe\"\n", "\t\"slices\"\n\t\"sync\"\n\t\"unsafe\"\n"},
		{`var (
	systemDefaultDevice    mtl.Device
	systemDefaultDeviceErr error
)

func init() {
	// mtl.CreateSystemDefaultDevice must be called on the main thread (#2147).
	d, err := mtl.CreateSystemDefaultDevice()
	if err != nil {
		systemDefaultDeviceErr = err
		return
	}
	systemDefaultDevice = d
}
`, `var (
	systemDefaultDevice     mtl.Device
	systemDefaultDeviceErr  error
	systemDefaultDeviceOnce sync.Once
)

// createSystemDefaultDevice creates the device, once.
// mtl.CreateSystemDefaultDevice must be called on the main thread (#2147).
func createSystemDefaultDevice() {
	systemDefaultDeviceOnce.Do(func() {
		d, err := mtl.CreateSystemDefaultDevice()
		if err != nil {
			systemDefaultDeviceErr = err
			return
		}
		systemDefaultDevice = d
	})
}

func init() {
	// On iOS, NewGraphics and Initialize don't run on the main thread, so the device is created here.
	// On macOS, NewGraphics runs on the main thread and creates it there: a program that links
	// Ebitengine but never runs a game doesn't pay for the device when it starts.
	if runtime.GOOS == "ios" {
		createSystemDefaultDevice()
	}
}
`},
		{`func NewGraphics(colorSpace color.ColorSpace) (graphicsdriver.Graphics, error) {
`, `func NewGraphics(colorSpace color.ColorSpace) (graphicsdriver.Graphics, error) {
	createSystemDefaultDevice()

`},
	},
}

// edit replaces the text old, which must be there once, with new.
type edit struct{ old, new string }

// apply makes the edits on text, or says which one doesn't fit it: a new
// version of Ebitengine needs the patch looked at again.
func apply(text string, edits []edit) (string, error) {
	for i, e := range edits {
		if n := strings.Count(text, e.old); n != 1 {
			return "", fmt.Errorf("edit %d of the patch is found %d times, not once: look at it again for this version", i+1, n)
		}
		text = strings.Replace(text, e.old, e.new, 1)
	}
	return text, nil
}

// patchFile applies the patch to the copy of its file.
func patchFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text, err := apply(string(data), patch.edits)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
