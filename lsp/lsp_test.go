package lsp

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// The tests open the files of testdata/garden in a server and ask what an
// editor asks. main.liz uses shapes.liz (without as) and the standard
// module flower/lily2d (as lily2d); other.liz uses shapes.liz as s.

func garden(t *testing.T) (*client, string) {
	root, _ := filepath.Abs("testdata/garden")
	c := newClient(t, root, 0)
	uri := c.open("garden/main.liz")
	c.diagnostics(uri)
	return c, uri
}

func TestDiagnostics(t *testing.T) {
	c := newClient(t, "testdata", 0)
	uri := c.open("garden/broken.liz", "let frogs = 1\nputs(forgs)\nfunction f() {\n  let unused = 2\n}\n")
	got := c.diagnostics(uri)

	tests := []struct {
		code     string
		severity int
		text     string // the text of its range
		message  string // a part of its message
		tags     int
	}{
		{"unknown-name", 1, "forgs", "hint: did you mean `frogs`?", 0},
		{"unused", 2, "unused", "unused is never used", 1},
	}
	if len(got) != len(tests) {
		t.Fatalf("got %d diagnostics, want %d: %+v", len(got), len(tests), got)
	}
	byCode := map[string]Diagnostic{}
	for _, d := range got {
		byCode[d.Code] = d
	}
	for _, tt := range tests {
		d, ok := byCode[tt.code]
		if !ok {
			t.Errorf("no %s diagnostic in %+v", tt.code, got)
			continue
		}
		if d.Severity != tt.severity || d.Source != "lizard" || !strings.Contains(d.Message, tt.message) || len(d.Tags) != tt.tags {
			t.Errorf("%s: got %+v", tt.code, d)
		}
		if tt.text != "" && c.textOf(uri, d.Range) != tt.text {
			t.Errorf("%s: its range has %q, want %q", tt.code, c.textOf(uri, d.Range), tt.text)
		}
	}

	// A syntax error.
	c.change(uri, "let frogs = 1\nputs(frogs\n", 2)
	if got := c.diagnostics(uri); len(got) != 1 || got[0].Code != "syntax" || got[0].Severity != 1 || got[0].Range.Start.Line != 1 {
		t.Errorf("the syntax error: %+v", got)
	}

	// Fixing the code clears them.
	c.change(uri, "let frogs = 1\nputs(frogs)\n", 3)
	if got := c.diagnostics(uri); len(got) != 0 {
		t.Errorf("after the fix: %+v", got)
	}
}

func TestDiagnosticsWaitForTheTypingToStop(t *testing.T) {
	c := newClient(t, "testdata", 50*time.Millisecond)
	uri := c.open("garden/typing.liz", "puts(1)\n")
	c.diagnostics(uri)
	for i, text := range []string{"puts(x", "puts(xy", "puts(xyz)\n"} {
		c.change(uri, text, i+2)
	}
	got := c.diagnostics(uri)
	if len(got) != 1 || !strings.Contains(got[0].Message, "xyz") {
		t.Fatalf("want one diagnostic about xyz, of the last text: %+v", got)
	}
	// Nothing else comes: the earlier changes made no diagnostics.
	m := c.call("shutdown", nil)
	for _, p := range c.pending {
		if p.Method == "textDocument/publishDiagnostics" {
			t.Errorf("more diagnostics: %s", p.Params)
		}
	}
	if m.Error != nil {
		t.Error(m.Error)
	}
}

func TestModulesBeingEditedAreReadFromTheEditor(t *testing.T) {
	c, _ := garden(t)
	shapes := c.open("garden/shapes.liz", "function volume(a) {\n  return a\n}\n")
	c.diagnostics(shapes)
	other := c.open("garden/other.liz")
	got := c.diagnostics(other)
	found := false
	for _, d := range got {
		found = found || strings.Contains(d.Message, "area")
	}
	if !found {
		t.Errorf("s.area should be unknown now that shapes.liz has no area: %+v", got)
	}
}

