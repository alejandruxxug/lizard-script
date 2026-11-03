package compiler

import (
	"lizardscript/ast"
	"lizardscript/code"
)

// How return, break and continue are compiled. In the evaluator they are
// values that travel up through Eval: blocks, ifs, layers and trys pass them
// on, loops take break and continue, a function call takes return. Here that
// is a jump to where they stop, closing the layers and trys in between.
//
// Two of the evaluator's corners are kept too:
//
//   - An if or loop used as a value (let x = if ...) is a "boundary": the
//     signal stops there and becomes the value, a RETURN_VALUE, BREAK or
//     CONTINUE.
//   - Such a value, read back as a statement (x;), acts like the signal it
//     holds. So a statement that could give one (a name, a call, an index,
//     an assignment) is followed by an OpCheckSignal, whose three ways out
//     are compiled at the end of the function.

type openKind int

const (
	openLayer openKind = iota
	openHandler
)

type targetKind int

const (
	loopTarget      targetKind = iota // takes break and continue
	boundaryTarget                    // an if or loop used as a value: takes everything
	statementTarget                   // a statement of the program: takes break and continue values
)

type target struct {
	kind targetKind
	open int // len(open) when it started
	// end is where a boundary or statement ends; for a loop, where a break
	// goes. next is where a continue goes.
	end, next *label
}

type signal int

const (
	signalReturn signal = iota
	signalBreak
	signalContinue
)

// stub is an OpCheckSignal and the targets and open constructs where it is.
type stub struct {
	at      int
	targets []*target
	open    []openKind
}

// pushTarget starts a construct that return, break or continue can stop at.
func (c *Compiler) pushTarget(kind targetKind) *target {
	t := &target{kind: kind, open: len(c.fs.open), end: c.newLabel(), next: c.newLabel()}
	c.fs.targets = append(c.fs.targets, t)
	return t
}

func (c *Compiler) popTarget() {
	c.fs.targets = c.fs.targets[:len(c.fs.targets)-1]
}

// boundary starts an if or loop used as a value, unless it is in a
// statement's place. done ends it.
func (c *Compiler) boundary(transparent bool) (done func()) {
	if transparent {
		return func() {}
	}
	t := c.pushTarget(boundaryTarget)
	return func() {
		c.popTarget()
		c.place(t.end)
	}
}

// emitSignal emits a return, break or continue from here. A signal written
// in the code (dynamic false) has the return value on the stack, or nothing
// for break and continue; one found as a value (dynamic true) has the value
// itself: a RETURN_VALUE, BREAK or CONTINUE.
func (c *Compiler) emitSignal(kind signal, dynamic bool) {
	fs := c.fs
	for i := len(fs.targets) - 1; i >= 0; i-- {
		t := fs.targets[i]
		switch {
		case t.kind == boundaryTarget:
			// The boundary's value is the signal itself.
			if !dynamic {
				switch kind {
				case signalReturn:
					c.emit(code.OpMakeReturn)
				case signalBreak:
					c.emit(code.OpBreakValue)
				default:
					c.emit(code.OpContinueValue)
				}
			}
			c.closeOpen(t.open)
			c.jump(code.OpJump, t.end)
			return

		case t.kind == loopTarget && kind != signalReturn:
			if dynamic {
				c.emit(code.OpPop)
			}
			c.closeOpen(t.open)
			if kind == signalBreak {
				c.jump(code.OpJump, t.end)
			} else {
				c.jump(code.OpJump, t.next)
			}
			return

		case t.kind == statementTarget && kind != signalReturn:
			// A break or continue value at the top of the program ends
			// the statement and is its value; the program goes on.
			c.closeOpen(t.open)
			c.jump(code.OpJump, t.end)
			return
		}
	}

	// The function (or the program) ends. A return value is unwrapped; a
	// break or continue value is the function's value as it is.
	if kind == signalReturn && dynamic {
		c.emit(code.OpUnwrapReturn)
	}
	c.emit(code.OpReturnValue)
}

// closeOpen closes the layers and trys opened since depth, innermost first.
func (c *Compiler) closeOpen(depth int) {
	open := c.fs.open
	for i := len(open) - 1; i >= depth; i-- {
		if open[i] == openLayer {
			c.emit(code.OpLayerExit)
		} else {
			c.emit(code.OpPopHandler)
		}
	}
}

// checkSignal emits an OpCheckSignal for the value of a statement. Its
// code, one piece per kind of signal, goes at the end of the function.
func (c *Compiler) checkSignal() {
	fs := c.fs
	at := c.emit(code.OpCheckSignal, 0, 0, 0)
	fs.stubs = append(fs.stubs, stub{
		at:      at,
		targets: append([]*target{}, fs.targets...),
		open:    append([]openKind{}, fs.open...),
	})
}

func (c *Compiler) emitStubs() {
	fs := c.fs
	targets, open := fs.targets, fs.open
	for _, s := range fs.stubs {
		fs.targets, fs.open = s.targets, s.open
		for i, kind := range []signal{signalReturn, signalBreak, signalContinue} {
			c.patch(operandAt(s.at, i), len(fs.ins))
			c.emitSignal(kind, true)
		}
	}
	fs.targets, fs.open = targets, open
	fs.stubs = nil
}

// mayBeSignal reports whether the value of an expression statement can be
// a RETURN_VALUE, BREAK or CONTINUE value: one read from a variable, an
// element or a call. An if in a statement's place has checked its own
// statements already.
func mayBeSignal(expression ast.Expression) bool {
	switch expression.(type) {
	case *ast.Identifier, *ast.CallExpression, *ast.IndexExpression, *ast.AssignExpression:
		return true
	}
	return false
}
