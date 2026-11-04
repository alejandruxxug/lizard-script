package compiler

import (
	"github.com/alejandruxxug/lizard-script/engine/code"
	"github.com/alejandruxxug/lizard-script/lang/ast"
)

// newFuncState starts compiling a function inside parent (nil for the
// program).
func newFuncState(parent *funcState) *funcState {
	return &funcState{parent: parent, freeIndex: map[*symbol]int{}, last: -1, landing: -1}
}

// compileProgram compiles the statements of the program, in the global
// scope. Its value is the one of the last statement.
func (c *Compiler) compileProgram(program *ast.Program) *code.Function {
	fs := newFuncState(nil)
	c.fs = fs
	fs.scope = &scope{global: true, enter: -1}

	for i, st := range program.Statements {
		t := &target{kind: statementTarget, open: len(fs.open), end: c.newLabel()}
		fs.targets = append(fs.targets, t)
		c.emit(code.OpStatement, c.statementIndex(st))
		c.compileStatement(st)
		fs.targets = fs.targets[:len(fs.targets)-1]
		c.place(t.end)
		if i < len(program.Statements)-1 {
			c.emit(code.OpPop)
		}
	}
	if len(program.Statements) == 0 {
		c.emit(code.OpNull)
	}
	c.emit(code.OpReturnValue)

	return c.finishFunction(nil, 0, 0)
}

// compileFunction compiles a function literal into a function of the unit
// and emits the OpClosure that makes it. A call runs, in the function's own
// scope: the arguments in place, the defaults of the missing ones, the body.
func (c *Compiler) compileFunction(lit *ast.FunctionLiteral) {
	parent := c.fs
	fs := newFuncState(parent)
	c.fs = fs

	params := c.openFunctionScope(lit)
	required := requiredParameters(lit)
	c.compileDefaults(lit, params, required)
	c.compileStatements(lit.Body.Statements)
	c.emit(code.OpReturnValue)

	fn := c.finishFunction(lit, len(params), required)
	captures := fs.free
	c.fs = parent

	c.unit.Functions = append(c.unit.Functions, fn)
	c.unit.Captures = append(c.unit.Captures, captures)
	c.emit(code.OpClosure, len(c.unit.Functions)-1, len(c.unit.Captures)-1)
}

// openFunctionScope declares the function's scope, which holds the
// parameters and the lets of the body, like the call's environment does in
// the evaluator. It returns the symbol of each parameter.
//
// Each argument arrives in the slot of its position. A name used by two
// parameters is one variable, in the slot of its first position, and a
// later argument is moved there, as the evaluator's second define wins.
func (c *Compiler) openFunctionScope(lit *ast.FunctionLiteral) []*symbol {
	fs := c.fs
	s := &scope{symbols: map[string]*symbol{}, wrap: true, enter: -1}
	fs.scope = s

	params := make([]*symbol, len(lit.Parameters))
	for i, p := range lit.Parameters {
		if sym, ok := s.symbols[p.Value]; ok {
			params[i] = sym
			continue
		}
		params[i] = c.declare(s, p.Value, i)
	}
	fs.nextSlot = len(params)
	fs.numLocals = len(params)
	for _, name := range letNames(lit.Body.Statements) {
		if _, ok := s.symbols[name]; !ok {
			c.declare(s, name, c.allocSlot())
		}
	}

	for i, sym := range params {
		if sym.slot != i {
			c.emit(code.OpMoveIfGiven, i, sym.slot)
		}
	}
	if len(s.order) > 0 {
		c.emitEnterScope(s)
	}
	return params
}

// compileDefaults gives the missing arguments their defaults, left to
// right. Default i runs when the call gave fewer than i+1 arguments, so the
// parameters before it are set and the ones from it on are not: a read of
// those skips to the scopes around the function, as in the evaluator. A
// return value that comes out of a default is unwrapped, like the
// evaluator's bindDefaults does.
func (c *Compiler) compileDefaults(lit *ast.FunctionLiteral, params []*symbol, required int) {
	for i := required; i < len(params); i++ {
		for _, before := range params[:i] {
			before.defined = true
		}
		skip := c.newLabel()
		at := c.emit(code.OpJumpIfGiven, i, 0)
		c.point(at, 1, skip)
		if i < len(lit.Defaults) && lit.Defaults[i] != nil {
			c.compileExpression(lit.Defaults[i], false)
		} else {
			c.emit(code.OpNull)
		}
		c.emit(code.OpUnwrapReturn)
		c.setLocal(params[i])
		c.place(skip)
	}
	for _, sym := range params {
		sym.defined = true
	}
}

// finishFunction writes the signal stubs, closes the function's scope and
// builds the function.
func (c *Compiler) finishFunction(lit *ast.FunctionLiteral, params, required int) *code.Function {
	fs := c.fs
	c.emitStubs()
	if !fs.scope.global {
		c.closeScope()
	}
	return &code.Function{
		Instructions: fs.ins,
		NumLocals:    fs.numLocals,
		NumParams:    params,
		Required:     required,
		MaxStack:     fs.count + 2,
		Literal:      lit,
		Name:         c.names[lit],
		Unit:         c.unit,
	}
}

// requiredParameters is how many arguments a call must give, like the
// evaluator's: the parameters before the first one with a default.
func requiredParameters(lit *ast.FunctionLiteral) int {
	for i := range lit.Parameters {
		if i < len(lit.Defaults) && lit.Defaults[i] != nil {
			return i
		}
	}
	return len(lit.Parameters)
}
