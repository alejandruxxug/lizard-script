// Package check finds mistakes in lizard-script code without running it:
// the syntax errors, names that are never defined, calls with the wrong
// number of arguments, and code that is probably not what was meant, like a
// variable that is never used or a comparison of two literals. Each finding
// is a Diagnostic with its position, for the command line (lizardscript
// check), an editor (lizard.check, a language server) or a test.
//
// It follows the language's scoping exactly where it can be sure, and gives
// the benefit of the doubt everywhere else: a name read inside a function
// may be defined later in a scope around it (see compiler/scopes.go), and a
// module whose file can't be read may define anything. So an error here is
// an error when the code runs; a program with none may still fail.
package check

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"os"
	"regexp"
	"sort"
	"strconv"
)

// Severity is how bad a finding is: an error fails when the code runs (or
// stops it from running at all), a warning is probably a mistake.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

// Diagnostic is one finding, from (Line, Column) to just before (EndLine,
// EndColumn), counted like token positions. Rule names the check that made
// it (see the rules below), and Hint, when there is one, says how to fix it,
// like "did you mean `len`?".
type Diagnostic struct {
	Severity  Severity `json:"severity"`
	Line      int      `json:"line"`
	Column    int      `json:"column"`
	EndLine   int      `json:"end_line"`
	EndColumn int      `json:"end_column"`
	Message   string   `json:"message"`
	Hint      string   `json:"hint,omitempty"`
	Rule      string   `json:"rule"`
}

// The rules, by the name in Diagnostic.Rule.
const (
	RuleSyntax       = "syntax"        // error: a syntax error
	RuleLexical      = "lexical"       // error: a lexical error, like a string that never ends
	RuleUnknownName  = "unknown-name"  // error: a name that is defined nowhere it could come from
	RuleAssign       = "assign"        // error: = to a name that was never declared
	RuleArguments    = "arguments"     // error: a call with the wrong number of arguments
	RuleModule       = "module"        // error: a use whose module can't be found or doesn't parse
	RuleReturn       = "return"        // error: a return whose value is on the next line
	RuleMember       = "member"        // error when called, warning when read: math.sine
	RuleUnused       = "unused"        // warning: a let or a parameter that is never read
	RuleShadow       = "shadow"        // warning: a let that hides a name of a scope around it
	RuleUnreachable  = "unreachable"   // warning: code after return, break or continue
	RuleLiterals     = "literals"      // warning: == (and !=, ===, !==) between two literals
	RuleUnusedModule = "unused-module" // warning: a use none of whose names are used
	RuleSelfAssign   = "self-assign"   // warning: x = x
)

// Globals are the names a program has without declaring them, besides the
// builtins and the libraries (which every program has): what the host
// gives, each with the names of its members when it is a module (a hash).
// The lizardscript command sets it to the Terrarium (see GlobalsOf).
var Globals = map[string][]string{}

// GlobalsOf lists the names env defines, with the keys of the ones that
// are hashes: a host fills an environment the way it fills a program's and
// hands it here, as in check.Globals = check.GlobalsOf(env).
func GlobalsOf(env *object.Environment) map[string][]string {
	names := map[string][]string{}
	for _, name := range env.Names() {
		value, _ := env.Get(name)
		names[name] = members(value)
	}
	return names
}

// members are the string keys of value when it is a hash, else nil.
func members(value object.Object) []string {
	h, ok := value.(*object.Hash)
	if !ok {
		return nil
	}
	keys := []string{}
	h.Each(func(pair object.HashPair) {
		if key, ok := pair.Key.(*object.String); ok {
			keys = append(keys, key.Value)
		}
	})
	return keys
}

// File checks the file at path, with Globals and the names of its project
// file (see ProjectNames). Its use statements look for modules the way a
// run does, next to it and in evaluator.ModulePath.
func File(path string) ([]Diagnostic, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	names, _, err := ProjectNames(path)
	if err != nil {
		return nil, err
	}
	return SourceWith(path, string(source), WithNames(Globals, names)), nil
}

// Source checks text, the code of the source called name (a file name, so
// its uses find their modules; it may be ""), with Globals.
func Source(name, text string) []Diagnostic {
	return SourceWith(name, text, Globals)
}

// SourceWith is Source with other globals.
func SourceWith(name, text string, globals map[string][]string) []Diagnostic {
	p := parser.New(lexer.NewNamed(text, name))
	program := p.ParseProgram()
	c := newChecker(globals)
	for _, message := range p.Errors() {
		c.diagnostics = append(c.diagnostics, syntaxDiagnostic(text, message))
	}
	c.broken = len(p.Errors()) > 0
	c.check(program)

	sort.SliceStable(c.diagnostics, func(i, j int) bool {
		a, b := c.diagnostics[i], c.diagnostics[j]
		return a.Line < b.Line || a.Line == b.Line && a.Column < b.Column
	})
	return c.diagnostics
}

var errorPosition = regexp.MustCompile(`^\[line (\d+), column (\d+)\] (syntax|lexical) error: (.*)$`)

// syntaxDiagnostic turns a message of the parser into a diagnostic that
// spans the token at its position. Its rule is "syntax" or "lexical", and
// the message is what follows.
func syntaxDiagnostic(text, message string) Diagnostic {
	d := Diagnostic{Severity: Error, Message: message, Rule: RuleSyntax}
	match := errorPosition.FindStringSubmatch(message)
	if match == nil {
		return d
	}
	d.Line, _ = strconv.Atoi(match[1])
	d.Column, _ = strconv.Atoi(match[2])
	d.Rule, d.Message = match[3], match[4]
	d.EndLine, d.EndColumn = tokenEnd(text, d.Line, d.Column)
	return d
}

// tokenEnd is where the token that starts at (line, column) ends, or the
// next column when none starts there.
func tokenEnd(text string, line, column int) (int, int) {
	l := lexer.New(text)
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		if tok.Line == line && tok.Column == column {
			return tok.EndLine, tok.EndColumn
		}
		if tok.Line > line {
			break
		}
	}
	return line, column + 1
}

// add records a finding about node.
func (c *checker) add(severity Severity, rule string, node ast.Node, hint, format string, args ...any) {
	span := node.Range()
	c.diagnostics = append(c.diagnostics, Diagnostic{
		Severity: severity, Rule: rule, Hint: hint, Message: fmt.Sprintf(format, args...),
		Line: span.StartLine, Column: span.StartColumn, EndLine: span.EndLine, EndColumn: span.EndColumn,
	})
}

// builtinNames are the names every program has, with the members of the
// libraries.
func builtinNames() map[string][]string {
	names := map[string][]string{}
	for _, name := range evaluator.BuiltinNames() {
		value, _ := evaluator.Builtin(name)
		names[name] = members(value)
	}
	return names
}
