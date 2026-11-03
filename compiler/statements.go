package compiler

import (
	"lizardscript/ast"
	"lizardscript/code"
)

// Every statement and expression leaves exactly one value on the stack,
// like Eval returns one. A block pops the values of all its statements but
// the last.

// compileStatements compiles the statements of a block, a function body or
// a layer, each announced to the OnStatement hook, keeping the last value.
// No statements give null.
func (c *Compiler) compileStatements(statements []ast.Statement) {
	if len(statements) == 0 {
		c.emit(code.OpNull)
		return
	}
	for i, st := range statements {
		c.emit(code.OpStatement, c.statementIndex(st))
		c.compileStatement(st)
		if i < len(statements)-1 {
			c.emit(code.OpPop)
		}
	}
}

// compileBlock compiles a block in a new scope.
func (c *Compiler) compileBlock(block *ast.BlockStatement) {
	c.openScope(letNames(block.Statements))
	c.compileStatements(block.Statements)
	c.closeScope()
}

// compileStatement compiles a statement in a statement's place: in a block,
// a function body, a layer or the program.
func (c *Compiler) compileStatement(st ast.Statement) {
	switch st := st.(type) {
	case *ast.LetStatement:
		c.compileExpression(st.Value, false)
		c.define(st.Name.Value)
		c.emit(code.OpNull)

	case *ast.ReturnStatement:
		if st.ReturnValue == nil {
			c.emit(code.OpNull)
		} else {
			c.compileExpression(st.ReturnValue, false)
		}
		c.emitSignal(signalReturn, false)

	case *ast.BreakStatement:
		c.emitSignal(signalBreak, false)

	case *ast.ContinueStatement:
		c.emitSignal(signalContinue, false)

	case *ast.TryStatement:
		c.compileTry(st)

	case *ast.LayerStatement:
		c.compileLayer(st)

	case *ast.BlockStatement:
		c.compileBlock(st)

	case *ast.ExpressionStatement:
		c.compileExpression(st.Expression, true)
		if mayBeSignal(st.Expression) {
			c.checkSignal()
		}

	default:
		c.emit(code.OpNull)
	}
}

// define stores the top value in name, in the innermost scope (let).
func (c *Compiler) define(name string) {
	s := c.fs.scope
	if s.global {
		c.emit(code.OpDefineGlobal, c.name(code.Name{Name: name, Final: -1}))
		return
	}
	sym, ok := s.symbols[name]
	if !ok {
		// Every let of a scope was declared when it opened; this is only
		// for a tree built by hand.
		sym = c.declare(s, name, c.allocSlot())
	}
	c.setLocal(sym)
	sym.defined = true
}

// compileTry compiles try { body } catch (name) { catch }. The VM jumps to
// the catch code with the stack as it was at the try and the error's hash
// on top.
func (c *Compiler) compileTry(st *ast.TryStatement) {
	fs := c.fs
	catch, end := c.newLabel(), c.newLabel()

	at := c.emit(code.OpPushHandler, 0)
	c.point(at, 0, catch)
	fs.open = append(fs.open, openHandler)
	c.compileBlock(st.Body)
	fs.open = fs.open[:len(fs.open)-1]
	c.emit(code.OpPopHandler)
	c.jump(code.OpJump, end)

	c.place(catch)
	var names []string
	if st.Name != nil {
		names = []string{st.Name.Value}
	}
	c.openScope(names)
	if st.Name != nil {
		c.define(st.Name.Value)
	} else {
		c.emit(code.OpPop)
	}
	c.compileBlock(st.Catch)
	c.closeScope()
	c.place(end)
}

// compileLayer compiles a layer: the hook is told when the body starts and
// ends, the body has its own scope, and the value is null.
func (c *Compiler) compileLayer(st *ast.LayerStatement) {
	fs := c.fs
	c.emit(code.OpLayerEnter, c.node(st))
	fs.open = append(fs.open, openLayer)
	c.openScope(letNames(st.Body.Statements))
	for _, inner := range st.Body.Statements {
		c.emit(code.OpStatement, c.statementIndex(inner))
		c.compileStatement(inner)
		c.emit(code.OpPop)
	}
	c.closeScope()
	fs.open = fs.open[:len(fs.open)-1]
	c.emit(code.OpLayerExit)
	c.emit(code.OpNull)
}

// compileIf compiles an if. transparent is whether it is in a statement's
// place, where a return, break or continue inside goes on through it.
func (c *Compiler) compileIf(node *ast.IfExpression, transparent bool) {
	done := c.boundary(transparent)
	alternative, end := c.newLabel(), c.newLabel()

	c.compileExpression(node.Condition, false)
	c.jump(code.OpJumpNotTruthy, alternative)
	c.compileBlock(node.Consequence)
	c.jump(code.OpJump, end)

	c.place(alternative)
	if node.Alternative != nil {
		c.compileBlock(node.Alternative)
	} else {
		c.emit(code.OpNull)
	}
	c.place(end)
	done()
}

// compileWhile compiles a while loop: a step and the condition before each
// round, the body in a new scope each round, and null at the end.
func (c *Compiler) compileWhile(node *ast.WhileExpression, transparent bool) {
	done := c.boundary(transparent)
	loop := &target{kind: loopTarget, open: len(c.fs.open), end: c.newLabel(), next: c.newLabel()}

	c.place(loop.next)
	c.emit(code.OpStep, c.token(node.Token))
	c.compileExpression(node.Condition, false)
	c.jump(code.OpJumpNotTruthy, loop.end)

	c.fs.targets = append(c.fs.targets, loop)
	c.compileBlock(node.Body)
	c.emit(code.OpPop)
	c.popTarget()
	c.jump(code.OpJump, loop.next)

	c.place(loop.end)
	c.emit(code.OpNull)
	done()
}

// compileFor compiles a for loop. The init runs once in a scope of its own
// around the loop (its value is dropped, and it is not announced to the
// hook); then each round is a step, the condition, the body in a new scope
// and the update, which is also where a continue goes.
func (c *Compiler) compileFor(node *ast.ForExpression, transparent bool) {
	done := c.boundary(transparent)
	loop := &target{kind: loopTarget, open: len(c.fs.open), end: c.newLabel(), next: c.newLabel()}
	start := c.newLabel()

	var names []string
	if let, ok := node.Init.(*ast.LetStatement); ok {
		names = []string{let.Name.Value}
	}
	c.openScope(names)
	switch init := node.Init.(type) {
	case nil:
	case *ast.LetStatement:
		c.compileExpression(init.Value, false)
		c.define(init.Name.Value)
	case *ast.ExpressionStatement:
		c.compileExpression(init.Expression, false)
		c.emit(code.OpPop)
	}

	c.place(start)
	c.emit(code.OpStep, c.token(node.Token))
	if node.Condition != nil {
		c.compileExpression(node.Condition, false)
		c.jump(code.OpJumpNotTruthy, loop.end)
	}

	c.fs.targets = append(c.fs.targets, loop)
	c.compileBlock(node.Body)
	c.emit(code.OpPop)
	c.popTarget()

	c.place(loop.next)
	if node.Update != nil {
		c.compileExpression(node.Update, false)
		c.emit(code.OpPop)
	}
	c.jump(code.OpJump, start)

	c.place(loop.end)
	c.closeScope()
	c.emit(code.OpNull)
	done()
}
