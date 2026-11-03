package lsp

import (
	"lizardscript/ast"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// document is the text of a file the editor has open, which may differ
// from the file on disk.
type document struct {
	uri     string
	path    string // the file, for the use statements to find their modules
	version int
	text    string
	lines   []string // the text cut at each "\n", without it
	index   *index   // the analysis of text, made when first needed
}

func newDocument(uri, path, text string, version int) *document {
	return &document{uri: uri, path: path, version: version, text: text, lines: strings.Split(text, "\n")}
}

// The positions of the language count lines and columns from 1, a column
// for each character; the protocol counts from 0, with a unit for each
// UTF-16 code unit (two for a character beyond U+FFFF).

// toPosition is the protocol's position of (line, column) of the language.
func (d *document) toPosition(line, column int) Position {
	return positionIn(d.lines, line, column)
}

// toRange is the protocol's range from (line, column) to (endLine,
// endColumn) of the language.
func (d *document) toRange(line, column, endLine, endColumn int) Range {
	return Range{positionIn(d.lines, line, column), positionIn(d.lines, endLine, endColumn)}
}

// positionIn is the protocol's position of (line, column) in lines.
func positionIn(lines []string, line, column int) Position {
	if line < 1 {
		return Position{}
	}
	if line > len(lines) {
		last := len(lines) - 1
		return Position{Line: last, Character: utf16Len(lines[last])}
	}
	text, units := lines[line-1], 0
	for i := 0; i < column-1 && len(text) > 0; i++ {
		r, size := utf8.DecodeRuneInString(text)
		units += utf16Units(r)
		text = text[size:]
	}
	return Position{Line: line - 1, Character: units}
}

// spanRange is the protocol's range of span in lines.
func spanRange(lines []string, span *ast.Span) Range {
	return Range{positionIn(lines, span.StartLine, span.StartColumn), positionIn(lines, span.EndLine, span.EndColumn)}
}

// fromPosition is (line, column) of the language at the protocol's p.
func (d *document) fromPosition(p Position) (int, int) {
	if p.Line < 0 || p.Line >= len(d.lines) {
		return p.Line + 1, 1
	}
	text, units, column := d.lines[p.Line], 0, 1
	for units < p.Character && len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		units += utf16Units(r)
		text = text[size:]
		column++
	}
	return p.Line + 1, column
}

// offset is the byte offset in the text of the protocol's p.
func (d *document) offset(p Position) int {
	if p.Line >= len(d.lines) {
		return len(d.text)
	}
	at := 0
	for _, line := range d.lines[:max(p.Line, 0)] {
		at += len(line) + 1
	}
	text, units := d.lines[max(p.Line, 0)], 0
	for units < p.Character && len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		units += utf16Units(r)
		text = text[size:]
		at += size
	}
	return at
}

// end is the position after the last character.
func (d *document) end() Position {
	last := len(d.lines) - 1
	return Position{Line: last, Character: utf16Len(d.lines[last])}
}

// edit replaces the text in r (the whole text when r is nil) with text.
func (d *document) edit(r *Range, text string) {
	if r == nil {
		d.text = text
	} else {
		start, end := d.offset(r.Start), d.offset(r.End)
		d.text = d.text[:start] + text + d.text[max(start, end):]
	}
	d.lines = strings.Split(d.text, "\n")
	d.index = nil
}

func utf16Units(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16Units(r)
	}
	return n
}

// uriToPath is the file of a file:// URI, or "" for another kind.
func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return ""
	}
	path := u.Path
	// file:///C:/x on Windows has the path /C:/x.
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.Clean(filepath.FromSlash(path))
}

// pathToURI is the file:// URI of path.
func pathToURI(path string) string {
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}
