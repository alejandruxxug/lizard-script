package build

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alejandruxxug/lizard-script/tools/bundle"
)

// writePNG writes a side by side PNG filled with c, with a clear left half
// when halfClear is true, and returns its path.
func writePNG(t *testing.T, side int, c color.NRGBA, halfClear bool) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			if !halfClear || x >= side/2 {
				img.SetNRGBA(x, y, c)
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	path := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func sideOf(t *testing.T, data []byte) int {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img.Bounds().Dx()
}

func TestLoadIconsErrors(t *testing.T) {
	notSquare := filepath.Join(t.TempDir(), "wide.png")
	var buf bytes.Buffer
	png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 512, 256)))
	os.WriteFile(notSquare, buf.Bytes(), 0o644)
	text := filepath.Join(t.TempDir(), "x.png")
	os.WriteFile(text, []byte("no"), 0o644)
	for _, tt := range []struct{ master, small, want string }{
		{filepath.Join(t.TempDir(), "none.png"), "", "--icon:"},
		{text, "", "not a PNG"},
		{notSquare, "", "must be square"},
		{writePNG(t, 128, color.NRGBA{A: 255}, false), "", "at least 256x256"},
		{writePNG(t, 512, color.NRGBA{A: 255}, false), text, "--icon-small"},
	} {
		if _, err := loadIcons(tt.master, tt.small); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: got %v, want %q", tt.master, err, tt.want)
		}
	}
}

func TestResize(t *testing.T) {
	// A half clear red picture shrinks to red with half the alpha at the edge
	// pixel, and the color is not darkened by the clear pixels.
	set, err := loadIcons(writePNG(t, 512, color.NRGBA{R: 200, A: 255}, true), "")
	if err != nil {
		t.Fatal(err)
	}
	if got := sideOf(t, set.png(512)); got != 512 {
		t.Errorf("the master at its own size is %d", got)
	}
	img, _ := png.Decode(bytes.NewReader(set.png(3)))
	if c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); c.A != 0 {
		t.Errorf("clear side is %v", c)
	}
	mid := color.NRGBAModel.Convert(img.At(1, 0)).(color.NRGBA)
	if mid.R != 200 || mid.A < 120 || mid.A > 135 {
		t.Errorf("edge pixel is %v, want red 200 with half alpha", mid)
	}
	if c := color.NRGBAModel.Convert(img.At(2, 0)).(color.NRGBA); c != (color.NRGBA{R: 200, A: 255}) {
		t.Errorf("solid side is %v", c)
	}
}

func TestSmallIconIsUsedAt16(t *testing.T) {
	set, err := loadIcons(writePNG(t, 512, color.NRGBA{R: 255, A: 255}, false), writePNG(t, 16, color.NRGBA{B: 255, A: 255}, false))
	if err != nil {
		t.Fatal(err)
	}
	for size, want := range map[int]uint8{16: 255, 32: 0} {
		img, _ := png.Decode(bytes.NewReader(set.png(size)))
		if _, _, b, _ := img.At(0, 0).RGBA(); uint8(b>>8) != want {
			t.Errorf("size %d: blue is %d, want %d", size, b>>8, want)
		}
	}
}

// icnsEntries parses an .icns file back into kind to PNG.
func icnsEntries(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	if string(data[:4]) != "icns" || int(binary.BigEndian.Uint32(data[4:])) != len(data) {
		t.Fatalf("bad icns header or length %d", len(data))
	}
	entries := map[string][]byte{}
	for rest := data[8:]; len(rest) > 0; {
		n := int(binary.BigEndian.Uint32(rest[4:]))
		entries[string(rest[:4])] = rest[8:n]
		rest = rest[n:]
	}
	return entries
}

func TestWriteICNS(t *testing.T) {
	set, err := loadIcons(writePNG(t, 1024, color.NRGBA{G: 90, A: 255}, false), "")
	if err != nil {
		t.Fatal(err)
	}
	entries := icnsEntries(t, writeICNS(set))
	want := map[string]int{"icp4": 16, "ic11": 32, "ic12": 64, "ic07": 128, "ic13": 256, "ic08": 256, "ic14": 512, "ic09": 512, "ic10": 1024}
	if len(entries) != len(want) {
		t.Errorf("got %d entries, want %d", len(entries), len(want))
	}
	for kind, side := range want {
		if got := sideOf(t, entries[kind]); got != side {
			t.Errorf("%s is %d pixels, want %d", kind, got, side)
		}
	}
	// A 512 master has no ic10.
	small, _ := loadIcons(writePNG(t, 512, color.NRGBA{A: 255}, false), "")
	if _, ok := icnsEntries(t, writeICNS(small))["ic10"]; ok {
		t.Error("a 512 master made an ic10")
	}
}

