package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"lizardscript/ast"
	"lizardscript/evaluator"
	"lizardscript/object"
	"lizardscript/terrarium"
	_ "lizardscript/terrarium/window" // so the globals have window and pen, as in lizardscript
	"strings"
	"testing"
)

func TestFraming(t *testing.T) {
	var b bytes.Buffer
	sent := &message{ID: json.RawMessage("7"), Method: "textDocument/hover", Params: json.RawMessage(`{"a":"ü"}`)}
	if err := writeMessage(&b, sent); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(b.String(), "Content-Length: ") {
		t.Fatalf("no header: %q", b.String())
	}
	got, err := readMessage(bufio.NewReader(&b))
	if err != nil {
		t.Fatal(err)
	}
	if got.JSONRPC != "2.0" || string(got.ID) != "7" || got.Method != sent.Method || string(got.Params) != string(sent.Params) {
		t.Errorf("got %+v", got)
	}

	bad := []string{
		"Content-Type: x\r\n\r\n{}",     // no length
		"Content-Length: 10\r\n\r\n{}",  // too short
		"Content-Length: two\r\n\r\n{}", // not a number
	}
	for _, input := range bad {
		if _, err := readMessage(bufio.NewReader(strings.NewReader(input))); err == nil {
			t.Errorf("%q: no error", input)
		}
	}
	m, err := readMessage(bufio.NewReader(strings.NewReader("Content-Length: 3\r\n\r\n{x}")))
	if err != nil || m.Error == nil || m.Error.Code != codeParseError {
		t.Errorf("bad JSON: %+v %v", m, err)
	}
}

func TestPositions(t *testing.T) {
	// "🦎" is one column of the language and two UTF-16 units.
	d := newDocument("", "", "let a = \"🦎\" + b\nlet c\n", 0)
	tests := []struct {
		line, column int
		want         Position
	}{
		{1, 1, Position{0, 0}},
		{1, 10, Position{0, 9}},  // the lizard
		{1, 11, Position{0, 11}}, // after it
		{1, 15, Position{0, 15}}, // b
		{2, 5, Position{1, 4}},
		{9, 1, Position{2, 0}}, // past the end
	}
	for _, tt := range tests {
		got := d.toPosition(tt.line, tt.column)
		if got != tt.want {
			t.Errorf("(%d, %d): got %+v, want %+v", tt.line, tt.column, got, tt.want)
		}
		if tt.line <= 2 {
			if line, column := d.fromPosition(got); line != tt.line || column != tt.column {
				t.Errorf("%+v back: got (%d, %d)", got, line, column)
			}
		}
	}
	if got := d.offset(Position{0, 11}); d.text[got:got+3] != "\" +" {
		t.Errorf("offset after the lizard: %d", got)
	}

	d.edit(&Range{Position{0, 15}, Position{0, 16}}, "bee")
	if d.lines[0] != "let a = \"🦎\" + bee" {
		t.Errorf("edit: %q", d.lines[0])
	}
}

func TestURIs(t *testing.T) {
	tests := []struct{ uri, path string }{
		{"file:///Users/me/Langs%26Compilers/4th%20Semester/a.liz", "/Users/me/Langs&Compilers/4th Semester/a.liz"},
		{"file:///tmp/a.liz", "/tmp/a.liz"},
		{"untitled:Untitled-1", ""},
	}
	for _, tt := range tests {
		if got := uriToPath(tt.uri); got != tt.path {
			t.Errorf("%s: got %q", tt.uri, got)
		}
		if tt.path != "" && uriToPath(pathToURI(tt.path)) != tt.path {
			t.Errorf("%s: the URI %s doesn't come back", tt.path, pathToURI(tt.path))
		}
	}
}

// TestDocCommentsAreReadLikeLizardDocs checks that the server reads a doc
// comment the way lizard.docs does.
func TestDocCommentsAreReadLikeLizardDocs(t *testing.T) {
	code := strings.Join([]string{
		"-~> A circle around center,",
		"-~> drawn in one go.",
		"-~>",
		"-~> center: the middle",
		"-~>   of the circle",
		"-~> radius: how big",
		"-~> example: circle((10, 10), 5)",
		"-~> circle((0, 0))",
		"function circle(center, radius = 1) { center }",
		"",
		"function bare(x) { x }",
	}, "\n")
	env := object.NewEnvironment()
	terrarium.Install(env)
	lizard, _ := env.Get("lizard")
	docs := lizard.(*object.Hash)
	var fn object.Object
	docs.Each(func(pair object.HashPair) {
		if pair.Key.Inspect() == "docs" {
			fn = pair.Value
		}
	})
	result := fn.(*object.Builtin).Fn(&object.String{Value: code})
	want, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("lizard.docs: %s", result.Inspect())
	}

	ix := newIndex("", code, builtinGlobals(), nil)
	for _, sym := range ix.symbols {
		if sym.let == nil {
			continue
		}
		f, _ := sym.let.Value.(*ast.FunctionLiteral)
		got := ix.docOf(sym.let.Range().StartLine, f)
		entry := field(want, sym.name).(*object.Hash)
		if s := field(entry, "summary").Inspect(); s != got.summary {
			t.Errorf("%s summary: got %q, lizard.docs %q", sym.name, got.summary, s)
		}
		example := field(entry, "example")
		if wantExample := map[bool]string{true: example.Inspect(), false: ""}[example != evaluator.NULL]; wantExample != got.example {
			t.Errorf("%s example: got %q, lizard.docs %q", sym.name, got.example, wantExample)
		}
		for _, p := range field(entry, "params").(*object.Array).Elements {
			name := field(p.(*object.Hash), "name").Inspect()
			doc := field(p.(*object.Hash), "doc")
			wantDoc := ""
			if doc != evaluator.NULL {
				wantDoc = doc.Inspect()
			}
			if got.params[name] != wantDoc {
				t.Errorf("%s %s: got %q, lizard.docs %q", sym.name, name, got.params[name], wantDoc)
			}
		}
	}
}

func field(h *object.Hash, key string) object.Object {
	var out object.Object = evaluator.NULL
	h.Each(func(pair object.HashPair) {
		if pair.Key.Inspect() == key {
			out = pair.Value
		}
	})
	return out
}

func TestParseSignature(t *testing.T) {
	tests := []struct {
		label  string
		params []string
	}{
		{"len(x)", []string{"x"}},
		{"math.random()", nil},
		{"text.find(s, sub, [from])", []string{"s", "sub", "[from]"}},
		{"fs.list(dir, options = {})", []string{"dir", "options = {}"}},
		{"img.composite(target, source, mode = \"normal\", opacity = 1, pos = (0, 0))",
			[]string{"target", "source", "mode = \"normal\"", "opacity = 1", "pos = (0, 0)"}},
	}
	for _, tt := range tests {
		got := parseSignature(tt.label).params
		if strings.Join(got, "|") != strings.Join(tt.params, "|") {
			t.Errorf("%s: got %q", tt.label, got)
		}
	}
}

func TestEveryBuiltinIsDocumented(t *testing.T) {
	globals := builtinGlobals()
	for name, value := range terrarium.CheckGlobals() {
		globals[name] = value
	}
	for _, name := range globalNames(globals) {
		if len(globals[name]) == 0 {
			if _, ok := builtinDocs[name]; !ok {
				t.Errorf("%s has no documentation", name)
			}
			continue
		}
		for _, member := range globals[name] {
			in := describeGlobal(globals, name+"."+member)
			if in.doc == "" {
				t.Errorf("%s.%s has no documentation", name, member)
			}
		}
	}
}
