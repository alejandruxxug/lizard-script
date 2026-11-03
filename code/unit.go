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
	Unit    *Unit
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
