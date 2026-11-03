package check

import (
	"lizardscript/ast"
)

// How names are found, the evaluator's way: a scope for the program, each
// block, each loop round, each function call, each layer and each catch; a
// name is looked up from the innermost scope out, then in the modules the
// file uses, then among the globals and the builtins.
//
// The checker walks the tree twice. The first walk makes the scopes and
// declares every name in them; the second resolves each name that is read
// or assigned, knowing every declaration, also the ones further down. That
// matters because of when a let has run (see compiler/scopes.go):
//
//   - Read by the code of its own function, a scope's let counts only if it
//     comes before the read: the statements of a scope run in order, and
//     the scope is new each time it starts (each round, each call). A read
//     before every let of the name in that scope means the next scope out.
//   - Read from inside a function made in the scope, any let of the scope
//     counts, wherever it is: the function may run after it.

// pos is a place in the source, to tell what comes first.
type pos struct{ line, column int }

func (a pos) before(b pos) bool {
	return a.line < b.line || a.line == b.line && a.column < b.column
}

func startOf(node ast.Node) pos { return pos{node.Range().StartLine, node.Range().StartColumn} }
func endOf(node ast.Node) pos   { return pos{node.Range().EndLine, node.Range().EndColumn} }

type declKind int

const (
	letDecl   declKind = iota // let x = ..., function x() {}
	paramDecl                 // a function's parameter
	catchDecl                 // the name in catch (e)
	useDecl                   // the name in use "x" as m
)

// decl is a name declared in a scope.
type decl struct {
	name     string
	kind     declKind
	node     *ast.Identifier
	at       pos                  // from where it is defined: after the let
	function *ast.FunctionLiteral // its value, when it is one
	module   *module              // for a useDecl
	scope    *scope
	used     bool // read somewhere
	assigned bool // changed with = somewhere, so its value is not surely function
}

// scope is one scope of the evaluator.
type scope struct {
	parent   *scope
	function int // which function its code belongs to, 0 for the program
	decls    map[string][]*decl
}

func (s *scope) declare(d *decl) {
	d.scope = s
	s.decls[d.name] = append(s.decls[d.name], d)
}

// found is what a name means at one place.
type found struct {
	decls  []*decl // the declarations it may be, innermost first
	module *module // or a module the file uses has it
	global bool    // or it is a global or a builtin
	later  *decl   // the let it was read before, in its own function
}

func (f found) known() bool { return len(f.decls) > 0 || f.module != nil || f.global }

// resolve finds what name means when it is used at place, in scope s.
func (c *checker) resolve(name string, place pos, s *scope) found {
	var f found
	for sc := s; sc != nil; sc = sc.parent {
		decls := sc.decls[name]
		if len(decls) == 0 {
			continue
		}
		// The last let that has surely run at place, if any.
		var ran *decl
		for _, d := range decls {
			if !place.before(d.at) {
				ran = d
			}
		}
		if sc.function == s.function {
			if ran != nil {
				f.decls = append(f.decls, ran)
				return f
			}
			if f.later == nil && len(f.decls) == 0 {
				f.later = decls[0]
			}
			continue
		}
		f.decls = append(f.decls, decls...)
		if ran != nil {
			return f
		}
	}
	if m := c.moduleWith(name); m != nil {
		f.module = m
		return f
	}
	if _, ok := c.globals[name]; ok {
		f.global = true
	}
	return f
}

// moduleWith is the module, of the ones the file uses without as, whose
// name name is: the last used first, like the evaluator.
func (c *checker) moduleWith(name string) *module {
	for i := len(c.modules) - 1; i >= 0; i-- {
		if _, ok := c.modules[i].exports[name]; ok {
			return c.modules[i]
		}
	}
	return nil
}

// visibleNames are the names that could be meant at scope s: for a "did
// you mean" hint.
func (c *checker) visibleNames(s *scope) []string {
	var names []string
	for sc := s; sc != nil; sc = sc.parent {
		for name := range sc.decls {
			names = append(names, name)
		}
	}
	for _, m := range c.modules {
		for name := range m.exports {
			names = append(names, name)
		}
	}
	for name := range c.globals {
		names = append(names, name)
	}
	return names
}

// outerDecl is a declaration of d's name in a scope around d's that d
// hides, if any: in the same function only one that has run by then.
func outerDecl(d *decl) *decl {
	for sc := d.scope.parent; sc != nil; sc = sc.parent {
		for _, o := range sc.decls[d.name] {
			if sc.function != d.scope.function || !d.at.before(o.at) {
				return o
			}
		}
	}
	return nil
}