func TestWriteICO(t *testing.T) {
	set, err := loadIcons(writePNG(t, 1024, color.NRGBA{G: 90, A: 255}, false), "")
	if err != nil {
		t.Fatal(err)
	}
	data := writeICO(icoImages(set))
	if binary.LittleEndian.Uint16(data[2:]) != 1 || binary.LittleEndian.Uint16(data[4:]) != 4 {
		t.Fatalf("header %v", data[:6])
	}
	for i, want := range []int{16, 32, 48, 256} {
		e := data[6+16*i:]
		size := int(binary.LittleEndian.Uint32(e[8:]))
		offset := int(binary.LittleEndian.Uint32(e[12:]))
		if side := sideOf(t, data[offset:offset+size]); side != want {
			t.Errorf("picture %d is %d pixels, want %d", i, side, want)
		}
		if got := int(e[0]); got != want%256 {
			t.Errorf("picture %d says its width is %d", i, got)
		}
	}
}

// fakePE is a small 64 bit PE file: one .text section, headers of 0x400.
func fakePE() []byte {
	exe := make([]byte, 0x600)
	copy(exe, "MZ")
	binary.LittleEndian.PutUint32(exe[0x3c:], 0x80)
	copy(exe[0x80:], "PE\x00\x00")
	c := exe[0x84:] // the COFF header
	binary.LittleEndian.PutUint16(c[0:], 0x8664)
	binary.LittleEndian.PutUint16(c[2:], 1)
	binary.LittleEndian.PutUint16(c[16:], 240)
	binary.LittleEndian.PutUint16(c[18:], 0x22)
	o := exe[0x98:] // the optional header
	binary.LittleEndian.PutUint16(o[0:], 0x20b)
	binary.LittleEndian.PutUint32(o[4:], 0x200)   // size of code
	binary.LittleEndian.PutUint32(o[32:], 0x1000) // section alignment
	binary.LittleEndian.PutUint32(o[36:], 0x200)  // file alignment
	binary.LittleEndian.PutUint32(o[56:], 0x2000) // size of image
	binary.LittleEndian.PutUint32(o[60:], 0x400)  // size of headers
	binary.LittleEndian.PutUint32(o[108:], 16)    // data directories
	s := exe[0x98+240:]                           // the section table
	copy(s, ".text")
	binary.LittleEndian.PutUint32(s[8:], 0x100)
	binary.LittleEndian.PutUint32(s[12:], 0x1000)
	binary.LittleEndian.PutUint32(s[16:], 0x200)
	binary.LittleEndian.PutUint32(s[20:], 0x400)
	return exe
}

// resourceBlobs walks the resource tree of f's .rsrc section and returns
// the data of every leaf under the type (3 or 14), by name id.
func resourceBlobs(t *testing.T, f *pe.File, section []byte, typ uint32) map[uint32][]byte {
	t.Helper()
	rsrc := f.Section(".rsrc")
	rva := rsrc.VirtualAddress
	entries := func(at uint32) [][2]uint32 {
		n := int(binary.LittleEndian.Uint16(section[at+12:])) + int(binary.LittleEndian.Uint16(section[at+14:]))
		var out [][2]uint32
		for i := 0; i < n; i++ {
			e := section[at+16+8*uint32(i):]
			out = append(out, [2]uint32{binary.LittleEndian.Uint32(e), binary.LittleEndian.Uint32(e[4:]) &^ 0x80000000})
		}
		return out
	}
	blobs := map[uint32][]byte{}
	for _, top := range entries(0) {
		if top[0] != typ {
			continue
		}
		for _, name := range entries(top[1]) {
			langs := entries(name[1])
			if len(langs) != 1 {
				t.Fatalf("name %d has %d languages", name[0], len(langs))
			}
			d := section[langs[0][1]:]
			addr, size := binary.LittleEndian.Uint32(d)-rva, binary.LittleEndian.Uint32(d[4:])
			blobs[name[0]] = section[addr : addr+size]
		}
	}
	return blobs
}

