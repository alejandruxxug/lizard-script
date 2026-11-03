package lsp

import (
	"encoding/json"
	"lizardscript/ast"
	"regexp"
	"sort"
	"strings"
)

// keywords are the words of the language, with the ones that are only
// keywords where the parser reads them so (layer, use, as, try, catch,
// continue).
var keywords = []string{
	"let", "function", "return", "if", "elif", "else", "while", "for", "break",
	"continue", "true", "false", "null", "try", "catch", "layer", "use", "as",
}

// snippets are the statements a completion can write whole. "\t" is one
// level of the editor's indentation.
var snippets = []struct{ label, detail, body string }{
	{"function", "function name(...) { }", "function ${1:name}(${2}) {\n\t$0\n}"},
	{"if", "if (...) { }", "if (${1:condition}) {\n\t$0\n}"},
	{"if else", "if (...) { } else { }", "if (${1:condition}) {\n\t$2\n} else {\n\t$0\n}"},
	{"for", "for (let i = 0; i < n; i += 1) { }", "for (let ${1:i} = 0; ${1:i} < ${2:n}; ${1:i} += 1) {\n\t$0\n}"},
	{"while", "while (...) { }", "while (${1:condition}) {\n\t$0\n}"},
	{"layer", "layer name", "layer ${1:name}\n\t$0"},
	{"try", "try { } catch (e) { }", "try {\n\t$1\n} catch (${2:e}) {\n\t$0\n}"},
	{"use", "use \"module\" as name", "use \"${1:module}\" as ${2:name}"},
}

// memberBefore matches the end of a line that asks for a member: a name or
// a chain of names, a dot and the start of the member.
var memberBefore = regexp.MustCompile(`([A-Za-z_]\w*(?:\s*\.\s*[A-Za-z_]\w*)*)\s*\.\s*(\w*)$`)

func (s *Server) completion(params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	d, ix, at, err := s.at(p)
	if err != nil {
		return nil, err
	}
	line := ""
	if p.Position.Line < len(d.lines) {
		line = d.text[d.offset(Position{Line: p.Position.Line}):d.offset(p.Position)]
	}
	if inCommentOrString(line) {
		return CompletionList{Items: []CompletionItem{}}, nil
	}
	if m := memberBefore.FindStringSubmatch(line); m != nil {
		chain := strings.Split(strings.Join(strings.Fields(m[1]), ""), ".")
		column := at.column - len([]rune(m[0]))
		t := ix.resolve(chain[0], pos{at.line, column}, ix.scopeAt(at))
		for _, name := range chain[1:] {
			t = t.member(name)
		}
		return CompletionList{Items: membersOf(ix, t)}, nil
	}
	return CompletionList{Items: s.names(ix, at)}, nil
}

// inCommentOrString reports whether the end of line is in a comment or a
// string, where no name is written.
func inCommentOrString(line string) bool {
	quoted := false
	for i := 0; i < len(line); i++ {
		switch {
		case quoted && line[i] == '\\':
			i++
		case line[i] == '"':
			quoted = !quoted
		case !quoted && strings.HasPrefix(line[i:], "-~>"):
			return true
		}
	}
	return quoted
}

// names are the completions of a name at p: the names of the file it can
// see, the names of the modules it uses, the globals and the builtins, the
// keywords and the snippets.
func (s *Server) names(ix *index, p pos) []CompletionItem {
	items := []CompletionItem{}
	seen := map[string]bool{}
	add := func(name, sort string, in info) {
		if seen[name] {
			return
		}
		seen[name] = true
		items = append(items, item(name, sort, in))
	}
	for _, sym := range ix.visible(p) {
		add(sym.name, "0", describeSymbol(ix, sym))
	}
	for i := len(ix.modules) - 1; i >= 0; i-- {
		m := ix.modules[i]
		for _, sym := range m.index.exports() {
			add(sym.name, "1", describeSymbol(m.index, sym))
		}
	}
	for _, name := range globalNames(ix.globals) {
		add(name, "2", describeGlobal(ix.globals, name))
	}
	for _, word := range keywords {
		if !seen[word] {
			seen[word] = true
			items = append(items, CompletionItem{Label: word, Kind: completionKeyword, SortText: "3" + word})
		}
	}
	for _, sn := range snippets {
		items = append(items, CompletionItem{Label: sn.label, Kind: completionSnippet, Detail: sn.detail,
			InsertText: sn.body, InsertTextFormat: 2, SortText: "4" + sn.label})
	}
	return items
}

// membersOf are the completions after t and a dot: the functions of a
// library or a module, or the keys of a hash literal.
func membersOf(ix *index, t target) []CompletionItem {
	items := []CompletionItem{}
	switch {
	case t.sym != nil && t.sym.kind == useDecl && t.path == "":
		if t.sym.module != nil {
			for _, sym := range t.sym.module.index.exports() {
				items = append(items, item(sym.name, "0", describeSymbol(t.sym.module.index, sym)))
			}
		}
	case t.global && !strings.Contains(t.path, "."):
		members := append([]string(nil), ix.globals[t.path]...)
		sort.Strings(members)
		for _, name := range members {
			items = append(items, item(name, "0", describeGlobal(ix.globals, t.path+"."+name)))
		}
	case t.sym != nil || t.module != nil:
		h, ok := valueOf(t).(*ast.HashLiteral)
		if !ok {
			break
		}
		where := ix
		if t.module != nil {
			where = t.module.index
		}
		seen := map[string]bool{}
		for _, pair := range h.Pairs {
			key, ok := pair.Key.(*ast.StringLiteral)
			if !ok || seen[key.Value] || !plainName.MatchString(key.Value) {
				continue
			}
			seen[key.Value] = true
			items = append(items, item(key.Value, "0", describe(where, t.member(key.Value))))
		}
	}
	return items
}

// item is a completion item for name.
func item(name, sortText string, in info) CompletionItem {
	it := CompletionItem{Label: name, Kind: in.kind, Detail: in.detail, SortText: sortText + name}
	if it.Kind == 0 {
		it.Kind = completionVariable
	}
	if in.doc != "" {
		it.Documentation = &MarkupContent{Kind: "markdown", Value: in.doc}
	}
	return it
}
