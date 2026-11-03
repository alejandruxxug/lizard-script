package bundle

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// writeBuilt makes a fake executable: some bytes, then a built program.
func writeBuilt(t *testing.T, runtime []byte, program Program, files []File) string {
	t.Helper()
	var out bytes.Buffer
	out.Write(runtime)
	if err := Write(&out, program, files); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(path, out.Bytes(), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWriteAndOpen(t *testing.T) {
	runtime := []byte("pretend this is lizardscript")
	exe := writeBuilt(t, runtime, Program{Main: "main.liz", VM: true}, []File{
		{Name: "main.liz", Data: []byte("puts(1)")},
		{Name: "assets/a.txt", Data: []byte("data")},
	})
	program, err := Open(exe)
	if err != nil || program == nil {
		t.Fatalf("Open: %v, %v", program, err)
	}
	if program.Main != "main.liz" || !program.VM {
		t.Errorf("got main %q, vm %t", program.Main, program.VM)
	}
	if data, _ := fs.ReadFile(program.Files, "assets/a.txt"); string(data) != "data" {
		t.Errorf("assets/a.txt is %q", data)
	}
	if size, err := RuntimeSize(exe); err != nil || size != int64(len(runtime)) {
		t.Errorf("RuntimeSize = %d, %v, want %d", size, err, len(runtime))
	}
}

func TestOpenAPlainFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain")
	os.WriteFile(path, []byte("no program here, just bytes"), 0o755)
	if program, err := Open(path); program != nil || err != nil {
		t.Errorf("a plain file gave %v, %v", program, err)
	}
	if size, _ := RuntimeSize(path); size != 27 {
		t.Errorf("RuntimeSize of a plain file = %d", size)
	}
	os.WriteFile(path, append([]byte("xx"), []byte("\xff\xff\xff\xff\xff\xff\xff\x7f"+magic)...), 0o755)
	if _, err := Open(path); err == nil {
		t.Error("a damaged trailer gave no error")
	}
}

func TestPaths(t *testing.T) {
	saved := Current
	defer func() { Current = saved }()
	exe := writeBuilt(t, nil, Program{Main: "main.liz"}, []File{{Name: "main.liz", Data: []byte("x")}, {Name: "d/f", Data: []byte("f")}})
	Current, _ = Open(exe)

	for _, tc := range []struct{ path, want string }{
		{"bundle:/main.liz", "x"}, {"bundle:/d/../d/f", "f"}, {`bundle:\d\f`, "f"}, {"bundle:main.liz", "x"},
	} {
		if data, err := ReadFile(tc.path); err != nil || string(data) != tc.want {
			t.Errorf("ReadFile(%q) = %q, %v", tc.path, data, err)
		}
	}
	if entries, err := ReadDir("bundle:"); err != nil || len(entries) != 2 {
		t.Errorf("ReadDir(bundle:) = %v, %v", entries, err)
	}
	if info, err := Stat("bundle:/d"); err != nil || !info.IsDir() {
		t.Errorf("Stat(bundle:/d) = %v, %v", info, err)
	}
	if _, err := ReadFile("bundle:/nope"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a missing file gave %v", err)
	}
	if Path("main.liz") != "bundle:/main.liz" || !IsPath("bundle:") || IsPath("main.liz") {
		t.Error("Path or IsPath")
	}

	Current = nil
	if _, err := ReadFile("bundle:/main.liz"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("with no built program: %v", err)
	}
	disk := filepath.Join(t.TempDir(), "disk.txt")
	os.WriteFile(disk, []byte("on disk"), 0o644)
	if data, err := ReadFile(disk); err != nil || string(data) != "on disk" {
		t.Errorf("a disk file gave %q, %v", data, err)
	}
}
