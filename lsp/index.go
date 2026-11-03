package lsp

import (
	"lizardscript/ast"
	"lizardscript/lexer"
	"lizardscript/parser"
	"lizardscript/token"
	"sort"
	"strings"
)

// An index is what the server knows about one source: its tree, its
// scopes, the names it declares and what each name in it means. Names are
// found the evaluator's way, like package check does (see check/scopes.go):
// a scope for the program, each block, each for loop, each function and
// each catch; a name is looked up from the innermost scope out, then in the
// modules the file uses, then among the globals and the builtins.
//
// Lizard binds late, so a let counts from where it is for the code of its
// own function, but for a function made in its scope it counts wherever it
// is: the function may run after it. The index walks the tree twice, first
// to declare every name, then to resolve every name with all of them known.

// pos is a place in the source, counted like token positions.
type pos struct{ line, column int }

func (a pos) before(b pos) bool {
	return a.line < b.line || a.line == b.line && a.column < b.column
}

func startOf(node ast.Node) pos { return pos{node.Range().StartLine, node.Range().StartColumn} }
func endOf(node ast.Node) pos   { return pos{node.Range().EndLine, node.Range().EndColumn} }

// inside reports whether p is in span, its end included (the cursor right
// after a name is still on it).
func inside(p pos, span *ast.Span) bool {
	return !p.before(pos{span.StartLine, span.StartColumn}) && !(pos{span.EndLine, span.EndColumn}).before(p)
}

type declKind int

const (
	letDecl   declKind = iota // let x = ..., function x() {}
	paramDecl                 // a function's parameter
	catchDecl                 // the name in catch (e)
	useDecl                   // the name in use "x" as m
)

// symbol is a name declared in a scope.
type symbol struct {
	name   string
	kind   declKind
	ident  *ast.Identifier      // where it is declared
	let    *ast.LetStatement    // for a let
	at     pos                  // from where it is defined
	scope  *scope               // where it is declared
	use    *ast.UseStatement    // for a useDecl
	module *moduleFile          // for a useDecl, when its file was found
	fn     *ast.FunctionLiteral // the function it declares a parameter of
}

// value is the expression a let gives, or nil.
func (s *symbol) value() ast.Expression {
	if s.let == nil {
		return nil
	}
	return s.let.Value
}

// scope is one scope of the evaluator.
type scope struct {
	parent   *scope
	function int       // which function its code belongs to, 0 for the program
	span     *ast.Span // where its code is
	decls    map[string][]*symbol
	children []*scope
}

// target is what a name means: a name declared in this file (sym), a name
// of a module the file uses (module), or a global or a builtin (global).
// path goes on from there with the names after dots: config.size is
// {sym: config, path: "size"}, lily2d.circle (lily2d used with as) is
// {module: lily2d, path: "circle"}, and math.sin is {global, "math.sin"}.
type target struct {
	sym    *symbol
	module *moduleFile
	global bool
	path   string
}

func (t target) known() bool { return t.sym != nil || t.module != nil || t.global }

