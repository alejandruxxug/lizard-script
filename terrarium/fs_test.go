package terrarium

import (
	"lizardscript/bundle"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
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

// TestFSInfoRenameCopy runs on both engines in one folder, so each case
// that makes files removes them.
func TestFSInfoRenameCopy(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir())
	d := quote(dir)
	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	os.WriteFile(filepath.Join(dir, "dated.txt"), []byte("12345"), 0o644)
	os.Chtimes(filepath.Join(dir, "dated.txt"), stamp, stamp)
	os.Mkdir(filepath.Join(dir, "sub"), 0o755)
	check(t, []testCase{
		{"info of a file", `fs.info(fs.join(` + d + `, "dated.txt"))`, `{"size": 5, "mtime": 1767323045, "dir": false}`, false},
		{"info of a folder", `fs.info(fs.join(` + d + `, "sub")).dir`, "true", false},
		{"list with info", `fs.list(` + d + `, {"info": true})[0]`, `{"name": "dated.txt", "size": 5, "mtime": 1767323045, "dir": false}`, false},
		{"list without info", `fs.list(` + d + `, {"info": false})`, `["dated.txt", "sub"]`, false},
		{"rename", `let a = fs.join(` + d + `, "r1"); let b = fs.join(` + d + `, "r2"); fs.write(a, "r"); fs.rename(a, b); let r = [fs.exists(a), fs.read(b)]; fs.remove(b); r`, `[false, "r"]`, false},
		{"rename a folder", `let a = fs.join(` + d + `, "m1"); let b = fs.join(` + d + `, "m2"); fs.mkdir(a); fs.rename(a, b); let r = fs.info(b).dir; fs.remove(b); r`, "true", false},
		{"copy", `let a = fs.join(` + d + `, "c1"); let b = fs.join(` + d + `, "c2"); fs.write(a, "copied"); fs.copy(a, b); let r = [fs.read(a), fs.read(b)]; fs.remove(a); fs.remove(b); r`, `["copied", "copied"]`, false},

		{"info missing", `fs.info(fs.join(` + d + `, "nope"))`, "can't read " + dir + "/nope: no such file or directory", true},
		{"list an unknown option", `fs.list(` + d + `, {"sizes": true})`, `unknown option "sizes", the only one is info`, true},
		{"list info not a boolean", `fs.list(` + d + `, {"info": 1})`, "the option info must be true or false, got INTEGER", true},
		{"list options not a hash", `fs.list(` + d + `, true)`, "want a HASH options, got BOOLEAN", true},
		{"rename over a folder", `fs.rename(fs.join(` + d + `, "dated.txt"), fs.join(` + d + `, "sub"))`, "there is already something there", true},
		{"rename missing", `fs.rename(fs.join(` + d + `, "nope"), fs.join(` + d + `, "nope2"))`, "can't move " + dir + "/nope: no such file or directory", true},
		{"copy over a file", `fs.copy(fs.join(` + d + `, "dated.txt"), fs.join(` + d + `, "dated.txt"))`, "can't copy " + dir + "/dated.txt to " + dir + "/dated.txt: there is already something there", true},
		{"copy a folder", `fs.copy(fs.join(` + d + `, "sub"), fs.join(` + d + `, "sub2"))`, "it is a folder", true},
		{"copy missing", `fs.copy(fs.join(` + d + `, "nope"), fs.join(` + d + `, "x"))`, "can't copy " + dir + "/nope: no such file or directory", true},
	})
}

func TestBundlePaths(t *testing.T) {
	saved := bundle.Current
	defer func() { bundle.Current = saved }()
	bundle.Current = &bundle.Program{Main: "main.liz", Files: fstest.MapFS{
		"main.liz":         {Data: []byte("puts(1)")},
		"assets/hello.txt": {Data: []byte("hi from the bundle")},
	}}
	ProgramFile = "bundle:/main.liz"
	defer func() { ProgramFile = "" }()
	check(t, []testCase{
		{"program dir", `os.program_dir()`, "bundle:", false},
		{"read", `fs.read(fs.join(os.program_dir(), "assets/hello.txt"))`, "hi from the bundle", false},
		{"exists", `[fs.exists("bundle:/main.liz"), fs.exists("bundle:/nope")]`, "[true, false]", false},
		{"list", `fs.list("bundle:")`, `["assets", "main.liz"]`, false},
		{"list a folder", `fs.list("bundle:/assets")`, `["hello.txt"]`, false},
		{"info", `fs.info("bundle:/main.liz").size`, "7", false},
		{"copy out", `let to = fs.join(` + quote(filepath.ToSlash(t.TempDir())) + `, "m"); fs.copy("bundle:/main.liz", to); let r = fs.read(to); fs.remove(to); r`, "puts(1)", false},
		{"write", `fs.write("bundle:/main.liz", "")`, "can't write bundle:/main.liz: the files of a built program can't be changed", true},
		{"remove", `fs.remove("bundle:/main.liz")`, "can't be changed", true},
		{"trash", `fs.trash("bundle:/main.liz")`, "can't be changed", true},
		{"missing", `fs.read("bundle:/nope.txt")`, "can't read bundle:/nope.txt: file does not exist", true},
	})
}

func TestProgramDir(t *testing.T) {
	defer func() { ProgramFile = "" }()
	ProgramFile = "tests/x.liz"
	wd, _ := os.Getwd()
	check(t, []testCase{
		{"next to the file", `os.program_dir()`, filepath.ToSlash(filepath.Join(wd, "tests")), false},
	})
	ProgramFile = ""
	check(t, []testCase{
		{"no file", `os.program_dir()`, filepath.ToSlash(wd), false},
	})
}