func TestAddIcon(t *testing.T) {
	set, err := loadIcons(writePNG(t, 1024, color.NRGBA{G: 90, A: 255}, false), "")
	if err != nil {
		t.Fatal(err)
	}
	exe := fakePE()
	out, err := addIcon(exe, icoImages(set))
	if err != nil {
		t.Fatal(err)
	}
	f, err := pe.NewFile(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("the result is not a PE file: %v", err)
	}
	if len(f.Sections) != 2 || f.Sections[1].Name != ".rsrc" {
		t.Fatalf("sections: %v", f.Sections)
	}
	rsrc := f.Sections[1]
	if rsrc.VirtualAddress != 0x2000 || rsrc.Offset != 0x600 || rsrc.Offset%0x200 != 0 {
		t.Errorf("the section is at rva %#x, offset %#x", rsrc.VirtualAddress, rsrc.Offset)
	}
	oh := f.OptionalHeader.(*pe.OptionalHeader64)
	if oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress != 0x2000 {
		t.Errorf("resource directory %+v", oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE])
	}
	if want := alignUp(0x2000+rsrc.VirtualSize, 0x1000); oh.SizeOfImage != want {
		t.Errorf("size of image %#x, want %#x", oh.SizeOfImage, want)
	}
	section, _ := rsrc.Data()
	icons := resourceBlobs(t, f, section, rtIcon)
	if len(icons) != 4 {
		t.Fatalf("%d icons", len(icons))
	}
	for i, want := range []int{16, 32, 48, 256} {
		if side := sideOf(t, icons[uint32(i+1)]); side != want {
			t.Errorf("icon %d is %d pixels, want %d", i+1, side, want)
		}
	}
	group := resourceBlobs(t, f, section, rtGroupIcon)[1]
	if len(group) != 6+14*4 || binary.LittleEndian.Uint16(group[4:]) != 4 {
		t.Fatalf("the group is %v", group)
	}
	for i := 0; i < 4; i++ {
		e := group[6+14*i:]
		if id := binary.LittleEndian.Uint16(e[12:]); int(id) != i+1 {
			t.Errorf("group entry %d names icon %d", i, id)
		}
		if size := binary.LittleEndian.Uint32(e[8:]); int(size) != len(icons[uint32(i+1)]) {
			t.Errorf("group entry %d says %d bytes", i, size)
		}
	}
	// The old file's bytes are all still where they were.
	if !bytes.Equal(out[:len(exe)], exeWithPatchedHeaders(exe, out)) {
		t.Error("the sections before the new one changed")
	}
}

// exeWithPatchedHeaders is exe with the header fields addIcon changes taken
// from out, to compare everything else.
func exeWithPatchedHeaders(exe, out []byte) []byte {
	patched := append([]byte{}, exe...)
	for _, r := range [][2]int{{0x86, 2}, {0x98 + 8, 4}, {0x98 + 56, 4}, {0x98 + 112 + 16, 8}, {0x98 + 240 + 40, 40}} {
		copy(patched[r[0]:r[0]+r[1]], out[r[0]:r[0]+r[1]])
	}
	return patched
}

func TestAddIconErrors(t *testing.T) {
	images := []icoImage{{16, []byte("png")}}
	if _, err := addIcon([]byte("RUNTIME"), images); err == nil || !strings.Contains(err.Error(), "not a Windows program") {
		t.Errorf("got %v", err)
	}
	withRsrc := fakePE()
	binary.LittleEndian.PutUint32(withRsrc[0x98+112+16:], 0x5000)
	if _, err := addIcon(withRsrc, images); err == nil || !strings.Contains(err.Error(), "already has resources") {
		t.Errorf("got %v", err)
	}
	full := fakePE()
	binary.LittleEndian.PutUint32(full[0x98+60:], 0x98+240+40) // headers end after one section
	binary.LittleEndian.PutUint32(full[0x98+240+20:], 0x98+240+40)
	if _, err := addIcon(full, images); err == nil || !strings.Contains(err.Error(), "no room") {
		t.Errorf("got %v", err)
	}
}