func TestFormatting(t *testing.T) {
	c := newClient(t, "testdata", 0)
	uri := c.open("garden/messy.liz", "let  x=1\nif(x>0){puts( x )}\n")
	c.diagnostics(uri)
	var edits []TextEdit
	c.request("textDocument/formatting", map[string]any{"textDocument": map[string]any{"uri": uri},
		"options": map[string]any{"tabSize": 2, "insertSpaces": true}}, &edits)
	if len(edits) != 1 || edits[0].NewText != "let x = 1\nif (x > 0) { puts(x) }\n" {
		t.Fatalf("got %+v", edits)
	}

	// Code with a syntax error is left alone.
	c.change(uri, "let = 1\n", 2)
	c.diagnostics(uri)
	c.request("textDocument/formatting", map[string]any{"textDocument": map[string]any{"uri": uri}}, &edits)
	if len(edits) != 0 {
		t.Errorf("broken code was formatted: %+v", edits)
	}
}

func labels(items []CompletionItem) map[string]CompletionItem {
	out := map[string]CompletionItem{}
	for _, it := range items {
		out[it.Label] = it
	}
	return out
}

func TestCompletion(t *testing.T) {
	c, uri := garden(t)
	tests := []struct {
		name   string
		text   string // the file's new text, with | at the cursor
		want   []string
		absent []string
		doc    map[string]string // a part of the doc or the detail of an item
	}{
		{"names in scope", "let apple = 1\nfunction f(seed) {\n  let inner = 2\n  |\n}\nlet later = 3\n",
			[]string{"apple", "seed", "inner", "later", "len", "math", "fs", "let", "function", "for"},
			nil, map[string]string{"len": "The number of characters", "seed": "parameter seed of f(seed)"}},
		{"a let of the same function that comes later", "let a = 1\n|\nlet b = 2\n",
			[]string{"a"}, []string{"b"}, nil},
		{"names of a module used without as", "use \"shapes\"\n|\n",
			[]string{"area", "unit"}, []string{"_hidden"}, map[string]string{"area": "The area of a rectangle."}},
		{"math", "puts(math.|)\n", []string{"sin", "pi", "random"}, []string{"len"},
			map[string]string{"sin": "math.sin(x)", "random": "math.random(a, b)"}},
		{"text", "text.sp|", []string{"split", "slice"}, nil, nil},
		{"list and hash", "list.|\nhash.\n", []string{"sort", "insert"}, nil, nil},
		{"a Terrarium module", "fs.|", []string{"read", "write"}, nil, map[string]string{"read": "The text of the file"}},
		{"a drawing target function", "img.|", []string{"fill", "spans"}, nil, map[string]string{"fill": "Paints the rectangle"}},
		{"a module used with as", "use \"shapes\" as s\ns.|\n", []string{"area", "unit"}, []string{"_hidden"},
			map[string]string{"area": "`w`: its width"}},
		{"a standard module", "use \"flower/lily2d\" as lily2d\nlily2d.|\n", []string{"circle", "ellipse"}, nil,
			map[string]string{"circle": "Draws a circle."}},
		{"a standard module that renames its parts' functions", "use \"flower/lily3d\" as lily3d\nlily3d.|\n",
			[]string{"dot", "cross", "cube", "render"}, nil, map[string]string{"dot": "dot(a, b)"}},
		{"names that are each other", "let a = b\nlet b = a\n|\n", []string{"a", "b"}, nil, nil},
		{"the keys of a hash", "let config = {\"width\": 1, \"grow\": function(by) { by }}\nconfig.|\n",
			[]string{"width", "grow"}, nil, map[string]string{"grow": "grow(by)"}},
		{"nested keys", "let c = {\"inner\": {\"deep\": 1}}\nc.inner.|\n", []string{"deep"}, nil, nil},
		{"nothing in a comment", "-~> math.|\n", nil, []string{"sin", "let"}, nil},
		{"nothing in a string", "puts(\"le|\n", nil, []string{"let"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := strings.Replace(tt.text, "|", "", 1)
			c.change(uri, text, 2)
			c.diagnostics(uri)
			cursor := strings.Index(tt.text, "|")
			before := tt.text[:cursor]
			p := Position{Line: strings.Count(before, "\n"), Character: utf16Len(before[strings.LastIndex(before, "\n")+1:])}
			var list CompletionList
			c.request("textDocument/completion", map[string]any{"textDocument": map[string]any{"uri": uri}, "position": p}, &list)
			got := labels(list.Items)
			for _, want := range tt.want {
				if _, ok := got[want]; !ok {
					t.Errorf("no %s in %v", want, sortedLabels(got))
				}
			}
			for _, absent := range tt.absent {
				if _, ok := got[absent]; ok {
					t.Errorf("%s should not be offered", absent)
				}
			}
			for name, part := range tt.doc {
				it := got[name]
				doc := it.Detail
				if it.Documentation != nil {
					doc += "\n" + it.Documentation.Value
				}
				if !strings.Contains(doc, part) {
					t.Errorf("%s: %q doesn't say %q", name, doc, part)
				}
			}
		})
	}
}

