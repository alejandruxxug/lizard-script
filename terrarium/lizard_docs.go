package terrarium

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"regexp"
	"strings"
)

// lizard.docs(code) reads the doc comments of the functions at the top
// level of code: the -~> lines right above each one. It gives a hash from
// each function's name to {"name", "line", "summary", "params", "example"}.
// params has one {"name", "default", "doc"} per parameter: its default as
// written (or null) and the text of its "name: ..." line (or null). The
// summary is the lines before the first blank line, a parameter line or the
// "example:" line, joined with spaces; example is what follows "example:"
// (lines after it go on it), or null. Code with syntax errors gives the
// functions that parsed.
//
//	-~> A circle around center.
//	-~> center: the middle
//	-~> example: circle((10, 10), 5)
//	function circle(center, radius = 1) { ... }
func lizardDocs(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	code, err := arg.String(args, 0, "code")
	if err != nil {
		return err
	}
	src := newSource(code)
	program := parser.New(lexer.New(code)).ParseProgram()
	out := object.NewHash()
	for _, statement := range program.Statements {
		let, ok := statement.(*ast.LetStatement)
		if !ok {
			continue
		}
		function, ok := let.Value.(*ast.FunctionLiteral)
		if !ok {
			continue
		}
		arg.Set(out, let.Name.Value, functionDoc(src, let, function))
	}
	return out
}

var (
	paramLine   = regexp.MustCompile(`^(\w+):\s*(.*)$`)
	exampleLine = regexp.MustCompile(`^example:\s*(.*)$`)
)

// functionDoc is the entry of lizard.docs for one function.
func functionDoc(src source, let *ast.LetStatement, function *ast.FunctionLiteral) *object.Hash {
	start := let.Range().StartLine
	var lines []string
	for l := start - 1; l >= 1 && l <= len(src); l-- {
		text := strings.TrimSpace(string(src[l-1]))
		if !strings.HasPrefix(text, "-~>") {
			break
		}
		lines = append([]string{strings.TrimSpace(text[3:])}, lines...)
	}

	known := map[string]bool{}
	for _, p := range function.Parameters {
		known[p.Value] = true
	}
	paramDocs := map[string]string{}
	var summary []string
	example, hasExample := "", false
	summaryDone, last := false, "" // last is the parameter or "example" a line goes on
	for _, line := range lines {
		if m := exampleLine.FindStringSubmatch(line); m != nil {
			example, hasExample, last, summaryDone = m[1], true, "example", true
			continue
		}
		if m := paramLine.FindStringSubmatch(line); m != nil && known[m[1]] {
			paramDocs[m[1]] = m[2]
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
			example += "\n" + line
		case last != "":
			paramDocs[last] += " " + line
		}
	}

	params := &object.Array{}
	for i, p := range function.Parameters {
		h := object.NewHash()
		arg.Set(h, "name", arg.Str(p.Value))
		def := object.Object(arg.Null)
		if i < len(function.Defaults) && function.Defaults[i] != nil {
			s := function.Defaults[i].Range()
			def = arg.Str(src.text(s.StartLine, s.StartColumn, s.EndLine, s.EndColumn))
		}
		arg.Set(h, "default", def)
		doc := object.Object(arg.Null)
		if text, ok := paramDocs[p.Value]; ok {
			doc = arg.Str(text)
		}
		arg.Set(h, "doc", doc)
		params.Elements = append(params.Elements, h)
	}

	h := object.NewHash()
	arg.Set(h, "name", arg.Str(let.Name.Value))
	arg.Set(h, "line", arg.Int64(int64(start)))
	arg.Set(h, "summary", arg.Str(strings.Join(summary, " ")))
	arg.Set(h, "params", params)
	if hasExample {
		arg.Set(h, "example", arg.Str(example))
	} else {
		arg.Set(h, "example", arg.Null)
	}
	return h
}
