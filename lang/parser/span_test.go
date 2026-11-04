package parser

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spanText is the source text a span covers. Lines and columns count
// characters, like the lexer.
func spanText(source string, span *ast.Span) string {
	lines := strings.Split(source, "\n")
	var out []string
	for line := span.StartLine; line <= span.EndLine && line <= len(lines); line++ {
		runes := []rune(lines[line-1])
		from, to := 0, len(runes)
		if line == span.StartLine {
			from = min(span.StartColumn-1, len(runes))
		}
		if line == span.EndLine {
			to = min(span.EndColumn-1, len(runes))
		}
		out = append(out, string(runes[from:max(from, to)]))
	}
	return strings.Join(out, "\n")
}

// describeSpans lists each node of program as "Type: text".
func describeSpans(source string, program *ast.Program) []string {
	var out []string
	for _, statement := range program.Statements {
		ast.Inspect(statement, func(n ast.Node) bool {
			name := strings.TrimPrefix(fmt.Sprintf("%T", n), "*ast.")
			out = append(out, name+": "+spanText(source, n.Range()))
			return true
		})
	}
	return out
}

// TestSpans checks the text that each node of a program covers.
func TestSpans(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"let x = 1 + 2 * y;", []string{
			"LetStatement: let x = 1 + 2 * y;", "Identifier: x", "InfixExpression: 1 + 2 * y",
			"IntegerLiteral: 1", "InfixExpression: 2 * y", "IntegerLiteral: 2", "Identifier: y"}},
		{"(1 + 2) * 3", []string{
			"ExpressionStatement: (1 + 2) * 3", "InfixExpression: (1 + 2) * 3",
			"InfixExpression: (1 + 2)", "IntegerLiteral: 1", "IntegerLiteral: 2", "IntegerLiteral: 3"}},
		{`f(a.b, [1], {"k": 2})`, []string{
			`ExpressionStatement: f(a.b, [1], {"k": 2})`, `CallExpression: f(a.b, [1], {"k": 2})`,
			"Identifier: f", "IndexExpression: a.b", "Identifier: a", "StringLiteral: b",
			"ArrayLiteral: [1]", "IntegerLiteral: 1",
			`HashLiteral: {"k": 2}`, `StringLiteral: "k"`, "IntegerLiteral: 2"}},
		{"function add(a, b = 2) {\n  return a + b\n}", []string{
			"LetStatement: function add(a, b = 2) {\n  return a + b\n}", "Identifier: add",
			"FunctionLiteral: function add(a, b = 2) {\n  return a + b\n}",
			"Identifier: a", "Identifier: b", "IntegerLiteral: 2",
			"BlockStatement: {\n  return a + b\n}", "ReturnStatement: return a + b",
			"InfixExpression: a + b", "Identifier: a", "Identifier: b"}},
		{"if a { 1 } elif b { 2 } else { 3 }", []string{
			"ExpressionStatement: if a { 1 } elif b { 2 } else { 3 }",
			"IfExpression: if a { 1 } elif b { 2 } else { 3 }", "Identifier: a",
			"BlockStatement: { 1 }", "ExpressionStatement: 1", "IntegerLiteral: 1",
			"BlockStatement: elif b { 2 } else { 3 }", "ExpressionStatement: elif b { 2 } else { 3 }",
			"IfExpression: elif b { 2 } else { 3 }", "Identifier: b",
			"BlockStatement: { 2 }", "ExpressionStatement: 2", "IntegerLiteral: 2",
			"BlockStatement: { 3 }", "ExpressionStatement: 3", "IntegerLiteral: 3"}},
		{"for (let i = 0; i < 3; i += 1) { puts(i) }", []string{
			"ExpressionStatement: for (let i = 0; i < 3; i += 1) { puts(i) }",
			"ForExpression: for (let i = 0; i < 3; i += 1) { puts(i) }",
			"LetStatement: let i = 0;", "Identifier: i", "IntegerLiteral: 0",
			"InfixExpression: i < 3", "Identifier: i", "IntegerLiteral: 3",
			"AssignExpression: i += 1", "Identifier: i", "IntegerLiteral: 1",
			"BlockStatement: { puts(i) }", "ExpressionStatement: puts(i)", "CallExpression: puts(i)",
			"Identifier: puts", "Identifier: i"}},
		{"layer sun\n  circle((1, 2))\n  x", []string{
			"LayerStatement: layer sun\n  circle((1, 2))\n  x", "BlockStatement: circle((1, 2))\n  x",
			"ExpressionStatement: circle((1, 2))", "CallExpression: circle((1, 2))", "Identifier: circle",
			"PointLiteral: (1, 2)", "IntegerLiteral: 1", "IntegerLiteral: 2",
			"ExpressionStatement: x", "Identifier: x"}},
		{"try { f() } catch (e) { -1 }", []string{
			"TryStatement: try { f() } catch (e) { -1 }", "BlockStatement: { f() }",
			"ExpressionStatement: f()", "CallExpression: f()", "Identifier: f", "Identifier: e",
			"BlockStatement: { -1 }", "ExpressionStatement: -1", "PrefixExpression: -1", "IntegerLiteral: 1"}},
		{`let s = "é\nü" + #fff`, []string{
			"LetStatement: let s = \"é\\nü\" + #fff", "Identifier: s", "InfixExpression: \"é\\nü\" + #fff",
			"StringLiteral: \"é\\nü\"", "ColorLiteral: #fff"}},
		{"let t = \"two\nlines\"", []string{
			"LetStatement: let t = \"two\nlines\"", "Identifier: t", "StringLiteral: \"two\nlines\""}},
		{`use "draw/shapes" as shapes`, []string{
			`UseStatement: use "draw/shapes" as shapes`, `StringLiteral: "draw/shapes"`, "Identifier: shapes"}},
		{"while x { break }", []string{
			"ExpressionStatement: while x { break }", "WhileExpression: while x { break }", "Identifier: x",
			"BlockStatement: { break }", "BreakStatement: break"}},
	}

	for _, tt := range tests {
		program := parse(t, tt.input)
		got := describeSpans(tt.input, program)
		if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
			t.Errorf("input %q\n--- got:\n%s\n--- want:\n%s", tt.input, strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
		}
	}
}

