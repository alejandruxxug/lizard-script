package terrarium

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFS(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir())
	d := quote(dir)
	check(t, []testCase{
		{"write then read", `fs.write(fs.join(` + d + `, "a.txt"), "hi é"); fs.read(fs.join(` + d + `, "a.txt"))`, "hi é", false},
		{"write replaces", `let p = fs.join(` + d + `, "b.txt"); fs.write(p, "long text"); fs.write(p, "x"); fs.read(p)`, "x", false},
		{"append makes the file", `let p = fs.join(` + d + `, "c.txt"); if (fs.exists(p)) { fs.remove(p) }; fs.append(p, "a"); fs.append(p, "b"); fs.read(p)`, "ab", false},
		{"exists", `[fs.exists(` + d + `), fs.exists(fs.join(` + d + `, "nope"))]`, "[true, false]", false},
		{"mkdir and list", `fs.mkdir(fs.join(` + d + `, "l/z/deep")); fs.write(fs.join(` + d + `, "l/b"), ""); fs.list(fs.join(` + d + `, "l"))`, `["b", "z"]`, false},
		{"mkdir twice is fine", `fs.mkdir(` + d + `); fs.mkdir(` + d + `)`, "null", false},
		{"remove a file", `let p = fs.join(` + d + `, "r.txt"); fs.write(p, ""); fs.remove(p); fs.exists(p)`, "false", false},
		{"remove an empty folder", `let p = fs.join(` + d + `, "empty"); fs.mkdir(p); fs.remove(p); fs.exists(p)`, "false", false},
		{"join tidies", `fs.join("a/", "./b", "../c")`, "a/c", false},
		{"join one", `fs.join("a")`, "a", false},
		{"home", `fs.home() == ""`, "false", false},

		{"read missing", `fs.read(fs.join(` + d + `, "missing.txt"))`, "can't read " + dir + "/missing.txt: no such file or directory", true},
		{"read a number", `fs.read(5)`, "want a STRING path, got INTEGER", true},
		{"write count", `fs.write("x")`, "wrong number of arguments: want 2, got 1", true},
		{"write a number", `fs.write("x", 5)`, "want a STRING text, got INTEGER", true},
		{"write into a missing folder", `fs.write(fs.join(` + d + `, "no/such/x"), "")`, "can't write", true},
		{"write over a folder", `fs.write(` + d + `, "")`, "it is a folder", true},
		{"list missing", `fs.list(fs.join(` + d + `, "missing"))`, "can't list", true},
		{"remove a full folder", `let p = fs.join(` + d + `, "full"); fs.mkdir(p); fs.write(fs.join(p, "f"), ""); fs.remove(p)`, "the folder is not empty", true},
		{"remove missing", `fs.remove(fs.join(` + d + `, "missing"))`, "can't remove", true},
		{"join nothing", `fs.join()`, "want at least 1, got 0", true},
		{"join a number", `fs.join("a", 1)`, "want a STRING part, got INTEGER", true},
		{"home takes nothing", `fs.home(1)`, "want 0, got 1", true},
		{"errors are catchable", `let m = ""; try { fs.read(5) } catch (e) { m = e.message }; m`, "fs.read: want a STRING path, got INTEGER", false},
	})
}

func TestWriteFileAtomicKeepsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode is %v, want 0600", info.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("the folder has %d files, want only the one written", len(entries))
	}
}
