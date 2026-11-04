package format

import (
	"bytes"
	"github.com/alejandruxxug/lizard-script/internal/corpus"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/tools/repl"
	"math/rand"
	"strings"
	"testing"
)

func TestRules(t *testing.T) {
	tests := []struct {
		name, input, expected string
	}{
		{"spaces around operators", "let x=1+2*3", "let x = 1 + 2 * 3\n"},
		{"space after commas", "puts(1,2 ,3)", "puts(1, 2, 3)\n"},
		{"no space inside brackets", "let a = [ 1, ( 2 + 3 ) ]", "let a = [1, (2 + 3)]\n"},
		{"prefix operators", "let a = - x + ! b", "let a = -x + !b\n"},
		{"minus after an operator", "let a = 1 -  -2", "let a = 1 - -2\n"},
		{"calls and indexes", "f (1) [0] . name", "f(1)[0].name\n"},
		{"function parameters", "let f = function (a,b = 2) { a }", "let f = function(a, b = 2) { a }\n"},
		{"function declaration", "function  add (a, b) {\n    a + b\n}", "function add(a, b) {\n  a + b\n}\n"},
		{"empty block", "function f() { }", "function f() {}\n"},
		{"keywords and parentheses", "if(a){ 1 }\nwhile(x){}", "if (a) { 1 }\nwhile (x) {}\n"},
		{"for header", "for(let i=0;i<3;i+=1){ puts(i) }", "for (let i = 0; i < 3; i += 1) { puts(i) }\n"},
		{"empty for header", "for (;;) { break }", "for (;;) { break }\n"},
		{"hash on one line", "let h = { \"a\" : 1, \"b\":2 }", "let h = {\"a\": 1, \"b\": 2}\n"},
		{"hash over lines", "let h = {\"a\": 1, \"b\": 2,\n  \"c\": 3}",
			"let h = {\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3\n}\n"},
		{"hash over lines keeps a trailing comma", "let h = {\"a\": 1,\n}", "let h = {\n  \"a\": 1,\n}\n"},
		{"empty hash over lines", "let h = {\n}", "let h = {}\n"},
		{"a pair that goes on", "let h = {\"a\": 1 +\n2,\n\"b\": 2}", "let h = {\n  \"a\": 1 +\n    2,\n  \"b\": 2\n}\n"},
		{"indentation", "if (a) {\nif (b) {\n    x()\n        }\n}", "if (a) {\n  if (b) {\n    x()\n  }\n}\n"},
		{"tabs become spaces", "if (a) {\n\tx()\n}", "if (a) {\n  x()\n}\n"},
		{"brace on the same line", "if (a)\n{\n  x()\n}", "if (a) {\n  x()\n}\n"},
		{"else on the line of the brace", "if (a) {\n  x()\n}\nelse {\n  y()\n}", "if (a) {\n  x()\n} else {\n  y()\n}\n"},
		{"elif on the line of the brace", "if (a) {\n  x()\n}\nelif (b) {\n  y()\n}", "if (a) {\n  x()\n} elif (b) {\n  y()\n}\n"},
		{"one-line blocks stay one per line", "if (a) { 1 }\nelse { 2 }", "if (a) { 1 }\nelse { 2 }\n"},
		{"catch on the line of the brace", "try {\n  x()\n}\ncatch (e) {\n  y()\n}", "try {\n  x()\n} catch (e) {\n  y()\n}\n"},
		{"a statement that goes on", "let total = first +\nsecond +\n      third", "let total = first +\n  second +\n  third\n"},
		{"arguments over lines", "puts(1,\n2,\n       3)", "puts(1,\n  2,\n  3)\n"},
		{"closing bracket on its own line", "let a = [\n1,\n2\n    ]", "let a = [\n  1,\n  2\n]\n"},
		{"function inside a call", "each(xs, function(x) {\nputs(x)\n})", "each(xs, function(x) {\n  puts(x)\n})\n"},
		{"blank lines collapse", "a()\n\n\n\nb()", "a()\n\nb()\n"},
		{"blank lines at the ends go", "\n\na()\n\n", "a()\n"},
		{"trailing spaces go", "a()   \nb() -~> note   ", "a()\nb() -~> note\n"},
		{"final newline", "a()", "a()\n"},
		{"empty program", "", ""},
		{"comments align", "let a = 1 -~> one\nlet bbb = 22 -~> two\n\nlet c = 3   -~> three",
			"let a = 1    -~> one\nlet bbb = 22 -~> two\n\nlet c = 3 -~> three\n"},
		{"comment lines are indented", "if (a) {\n-~> why\nx()\n}", "if (a) {\n  -~> why\n  x()\n}\n"},
		{"a comment keeps the brace on the next line", "if (a) -~> note\n{\n  x()\n}", "if (a) -~> note\n{\n  x()\n}\n"},
		{"semicolons", "a() ; b() ;", "a(); b();\n"},
		{"empty statements", "a()\n;;", "a()\n;;\n"},
		{"strings and units as written", "let s = \"a\\tb\"   +\"\\u{1F98E}\"\nlet r = 45deg", "let s = \"a\\tb\" + \"\\u{1F98E}\"\nlet r = 45deg\n"},
		{"a string over lines is left alone", "let s = \"one   \n   two\"", "let s = \"one   \n   two\"\n"},
		{"points and colors", "let p = ( 1,2 )\nlet c = #FFaa00", "let p = (1, 2)\nlet c = #FFaa00\n"},
		{"dot access on a number", "let x = 1 .y", "let x = 1.y\n"},
		{"use", "use   \"shapes\"   as   s", "use \"shapes\" as s\n"},
		{"layer body", "layer sky\n      let x = 1\n      puts(x)\nputs(2)", "layer sky\n  let x = 1\n  puts(x)\nputs(2)\n"},
		{"nested layers", "layer a\n layer b\n    x()\n y()\nz()", "layer a\n  layer b\n    x()\n  y()\nz()\n"},
		{"layer in a block", "if (a) {\n    layer sun\n        x()\n    y()\n}", "if (a) {\n  layer sun\n    x()\n  y()\n}\n"},
		{"a layer starts its line", "a(); layer b\n         c()", "a();\nlayer b\n  c()\n"},
		{"comment inside a layer body", "layer a\n  x()\n  -~> more\nlayer b\n  y()", "layer a\n  x()\n  -~> more\nlayer b\n  y()\n"},
		{"comment after a layer body", "layer a\n  x()\n-~> next\ny()", "layer a\n  x()\n-~> next\ny()\n"},
		{"return value", "function f() {\n  return  1\n}", "function f() {\n  return 1\n}\n"},
		{"CRLF line ends", "a()\r\nif (b) {\r\n  c()\r\n}\r\n", "a()\nif (b) {\n  c()\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Source(tt.input)
			if err != nil {
				t.Fatalf("Source(%q): %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("Source(%q)\n--- expected:\n%s--- got:\n%s", tt.input, tt.expected, got)
			}
			again, err := Source(got)
			if err != nil || again != got {
				t.Errorf("formatting %q again changed it:\n%s (%v)", got, again, err)
			}
		})
	}
}

