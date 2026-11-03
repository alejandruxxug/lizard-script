// Command docgen writes lsp/builtin_docs.go: the signature and the text of
// every builtin, library function and Terrarium function, read from the
// tables of docs/SEMANTICS.md and the headings of docs/TERRARIUM.md, so a
// hover in an editor says what the documentation says. It runs with
//
//	go generate ./lsp
//
// and a test fails when the file is older than the documents.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	docs := flag.String("docs", "../docs", "the folder with SEMANTICS.md and TERRARIUM.md")
	out := flag.String("o", "builtin_docs.go", "the file to write")
	flag.Parse()

	code, err := generate(*docs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "docgen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, code, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "docgen:", err)
		os.Exit(1)
	}
}

// entry is what the documents say about one name.
type entry struct {
	signatures []string
	doc        string
}

// generate reads the documents in dir and returns the Go file.
func generate(dir string) ([]byte, error) {
	semantics, err := os.ReadFile(filepath.Join(dir, "SEMANTICS.md"))
	if err != nil {
		return nil, err
	}
	terrarium, err := os.ReadFile(filepath.Join(dir, "TERRARIUM.md"))
	if err != nil {
		return nil, err
	}
	entries := map[string]*entry{}
	readTables(string(semantics), entries)
	modules := readHeadings(string(terrarium), entries)
	for name, e := range extras {
		if entries[name] == nil {
			entries[name] = e
		}
	}
	return write(entries, modules)
}

// extras are the builtins that the documents describe in prose, not in a
// table.
var extras = map[string]*entry{
	"point": {signatures: []string{"point(x, y)", "point(x, y, z)"},
		doc: "A point, the same value as `(x, y)` or `(x, y, z)`. Every coordinate must be an integer or a float, and keeps its type."},
	"null": {doc: "The value of nothing: what a function without a value gives. `type(null)` is `\"NULL\"`; it is false in conditions and equal only to `null`."},
}

// The sections of SEMANTICS.md whose tables document builtins, and the
// library each one's names belong to.
var tableSections = map[string]bool{
	"Colors": true, "Builtins": true, "The math library": true,
	"The text library": true, "The list library": true, "The hash library": true,
}

var (
	codeSpan  = regexp.MustCompile("`([^`]+)`")
	signature = regexp.MustCompile(`^([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?)(\(.*\))?$`)
	anchor    = regexp.MustCompile(`\[([^\]]+)\]\(#[^)]*\)`)
)

// readTables adds the rows of the builtin tables of SEMANTICS.md: the first
// cell has one or more signatures in backticks, the second what they do.
func readTables(text string, entries map[string]*entry) {
	section := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			section = strings.TrimPrefix(line, "## ")
			continue
		}
		if !tableSections[section] || !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "| "), " | ")
		if len(cells) < 2 {
			continue
		}
		doc := clean(strings.Join(cells[1:], " | "))
		for _, span := range codeSpan.FindAllStringSubmatch(cells[0], -1) {
			m := signature.FindStringSubmatch(span[1])
			if m == nil {
				continue
			}
			add(entries, m[1], span[1], m[2] != "", doc)
		}
	}
}

// readHeadings adds the "### module.name(params)" (or ####) sections of TERRARIUM.md
// and returns the summary of each module, from the table at its top. A
// heading right above another shares its text, and a heading without a
// module (the drawing target functions) is stored as "*.name": every module
// that has the name has the function.
func readHeadings(text string, entries map[string]*entry) map[string]string {
	modules := map[string]string{}
	moduleRow := regexp.MustCompile("^\\| \\[`([a-z]+)`\\]\\(#[a-z-]+\\) \\| (.+) \\|$")
	var pending []string // the headings that wait for their text
	var body []string
	flush := func() {
		doc := clean(strings.TrimSpace(strings.Join(fenced(body), "\n")))
		for _, heading := range pending {
			m := signature.FindStringSubmatch(heading)
			if m == nil || m[2] == "" {
				continue
			}
			name := m[1]
			if !strings.Contains(name, ".") {
				name = "*." + name
			}
			add(entries, name, heading, true, doc)
		}
		pending, body = nil, nil
	}
	for _, line := range strings.Split(text, "\n") {
		if m := moduleRow.FindStringSubmatch(line); m != nil {
			modules[m[1]] = clean(m[2])
		}
		switch {
		case strings.HasPrefix(line, "### "), strings.HasPrefix(line, "#### "):
			if len(body) > 0 && strings.TrimSpace(strings.Join(body, "")) != "" {
				flush()
			}
			body = nil
			pending = append(pending, strings.TrimLeft(line, "# "))
		case strings.HasPrefix(line, "## "), strings.HasPrefix(line, "# "):
			flush()
		case len(pending) > 0:
			body = append(body, line)
		}
	}
	flush()
	return modules
}

// fenced names the language of the code blocks of lines, so an editor
// colors them: a fence that opens a block becomes "```lizard".
func fenced(lines []string) []string {
	out, open := make([]string, len(lines)), false
	for i, line := range lines {
		out[i] = line
		if line == "```" {
			if !open {
				out[i] = "```lizard"
			}
			open = !open
		} else if strings.HasPrefix(line, "```") {
			open = true
		}
	}
	return out
}

// add records one signature (or a plain name, like math.pi) of name.
func add(entries map[string]*entry, name, sig string, isCall bool, doc string) {
	e := entries[name]
	if e == nil {
		e = &entry{}
		entries[name] = e
	}
	if isCall {
		e.signatures = append(e.signatures, sig)
	}
	if !strings.Contains(e.doc, doc) {
		if e.doc != "" {
			e.doc += "\n\n"
		}
		e.doc += doc
	}
}

// clean takes out the links to anchors of the document, which mean nothing
// in an editor.
func clean(text string) string {
	return anchor.ReplaceAllString(text, "$1")
}

// write makes the Go file.
func write(entries map[string]*entry, modules map[string]string) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("// Code generated by lsp/internal/docgen from docs/SEMANTICS.md and docs/TERRARIUM.md. DO NOT EDIT.\n\n")
	b.WriteString("package lsp\n\n")
	b.WriteString("// builtinDocs are the builtins, the libraries' members and the Terrarium's\n")
	b.WriteString("// functions, by name (\"len\", \"math.sin\", \"fs.read\"); \"*.name\" is a\n")
	b.WriteString("// drawing target function, which every module that has name shares.\n")
	b.WriteString("var builtinDocs = map[string]builtinDoc{\n")
	for _, name := range sortedKeys(entries) {
		e := entries[name]
		var sigs []string
		for _, sig := range e.signatures {
			sigs = append(sigs, fmt.Sprintf("%q", sig))
		}
		fmt.Fprintf(&b, "\t%q: {Signatures: []string{%s}, Doc: %q},\n", name, strings.Join(sigs, ", "), e.doc)
	}
	b.WriteString("}\n\n")
	b.WriteString("// moduleDocs say what each Terrarium module is for.\n")
	b.WriteString("var moduleDocs = map[string]string{\n")
	for _, name := range sortedKeys(modules) {
		fmt.Fprintf(&b, "\t%q: %q,\n", name, modules[name])
	}
	b.WriteString("}\n")
	return format.Source(b.Bytes())
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