// TestSpansOfEveryProgram parses the programs of the repository and checks
// that every node has a span inside its parent's, that ends after it starts.
func TestSpansOfEveryProgram(t *testing.T) {
	var files []string
	for _, pattern := range []string{"../../examples/*.liz", "../../examples/terrarium/*.liz", "../../tests/valid/*.liz", "../../terrarium/lib/*.liz"} {
		matches, _ := filepath.Glob(pattern)
		files = append(files, matches...)
	}
	if len(files) < 10 {
		t.Fatalf("found only %d programs", len(files))
	}
	before := func(l1, c1, l2, c2 int) bool { return l1 < l2 || l1 == l2 && c1 <= c2 }
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		p := New(lexer.New(string(source)))
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("%s: %v", file, p.Errors())
		}
		var check func(n ast.Node, parent *ast.Span)
		check = func(n ast.Node, parent *ast.Span) {
			s := n.Range()
			if s.StartLine == 0 || !before(s.StartLine, s.StartColumn, s.EndLine, s.EndColumn) {
				t.Fatalf("%s: %T %q has the span %+v", file, n, n.String(), *s)
			}
			if parent != nil && (!before(parent.StartLine, parent.StartColumn, s.StartLine, s.StartColumn) ||
				!before(s.EndLine, s.EndColumn, parent.EndLine, parent.EndColumn)) {
				t.Fatalf("%s: %T %q at %+v is outside its parent at %+v", file, n, n.String(), *s, *parent)
			}
			for _, child := range ast.Children(n) {
				check(child, s)
			}
		}
		check(program, nil)
	}
}
