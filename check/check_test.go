package check

import (
	"fmt"
	"lizardscript/lexer"
	"lizardscript/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// show writes a diagnostic the way the tests compare it.
func show(d Diagnostic) string {
	s := fmt.Sprintf("%d:%d %s %s: %s", d.Line, d.Column, d.Severity, d.Rule, d.Message)
	if d.Hint != "" {
		s += " (" + d.Hint + ")"
	}
	return s
}

func checkAll(name, source string, globals map[string][]string) []string {
	shown := []string{}
	for _, d := range SourceWith(name, source, globals) {
		shown = append(shown, show(d))
	}
	return shown
}

func TestRules(t *testing.T) {
	host := map[string][]string{"fs": {"read", "write"}, "canvas": nil}
	tests := []struct {
		name, input string
		expected    []string
	}{
		// Names.
		{"a defined name", "let x = 1\nputs(x)", nil},
		{"an unknown name", "puts(y)", []string{"1:6 error unknown-name: identifier not found: y"}},
		{"did you mean", "let count = 0\nputs(cuont)",
			[]string{"2:6 error unknown-name: identifier not found: cuont (did you mean `count`?)"}},
		{"did you mean a builtin", "puts(lenn([1]))",
			[]string{"1:6 error unknown-name: identifier not found: lenn (did you mean `len`?)"}},
		{"a short name has no hint", "puts(xy)", []string{"1:6 error unknown-name: identifier not found: xy"}},
		{"read before its let", "puts(x)\nlet x = 1",
			[]string{"1:6 error unknown-name: identifier not found: x: its let, at line 2, comes later"}},
		{"read in a block before the let around it", "if (true) { puts(x) }\nlet x = 1",
			[]string{"1:18 error unknown-name: identifier not found: x: its let, at line 2, comes later"}},
		{"its own let's value reads the outer one", "let x = 1\nif (true) { let x = x + 1; puts(x) }",
			[]string{"2:17 warning shadow: let x hides the x of line 1"}},
		{"a function sees a later let", "function f() { g() + total }\nfunction g() { 1 }\nlet total = 2\nputs(f())", nil},
		{"recursion", "function fact(n) { if (n < 2) { 1 } else { n * fact(n - 1) } }\nputs(fact(5))", nil},
		{"a let in a block is gone after it", "if (true) { let inner = 1; puts(inner) }\nputs(inner)",
			[]string{"2:6 error unknown-name: identifier not found: inner"}},
		{"a loop round doesn't see the round before", "let i = 0\nwhile (i < 2) { if (i > 0) { puts(v) }; let v = i; puts(v); i += 1 }",
			[]string{"2:35 error unknown-name: identifier not found: v: its let, at line 2, comes later"}},
		{"for header", "for (let i = 0; i < 3; i += 1) { puts(i) }\nputs(i)",
			[]string{"2:6 error unknown-name: identifier not found: i"}},
		{"parameters and defaults", "function f(a, b = a * 2) { a + b }\nputs(f(1))", nil},
		{"catch name", "try { error(\"x\") } catch (e) { puts(e.message) }", nil},
		{"layers are scopes", "layer sky\n  let sun = 1\n  puts(sun)\nputs(sun)",
			[]string{"4:6 error unknown-name: identifier not found: sun"}},
		{"builtins, null and libraries", "puts(len([1]), null, math.pi, text.upper(\"a\"), list.copy([]), hash.keys({}))", nil},
		{"host globals", "puts(fs.read(\"a\"), canvas)", nil},
		{"continue as a name", "puts(continue + 1)",
			[]string{"1:6 error unknown-name: identifier not found: continue (continue is a statement only when it is alone, at the end of its line or before a ;)"}},

		// Assignment.
		{"assign a declared name", "let x = 1\nx = 2\nputs(x)", nil},
		{"assign an undeclared name", "y = 2",
			[]string{"1:1 error assign: can't assign to y: it was never declared, use let y = ..."}},
		{"compound assign an undeclared name", "total += 1",
			[]string{"1:1 error unknown-name: identifier not found: total", "1:1 error assign: can't assign to total: it was never declared, use let total = ..."}},
		{"assign a builtin", "len = 1",
			[]string{"1:1 error assign: can't assign to len: it was never declared, use let len = ..."}},
		{"assign a host global", "canvas = 1", nil},
		{"a function assigns a later let", "function bump() { count += 1 }\nlet count = 0\nbump()", nil},

		// Arguments.
		{"right number", "function add(a, b) { a + b }\nputs(add(1, 2))", nil},
		{"too few", "function add(a, b) { a + b }\nputs(add(1))",
			[]string{"2:6 error arguments: wrong number of arguments to add: want 2, got 1"}},
		{"with defaults", "function f(a, b = 1) { a + b }\nputs(f(1), f(1, 2), f())",
			[]string{"2:21 error arguments: wrong number of arguments to f: want 1 to 2, got 0"}},
		{"a reassigned function is not known", "let f = function(a) { a }\nf = function(a, b) { a + b }\nputs(f(1, 2))", nil},
		{"a declared-twice function is not known", "function f(a) { a }\nfunction f(a, b) { a + b }\nputs(f(1, 2))", nil},

		// Members of modules.
		{"a library member", "puts(math.sin(1))", nil},
		{"an unknown library member called", "puts(math.sine(1))",
			[]string{"1:6 error member: math has no sine (did you mean `math.sin`?)"}},
		{"an unknown member read", "let f = fs.reed",
			[]string{"1:9 warning member: fs has no reed (did you mean `fs.read`?)"}},
		{"a member the program adds", "math.tau2 = 1\nputs(math.tau2)", nil},
		{"a shadowed library is not checked", "let math = {\"sine\": 1}\nputs(math.sine)", nil},

		// Return, unreachable code.
		{"code after return", "function f() {\n  return 1\n  puts(2)\n  puts(3)\n}\nf()",
			[]string{"3:3 warning unreachable: this code never runs: it comes after a return"}},
		{"code after break", "while (true) { break; puts(1) }",
			[]string{"1:23 warning unreachable: this code never runs: it comes after a break"}},
		{"a return value on the next line", "function f() {\n  return\n    1 + 2\n}\nf()",
			[]string{"2:3 error return: this return gives null: the value on the next line is a statement of its own (put the value on the line of the return)"}},

		// Warnings.
		{"unused let", "function f() { let tmp = 1; 2 }\nf()",
			[]string{"1:20 warning unused: tmp is never used (start its name with _ if that is on purpose)"}},
		{"unused parameter", "function f(a, b) { a }\nf(1, 2)",
			[]string{"1:15 warning unused: the parameter b is never used (start its name with _ if that is on purpose)"}},
		{"_ names are unused on purpose", "function f(a, _b) { let _c = 1; a }\nf(1, 2)", nil},
		{"top-level lets may be used by another file", "let unused = 1", nil},
		{"assigned but never read", "function f() { let x = 1; x = 2 }\nf()",
			[]string{"1:20 warning unused: x is never used (start its name with _ if that is on purpose)"}},
		{"shadowing a parameter", "function f(a) { if (a) { let a = 2; a } }\nf(1)",
			[]string{"1:30 warning shadow: let a hides the parameter a of line 1"}},
		{"a later outer let is not hidden", "if (true) { let x = 1; puts(x) }\nlet x = 2", nil},
		{"literals", "puts(1 == 1.0, \"a\" !== \"b\", -1 === 1)",
			[]string{"1:6 warning literals: both sides of == are literals, so it is always true",
				"1:16 warning literals: both sides of !== are literals, so it is always true",
				"1:29 warning literals: both sides of === are literals, so it is always false"}},
		{"self-assignment", "let x = 1\nx = x\nlet h = {}\nh.a = h.a",
			[]string{"2:1 warning self-assign: x is assigned to itself", "4:1 warning self-assign: (h[\"a\"]) is assigned to itself"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkAll("", tt.input, host)
			if strings.Join(got, "\n") != strings.Join(tt.expected, "\n") {
				t.Errorf("check(%q)\n--- expected:\n%s\n--- got:\n%s", tt.input,
					strings.Join(tt.expected, "\n"), strings.Join(got, "\n"))
			}
		})
	}
}

// The syntax errors are the parser's, every one, each spanning its token.
// Their messages are the parser's own, so they are compared with it.
func TestSyntaxErrors(t *testing.T) {
	source := "let = 1\nlet y = 2 3\nlet s = \"open"
	p := parser.New(lexer.New(source))
	p.ParseProgram()
	got := Source("", source)
	if len(got) != len(p.Errors()) || len(got) < 2 {
		t.Fatalf("expected the parser's %d errors, got %v", len(p.Errors()), got)
	}
	for i, message := range p.Errors() {
		d := got[i]
		position := fmt.Sprintf("[line %d, column %d] %s error: ", d.Line, d.Column, d.Rule)
		if d.Severity != Error || position+d.Message != message {
			t.Errorf("diagnostic %d is %s, the parser said %q", i, show(d), message)
		}
	}
	if s := got[0]; s.Rule != RuleSyntax || s.EndLine != 1 || s.EndColumn != 6 {
		t.Errorf("wrong first error: %+v", s)
	}
	if last := got[len(got)-1]; last.Rule != RuleLexical {
		t.Errorf("expected a lexical error last, got %+v", last)
	}

	// With syntax errors parts of the tree are missing, so the rules about
	// names are left out.
	for _, d := range Source("", "puts(nope)\nlet = 1") {
		if d.Rule != RuleSyntax {
			t.Errorf("unexpected %s with a syntax error", show(d))
		}
	}
}

func TestModules(t *testing.T) {
	dir := t.TempDir()
	write := func(name, code string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(code), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("shapes.liz", "let sides = 4\nfunction square(n) { n * n }\nlet _made = 0\n")
	write("broken.liz", "let = 1\n")
	main := filepath.Join(dir, "main.liz")

	tests := []struct {
		name, input string
		expected    []string
	}{
		{"names of a module", "use \"shapes\"\nputs(square(sides))", nil},
		{"a module's private name", "use \"shapes\"\nputs(square(1), _made)",
			[]string{"2:17 error unknown-name: identifier not found: _made (_made is private to the module \"shapes\": a name that starts with _ stays inside it)"}},
		{"a module's function arguments", "use \"shapes\"\nputs(square(1, 2))",
			[]string{"2:6 error arguments: wrong number of arguments to square: want 1, got 2"}},
		{"assign a module's name", "use \"shapes\"\nsides = 5\nputs(sides)", nil},
		{"unused module", "use \"shapes\"\nputs(1)",
			[]string{"1:1 warning unused-module: none of the names of \"shapes\" are used"}},
		{"as", "use \"shapes\" as s\nputs(s.square(s.sides))", nil},
		{"unused as", "use \"shapes\" as s",
			[]string{"1:17 warning unused-module: s is never used"}},
		{"a member of as", "use \"shapes\" as s\nputs(s.squares(1))",
			[]string{"2:6 error member: s has no squares (did you mean `s.square`?)"}},
		{"a private member of as", "use \"shapes\" as s\nputs(s._made)",
			[]string{"2:6 warning member: s._made is private to its module"}},
		{"arguments through as", "use \"shapes\" as s\nputs(s.square())",
			[]string{"2:6 error arguments: wrong number of arguments to s.square: want 1, got 0"}},
		{"a missing module", "use \"nowhere\"\nputs(anything)",
			[]string{"1:1 error module: can't find the module \"nowhere\": there is no " + filepath.ToSlash(filepath.Join(dir, "nowhere.liz"))}},
		{"a module with syntax errors", "use \"broken\"\nputs(anything)",
			[]string{"1:1 error module: the module \"broken\" has syntax errors: [line 1, column 5] syntax error: ..."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkAll(main, tt.input, nil)
			if len(got) == 1 && len(tt.expected) == 1 && strings.HasSuffix(tt.expected[0], "...") &&
				strings.HasPrefix(got[0], strings.TrimSuffix(tt.expected[0], "...")) {
				return // the parser's own message follows
			}
			if strings.Join(got, "\n") != strings.Join(tt.expected, "\n") {
				t.Errorf("check(%q)\n--- expected:\n%s\n--- got:\n%s", tt.input,
					strings.Join(tt.expected, "\n"), strings.Join(got, "\n"))
			}
		})
	}
}

func TestPositions(t *testing.T) {
	d := Source("", "let x = 1\nputs(yy_z)")
	if len(d) != 1 {
		t.Fatalf("expected one diagnostic, got %v", d)
	}
	if d[0].Line != 2 || d[0].Column != 6 || d[0].EndLine != 2 || d[0].EndColumn != 10 {
		t.Errorf("wrong span: %+v", d[0])
	}
	syntax := Source("", "let x = (")
	if s := syntax[0]; s.Line != 1 || s.Column != 10 || s.EndColumn != 11 {
		t.Errorf("wrong span of a syntax error: %+v", s)
	}
}

func TestDistance(t *testing.T) {
	tests := []struct {
		a, b string
		d    int
	}{
		{"len", "len", 0}, {"lenn", "len", 1}, {"sine", "sin", 1}, {"lenght", "length", 1},
		{"cuont", "count", 1}, {"abc", "xyz", 3}, {"", "ab", 2},
	}
	for _, tt := range tests {
		if got := distance(tt.a, tt.b); got != tt.d {
			t.Errorf("distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.d)
		}
	}
}
