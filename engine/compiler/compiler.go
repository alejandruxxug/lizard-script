// Package compiler turns a parsed program into bytecode (package code) for
// the VM (package vm). The VM must do exactly what the evaluator does, so
// the compiler follows the evaluator's rules rather than simpler ones; the
// two places where that takes care are explained where they are done:
// names and scopes in scopes.go, return, break and continue in signals.go.
//
// One file per part: this one has the compiler's state and the emitting of
// instructions and table entries; functions.go the program and functions,
// statements.go and expressions.go each kind of node.
package compiler

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/code"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"math"
)

// Compile compiles program. The error only happens for programs beyond the
// limits of the bytecode (more than 2^32 of something).
func Compile(program *ast.Program) (unit *code.Unit, err error) {
	c := &Compiler{
		unit:      &code.Unit{},
		integers:  map[int64]int{},
		strings:   map[string]int{},
		floats:    map[uint64]int{},
		fieldKeys: map[string]*object.String{},
		statement: map[ast.Statement]int{},
		names:     map[*ast.FunctionLiteral]string{},
	}
	defer func() {
		if r := recover(); r != nil {
			failure, ok := r.(compileError)
			if !ok {
				panic(r)
			}
			unit, err = nil, failure
		}
	}()
	c.unit.Main = c.compileProgram(program)
	return c.unit, nil
}

// compileError stops compiling a program that doesn't fit the bytecode.
type compileError string

func (e compileError) Error() string { return string(e) }

// Compiler holds the unit being built and the function being compiled.
type Compiler struct {
	unit *code.Unit
	fs   *funcState

	// The constants already in the unit, so each is stored once.
	integers  map[int64]int
	strings   map[string]int
	floats    map[uint64]int
	fieldKeys map[string]*object.String
	statement map[ast.Statement]int

	// names are the functions a let names, for code.Function.Name.
	names map[*ast.FunctionLiteral]string

	// inFormula is set while the usual instructions of a formula are
	// compiled (see formulas.go).
	inFormula bool
}

// funcState is one function being compiled (the program is one too).
type funcState struct {
	parent *funcState
	ins    []byte
	count  int // instructions emitted, a bound for MaxStack
	// last is where the last instruction starts, and landing the last
	// place a jump goes to, for fusing two instructions into one (-1 when
	// there is none).
	last, landing int

	scope     *scope
	nextSlot  int
	numLocals int

	// free lists where each captured variable comes from, in the order of
	// the closure's Free cells.
	free      []code.Capture
	freeIndex map[*symbol]int

	// targets are the constructs a return, break or continue can stop at,
	// innermost last, and open the layers and trys open right now, which a
	// jump out of them must close.
	targets []*target
	open    []openKind

	// stubs are the signal checks whose code goes at the end.
	stubs []stub
}

// label is a place in the code that jumps go to, maybe not known yet.
type label struct {
	at   int
	uses []int // the operands waiting for it
}

// Emitting code

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	for _, o := range operands {
		if o < 0 || o > math.MaxUint32 {
			panic(compileError("the program is too big to compile"))
		}
	}
	fs := c.fs
	at := len(fs.ins)
	fs.ins = append(fs.ins, code.Make(op, operands...)...)
	fs.count++
	fs.last = at
	return at
}

// lastIs reports whether the last instruction is op and the next one can
// be fused with it: no jump goes between them. It returns where it starts.
func (c *Compiler) lastIs(op code.Opcode) (int, bool) {
	fs := c.fs
	if fs.last < 0 || fs.landing == len(fs.ins) || code.Opcode(fs.ins[fs.last]) != op {
		return 0, false
	}
	return fs.last, true
}

// replaceLast puts op in the place of the last instruction, which starts
// at `at`, so it stays where the instructions that point at it expect.
func (c *Compiler) replaceLast(at int, op code.Opcode, operands ...int) {
	c.fs.ins = c.fs.ins[:at]
	c.emit(op, operands...)
}

// operandAt is where operand i of the instruction at `at` starts. All
// operands that point somewhere are 4 bytes.
func operandAt(at, i int) int {
	return at + 1 + 4*i
}

