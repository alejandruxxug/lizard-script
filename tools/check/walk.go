package check

import (
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
)

// checker is the state of one check.
type checker struct {
	globals     map[string][]string // the host's and the builtins, with their members
	host        map[string][]string // the host's alone, which can be assigned
	diagnostics []Diagnostic
	broken      bool // the code has syntax errors, so the tree misses parts

	declaring bool                // the first walk, which declares
	scopes    map[ast.Node]*scope // made by the first walk, by the node that opens each
	functions int                 // how many functions the first walk has numbered
	decls     []*decl             // every declaration, in order

	modules []*module          // the ones used without as, in order
	opaque  bool               // a module used without as couldn't be read: any name may be its
	loaded  map[string]*module // by the path in the use

	calls     []call       // the calls of names, checked at the end
	reads     []memberRead // the dot reads of modules, checked at the end
	setFields map[string]bool
}

func newChecker(globals map[string][]string) *checker {
	all := builtinNames()
	for name, members := range globals {
		all[name] = members
	}
	return &checker{globals: all, host: globals, scopes: map[ast.Node]*scope{},
		loaded: map[string]*module{}, setFields: map[string]bool{}}
}

// check walks program twice, then applies the rules that need every name.
func (c *checker) check(program *ast.Program) {
	top := &scope{decls: map[string][]*decl{}}
	c.declaring = true
	c.statements(program.Statements, top)
	c.declaring = false
	c.statements(program.Statements, top)
	c.finish()
}

// scopeOf is the scope that node opens inside parent: new in the first
// walk, the same one in the second. A function's scope starts a new
// function.
func (c *checker) scopeOf(node ast.Node, parent *scope) *scope {
	if s, ok := c.scopes[node]; ok {
		return s
	}
	s := &scope{parent: parent, function: parent.function, decls: map[string][]*decl{}}
	if _, ok := node.(*ast.FunctionLiteral); ok {
		c.functions++
		s.function = c.functions
	}
	c.scopes[node] = s
	return s
}

// declare adds a name to s in the first walk.
func (c *checker) declare(s *scope, kind declKind, name *ast.Identifier, at pos, value ast.Expression) *decl {
	if !c.declaring || name == nil {
		return nil
	}
	d := &decl{name: name.Value, kind: kind, node: name, at: at}
	d.function, _ = value.(*ast.FunctionLiteral)
	s.declare(d)
	c.decls = append(c.decls, d)
	return d
}

func (c *checker) statements(list []ast.Statement, s *scope) {
	for _, st := range list {
		c.statement(st, s)
	}
	if !c.declaring {
		c.unreachable(list)
	}
}

func (c *checker) statement(st ast.Statement, s *scope) {
	switch st := st.(type) {
	case *ast.LetStatement:
		c.expression(st.Value, s)
		c.declare(s, letDecl, st.Name, endOf(st), st.Value)
	case *ast.ReturnStatement:
		c.expression(st.ReturnValue, s)
	case *ast.ExpressionStatement:
		c.expression(st.Expression, s)
	case *ast.TryStatement:
		c.block(st.Body, s, nil)
		c.block(st.Catch, s, func(inner *scope) {
			c.declare(inner, catchDecl, st.Name, startOf(st.Catch), nil)
		})
	case *ast.LayerStatement:
		c.block(st.Body, s, nil)
	case *ast.UseStatement:
		if c.declaring {
			c.use(st, s)
		}
	}
}

// block walks the statements of b in a scope of its own, after declare
// (if any) adds the names it starts with.
func (c *checker) block(b *ast.BlockStatement, s *scope, declare func(inner *scope)) {
	if b == nil {
		return
	}
	inner := c.scopeOf(b, s)
	if declare != nil {
		declare(inner)
	}
	c.statements(b.Statements, inner)
}

