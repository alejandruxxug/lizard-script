package compiler

import (
	"lizardscript/ast"
	"lizardscript/code"
	"lizardscript/token"
)

// How names are resolved. The evaluator keeps every scope as a map filled at
// run time; here each name a scope declares gets a slot of the frame, and
// each read is decided now, by where it is:
//
//   - In the scope's own function, a let has run before every read that
//     comes after it in the scope, and has not before one that comes
//     earlier: a scope is entered fresh each time (each round of a loop,
//     each call), and its statements run in order. So a read before the let
//     skips that scope and means the next one out, exactly like the map
//     lookup that doesn't find the name yet.
//   - A closure can run at any moment, so for it the compiler only knows
//     whether the let had run when the closure was made. If it had, the
//     variable is surely set; if not, the read checks the variable's cell
//     at run time and, while it is still empty, falls back to the next
//     scope out, then the next, as the evaluator would.
//   - A local a closure uses is a cell, made new each time its scope
//     starts, so closures share it by reference, and closures made in
//     different rounds of a loop get different ones.
//   - The program's own scope is the host's environment, used by name, so a
//     host can add globals between runs; after it come the builtins, null and
//     the libraries.

// scope is one scope of the evaluator. The program's own scope is global:
// its names live in the host's environment.
type scope struct {
	outer     *scope
	symbols   map[string]*symbol
	order     []*symbol
	global    bool
	wrap      bool // a function's scope: the cells take the arguments
	enter     int  // where its OpEnterScope is, or -1
	startSlot int
}

// symbol is a name declared in a scope.
type symbol struct {
	name  string
	slot  int
	scope *scope
	fs    *funcState
	// defined is whether the let has run at the point being compiled.
	defined bool
	// captured is whether a closure uses it, which makes it a cell.
	captured bool
	// refs are the instructions that use it as a plain local, rewritten to
	// the cell ones if it ends up captured.
	refs []int
}

// letNames are the names the lets of a block declare in its scope.
func letNames(statements []ast.Statement) []string {
	var names []string
	for _, st := range statements {
		if let, ok := st.(*ast.LetStatement); ok {
			names = append(names, let.Name.Value)
		}
	}
	return names
}

func (c *Compiler) allocSlot() int {
	fs := c.fs
	slot := fs.nextSlot
	fs.nextSlot++
	if fs.nextSlot > fs.numLocals {
		fs.numLocals = fs.nextSlot
	}
	return slot
}

func (c *Compiler) declare(s *scope, name string, slot int) *symbol {
	sym := &symbol{name: name, slot: slot, scope: s, fs: c.fs}
	s.symbols[name] = sym
	s.order = append(s.order, sym)
	return sym
}

// openScope starts a scope that declares names, and emits its
// OpEnterScope when it declares any.
func (c *Compiler) openScope(names []string) *scope {
	fs := c.fs
	s := &scope{outer: fs.scope, symbols: map[string]*symbol{}, enter: -1, startSlot: fs.nextSlot}
	for _, name := range names {
		if _, ok := s.symbols[name]; !ok {
			c.declare(s, name, c.allocSlot())
		}
	}
	fs.scope = s
	if len(s.order) > 0 {
		c.emitEnterScope(s)
	}
	return s
}

func (c *Compiler) emitEnterScope(s *scope) {
	c.unit.Scopes = append(c.unit.Scopes, code.Scope{Wrap: s.wrap})
	s.enter = c.emit(code.OpEnterScope, len(c.unit.Scopes)-1)
}

// closeScope ends the innermost scope. The locals that closures captured
// become cells: their OpEnterScope makes them, and the instructions that
// used them as plain locals are rewritten. Its slots are free again.
func (c *Compiler) closeScope() {
	fs := c.fs
	s := fs.scope
	var slots []int
	for _, sym := range s.order {
		if !sym.captured {
			continue
		}
		slots = append(slots, sym.slot)
		for _, at := range sym.refs {
			fs.ins[at] = byte(cellOpcode(code.Opcode(fs.ins[at])))
		}
	}
	if s.enter >= 0 {
		index := int(code.ReadUint32(fs.ins[s.enter+1:]))
		c.unit.Scopes[index].Slots = slots
		if len(slots) == 0 {
			fs.ins[s.enter] = byte(code.OpSkipScope)
		}
	}
	fs.nextSlot = s.startSlot
	fs.scope = s.outer
}

