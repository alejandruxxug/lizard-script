package lsp

import (
	"encoding/json"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Go to definition, find references, highlights and rename all start from
// the name under the cursor and what it means (its target). A name declared
// at the top of a file that another file uses is the same name in both, so
// its references and its rename reach every file of the workspace that
// uses the module.

// identity is the name of a target the same in every file: a top-level
// name of a file ("file:/a/b.liz#circle"), a global ("global#len"), or ""
// for a name that only one file can see.
func (ix *index) identity(t target) string {
	switch {
	case t.module != nil:
		return "file:" + cleanName(t.module.path) + "#" + t.path
	case t.sym != nil && t.sym.scope == ix.top && t.sym.kind == letDecl && !strings.HasPrefix(t.sym.name, "_"):
		name := t.sym.name
		if t.path != "" {
			name += "." + t.path
		}
		return "file:" + cleanName(ix.name) + "#" + name
	case t.global:
		return "global#" + t.path
	}
	return ""
}

func cleanName(name string) string {
	if evaluator.IsStandardModule(name) {
		return name
	}
	return filepath.Clean(filepath.FromSlash(name))
}

// same reports whether a and b, both of ix, are the same name.
func (ix *index) same(a, b target) bool {
	if a.sym != nil && a.sym == b.sym {
		return a.path == b.path
	}
	id := ix.identity(a)
	return id != "" && id == ix.identity(b)
}

// uriOf is the URI of the source of ix.
func (s *Server) uriOf(ix *index) string {
	if d := s.docs[docKey(pathToURI(ix.name))]; d != nil && d.index == ix {
		return d.uri
	}
	for _, d := range s.docs {
		if d.index == ix {
			return d.uri
		}
	}
	return moduleURI(ix.name, ix.text)
}

// definitionOf is where t is declared: the index of its file and the span
// of its name. ok is false for a builtin or an unknown name.
func definitionOf(ix *index, t target) (*index, *ast.Span, bool) {
	switch {
	case t.sym != nil && t.path == "":
		return ix, t.sym.ident.Range(), true
	case t.sym != nil:
		if key := keyOf(t); key != nil {
			return ix, key.Range(), true
		}
	case t.module != nil && t.path == "":
		return t.module.index, &ast.Span{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 1}, true
	case t.module != nil:
		head, rest, _ := strings.Cut(t.path, ".")
		if sym := t.module.index.exported(head); sym != nil {
			return definitionOf(t.module.index, target{sym: sym, path: rest})
		}
	}
	return nil, nil, false
}

func (s *Server) definition(params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	_, ix, at, err := s.at(p)
	if err != nil {
		return nil, err
	}
	o := ix.occurrenceAt(at)
	if o == nil {
		return nil, nil
	}
	where, span, ok := definitionOf(ix, o.target)
	if !ok {
		return nil, nil
	}
	return Location{URI: s.uriOf(where), Range: spanRange(where.lines, span)}, nil
}

// referencesOf are the places of t, a target of ix, in ix and, for a
// top-level name, in every file of the workspace that uses it.
func (s *Server) referencesOf(ix *index, t target, withDecl bool) map[*index][]*occurrence {
	found := map[*index][]*occurrence{}
	add := func(in *index, match func(o *occurrence) bool) {
		for _, o := range in.occurrences {
			if (withDecl || !o.decl) && match(o) {
				found[in] = append(found[in], o)
			}
		}
	}
	add(ix, func(o *occurrence) bool { return ix.same(o.target, t) })

	id := ix.identity(t)
	if !strings.HasPrefix(id, "file:") {
		return found
	}
	home, _, _ := definitionOf(ix, t)
	for _, other := range s.workspaceIndexes() {
		if other == ix || other.name == ix.name {
			continue
		}
		add(other, func(o *occurrence) bool { return other.identity(o.target) == id })
	}
	// The declaration, when it is in a module the workspace doesn't list.
	listed := false
	for in := range found {
		listed = listed || home != nil && sameFile(in.name, home.name)
	}
	if home != nil && !listed && withDecl {
		for _, o := range home.occurrences {
			if o.decl && home.identity(o.target) == id {
				found[home] = append(found[home], o)
			}
		}
	}
	return found
}

func (s *Server) references(params json.RawMessage) (any, error) {
	var p referenceParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	_, ix, at, err := s.at(p.TextDocumentPositionParams)
	if err != nil {
		return nil, err
	}
	o := ix.occurrenceAt(at)
	if o == nil || !o.target.known() {
		return []Location{}, nil
	}
	locations := []Location{}
	for in, list := range s.referencesOf(ix, o.target, p.Context.IncludeDeclaration) {
		uri := s.uriOf(in)
		for _, ref := range list {
			locations = append(locations, Location{URI: uri, Range: spanRange(in.lines, ref.span)})
		}
	}
	sortLocations(locations)
	return locations, nil
}

// highlights are the places of the name under the cursor in its file.
func (s *Server) highlights(params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	_, ix, at, err := s.at(p)
	if err != nil {
		return nil, err
	}
	o := ix.occurrenceAt(at)
	if o == nil || !o.target.known() {
		return []any{}, nil
	}
	out := []map[string]any{}
	for _, ref := range ix.occurrences {
		if ix.same(ref.target, o.target) {
			kind := 2 // read
			if ref.decl {
				kind = 3 // write
			}
			out = append(out, map[string]any{"range": spanRange(ix.lines, ref.span), "kind": kind})
		}
	}
	return out, nil
}

// renameTarget is the name under the cursor when it can be renamed: a name
// declared in a file of the workspace, not a builtin, a member of a hash or
// a name of a standard module.
func (s *Server) renameTarget(p TextDocumentPositionParams) (*index, *occurrence, error) {
	_, ix, at, err := s.at(p)
	if err != nil {
		return nil, nil, err
	}
	o := ix.occurrenceAt(at)
	switch {
	case o == nil:
		return nil, nil, &responseError{codeRequestFailed, "there is no name here"}
	case o.target.global:
		return nil, nil, &responseError{codeRequestFailed, o.name + " is a builtin: it can't be renamed"}
	case !o.target.known():
		return nil, nil, &responseError{codeRequestFailed, o.name + " is not declared anywhere"}
	case o.target.module != nil && evaluator.IsStandardModule(o.target.module.path):
		return nil, nil, &responseError{codeRequestFailed, o.name + " is a name of a standard module"}
	case strings.Contains(o.target.path, "."), o.target.sym != nil && o.target.path != "":
		return nil, nil, &responseError{codeRequestFailed, "a key of a hash can't be renamed"}
	case o.target.module != nil && o.target.path == "":
		return nil, nil, &responseError{codeRequestFailed, "there is no name here"}
	}
	return ix, o, nil
}

func (s *Server) prepareRename(params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	ix, o, err := s.renameTarget(p)
	if err != nil {
		return nil, err
	}
	return map[string]any{"range": spanRange(ix.lines, o.span), "placeholder": o.name}, nil
}

var plainName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (s *Server) rename(params json.RawMessage) (any, error) {
	var p renameParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	if !plainName.MatchString(p.NewName) || token.LookupIdent(p.NewName) != token.IDENT {
		return nil, &responseError{codeInvalidParams, p.NewName + " can't be a name"}
	}
	ix, o, err := s.renameTarget(p.TextDocumentPositionParams)
	if err != nil {
		return nil, err
	}
	edit := WorkspaceEdit{Changes: map[string][]TextEdit{}}
	for in, list := range s.referencesOf(ix, o.target, true) {
		uri := s.uriOf(in)
		for _, ref := range list {
			edit.Changes[uri] = append(edit.Changes[uri], TextEdit{Range: spanRange(in.lines, ref.span), NewText: p.NewName})
		}
	}
	return edit, nil
}

// workspaceFile is a .liz file of the workspace that is not open, read from
// the disk.
type workspaceFile struct {
	modified time.Time
	index    *index
}

// maxWorkspaceFiles bounds how many files the server reads for workspace
// symbols and references, so a huge folder doesn't stop it.
const maxWorkspaceFiles = 2000

// workspaceIndexes are the analyses of the open documents and of every
// other .liz file of the workspace folders.
func (s *Server) workspaceIndexes() []*index {
	var out []*index
	open := map[string]bool{}
	for key, d := range s.docs {
		out = append(out, s.indexOf(d))
		open[key] = true
	}
	count := 0
	for _, root := range s.roots {
		filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			name := entry.Name()
			if entry.IsDir() {
				if path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "target") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(name, ".liz") || open[path] {
				return nil
			}
			if count++; count > maxWorkspaceFiles {
				return filepath.SkipAll
			}
			if ix := s.workspaceIndex(path, entry); ix != nil {
				out = append(out, ix)
			}
			return nil
		})
	}
	return out
}

// workspaceIndex is the analysis of the file at path, made again when the
// file changes.
func (s *Server) workspaceIndex(path string, entry fs.DirEntry) *index {
	info, err := entry.Info()
	if err != nil {
		return nil
	}
	if f := s.workspace[path]; f != nil && f.modified.Equal(info.ModTime()) {
		return f.index
	}
	text, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	f := &workspaceFile{modified: info.ModTime(), index: newIndex(path, string(text), s.globals, s.loader)}
	s.workspace[path] = f
	return f.index
}

// sortLocations puts locations in the order of their files and places.
func sortLocations(list []Location) {
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.URI != b.URI {
			return a.URI < b.URI
		}
		if a.Range.Start.Line != b.Range.Start.Line {
			return a.Range.Start.Line < b.Range.Start.Line
		}
		return a.Range.Start.Character < b.Range.Start.Character
	})
}