func (c *checker) expression(e ast.Expression, s *scope) {
	switch e := e.(type) {
	case nil:
	case *ast.Identifier:
		c.read(e, s)
	case *ast.PointLiteral:
		c.expressions(e.Coords, s)
	case *ast.PrefixExpression:
		c.expression(e.Right, s)
	case *ast.InfixExpression:
		c.expression(e.Left, s)
		c.expression(e.Right, s)
		c.literals(e)
	case *ast.AssignExpression:
		c.assignment(e, s)
	case *ast.IfExpression:
		c.expression(e.Condition, s)
		c.block(e.Consequence, s, nil)
		c.block(e.Alternative, s, nil)
	case *ast.WhileExpression:
		c.expression(e.Condition, s)
		c.block(e.Body, s, nil)
	case *ast.ForExpression:
		header := c.scopeOf(e, s)
		if e.Init != nil {
			c.statement(e.Init, header)
		}
		c.expression(e.Condition, header)
		c.expression(e.Update, header)
		c.block(e.Body, header, nil)
	case *ast.FunctionLiteral:
		c.function(e, s)
	case *ast.CallExpression:
		c.expression(e.Function, s)
		c.expressions(e.Arguments, s)
		c.call(e, s)
	case *ast.ArrayLiteral:
		c.expressions(e.Elements, s)
	case *ast.IndexExpression:
		c.expression(e.Left, s)
		c.expression(e.Index, s)
		c.memberRead(e, s)
	case *ast.HashLiteral:
		for _, pair := range e.Pairs {
			c.expression(pair.Key, s)
			c.expression(pair.Value, s)
		}
	}
}

func (c *checker) expressions(list []ast.Expression, s *scope) {
	for _, e := range list {
		c.expression(e, s)
	}
}

// function walks a function literal in its own scope. A parameter is
// defined from where it is written, so a default sees the ones before it;
// the body runs in the same scope as the parameters, like the evaluator's.
func (c *checker) function(f *ast.FunctionLiteral, s *scope) {
	inner := c.scopeOf(f, s)
	for i, p := range f.Parameters {
		c.declare(inner, paramDecl, p, startOf(p), nil)
		if i < len(f.Defaults) {
			c.expression(f.Defaults[i], inner)
		}
	}
	if f.Body != nil {
		c.statements(f.Body.Statements, inner)
	}
}

// read resolves a name that is read, in the second walk.
func (c *checker) read(name *ast.Identifier, s *scope) {
	if c.declaring {
		return
	}
	f := c.resolve(name.Value, startOf(name), s)
	for _, d := range f.decls {
		d.used = true
	}
	if f.module != nil {
		f.module.used = true
	}
	if f.known() || c.opaque || c.broken {
		return
	}
	if f.later != nil {
		c.add(Error, RuleUnknownName, name, "", "identifier not found: %s: its let, at line %d, comes later",
			name.Value, f.later.node.Token.Line)
		return
	}
	c.add(Error, RuleUnknownName, name, c.suggestName(name.Value, s), "identifier not found: %s", name.Value)
}

// assignment walks x = e, x += e and a[i] = e.
func (c *checker) assignment(e *ast.AssignExpression, s *scope) {
	if name, ok := e.Left.(*ast.Identifier); ok {
		if e.Operator != "" {
			c.read(name, s)
		}
		c.expression(e.Value, s)
		c.assign(name, s)
	} else {
		c.expression(e.Left, s)
		c.expression(e.Value, s)
		c.fieldSet(e.Left)
	}
	c.selfAssignment(e)
}

// assign resolves a name that is assigned, in the second walk. Only a
// declared name can be: a builtin or a library can't.
func (c *checker) assign(name *ast.Identifier, s *scope) {
	if c.declaring {
		return
	}
	f := c.resolve(name.Value, startOf(name), s)
	for _, d := range f.decls {
		d.assigned = true
	}
	if f.module != nil {
		// Writing a module's variable is a use of the module.
		f.module.exports[name.Value].assigned = true
		f.module.used = true
	}
	_, host := c.host[name.Value]
	if len(f.decls) > 0 || f.module != nil || host || c.opaque || c.broken {
		return
	}
	c.add(Error, RuleAssign, name, c.suggestName(name.Value, s),
		"can't assign to %s: it was never declared, use let %s = ...", name.Value, name.Value)
}

// isDot reports whether index is a.name, not a[...]: its key is the name
// after the dot.
func isDot(index *ast.IndexExpression) (string, bool) {
	key, ok := index.Index.(*ast.StringLiteral)
	if !ok || key.Token.Type != token.IDENT {
		return "", false
	}
	return key.Value, true
}