func cellOpcode(op code.Opcode) code.Opcode {
	switch op {
	case code.OpGetLocal:
		return code.OpGetCell
	case code.OpSetLocal:
		return code.OpSetCell
	case code.OpAssignLocal:
		return code.OpAssignCell
	}
	return op
}

// localOp emits a local instruction for sym, the cell one when it is
// already captured, remembering it otherwise.
func (c *Compiler) localOp(op code.Opcode, sym *symbol) {
	if sym.captured {
		c.emit(cellOpcode(op), sym.slot)
		return
	}
	at := c.emit(op, sym.slot)
	sym.refs = append(sym.refs, at)
}

func (c *Compiler) setLocal(sym *symbol) {
	c.localOp(code.OpSetLocal, sym)
}

// Names

type locationKind int

const (
	atLocal locationKind = iota // a local of this function, surely set
	atFree                      // a captured variable, surely set
	atName                      // through a code.Name
)

type location struct {
	kind  locationKind
	sym   *symbol
	index int
	// for atName
	checked []int
	final   int
}

// resolve decides what a name means at this point of the code, following
// the evaluator's lookup: innermost scope out, only the lets that have run.
func (c *Compiler) resolve(name string) location {
	fs := c.fs
	for s := fs.scope; s != nil; s = s.outer {
		if s.global {
			return location{kind: atName, final: -1}
		}
		if sym, ok := s.symbols[name]; ok && sym.defined {
			return location{kind: atLocal, sym: sym}
		}
	}

	// The scopes of the functions around this one, as they are where this
	// function is made. A let that has run there is surely set when this
	// function runs; one that hasn't might be by then, so it is checked.
	var checked []int
	for outer := fs.parent; outer != nil; outer = outer.parent {
		for s := outer.scope; s != nil; s = s.outer {
			if s.global {
				return c.chain(checked, -1)
			}
			sym, ok := s.symbols[name]
			if !ok {
				continue
			}
			index := c.capture(fs, sym)
			if sym.defined {
				return c.chain(checked, index)
			}
			checked = append(checked, index)
		}
	}
	return c.chain(checked, -1)
}

func (c *Compiler) chain(checked []int, final int) location {
	if len(checked) == 0 && final >= 0 {
		return location{kind: atFree, index: final}
	}
	return location{kind: atName, checked: checked, final: final}
}

// capture makes sym, a variable of a function around fs, one of fs's
// captured variables, and returns its index. A function in between captures
// it too, to pass it on.
func (c *Compiler) capture(fs *funcState, sym *symbol) int {
	if index, ok := fs.freeIndex[sym]; ok {
		return index
	}
	var from code.Capture
	if sym.fs == fs.parent {
		sym.captured = true
		from = code.Capture{Local: true, Index: sym.slot}
	} else {
		from = code.Capture{Local: false, Index: c.capture(fs.parent, sym)}
	}
	fs.free = append(fs.free, from)
	fs.freeIndex[sym] = len(fs.free) - 1
	return len(fs.free) - 1
}

// read emits the read of a name at tok.
func (c *Compiler) read(name string, tok token.Token) {
	loc := c.resolve(name)
	switch loc.kind {
	case atLocal:
		c.localOp(code.OpGetLocal, loc.sym)
	case atFree:
		c.emit(code.OpGetFree, loc.index)
	default:
		c.emit(code.OpGetName, c.name(code.Name{Name: name, Token: tok, Checked: loc.checked, Final: loc.final}))
	}
}

// assign emits name = (the top value), which stays on the stack.
func (c *Compiler) assign(name string, tok token.Token) {
	loc := c.resolve(name)
	switch loc.kind {
	case atLocal:
		c.localOp(code.OpAssignLocal, loc.sym)
	case atFree:
		c.emit(code.OpAssignFree, loc.index)
	default:
		c.emit(code.OpAssignName, c.name(code.Name{Name: name, Token: tok, Checked: loc.checked, Final: loc.final}))
	}
}