// buildOptions are a program folder and a stand-in runtime for the output
// system, with an icon.
func buildFor(t *testing.T, goos string, runtimeBytes []byte, output string, icon bool) Options {
	t.Helper()
	dir := writeTree(t, map[string]string{"main.liz": "puts(1)"})
	runtimeFile := filepath.Join(t.TempDir(), "runtime")
	os.WriteFile(runtimeFile, runtimeBytes, 0o755)
	o := Options{Dir: dir, Output: output, Runtime: runtimeFile, VM: true, GOOS: goos, GOARCH: "amd64"}
	if icon {
		o.Icon = writePNG(t, 1024, color.NRGBA{G: 90, A: 255}, false)
	}
	return o
}

func TestBuildWindowsIcon(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app.exe")
	if _, err := Build(buildFor(t, "windows", fakePE(), out, true)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	if f, err := pe.NewFile(bytes.NewReader(data)); err != nil || f.Section(".rsrc") == nil {
		t.Fatalf("no icon section: %v", err)
	}
	program, err := bundle.Open(out)
	if err != nil || program == nil || !program.VM {
		t.Fatalf("the program is not after the runtime: %+v, %v", program, err)
	}
}

func TestBuildLinuxIcon(t *testing.T) {
	dir := t.TempDir()
	if _, err := Build(buildFor(t, "linux", []byte("RUNTIME"), filepath.Join(dir, "tool"), true)); err != nil {
		t.Fatal(err)
	}
	for file, side := range map[string]int{"icon.png": 512, "icon-256.png": 256} {
		data, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil || sideOf(t, data) != side {
			t.Errorf("%s: %v", file, err)
		}
	}
	desktop, _ := os.ReadFile(filepath.Join(dir, "tool.desktop"))
	if !strings.Contains(string(desktop), "Exec=tool\n") || !strings.Contains(string(desktop), "Icon=tool\n") {
		t.Errorf("desktop entry:\n%s", desktop)
	}
}

func TestBuildMacApp(t *testing.T) {
	out := filepath.Join(t.TempDir(), "My Tool.app")
	if _, err := Build(buildFor(t, "darwin", []byte("RUNTIME"), out, true)); err != nil {
		t.Fatal(err)
	}
	icns, err := os.ReadFile(filepath.Join(out, "Contents", "Resources", "icon.icns"))
	if err != nil || len(icnsEntries(t, icns)) != 9 {
		t.Fatalf("icon.icns: %v", err)
	}
	plist, _ := os.ReadFile(filepath.Join(out, "Contents", "Info.plist"))
	for _, want := range []string{"<string>My Tool</string>", "app.lizardscript.My-Tool", "CFBundleIconFile"} {
		if !strings.Contains(string(plist), want) {
			t.Errorf("Info.plist has no %q", want)
		}
	}
	exe := filepath.Join(out, "Contents", "MacOS", "My Tool")
	if data, err := os.ReadFile(exe); err != nil || string(data) != "RUNTIME" {
		t.Errorf("the executable was changed: %q, %v", data, err)
	}
	// The program's files are found through the executable.
	program, err := bundle.Open(exe)
	if err != nil || program == nil || !program.VM {
		t.Fatalf("Open: %+v, %v", program, err)
	}
	// And through a link to it, like the linkz the installer puts in a bin folder.
	link := filepath.Join(t.TempDir(), "tool")
	if err := os.Symlink(exe, link); err == nil {
		if program, err := bundle.Open(link); err != nil || program == nil {
			t.Errorf("Open through a link: %+v, %v", program, err)
		}
	}
}

func TestBuildIconErrors(t *testing.T) {
	dir := t.TempDir()
	for _, tt := range []struct {
		name   string
		goos   string
		output string
		icon   bool
		want   string
	}{
		{"icon on a mac needs an app", "darwin", filepath.Join(dir, "tool"), true, "needs an app"},
		{"an app on another system", "linux", filepath.Join(dir, "x.app"), false, "only a build for macOS"},
		{"a runtime that is no exe", "windows", filepath.Join(dir, "x.exe"), true, "not a Windows program"},
	} {
		if _, err := Build(buildFor(t, tt.goos, []byte("RUNTIME"), tt.output, tt.icon)); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: got %v, want %q", tt.name, err, tt.want)
		}
	}
	o := buildFor(t, "linux", []byte("RUNTIME"), filepath.Join(dir, "y"), false)
	o.IconSmall = writePNG(t, 16, color.NRGBA{A: 255}, false)
	if _, err := Build(o); err == nil || !strings.Contains(err.Error(), "--icon-small needs --icon") {
		t.Errorf("got %v", err)
	}
}