func (c *Compiler) newLabel() *label {
	return &label{at: -1}
}

// place puts l here and fills the jumps that were waiting for it.
func (c *Compiler) place(l *label) {
	l.at = len(c.fs.ins)
	c.fs.landing = l.at
	for _, use := range l.uses {
		c.patch(use, l.at)
	}
	l.uses = nil
}

func (c *Compiler) patch(operand, value int) {
	ins := c.fs.ins
	if value == len(ins) {
		c.fs.landing = value
	}
	ins[operand] = byte(value >> 24)
	ins[operand+1] = byte(value >> 16)
	ins[operand+2] = byte(value >> 8)
	ins[operand+3] = byte(value)
}

// point fills operand i of the instruction at `at` with l, now or once l is
// placed.
func (c *Compiler) point(at, i int, l *label) {
	operand := operandAt(at, i)
	if l.at >= 0 {
		c.patch(operand, l.at)
		return
	}
	l.uses = append(l.uses, operand)
}

// jump emits a jump instruction whose only operand is l.
func (c *Compiler) jump(op code.Opcode, l *label) {
	at := c.emit(op, 0)
	c.point(at, 0, l)
}

// Tables

func (c *Compiler) token(tok token.Token) int {
	c.unit.Tokens = append(c.unit.Tokens, tok)
	return len(c.unit.Tokens) - 1
}

func (c *Compiler) node(n ast.Node) int {
	c.unit.Nodes = append(c.unit.Nodes, n)
	return len(c.unit.Nodes) - 1
}

func (c *Compiler) statementIndex(st ast.Statement) int {
	if i, ok := c.statement[st]; ok {
		return i
	}
	c.unit.Statements = append(c.unit.Statements, st)
	c.statement[st] = len(c.unit.Statements) - 1
	return len(c.unit.Statements) - 1
}

func (c *Compiler) constant(obj object.Object) int {
	c.unit.Constants = append(c.unit.Constants, obj)
	c.unit.Values = append(c.unit.Values, object.ValueOf(obj))
	return len(c.unit.Constants) - 1
}

func (c *Compiler) integer(value int64) int {
	if i, ok := c.integers[value]; ok {
		return i
	}
	i := c.constant(&object.Integer{Value: value})
	c.integers[value] = i
	return i
}

func (c *Compiler) str(value string) int {
	if i, ok := c.strings[value]; ok {
		return i
	}
	i := c.constant(&object.String{Value: value})
	c.strings[value] = i
	return i
}

func (c *Compiler) float(value float64) int {
	bits := math.Float64bits(value)
	if i, ok := c.floats[bits]; ok {
		return i
	}
	i := c.constant(&object.Float{Value: value})
	c.floats[bits] = i
	return i
}

// pointCoords are the names of a point's coordinates, in order, like the
// evaluator's.
var pointCoords = map[string]int{"x": 0, "y": 1, "z": 2}

// field is a new entry of the unit's fields for a dot access with this
// name. Each dot access gets its own, so the VM's cache of what it read
// (code.FieldCache) is about that place of the code only.
func (c *Compiler) field(name string) int {
	key, ok := c.fieldKeys[name]
	if !ok {
		key = &object.String{Value: name}
		c.fieldKeys[name] = key
	}
	coord, ok := pointCoords[name]
	if !ok {
		coord = -1
	}
	c.unit.Fields = append(c.unit.Fields, code.Field{Name: key, Key: key.HashKey(), Coord: coord})
	return len(c.unit.Fields) - 1
}

func (c *Compiler) name(n code.Name) int {
	c.unit.Names = append(c.unit.Names, n)
	return len(c.unit.Names) - 1
}

// fail emits an error the program reports when it gets here.
func (c *Compiler) fail(tok token.Token, format string, args ...any) {
	c.unit.Failures = append(c.unit.Failures, code.Failure{Token: tok, Message: fmt.Sprintf(format, args...)})
	c.emit(code.OpFail, len(c.unit.Failures)-1)
}