func TestCompletionHasSnippets(t *testing.T) {
	c, uri := garden(t)
	var list CompletionList
	c.request("textDocument/completion", c.position(uri, "5, leaf)|"), &list)
	got := labels(list.Items)
	for _, name := range []string{"function", "if", "for", "layer", "try"} {
		found := false
		for _, it := range list.Items {
			found = found || it.Label == name && it.InsertTextFormat == 2
		}
		if !found {
			t.Errorf("no %s snippet", name)
		}
	}
	if got["double"].Kind != completionFunction || got["size"].Kind != completionVariable {
		t.Errorf("kinds: double %d, size %d", got["double"].Kind, got["size"].Kind)
	}
}

func sortedLabels(items map[string]CompletionItem) []string {
	var out []string
	for name := range items {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func TestHover(t *testing.T) {
	c, uri := garden(t)
	tests := []struct {
		marker string
		want   []string
	}{
		{"function dou|ble", []string{"double(x)", "Doubles x.", "`x`: a number"}},
		{"cloud = dou|ble(size)", []string{"double(x)", "Doubles x."}},
		{"double(si|ze)", []string{"let size = 90deg", "`90deg = 1.5708 rad`"}},
		{"size = 9|0deg", []string{"FLOAT `90deg = 1.5708 rad`"}},
		{"leaf = #3f|a34d", []string{"COLOR `#3fa34d`", "rgb(63, 163, 77)", "data:image/svg+xml;base64,"}},
		{"spot = (1|0, 20.5)", []string{"INTEGER `10`"}},
		{"spot = |(10, 20.5)", []string{"POINT 2D", "x = 10", "y = 20.5"}},
		{"le|n(\"abc\")", []string{"len(x)", "The number of characters"}},
		{"math.si|n(1)", []string{"math.sin(x)", "As a float."}},
		{"ma|th.sin(1)", []string{"module math"}},
		{"puts(ar|ea(3, 4)", []string{"area(w, h = 1)", "The area of a rectangle.", "`w`: its width", "area(3, 4)"}},
		{"config.wid|th", []string{"config.width: 640", "INTEGER `640`"}},
		{"config.gr|ow(2)", []string{"grow(by)"}},
		{"lily2d.cir|cle", []string{"circle(target, center, r, style)", "Draws a circle.", "`center`: its middle, a point"}},
		{"as lily|2d", []string{"module \"flower/lily2d\"", "anti-aliased 2D vector graphics"}},
		{"function double(|x)", []string{"parameter x of double(x)", "a number"}},
		{"len(\"a|bc\")", []string{"STRING of 3 characters"}},
	}
	for _, tt := range tests {
		var h *Hover
		c.request("textDocument/hover", c.position(uri, tt.marker), &h)
		if h == nil {
			t.Errorf("%s: no hover", tt.marker)
			continue
		}
		for _, want := range tt.want {
			if !strings.Contains(h.Contents.Value, want) {
				t.Errorf("%s: %q doesn't say %q", tt.marker, h.Contents.Value, want)
			}
		}
	}
}

func TestSignatureHelp(t *testing.T) {
	c := newClient(t, "testdata", 0)
	text := "use \"flower/lily2d\" as lily2d\nfunction area(w, h = 1) { w * h }\nputs(area(3, (1, 2), \nlily2d.circle(null, \ntext.find(\"a\", \"b\", \nmath.random(1, \nputs([1, \n"
	abs, _ := filepath.Abs("testdata/garden/sig.liz")
	uri := c.open("garden/sig.liz", text)
	_ = abs
	c.diagnostics(uri)

	tests := []struct {
		marker string
		label  string
		active int
	}{
		{"area(3, (1, 2), |\nlily2d", "area(w, h = 1)", 2},
		{"puts(|area", "puts(a, b, ...)", 0},
		{"area(|3", "area(w, h = 1)", 0},
		{"area(3, (1, |2)", "area(w, h = 1)", 1},
		{"circle(null, |", "lily2d.circle(target, center, r, style)", 1},
		{"find(\"a\", \"b\", |", "text.find(s, sub, [from])", 2},
		{"random(1, |", "math.random(a, b)", 1},
	}
	for _, tt := range tests {
		var help *SignatureHelp
		c.request("textDocument/signatureHelp", c.position(uri, tt.marker), &help)
		if help == nil {
			t.Errorf("%s: no signature help", tt.marker)
			continue
		}
		sig := help.Signatures[help.ActiveSignature]
		label := strings.Replace(sig.Label, "circle(", "lily2d.circle(", 1)
		if !strings.HasSuffix(label, strings.TrimPrefix(tt.label, "lily2d.")) || help.ActiveParameter != tt.active {
			t.Errorf("%s: got %s at %d, want %s at %d", tt.marker, sig.Label, help.ActiveParameter, tt.label, tt.active)
		}
		if tt.active < len(sig.Parameters) {
			r := sig.Parameters[tt.active].Label
			if r[0] >= r[1] || r[1] > len(sig.Label) {
				t.Errorf("%s: parameter range %v of %q", tt.marker, r, sig.Label)
			}
		}
	}

	var help *SignatureHelp
	c.request("textDocument/signatureHelp", c.position(uri, "puts([1, |"), &help)
	if help != nil {
		t.Errorf("inside an array: %+v", help)
	}
}

func TestDefinition(t *testing.T) {
	c, uri := garden(t)
	shapes, _ := filepath.Abs("testdata/garden/shapes.liz")
	tests := []struct {
		marker string
		file   string // "" for main.liz
		text   string // the text of the range
		line   int    // its line, from 0
	}{
		{"cloud = dou|ble(size)", "", "double", 10},
		{"return res|ult", "", "result", 11},
		{"let result = |x * 2", "", "x", 10},
		{"return x + la|ter", "", "later", 24},
		{"return |x + later", "", "x", 20},
		{"puts(ar|ea(3, 4)", shapes, "area", 6},
		{"config.wid|th", "", "\"width\"", 6},
		{"use \"sha|pes\"", shapes, "", 0},
	}
	for _, tt := range tests {
		var loc *Location
		c.request("textDocument/definition", c.position(uri, tt.marker), &loc)
		if loc == nil {
			t.Errorf("%s: no definition", tt.marker)
			continue
		}
		want := uri
		if tt.file != "" {
			want = pathToURI(tt.file)
		}
		if loc.URI != want || loc.Range.Start.Line != tt.line || c.textOf(loc.URI, loc.Range) != tt.text {
			t.Errorf("%s: got %s %+v %q", tt.marker, loc.URI, loc.Range, c.textOf(loc.URI, loc.Range))
		}
	}

	// A standard module is opened from a copy of its text.
	var loc *Location
	c.request("textDocument/definition", c.position(uri, "lily2d.cir|cle"), &loc)
	if loc == nil || !strings.HasSuffix(loc.URI, "/lizardscript/std/flower/lily2d.liz") {
		t.Fatalf("lily2d.circle: %+v", loc)
	}
	if got := c.textOf(loc.URI, loc.Range); got != "circle" {
		t.Errorf("lily2d.circle: its range has %q", got)
	}

	// A builtin has none.
	c.request("textDocument/definition", c.position(uri, "le|n(\"abc\")"), &loc)
	if loc != nil {
		t.Errorf("len: %+v", loc)
	}
}

func TestReferences(t *testing.T) {
	c, uri := garden(t)
	other, _ := filepath.Abs("testdata/garden/other.liz")
	shapes, _ := filepath.Abs("testdata/garden/shapes.liz")
	tests := []struct {
		marker string
		want   []string // file:line of each, "main" for main.liz
	}{
		{"function dou|ble", []string{"main:10", "main:16"}},
		{"let |x = 1", []string{"main:20", "main:22"}},
		// area is declared in shapes.liz, used by main.liz and, as s.area, by other.liz.
		{"puts(ar|ea", []string{"main:28", "other:2", "shapes:6"}},
	}
	names := map[string]string{uri: "main", pathToURI(other): "other", pathToURI(shapes): "shapes"}
	for _, tt := range tests {
		var locs []Location
		c.request("textDocument/references", map[string]any{"textDocument": map[string]any{"uri": uri},
			"position": c.at(uri, tt.marker), "context": map[string]any{"includeDeclaration": true}}, &locs)
		var got []string
		for _, l := range locs {
			got = append(got, names[l.URI]+":"+itoa(l.Range.Start.Line))
		}
		sort.Strings(got)
		if strings.Join(got, " ") != strings.Join(tt.want, " ") {
			t.Errorf("%s: got %v, want %v", tt.marker, got, tt.want)
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestRename(t *testing.T) {
	c := newClient(t, "testdata", 0)
	text := "let n = 1\nfunction f(n) {\n  return n + 1\n}\nputs(n, f(n))\n"
	uri := c.open("garden/rename.liz", text)
	c.diagnostics(uri)

	var edit WorkspaceEdit
	c.request("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uri},
		"position": c.at(uri, "let |n = 1"), "newName": "count"}, &edit)
	got := applyEdits(text, edit.Changes[uri])
	want := "let count = 1\nfunction f(n) {\n  return n + 1\n}\nputs(count, f(count))\n"
	if got != want {
		t.Errorf("the outer n: got\n%s", got)
	}

	c.request("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uri},
		"position": c.at(uri, "return |n"), "newName": "m"}, &edit)
	got = applyEdits(text, edit.Changes[uri])
	want = "let n = 1\nfunction f(m) {\n  return m + 1\n}\nputs(n, f(n))\n"
	if got != want {
		t.Errorf("the parameter: got\n%s", got)
	}

	errorsFor := []struct{ marker, name string }{
		{"pu|ts", "x"},          // a builtin
		{"let |n", "let"},       // a keyword
		{"let |n", "two words"}, // not a name
	}
	for _, tt := range errorsFor {
		m := c.call("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uri},
			"position": c.at(uri, tt.marker), "newName": tt.name})
		if m.Error == nil {
			t.Errorf("%s to %s: no error", tt.marker, tt.name)
		}
	}

	var prepared struct {
		Range       Range  `json:"range"`
		Placeholder string `json:"placeholder"`
	}
	c.request("textDocument/prepareRename", c.position(uri, "f(|n)"), &prepared)
	if prepared.Placeholder != "n" {
		t.Errorf("prepareRename: %+v", prepared)
	}
}

func TestRenameAcrossModules(t *testing.T) {
	c, uri := garden(t)
	other, _ := filepath.Abs("testdata/garden/other.liz")
	shapes, _ := filepath.Abs("testdata/garden/shapes.liz")
	var edit WorkspaceEdit
	c.request("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uri},
		"position": c.at(uri, "puts(ar|ea"), "newName": "surface"}, &edit)
	for _, file := range []string{uri, pathToURI(other), pathToURI(shapes)} {
		if len(edit.Changes[file]) != 1 || edit.Changes[file][0].NewText != "surface" {
			t.Errorf("%s: %+v", file, edit.Changes[file])
		}
	}
}

// applyEdits applies edits that don't overlap to text.
func applyEdits(text string, edits []TextEdit) string {
	d := newDocument("", "", text, 0)
	sort.Slice(edits, func(i, j int) bool { return d.offset(edits[i].Range.Start) > d.offset(edits[j].Range.Start) })
	for _, e := range edits {
		r := e.Range
		d.edit(&r, e.NewText)
	}
	return d.text
}

func TestDocumentSymbols(t *testing.T) {
	c, uri := garden(t)
	var symbols []DocumentSymbol
	c.request("textDocument/documentSymbol", map[string]any{"textDocument": map[string]any{"uri": uri}}, &symbols)
	got := outlineText(symbols, "")
	want := strings.Join([]string{
		"lily2d 2", "size 13", "leaf 13", "spot 13", "config 2", "double 12",
		"sky 3", "  cloud 13", "outer 12", "  inner 12",
	}, "\n")
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	for _, s := range symbols {
		if s.Name == "sky" && c.textOf(uri, s.SelectionRange) != "sky" {
			t.Errorf("the layer's name: %q", c.textOf(uri, s.SelectionRange))
		}
	}
}

func outlineText(symbols []DocumentSymbol, indent string) string {
	var lines []string
	for _, s := range symbols {
		lines = append(lines, indent+s.Name+" "+itoa(s.Kind))
		if inner := outlineText(s.Children, indent+"  "); inner != "" {
			lines = append(lines, inner)
		}
	}
	return strings.Join(lines, "\n")
}

func TestWorkspaceSymbols(t *testing.T) {
	c, _ := garden(t)
	var symbols []SymbolInformation
	c.request("workspace/symbol", map[string]any{"query": "ara"}, &symbols)
	if len(symbols) != 1 || symbols[0].Name != "area" || !strings.HasSuffix(symbols[0].Location.URI, "shapes.liz") {
		t.Errorf("got %+v", symbols)
	}
}

func TestFoldingRanges(t *testing.T) {
	c := newClient(t, "testdata", 0)
	text := "-~> one\n-~> two\nfunction f() {\n  return 1\n}\nlayer sky\n  puts(1)\n  puts(2)\nlet h = {\n  \"a\": 1\n}\n"
	uri := c.open("garden/fold.liz", text)
	c.diagnostics(uri)
	var ranges []FoldingRange
	c.request("textDocument/foldingRange", map[string]any{"textDocument": map[string]any{"uri": uri}}, &ranges)
	var got []string
	for _, r := range ranges {
		got = append(got, itoa(r.StartLine)+"-"+itoa(r.EndLine)+r.Kind)
	}
	sort.Strings(got)
	want := "0-1comment 2-3 5-7 8-9"
	if strings.Join(got, " ") != want {
		t.Errorf("got %v, want %s", got, want)
	}
}

func TestSemanticTokens(t *testing.T) {
	c := newClient(t, "testdata", 0)
	text := "use \"flower/lily2d\" as lily2d\nlet c = #ff8800\nfunction f(p) {\n  return p + (1, 2)\n}\nlayer sky\n  puts(math.pi, f(c), len(\"x\")) -~> done\n"
	uri := c.open("garden/tokens.liz", text)
	c.diagnostics(uri)
	var result SemanticTokens
	c.request("textDocument/semanticTokens/full", map[string]any{"textDocument": map[string]any{"uri": uri}}, &result)

	lines := strings.Split(text, "\n")
	got := map[string]string{}
	line, char := 0, 0
	for i := 0; i+4 < len(result.Data); i += 5 {
		if result.Data[i] > 0 {
			char = 0
		}
		line += result.Data[i]
		char += result.Data[i+1]
		word := string([]rune(lines[line])[char : char+result.Data[i+2]])
		kind := tokenTypes[result.Data[i+3]]
		if result.Data[i+4]&modDefaultLibrary != 0 {
			kind += "+lib"
		}
		if result.Data[i+4]&modDeclaration != 0 {
			kind += "+decl"
		}
		got[word] = kind
	}
	tests := map[string]string{
		"use": "keyword", "as": "keyword", "lily2d": "namespace+decl", "#ff8800": "color",
		"f": "function", "p": "parameter", "(": "point", "layer": "keyword", "sky": "label",
		"puts": "function+lib", "math": "namespace+lib", "pi": "property+lib", "len": "function+lib",
		"\"x\"": "string", "-~> done": "comment", "1": "number", "function": "keyword",
	}
	for word, want := range tests {
		if got[word] != want {
			t.Errorf("%s: got %q, want %q", word, got[word], want)
		}
	}
}

func TestColors(t *testing.T) {
	c := newClient(t, "testdata", 0)
	uri := c.open("garden/colors.liz", "let a = #f80\nlet b = #3fa34d80\n")
	c.diagnostics(uri)
	var colors []ColorInformation
	c.request("textDocument/documentColor", map[string]any{"textDocument": map[string]any{"uri": uri}}, &colors)
	if len(colors) != 2 || c.textOf(uri, colors[0].Range) != "#f80" || colors[0].Color.Red != 1 || colors[0].Color.Blue != 0 {
		t.Fatalf("got %+v", colors)
	}
	if a := colors[1].Color.Alpha; a < 0.5 || a > 0.51 {
		t.Errorf("the alpha of #3fa34d80: %v", a)
	}

	var presentations []ColorPresentation
	c.request("textDocument/colorPresentation", map[string]any{"textDocument": map[string]any{"uri": uri},
		"color": Color{Red: 1, Green: 0.5, Blue: 0, Alpha: 1}, "range": colors[0].Range}, &presentations)
	if len(presentations) != 2 || presentations[0].Label != "#ff8000" || presentations[1].Label != "rgb(255, 128, 0)" {
		t.Errorf("got %+v", presentations)
	}
	c.request("textDocument/colorPresentation", map[string]any{"textDocument": map[string]any{"uri": uri},
		"color": Color{Red: 0, Green: 0, Blue: 0, Alpha: 0.5}, "range": colors[0].Range}, &presentations)
	if presentations[0].Label != "#00000080" || presentations[0].TextEdit.NewText != "#00000080" {
		t.Errorf("with alpha: %+v", presentations)
	}
}

func TestProtocolErrors(t *testing.T) {
	c := newClient(t, "testdata", 0)
	tests := []struct {
		method string
		params any
		code   int
	}{
		{"textDocument/unknown", nil, codeMethodNotFound},
		{"textDocument/hover", map[string]any{"textDocument": map[string]any{"uri": "file:///not/open.liz"}}, codeInvalidParams},
		{"textDocument/hover", "not an object", codeInvalidParams},
	}
	for _, tt := range tests {
		m := c.call(tt.method, tt.params)
		if m.Error == nil || m.Error.Code != tt.code {
			t.Errorf("%s: got %+v, want code %d", tt.method, m.Error, tt.code)
		}
	}
}

func TestRequestsBeforeInitialize(t *testing.T) {
	s := New(Options{})
	m := &message{ID: json.RawMessage("1"), Method: "textDocument/hover", Params: json.RawMessage("{}")}
	if _, err := s.call(m); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("got %v", err)
	}
}
