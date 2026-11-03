package code

import (
	"lizardscript/ast"
	"lizardscript/object"
	"lizardscript/token"
)

// Function is a compiled function, or the whole program for Unit.Main.
type Function struct {
	Instructions Instructions
	// NumLocals is how many slots the frame keeps for its variables; the
	// arguments are the first NumParams of them.
	NumLocals int
	NumParams int
	// Required is how many arguments a call must give: the parameters
	// before the first one with a default value.
	Required int
	// MaxStack is the most values the function can have on the stack at
	// once, above its locals.
	MaxStack int
	// Literal is the source of the function, nil for the program.
	Literal *ast.FunctionLiteral
	// Name is the variable a let gave the function when it was made
	// (function f() {} or let f = function...), "" otherwise. Only the
	// profiler shows it.
	Name string
	Unit *Unit
}

// Unit is one compiled program: its functions and the tables their
// instructions point into.
type Unit struct {
	Main       *Function
	Functions  []*Function
	Constants  []object.Object
	Tokens     []token.Token
	Statements []ast.Statement
	Nodes      []ast.Node
	Names      []Name
	Scopes     []Scope
	Captures   [][]Capture
	Failures   []Failure
	Fields     []Field
	Formulas   []Formula
}

// Name is how a variable that is not a plain local is found. The captured
// variables in Checked are tried in order, skipping the ones not set yet.
// Then the one in Final, which surely is, or, when Final is -1, the global
// environment by Name, and then the builtins. Token is where an error goes.
type Name struct {
	Name    string
	Token   token.Token
	Checked []int
	Final   int
	// Global is the VM's memory of this name's last read in the global
	// environment and the builtins.
	Global Global
}

// Global remembers that a name read Value in Env, so the VM doesn't look it
// up again while Env keeps its Version and the libraries their Resets (see
// object.Environment.Version and evaluator.LibraryResets). Env is nil until
// the first read.
type Global struct {
	Env     *object.Environment
	Version uint64
	Resets  uint64
	Value   object.Object
	// Misses counts the reads in a row that found Env changed.
	Misses uint32
}

// Field is the constant name of one dot access (a.b, or a["b"]) of the
// code, ready for the kinds of values it reads the most: Key for a hash, and
// Coord, the coordinate of x, y or z for a point (-1 for any other name).
type Field struct {
	Name  *object.String
	Key   object.HashKey
	Coord int
	// Cache is the VM's memory of this dot access's last read of a hash.
	Cache FieldCache
}

// FieldCache remembers that a dot access read Value in Hash, so the VM
// doesn't look the key up again while Hash keeps its Version (see
// object.Hash.Version). Hash is nil until the first read of a hash. The
// cache keeps that hash alive while the unit is.
type FieldCache struct {
	Hash    *object.Hash
	Version uint64
	Value   object.Object
}

// Formula is arithmetic (+ - * /, with at most one comparison < > <= >= at
// the top) on leaves, values that can be read without running any code:
// variables, number constants and their dot accesses. The VM computes it
// in one instruction, OpFormula, without making objects for the numbers in
// the middle; when it can't, it runs the usual instructions of the
// expression, which follow the OpFormula.
type Formula struct {
	Leaves []Leaf
	// Steps are the leaves and operators in the order the instructions
	// would run them: postfix.
	Steps []Step
	// Skip is the VM's: how many runs left go straight to the usual
	// instructions, after one whose values were not all numbers.
	Skip uint32
}

// MaxFormulaDepth bounds how many values a formula has waiting at once
// while it is computed, so the VM computes it in a small fixed buffer.
const MaxFormulaDepth = 8

// LeafKind says where a formula's leaf starts.
type LeafKind byte

const (
	LeafLocal    LeafKind = iota // Index is a slot, like OpGetLocal
	LeafCell                     // Index is a slot, like OpGetCell
	LeafFree                     // Index is a captured variable, like OpGetFree
	LeafName                     // Index is in Unit.Names, like OpGetName
	LeafConstant                 // Index is in Unit.Constants, like OpConstant
)

// Leaf is a value of a formula: a start and the dot accesses on it, in
// order (p.next.z).
type Leaf struct {
	Kind   LeafKind
	Index  int
	Fields []FieldRead
}

// FieldRead is one dot access of a leaf, like OpGetField: Unit.Fields[Field]
// with errors at Unit.Tokens[Token].
type FieldRead struct {
	Field, Token int
}

// Step is one step of a formula: push Leaves[Leaf], or, when Leaf is -1,
// apply the infix operator Op to the two values on top, with errors at
// Unit.Tokens[Token].
type Step struct {
	Leaf  int
	Op    Opcode
	Token int
}

// Scope lists the locals of a scope that closures capture: each gets a new
// cell when the scope starts. With Wrap (a function's own scope) the cell
// takes the value already in the slot, the argument.
type Scope struct {
	Slots []int
	Wrap  bool
}

// Capture says where a closure takes one of its captured variables from:
// a local of the function that makes it, or one of that function's own
// captured variables.
type Capture struct {
	Local bool
	Index int
}

// Failure is an error found while compiling that the program must report
// when it gets there, like the tree-walker does.
type Failure struct {
	Token   token.Token
	Message string
}
