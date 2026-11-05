package window

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alejandruxxug/lizard-script/engine/object"
)

// writeIcon writes a 32 by 32 PNG and returns its path.
func writeIcon(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "icon.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadIcon(t *testing.T) {
	icon, err := loadIcon(writeIcon(t))
	if err != nil || icon.image.Bounds().Dx() != 32 || len(icon.data) == 0 {
		t.Fatalf("got %+v, %v", icon, err)
	}
	text := filepath.Join(t.TempDir(), "x.png")
	os.WriteFile(text, []byte("no"), 0o644)
	for _, tt := range []struct{ path, want string }{
		{filepath.Join(t.TempDir(), "none.png"), "can't load the icon"},
		{text, "not a PNG or JPEG image"},
	} {
		if _, err := loadIcon(tt.path); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: got %v, want %q", tt.path, err, tt.want)
		}
	}
}

func TestIconOption(t *testing.T) {
	path := writeIcon(t)
	for _, call := range []string{
		`window.open(1, 1, "", {"icon": %q})`,
		`window.set(window.open(1, 1, ""), {"icon": %q})`,
	} {
		if got := runProgram(t, fmt.Sprintf(call, path), nil); got.Type() == object.ERROR_OBJ {
			t.Errorf("%s: %s", call, got.Inspect())
		}
	}
	got := runProgram(t, `window.open(1, 1, "", {"icon": "nope.png"})`, nil)
	if err, ok := got.(*object.Error); !ok || !strings.Contains(err.Message, "can't load the icon nope.png") {
		t.Errorf("got %s", got.Inspect())
	}
	got = runProgram(t, `window.open(1, 1, "", {"icon": 5})`, nil)
	if err, ok := got.(*object.Error); !ok || !strings.Contains(err.Message, "want a STRING icon") {
		t.Errorf("got %s", got.Inspect())
	}
}
