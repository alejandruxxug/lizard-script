package check

import (
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"regexp"
	"strings"
)

// module is a file a use statement loads, read the way use reads it
// (evaluator.FindModule) but never run: what the checker needs is the names
// it defines at its top level.
type module struct {
	path    string // as the use writes it
	use     *ast.UseStatement
	exports map[string]*export // nil when it couldn't be read
	private map[string]bool    // the names it keeps, that start with _
	used    bool               // one of its names is used (without as)
}

// export is a name a module defines.
type export struct {
	function *ast.FunctionLiteral // its value, when surely a function
	assigned bool                 // changed with = by the file that uses it
}

// use declares what a use statement brings, in the first walk: the name
// after as, or the module's names. A module that can't be read is an
// error, and then any name may be one of its own.
func (c *checker) use(st *ast.UseStatement, top *scope) {
	m := c.load(st)
	if st.Name != nil {
		d := c.declare(top, useDecl, st.Name, endOf(st), nil)
		d.module = m
		return
	}
	c.modules = append(c.modules, m)
	if m.exports == nil {
		c.opaque = true
	}
}

// load reads the module of st, once per check.
func (c *checker) load(st *ast.UseStatement) *module {
	if m, ok := c.loaded[st.Path.Value]; ok {
		return m
	}
	m := &module{path: st.Path.Value, use: st}
	c.loaded[st.Path.Value] = m

	file, source, err := evaluator.FindModule(st)
	if err != nil {
		c.add(Error, RuleModule, st, "", "%s", withoutPosition(err))
		return m
	}
	p := parser.New(lexer.NewNamed(string(source), file))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		c.add(Error, RuleModule, st, "", "the module %q has syntax errors: %s", st.Path.Value, p.Errors()[0])
		return m
	}
	m.exports, m.private = exportsOf(program)
	return m
}

// exportsOf are the names a module's file defines at its top level, the
// ones another file sees, and the private ones, that start with _. A
// function is only surely one when the file declares it once and never
// assigns it.
func exportsOf(program *ast.Program) (map[string]*export, map[string]bool) {
	exports, private := map[string]*export{}, map[string]bool{}
	declared := map[string]int{}
	for _, st := range program.Statements {
		var name string
		var value ast.Expression
		switch st := st.(type) {
		case *ast.LetStatement:
			name, value = st.Name.Value, st.Value
		case *ast.UseStatement:
			if st.Name == nil {
				continue
			}
			name = st.Name.Value
		default:
			continue
		}
		if !object.Exported(name) {
			private[name] = true
			continue
		}
		declared[name]++
		e := &export{}
		e.function, _ = value.(*ast.FunctionLiteral)
		exports[name] = e
	}
	ast.Inspect(program, func(node ast.Node) bool {
		if a, ok := node.(*ast.AssignExpression); ok {
			if name, ok := a.Left.(*ast.Identifier); ok && exports[name.Value] != nil {
				exports[name.Value].function = nil
			}
		}
		return true
	})
	for name, count := range declared {
		if count > 1 {
			exports[name].function = nil
		}
	}
	return exports, private
}

var runtimePosition = regexp.MustCompile(`^\[line \d+, column \d+\] runtime error: `)

// withoutPosition is the message of err without its position, which a
// diagnostic has apart.
func withoutPosition(err *object.Error) string {
	return runtimePosition.ReplaceAllString(strings.TrimSpace(err.Message), "")
}
