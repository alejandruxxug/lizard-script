// Package bundle is the format of a program built with lizardscript build:
// a copy of the lizardscript program with the program's files appended to
// it, as a zip archive followed by a short trailer. When lizardscript
// starts, it looks for the trailer at the end of its own file; a built
// program finds it and runs the main file of its archive.
//
// While a built program runs, its files are at paths that start with
// "bundle:" (Root): "bundle:/main.liz", "bundle:/draw/shapes.liz". use finds
// its modules there, and the Terrarium's fs reads them (they can't be
// changed). Every other path is a file on disk, as always.
package bundle

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

// Root is the folder of a built program's own files.
const Root = "bundle:"

// magic ends every built program, after the size of its archive.
const magic = "LIZBUNDL"

// trailerSize is the size of the archive (8 bytes) and the magic.
const trailerSize = 8 + len(magic)

// Program is a built program: its files and how to run them.
type Program struct {
	Files fs.FS  // the files, with paths relative to the program's folder
	Main  string // the main file, like "main.liz"
	VM    bool   // run it on the bytecode VM
}

// Current is the built program that is running, or nil.
var Current *Program

// IsPath reports whether p is the path of a built program's file.
func IsPath(p string) bool {
	return strings.HasPrefix(p, Root)
}

// Path is the path of the file name of the built program: Path("main.liz")
// is "bundle:/main.liz".
func Path(name string) string {
	return Root + "/" + strings.TrimPrefix(name, "/")
}

// name turns a bundle path into a path of Files: "bundle:/a/../b" is "b",
// and "bundle:" is ".".
func name(p string) string {
	clean := path.Clean("/" + strings.ReplaceAll(strings.TrimPrefix(p, Root), `\`, "/"))
	if clean == "/" {
		return "."
	}
	return clean[1:]
}

// files is the file system that holds p, and p's name in it.
func files(p string) (fs.FS, string, error) {
	if Current == nil {
		return nil, "", &fs.PathError{Op: "open", Path: p, Err: fs.ErrNotExist}
	}
	return Current.Files, name(p), nil
}

// ReadFile reads a file: a built program's file for a bundle path, a file
// on disk for any other.
func ReadFile(p string) ([]byte, error) {
	if !IsPath(p) {
		return os.ReadFile(p)
	}
	fsys, n, err := files(p)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(fsys, n)
}

// Stat describes a file, like ReadFile reads it.
func Stat(p string) (fs.FileInfo, error) {
	if !IsPath(p) {
		return os.Stat(p)
	}
	fsys, n, err := files(p)
	if err != nil {
		return nil, err
	}
	return fs.Stat(fsys, n)
}

// ReadDir lists a folder, like ReadFile reads a file.
func ReadDir(p string) ([]fs.DirEntry, error) {
	if !IsPath(p) {
		return os.ReadDir(p)
	}
	fsys, n, err := files(p)
	if err != nil {
		return nil, err
	}
	return fs.ReadDir(fsys, n)
}

// ErrReadOnly is the error of a change to a built program's file.
var ErrReadOnly = errors.New("the files of a built program can't be changed")

// File is one file to put in a built program.
type File struct {
	Name string // its path in the program's folder, with "/"
	Data []byte
}

// Write writes the archive of a built program and its trailer to w.
func Write(w io.Writer, program Program, contents []File) error {
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	now := time.Now()
	for _, f := range contents {
		fw, err := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: zip.Deflate, Modified: now})
		if err != nil {
			return err
		}
		if _, err := fw.Write(f.Data); err != nil {
			return err
		}
	}
	engine := "evaluator"
	if program.VM {
		engine = "vm"
	}
	if err := zw.SetComment("main=" + program.Main + "\nengine=" + engine); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	var size [8]byte
	binary.LittleEndian.PutUint64(size[:], uint64(archive.Len()))
	for _, part := range [][]byte{archive.Bytes(), size[:], []byte(magic)} {
		if _, err := w.Write(part); err != nil {
			return err
		}
	}
	return nil
}

// Open reads the built program at the end of the file exe. It returns nil
// and no error when exe is a plain lizardscript, with nothing appended.
// The file stays open while the program runs.
func Open(exe string) (*Program, error) {
	f, err := os.Open(exe)
	if err != nil {
		return nil, err
	}
	program, err := read(f)
	if program == nil {
		f.Close()
	}
	return program, err
}

// RuntimeSize is the size of the lizardscript at the start of the file exe:
// the whole file, or what comes before a built program's files.
func RuntimeSize(exe string) (int64, error) {
	f, err := os.Open(exe)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	start, _, err := archive(f)
	return start, err
}

// archive finds the archive at the end of f: where it starts and its size.
// With no trailer, start is the size of f and size is 0.
func archive(f *os.File) (start, size int64, err error) {
	info, err := f.Stat()
	if err != nil {
		return 0, 0, err
	}
	end := info.Size()
	if end < int64(trailerSize) {
		return end, 0, nil
	}
	var trailer [trailerSize]byte
	if _, err := f.ReadAt(trailer[:], end-int64(trailerSize)); err != nil {
		return 0, 0, err
	}
	if string(trailer[8:]) != magic {
		return end, 0, nil
	}
	size = int64(binary.LittleEndian.Uint64(trailer[:8]))
	start = end - int64(trailerSize) - size
	if size <= 0 || start < 0 {
		return 0, 0, fmt.Errorf("%s: the program's files are damaged", f.Name())
	}
	return start, size, nil
}

// read reads the trailer and the archive before it.
func read(f *os.File) (*Program, error) {
	start, size, err := archive(f)
	if err != nil || size == 0 {
		return nil, err
	}
	zr, err := zip.NewReader(io.NewSectionReader(f, start, size), size)
	if err != nil {
		return nil, fmt.Errorf("%s: the program's files are damaged: %v", f.Name(), err)
	}
	program := &Program{Files: zr, Main: "main.liz"}
	for _, line := range strings.Split(zr.Comment, "\n") {
		key, value, _ := strings.Cut(line, "=")
		switch key {
		case "main":
			program.Main = value
		case "engine":
			program.VM = value == "vm"
		}
	}
	return program, nil
}
