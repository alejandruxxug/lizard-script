package lsp

import (
	"lizardscript/ast"
	"lizardscript/evaluator"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// moduleFile is a file a use statement loads, read the way use reads it
// (evaluator.FindModule) but never run.
type moduleFile struct {
	path  string // its source name: a file, or "std:/..." for a standard module
	index *index // its analysis
}

// loader finds and reads the modules of use statements, once for each
// version of their text. A module's own uses are read too, so a name it
// takes from another module (let dot = _vec.dot) is known.
type loader struct {
	globals map[string][]string
	files   map[string]*moduleFile
	reading map[string]bool // the modules being read, to stop at a cycle
}

func newLoader(globals map[string][]string) *loader {
	return &loader{globals: globals, files: map[string]*moduleFile{}, reading: map[string]bool{}}
}

// load is the module of st, or nil when it can't be found or is part of a
// cycle of uses (which the program can't run either).
func (l *loader) load(st *ast.UseStatement) *moduleFile {
	file, source, err := evaluator.FindModule(st)
	if err != nil || l.reading[file] {
		return nil
	}
	if m, ok := l.files[file]; ok && m.index.text == string(source) {
		return m
	}
	l.reading[file] = true
	defer delete(l.reading, file)
	m := &moduleFile{path: file, index: newIndex(file, string(source), l.globals, l)}
	l.files[file] = m
	return m
}

// moduleURI is the URI of a module's file. A standard module is part of
// the lizardscript binary, so its text is written to a folder of the user's
// cache for the editor to open.
func moduleURI(path, text string) string {
	if !evaluator.IsStandardModule(path) {
		return pathToURI(path)
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	file := filepath.Join(dir, "lizardscript", "std", filepath.FromSlash(strings.TrimPrefix(path, evaluator.StandardRoot+"/")))
	if old, err := os.ReadFile(file); err != nil || string(old) != text {
		if os.MkdirAll(filepath.Dir(file), 0o755) == nil {
			os.WriteFile(file, []byte(text), 0o644)
		}
	}
	return pathToURI(file)
}

// overlay is the text of the files the editor has open, which a use reads
// instead of the file on disk, so a module being edited is seen as it is.
type overlay struct {
	mu    sync.RWMutex
	texts map[string]string // by clean path
	next  func(string) ([]byte, error)
}

func (o *overlay) read(path string) ([]byte, error) {
	o.mu.RLock()
	text, ok := o.texts[filepath.Clean(path)]
	o.mu.RUnlock()
	if ok {
		return []byte(text), nil
	}
	return o.next(path)
}

func (o *overlay) set(path, text string) {
	o.mu.Lock()
	o.texts[path] = text
	o.mu.Unlock()
}

func (o *overlay) remove(path string) {
	o.mu.Lock()
	delete(o.texts, path)
	o.mu.Unlock()
}