// member is the target of t.name.
func (t target) member(name string) target {
	if t.sym != nil && t.sym.kind == useDecl && t.path == "" {
		if t.sym.module == nil {
			return target{}
		}
		return target{module: t.sym.module, path: name}
	}
	if !t.known() {
		return target{}
	}
	t.path = joinPath(t.path, name)
	return t
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// occurrence is a name in the source: an identifier, or the name after a
// dot. decl is true where the name is declared.
type occurrence struct {
	span   *ast.Span
	name   string // as written
	target target
	decl   bool
	dot    bool // the name after a dot
	call   bool // the name is called: f(...) or m.f(...)
}

// index is the analysis of one source.
type index struct {
	name        string // the source name, a file
	text        string
	lines       []string
	program     *ast.Program
	errors      []string
	top         *scope
	scopes      map[ast.Node]*scope
	symbols     []*symbol     // in the order they are declared
	occurrences []*occurrence // in the order of the source
	modules     []*moduleFile // the ones used without as, in order
	uses        map[*ast.UseStatement]*moduleFile
	globals     map[string][]string // the host's and the builtins, with their members

	functions int
	declaring bool
	loader    *loader
	called    map[ast.Node]bool // the names that are the function of a call
}

// newIndex analyses text, the source called name. loader finds the files of
// its use statements; with a nil loader they are not read.
func newIndex(name, text string, globals map[string][]string, loader *loader) *index {
	p := parser.New(lexer.NewNamed(text, name))
	program := p.ParseProgram()
	ix := &index{
		name: name, text: text, lines: strings.Split(text, "\n"),
		program: program, errors: p.Errors(), globals: globals, loader: loader,
		scopes: map[ast.Node]*scope{}, uses: map[*ast.UseStatement]*moduleFile{},
		called: map[ast.Node]bool{},
	}
	ix.top = &scope{span: &ast.Span{StartLine: 1, StartColumn: 1, EndLine: len(ix.lines) + 1, EndColumn: 1}, decls: map[string][]*symbol{}}
	ix.declaring = true
	ix.statements(program.Statements, ix.top)
	ix.declaring = false
	ix.statements(program.Statements, ix.top)
	sort.SliceStable(ix.occurrences, func(i, j int) bool {
		a, b := ix.occurrences[i].span, ix.occurrences[j].span
		return (pos{a.StartLine, a.StartColumn}).before(pos{b.StartLine, b.StartColumn})
	})
	return ix
}

// scopeOf is the scope that node opens inside parent: new in the first
// walk, the same one in the second.
func (ix *index) scopeOf(node ast.Node, parent *scope) *scope {
	if s, ok := ix.scopes[node]; ok {
		return s
	}
	s := &scope{parent: parent, function: parent.function, span: node.Range(), decls: map[string][]*symbol{}}
	if _, ok := node.(*ast.FunctionLiteral); ok {
		ix.functions++
		s.function = ix.functions
	}
	parent.children = append(parent.children, s)
	ix.scopes[node] = s
	return s
}

// declare adds a name to s, in the first walk, and notes where it is
// declared, in the second.
func (ix *index) declare(s *scope, kind declKind, name *ast.Identifier, at pos) *symbol {
	if name == nil {
		return nil
	}
	if !ix.declaring {
		for _, sym := range s.decls[name.Value] {
			if sym.ident == name {
				ix.note(name, target{sym: sym}, true)
				return sym
			}
		}
		return nil
	}
	sym := &symbol{name: name.Value, kind: kind, ident: name, at: at, scope: s}
	s.decls[name.Value] = append(s.decls[name.Value], sym)
	ix.symbols = append(ix.symbols, sym)
	return sym
}

// note records a name of the source, in the second walk.
func (ix *index) note(name *ast.Identifier, t target, decl bool) {
	ix.occurrences = append(ix.occurrences, &occurrence{span: name.Range(), name: name.Value, target: t, decl: decl, call: ix.called[name]})
}

func (ix *index) statements(list []ast.Statement, s *scope) {
	for _, st := range list {
		ix.statement(st, s)
	}
}

func (ix *index) statement(st ast.Statement, s *scope) {
	switch st := st.(type) {
	case *ast.LetStatement:
		ix.expression(st.Value, s)
		if sym := ix.declare(s, letDecl, st.Name, endOf(st)); sym != nil {
			sym.let = st
		}
	case *ast.ReturnStatement:
		ix.expression(st.ReturnValue, s)
	case *ast.ExpressionStatement:
		ix.expression(st.Expression, s)
	case *ast.TryStatement:
		ix.block(st.Body, s, nil)
		ix.block(st.Catch, s, func(inner *scope) {
			ix.declare(inner, catchDecl, st.Name, startOf(st.Catch))
		})
	case *ast.LayerStatement:
		ix.block(st.Body, s, nil)
	case *ast.UseStatement:
		ix.useStatement(st, s)
	}
}

// useStatement declares the name after as, or adds the module's names to
// the ones the file has.
func (ix *index) useStatement(st *ast.UseStatement, s *scope) {
	if ix.declaring {
		var m *moduleFile
		if ix.loader != nil {
			m = ix.loader.load(st)
		}
		ix.uses[st] = m
		if st.Name == nil {
			if m != nil {
				ix.modules = append(ix.modules, m)
			}
			return
		}
	}
	if !ix.declaring && ix.uses[st] != nil {
		// The path, to go to the module's file from it.
		ix.occurrences = append(ix.occurrences, &occurrence{span: st.Path.Range(), target: target{module: ix.uses[st]}})
	}
	if sym := ix.declare(s, useDecl, st.Name, endOf(st)); sym != nil {
		sym.use, sym.module = st, ix.uses[st]
	}
}

// block walks the statements of b in a scope of its own, after declare
// (if any) adds the names it starts with.
func (ix *index) block(b *ast.BlockStatement, s *scope, declare func(inner *scope)) {
	if b == nil {
		return
	}
	inner := ix.scopeOf(b, s)
	if declare != nil {
		declare(inner)
	}
	ix.statements(b.Statements, inner)
}

func (ix *index) expression(e ast.Expression, s *scope) {
	switch e := e.(type) {
	case nil:
	case *ast.Identifier:
		ix.read(e, s)
	case *ast.PointLiteral:
		ix.expressions(e.Coords, s)
	case *ast.PrefixExpression:
		ix.expression(e.Right, s)
	case *ast.InfixExpression:
		ix.expression(e.Left, s)
		ix.expression(e.Right, s)
	case *ast.AssignExpression:
		ix.expression(e.Left, s)
		ix.expression(e.Value, s)
	case *ast.IfExpression:
		ix.expression(e.Condition, s)
		ix.block(e.Consequence, s, nil)
		ix.block(e.Alternative, s, nil)
	case *ast.WhileExpression:
		ix.expression(e.Condition, s)
		ix.block(e.Body, s, nil)
	case *ast.ForExpression:
		header := ix.scopeOf(e, s)
		if e.Init != nil {
			ix.statement(e.Init, header)
		}
		ix.expression(e.Condition, header)
		ix.expression(e.Update, header)
		ix.block(e.Body, header, nil)
	case *ast.FunctionLiteral:
		ix.function(e, s)
	case *ast.CallExpression:
		ix.markCalled(e.Function)
		ix.expression(e.Function, s)
		ix.expressions(e.Arguments, s)
	case *ast.ArrayLiteral:
		ix.expressions(e.Elements, s)
	case *ast.IndexExpression:
		ix.expression(e.Left, s)
		if name, ok := dotName(e); ok {
			ix.dot(e, name, s)
		} else {
			ix.expression(e.Index, s)
		}
	case *ast.HashLiteral:
		for _, pair := range e.Pairs {
			ix.expression(pair.Key, s)
			ix.expression(pair.Value, s)
		}
	}
}

func (ix *index) expressions(list []ast.Expression, s *scope) {
	for _, e := range list {
		ix.expression(e, s)
	}
}

// markCalled notes that the name of fn, f or the name after m., is called.
func (ix *index) markCalled(fn ast.Expression) {
	switch fn := fn.(type) {
	case *ast.Identifier:
		ix.called[fn] = true
	case *ast.IndexExpression:
		if name, ok := dotName(fn); ok {
			ix.called[name] = true
		}
	}
}

// function walks a function literal in its own scope. A parameter is
// defined from where it is written, so a default sees the ones before it.
func (ix *index) function(f *ast.FunctionLiteral, s *scope) {
	inner := ix.scopeOf(f, s)
	for i, p := range f.Parameters {
		if sym := ix.declare(inner, paramDecl, p, startOf(p)); sym != nil {
			sym.fn = f
		}
		if i < len(f.Defaults) {
			ix.expression(f.Defaults[i], inner)
		}
	}
	if f.Body != nil {
		ix.statements(f.Body.Statements, inner)
	}
}

// read resolves a name that is read or assigned, in the second walk.
func (ix *index) read(name *ast.Identifier, s *scope) {
	if ix.declaring {
		return
	}
	ix.note(name, ix.resolve(name.Value, startOf(name), s), false)
}

// dot resolves the name after the dot of e, in the second walk.
func (ix *index) dot(e *ast.IndexExpression, name *ast.StringLiteral, s *scope) {
	if ix.declaring {
		return
	}
	ix.occurrences = append(ix.occurrences, &occurrence{span: name.Range(), name: name.Value,
		target: ix.targetOf(e, s), dot: true, call: ix.called[name]})
}

// targetOf is what e means when it is a name or a chain of names with
// dots, like a.b.c; else nothing.
func (ix *index) targetOf(e ast.Expression, s *scope) target {
	switch e := e.(type) {
	case *ast.Identifier:
		return ix.resolve(e.Value, startOf(e), s)
	case *ast.IndexExpression:
		if name, ok := dotName(e); ok {
			return ix.targetOf(e.Left, s).member(name.Value)
		}
	}
	return target{}
}

// resolve finds what name means when it is used at place, in scope s.
func (ix *index) resolve(name string, place pos, s *scope) target {
	for sc := s; sc != nil; sc = sc.parent {
		decls := sc.decls[name]
		if len(decls) == 0 {
			continue
		}
		// The last declaration that has surely run at place, if any.
		var ran *symbol
		for _, d := range decls {
			if !place.before(d.at) {
				ran = d
			}
		}
		if ran != nil {
			return target{sym: ran}
		}
		if sc.function != s.function {
			// From inside a function, a later let of a scope around it counts.
			return target{sym: decls[0]}
		}
	}
	for i := len(ix.modules) - 1; i >= 0; i-- {
		if ix.modules[i].index.exported(name) != nil {
			return target{module: ix.modules[i], path: name}
		}
	}
	if _, ok := ix.globals[name]; ok {
		return target{global: true, path: name}
	}
	return target{}
}

// exported is the symbol of the top-level name another file sees, or nil.
func (ix *index) exported(name string) *symbol {
	if ix == nil || strings.HasPrefix(name, "_") {
		return nil
	}
	decls := ix.top.decls[name]
	if len(decls) == 0 {
		return nil
	}
	return decls[len(decls)-1]
}

// exports are the top-level names another file sees, sorted.
func (ix *index) exports() []*symbol {
	var out []*symbol
	for name := range ix.top.decls {
		if sym := ix.exported(name); sym != nil {
			out = append(out, sym)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// dotName is the name after the dot when e is a.name, not a[...].
func dotName(e *ast.IndexExpression) (*ast.StringLiteral, bool) {
	key, ok := e.Index.(*ast.StringLiteral)
	if !ok || key.Token.Type != token.IDENT {
		return nil, false
	}
	return key, true
}

// occurrenceAt is the name at p, or nil.
func (ix *index) occurrenceAt(p pos) *occurrence {
	for _, o := range ix.occurrences {
		if inside(p, o.span) {
			return o
		}
	}
	return nil
}

// scopeAt is the innermost scope whose code has p.
func (ix *index) scopeAt(p pos) *scope {
	s := ix.top
	for {
		next := (*scope)(nil)
		for _, child := range s.children {
			if inside(p, child.span) {
				next = child
			}
		}
		if next == nil {
			return s
		}
		s = next
	}
}

// visible are the names declared in this file that code at p can read,
// the innermost first, one symbol for each name.
func (ix *index) visible(p pos) []*symbol {
	at := ix.scopeAt(p)
	seen := map[string]bool{}
	var out []*symbol
	for sc := at; sc != nil; sc = sc.parent {
		names := make([]string, 0, len(sc.decls))
		for name := range sc.decls {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if seen[name] {
				continue
			}
			for _, d := range sc.decls[name] {
				if sc.function != at.function || !p.before(d.at) || d.kind == paramDecl {
					seen[name] = true
					out = append(out, d)
					break
				}
			}
		}
	}
	return out
}

// valueOf is the expression that t is known to hold: the value of a let,
// a value of a hash literal along t's path, or nil.
func valueOf(t target) ast.Expression {
	var value ast.Expression
	path := t.path
	switch {
	case t.sym != nil:
		value = t.sym.value()
	case t.module != nil:
		head, rest, _ := strings.Cut(path, ".")
		sym := t.module.index.exported(head)
		if sym == nil {
			return nil
		}
		value, path = sym.value(), rest
	default:
		return nil
	}
	for _, name := range splitPath(path) {
		pair := pairOf(value, name)
		if pair == nil {
			return nil
		}
		value = pair.Value
	}
	return value
}

// pairOf is the pair of the hash literal value whose key is the string
// name, or nil.
func pairOf(value ast.Expression, name string) *ast.HashPair {
	h, ok := value.(*ast.HashLiteral)
	if !ok {
		return nil
	}
	for i := len(h.Pairs) - 1; i >= 0; i-- {
		if key, ok := h.Pairs[i].Key.(*ast.StringLiteral); ok && key.Value == name {
			return &h.Pairs[i]
		}
	}
	return nil
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

// signature writes a function the way it is declared: name(a, b = 2).
func (ix *index) signature(name string, f *ast.FunctionLiteral) string {
	params := make([]string, len(f.Parameters))
	for i, p := range f.Parameters {
		params[i] = p.Value
		if i < len(f.Defaults) && f.Defaults[i] != nil {
			params[i] += " = " + ix.source(f.Defaults[i].Range())
		}
	}
	return name + "(" + strings.Join(params, ", ") + ")"
}

// source is the text of the source in span.
func (ix *index) source(span *ast.Span) string {
	if span.StartLine < 1 || span.EndLine > len(ix.lines) {
		return ""
	}
	var b strings.Builder
	for line := span.StartLine; line <= span.EndLine; line++ {
		runes := []rune(ix.lines[line-1])
		from, to := 0, len(runes)
		if line == span.StartLine {
			from = min(span.StartColumn-1, len(runes))
		}
		if line == span.EndLine {
			to = min(span.EndColumn-1, len(runes))
		}
		if line > span.StartLine {
			b.WriteByte('\n')
		}
		if from < to {
			b.WriteString(string(runes[from:to]))
		}
	}
	return b.String()
}