func TestSyntaxErrors(t *testing.T) {
	_, err := Source("let = 1\nlet y = (")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.HasPrefix(err.Error(), "[line 1, column 5] syntax error:") {
		t.Errorf("wrong error: %v", err)
	}
}

// programs are the repository's programs that parse: the .liz files and
// the guide's code blocks.
func programs() []corpus.Program {
	all := append(corpus.Files("tests", "examples", "terrarium/lib", "terrarium/std", "bench"), corpus.GuideBlocks()...)
	var ok []corpus.Program
	for _, p := range all {
		parsed := parser.New(lexer.New(p.Source))
		parsed.ParseProgram()
		if len(parsed.Errors()) == 0 {
			ok = append(ok, p)
		}
	}
	return ok
}

// Every program of the repository formats to one with the same tree, and
// formatting that again changes nothing. Source checks the tree itself, so
// an error here is a broken guarantee.
func TestRepositoryKeepsMeaning(t *testing.T) {
	list := programs()
	if len(list) < 100 {
		t.Fatalf("found only %d programs", len(list))
	}
	for _, p := range list {
		out, err := Source(p.Source)
		if err != nil {
			t.Errorf("%s: %v", p.Name, err)
			continue
		}
		if again, err := Source(out); err != nil || again != out {
			t.Errorf("%s: formatting twice changed it (%v)", p.Name, err)
		}
	}
}

