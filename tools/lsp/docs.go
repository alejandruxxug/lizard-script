package lsp

import (
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"regexp"
	"strings"
)

// The doc comment of a declaration is the -~> lines right above it, read
// the way lizard.docs reads them (terrarium/lizard_docs.go):
//
//	-~> A circle around center.
//	-~> center: the middle
//	-~> example: circle((10, 10), 5)
//	function circle(center, radius = 1) { ... }
//
// The summary is the lines before the first blank line, a parameter line or
// the "example:" line, joined with spaces; a "name: ..." line of a
// parameter documents it, and the lines after "example:" go on the example.

// declDoc is the doc comment of a declaration.
type declDoc struct {
	summary string
	params  map[string]string
	example string // "" when there is none
}

var (
	paramLine   = regexp.MustCompile(`^(\w+):\s*(.*)$`)
	exampleLine = regexp.MustCompile(`^example:\s*(.*)$`)
)

// docOf reads the doc comment above the declaration that starts at line,
// a let of the function f (nil for another value).
func (ix *index) docOf(line int, f *ast.FunctionLiteral) declDoc {
	var lines []string
	for l := line - 1; l >= 1 && l <= len(ix.lines); l-- {
		text := strings.TrimSpace(ix.lines[l-1])
		if !strings.HasPrefix(text, "-~>") {
			break
		}
		lines = append([]string{strings.TrimSpace(text[3:])}, lines...)
	}

	known := map[string]bool{}
	if f != nil {
		for _, p := range f.Parameters {
			known[p.Value] = true
		}
	}
	doc := declDoc{params: map[string]string{}}
	var summary []string
	summaryDone, last := false, "" // last is the parameter or "example" a line goes on
	for _, line := range lines {
		if m := exampleLine.FindStringSubmatch(line); m != nil {
			doc.example, last, summaryDone = m[1], "example", true
			continue
		}
		if m := paramLine.FindStringSubmatch(line); m != nil && known[m[1]] {
			doc.params[m[1]] = m[2]
			last, summaryDone = m[1], true
			continue
		}
		switch {
		case line == "":
			summaryDone = summaryDone || len(summary) > 0
			last = ""
		case !summaryDone:
			summary = append(summary, line)
		case last == "example":
			doc.example += "\n" + line
		case last != "":
			doc.params[last] += " " + line
		}
	}
	doc.summary = strings.Join(summary, " ")
	return doc
}

// markdown writes the doc for a hover or a completion item.
func (d declDoc) markdown(f *ast.FunctionLiteral) string {
	var parts []string
	if d.summary != "" {
		parts = append(parts, d.summary)
	}
	if f != nil {
		var params []string
		for _, p := range f.Parameters {
			if text, ok := d.params[p.Value]; ok {
				params = append(params, "- `"+p.Value+"`: "+text)
			}
		}
		if len(params) > 0 {
			parts = append(parts, strings.Join(params, "\n"))
		}
	}
	if d.example != "" {
		parts = append(parts, "```lizard\n"+d.example+"\n```")
	}
	return strings.Join(parts, "\n\n")
}