// The programs of tests/valid print the same before and after formatting.
func TestValidProgramsPrintTheSame(t *testing.T) {
	for _, p := range corpus.Files("tests/valid") {
		out, err := Source(p.Source)
		if err != nil {
			t.Errorf("%s: %v", p.Name, err)
			continue
		}
		if out == p.Source {
			continue
		}
		name := corpus.Root() + "/" + p.Name
		if before, after := run(name, p.Source), run(name, out); before != after {
			t.Errorf("%s prints differently once formatted\n--- before:\n%s--- after:\n%s", p.Name, before, after)
		}
	}
}

func run(name, source string) string {
	var out bytes.Buffer
	repl.RunFile(name, source, &out, &out, false)
	return out.String()
}

// The layout depends on the tokens and the lines, not on the spaces: the
// repository's programs, with random spaces added and taken away between
// the tokens of a line, format to the same code as before.
func TestSpacesDontMatter(t *testing.T) {
	random := rand.New(rand.NewSource(1))
	for _, p := range programs() {
		want, err := Source(p.Source)
		if err != nil {
			continue
		}
		for round := 0; round < 3; round++ {
			messy := respace(p.Source, random)
			got, err := Source(messy)
			if err != nil {
				t.Errorf("%s with other spaces: %v\n%s", p.Name, err, messy)
				break
			}
			if got != want {
				t.Errorf("%s with other spaces formats differently:\n%s", p.Name, messy)
				break
			}
		}
	}
}

// respace writes the tokens of source on the lines they were on, at the
// columns they were at when the line has a layer (whose body is measured by
// columns), and else with a random indentation and random spaces between
// tokens, where no space is needed to keep two tokens apart.
func respace(source string, random *rand.Rand) string {
	items := readTokens(source)
	hasLayer := strings.Contains(source, "layer")
	var out strings.Builder
	line := 1
	for i, it := range items {
		if i == 0 || it.Line > items[i-1].EndLine {
			for ; line < it.Line; line++ {
				out.WriteString(strings.Repeat(" ", random.Intn(2)) + "\n")
			}
			if hasLayer {
				out.WriteString(strings.Repeat(" ", it.Column-1))
			} else {
				out.WriteString(strings.Repeat(" ", random.Intn(7)))
			}
		} else {
			out.WriteString(spaces(items[i-1], it, random))
		}
		out.WriteString(it.text)
		line = it.EndLine
	}
	out.WriteString("\n")
	return out.String()
}

// spaces is a random space between two tokens of a line: none when the
// tokens can't touch.
func spaces(prev, next item, random *rand.Rand) string {
	n := random.Intn(3)
	if n == 0 && glues(prev, next) {
		n = 1
	}
	return strings.Repeat(" ", n)
}

// glues reports whether the two tokens, written together, could read as
// other tokens.
func glues(prev, next item) bool {
	together := lexer.New(prev.text + next.text)
	first, second := together.NextToken(), together.NextToken()
	return first.Type != prev.Type || first.Literal != prev.Literal ||
		second.Type != next.Type || second.Literal != next.Literal
}

// Code that parses always formats, and formatting is idempotent. The seeds
// run with go test; go test -fuzz FuzzSource ./format looks for more.
func FuzzSource(f *testing.F) {
	for _, seed := range []string{
		"let x=1", "if (a) { 1 }\nelse { 2 }", "let h = {\"a\": 1,\n\"b\": [2,\n3]}",
		"layer a\n  layer b\n    x()\n  y()", "f(function() {\n  return 1 -~> one\n})",
		"try { a() }\ncatch { b() }", "let s = \"a\nb\" -~> c", "x = - -1 - (2, 3).x", "-~>\x830", "-~>\r",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		p := parser.New(lexer.New(text))
		p.ParseProgram()
		if len(p.Errors()) > 0 {
			return
		}
		out, err := Source(text)
		if err != nil {
			t.Fatalf("Source(%q): %v", text, err)
		}
		if again, err := Source(out); err != nil || again != out {
			t.Fatalf("not idempotent on %q:\n%q\n%q (%v)", text, out, again, err)
		}
	})
}
