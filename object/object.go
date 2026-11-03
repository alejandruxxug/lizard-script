// Package object defines the values that exist while a lizard-script program
// runs, and the environments that store variables.
package object

import (
	"fmt"
	"lizardscript/ast"
	"lizardscript/token"
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// ObjectType names the type of a value. It is what the type() builtin
// returns and what error messages show.
type ObjectType string

const (
	INTEGER_OBJ  = "INTEGER"
	FLOAT_OBJ    = "FLOAT"
	BOOLEAN_OBJ  = "BOOLEAN"
	STRING_OBJ   = "STRING"
	COLOR_OBJ    = "COLOR"
	POINT_OBJ    = "POINT"
	NULL_OBJ     = "NULL"
	ARRAY_OBJ    = "ARRAY"
	HASH_OBJ     = "HASH"
	FUNCTION_OBJ = "FUNCTION"
	BUILTIN_OBJ  = "BUILTIN"

	// Internal values, never stored in a variable.
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	BREAK_OBJ        = "BREAK"
	CONTINUE_OBJ     = "CONTINUE"
	ERROR_OBJ        = "ERROR"
)

// Object is implemented by every value.
type Object interface {
	Type() ObjectType
	// Inspect writes the value the way the REPL and puts show it.
	Inspect() string
}

// Integer is a 64-bit signed whole number.
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

// Float is a 64-bit decimal number. It is never infinite or NaN: the
// evaluator turns any operation that would produce one into an error.
type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }

// Inspect writes the shortest text that reads back as the same float. It
// always has a '.' or an exponent, so 1.0 never looks like the integer 1.
// Very big and very small numbers use an exponent: 1e+21, 1e-07.
func (f *Float) Inspect() string {
	abs := math.Abs(f.Value)
	var text string
	if abs >= 1e21 || (abs != 0 && abs < 1e-6) {
		text = strconv.FormatFloat(f.Value, 'e', -1, 64)
	} else {
		text = strconv.FormatFloat(f.Value, 'f', -1, 64)
	}
	if !strings.ContainsAny(text, ".e") {
		text += ".0"
	}
	return text
}

// Color is a color with four channels, each 0 to 255. It can't be changed
// and can't be a hash key.
type Color struct {
	R, G, B, A uint8
}

func (c *Color) Type() ObjectType { return COLOR_OBJ }

// Inspect writes #rrggbb in lowercase, or #rrggbbaa when the color is not
// fully opaque.
func (c *Color) Inspect() string {
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

// Point is a position with two or three coordinates (x, y) or (x, y, z).
// Every coordinate is an Integer or a Float and keeps its type. It can't be
// changed and can't be a hash key.
type Point struct {
	Coords []Object
}

func (p *Point) Type() ObjectType { return POINT_OBJ }
func (p *Point) Inspect() string {
	parts := make([]string, len(p.Coords))
	for i, c := range p.Coords {
		parts[i] = c.Inspect()
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// Boolean is true or false. There are only two of them, see the evaluator.
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

// String is a text. Its length and indexes count characters, not bytes.
//
// Reading by character index is O(1) after the first access: Len and RuneAt
// keep what they learned about Value. Value must not change once a String
// has been read this way, and a String is not safe for concurrent use.
type String struct {
	Value string

	scanned bool   // Value was looked at
	ascii   bool   // every byte of Value is below 0x80
	runes   []rune // the characters, only filled for non-ASCII text

	// buffer is where Value's bytes live when Concat made it, so another
	// piece can be added to it in place (see strings.go).
	buffer *strings.Builder

	hash uint64 // HashKey's value once it was computed, 0 before
}

// scan looks at Value once.
func (s *String) scan() {
	if s.scanned {
		return
	}
	s.scanned = true
	s.ascii = true
	for i := 0; i < len(s.Value); i++ {
		if s.Value[i] >= 0x80 {
			s.ascii = false
			s.runes = []rune(s.Value)
			return
		}
	}
}

// Len is the number of characters of the string.
func (s *String) Len() int {
	s.scan()
	if s.ascii {
		return len(s.Value)
	}
	return len(s.runes)
}

// Runes returns the characters of the string. The caller must not change
// the slice.
func (s *String) Runes() []rune {
	s.scan()
	if s.ascii {
		s.runes = []rune(s.Value)
		s.ascii = false // runes is filled now, use it from here on
	}
	return s.runes
}

// RuneAt returns the character at index i, which must be 0 <= i < Len().
func (s *String) RuneAt(i int) rune {
	s.scan()
	if s.ascii {
		return rune(s.Value[i])
	}
	return s.runes[i]
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

// Null is the absence of a value: what an if without else, a loop or a
// missing hash key produce.
type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

// Array is an ordered list of values. Arrays are shared, not copied: after
// let b = a, a change through b is seen through a.
type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string  { return inspectNested(a) }

// HashKey identifies a key of a Hash. Two keys are the same when they have
// the same type and value, so 1 and "1" are different keys.
type HashKey struct {
	Type  ObjectType
	Value uint64
}

// Hashable is implemented by the values that can be hash keys.
type Hashable interface {
	HashKey() HashKey
}

func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: uint64(i.Value)}
}

func (b *Boolean) HashKey() HashKey {
	var value uint64
	if b.Value {
		value = 1
	}
	return HashKey{Type: b.Type(), Value: value}
}

// HashKey of a string is its 64-bit FNV-1a hash, computed right here (the
// same as hash/fnv's New64a) so it makes no garbage, once: the string
// keeps it, since its bytes never change.
func (s *String) HashKey() HashKey {
	if s.hash == 0 {
		const offset, prime = 14695981039346656037, 1099511628211
		h := uint64(offset)
		for i := 0; i < len(s.Value); i++ {
			h ^= uint64(s.Value[i])
			h *= prime
		}
		s.hash = h
	}
	return HashKey{Type: STRING_OBJ, Value: s.hash}
}

// KnownHash is the value of s.HashKey() when it was computed before, else 0:
// for an engine's fast path, which can't stop to compute it.
func (s *String) KnownHash() uint64 {
	return s.hash
}

// HashPair keeps the original key next to the value, so the hash can show
// and return its keys.
type HashPair struct {
	Key   Object
	Value Object
}

// Hash is a dictionary. Like arrays, hashes are shared, not copied. Order
// keeps the keys in the order they were first added.
//
// Pairs and Order can be read directly, but are changed only through Set
// and Delete, which keep them in step and count the change in Version.
type Hash struct {
	Pairs map[HashKey]HashPair
	Order []HashKey

	// version counts the changes made by Set and Delete, for Version.
	version uint64
}

// NewHash returns an empty hash.
func NewHash() *Hash {
	return &Hash{Pairs: map[HashKey]HashPair{}}
}

// NewHashSized returns an empty hash with room for size keys, for a hash
// whose size is known, like a literal's.
func NewHashSized(size int) *Hash {
	if size <= smallHash {
		// The hash and its order in one block of memory.
		block := &smallHashBlock{}
		block.hash = Hash{Pairs: make(map[HashKey]HashPair, size), Order: block.order[:0]}
		return &block.hash
	}
	return &Hash{Pairs: make(map[HashKey]HashPair, size), Order: make([]HashKey, 0, size)}
}

// smallHash is the most keys a hash made by NewHashSized keeps its order
// next to it for; a key more moves the order out, like a full slice.
const smallHash = 4

type smallHashBlock struct {
	hash  Hash
	order [smallHash]HashKey
}

// Version changes whenever Set or Delete changes the hash. An engine that
// remembers what a key read in this hash (the VM's dot access) can trust it
// while the version stays.
func (h *Hash) Version() uint64 {
	return h.version
}

// Len is the number of keys.
func (h *Hash) Len() int {
	return len(h.Pairs)
}

// Get returns the value of key, and whether it is there.
func (h *Hash) Get(key Hashable) (Object, bool) {
	pair, ok := h.Pairs[key.HashKey()]
	return pair.Value, ok
}

// Set adds or replaces the value of key.
func (h *Hash) Set(key Hashable, value Object) {
	hashKey := key.HashKey()
	if _, ok := h.Pairs[hashKey]; !ok {
		h.Order = append(h.Order, hashKey)
	}
	h.Pairs[hashKey] = HashPair{Key: key.(Object), Value: value}
	h.version++
}

// Copy returns a new hash with the same keys and values, in the same order.
// The values themselves are shared, not copied.
func (h *Hash) Copy() *Hash {
	pairs := maps.Clone(h.Pairs)
	if pairs == nil {
		pairs = map[HashKey]HashPair{}
	}
	return &Hash{Pairs: pairs, Order: slices.Clone(h.Order)}
}

// Delete removes key, and reports whether it was there.
func (h *Hash) Delete(key Hashable) bool {
	hashKey := key.HashKey()
	if _, ok := h.Pairs[hashKey]; !ok {
		return false
	}
	delete(h.Pairs, hashKey)
	h.version++
	for i, k := range h.Order {
		if k == hashKey {
			h.Order = append(h.Order[:i], h.Order[i+1:]...)
			break
		}
	}
	return true
}

// Each calls f with every pair, in order.
func (h *Hash) Each(f func(pair HashPair)) {
	for _, k := range h.Order {
		f(h.Pairs[k])
	}
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string  { return inspectNested(h) }

// Quote writes text as a string literal that reads back as the same text:
// in quotes, with the escapes \" \\ \n \t and \r, and \u{...} for any
// other control character. Bytes of 0x80 and up are kept as they are.
func Quote(text string) string {
	var out strings.Builder
	out.WriteByte('"')
	for i := 0; i < len(text); i++ {
		ch := text[i]
		switch {
		case ch == '"' || ch == '\\':
			out.WriteByte('\\')
			out.WriteByte(ch)
		case ch == '\n':
			out.WriteString(`\n`)
		case ch == '\t':
			out.WriteString(`\t`)
		case ch == '\r':
			out.WriteString(`\r`)
		case ch < 0x20 || ch == 0x7f:
			fmt.Fprintf(&out, `\u{%x}`, ch)
		default:
			out.WriteByte(ch)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// inspectStep is one piece of output left to write: a value, or a text.
// After the text of a step with close, close is no longer being written.
type inspectStep struct {
	value Object
	text  string
	close Object
}

// inspectNested writes a value inside an array or hash. Strings get their
// quotes, so ["1"] and [1] look different.
//
// It keeps the pieces left to write in a stack instead of calling itself for
// each element, so an array nested a million times doesn't overflow Go's
// stack, and it writes everything into one buffer, so deep nesting doesn't
// copy the text again at every level.
//
// open has the arrays and hashes being written right now: meeting one of
// them again means it contains itself, and it is shown as [...] or {...}
// instead of looping forever.
func inspectNested(root Object) string {
	var out strings.Builder
	open := map[Object]bool{}
	stack := []inspectStep{{value: root}}

	for len(stack) > 0 {
		step := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if step.value == nil {
			out.WriteString(step.text)
			if step.close != nil {
				delete(open, step.close)
			}
			continue
		}

		// The pieces of an array or hash go on the stack last first, so
		// they come off in order.
		switch o := step.value.(type) {
		case *String:
			out.WriteString(Quote(o.Value))

		case *Array:
			if open[o] {
				out.WriteString("[...]")
				continue
			}
			open[o] = true
			out.WriteString("[")
			stack = append(stack, inspectStep{text: "]", close: o})
			for i := len(o.Elements) - 1; i >= 0; i-- {
				stack = append(stack, inspectStep{value: o.Elements[i]})
				if i > 0 {
					stack = append(stack, inspectStep{text: ", "})
				}
			}

		case *Hash:
			if open[o] {
				out.WriteString("{...}")
				continue
			}
			open[o] = true
			out.WriteString("{")
			stack = append(stack, inspectStep{text: "}", close: o})
			for i := len(o.Order) - 1; i >= 0; i-- {
				pair := o.Pairs[o.Order[i]]
				stack = append(stack,
					inspectStep{value: pair.Value},
					inspectStep{text: ": "},
					inspectStep{value: pair.Key})
				if i > 0 {
					stack = append(stack, inspectStep{text: ", "})
				}
			}

		default:
			out.WriteString(o.Inspect())
		}
	}

	return out.String()
}

// Function is a function value. Env is the environment where it was
// created, which is how it remembers the variables around it (a closure).
type Function struct {
	Parameters []*ast.Identifier
	// Defaults has one entry per parameter, nil for one without a default
	// value; it is nil when none has one. They run at each call.
	Defaults []ast.Expression
	Body     *ast.BlockStatement
	Env      *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	params := []string{}
	for i, p := range f.Parameters {
		params = append(params, ast.ParameterString(p, f.Defaults, i))
	}
	return "function(" + strings.Join(params, ", ") + ") " + f.Body.String()
}

// Callable is a function value that another engine runs, like the closures
// of the bytecode VM (package vm). It has the type FUNCTION and shows its
// source like a Function. The evaluator calls it where it would call a
// Function, so builtins that take a function (list.sort) work with both.
type Callable interface {
	Object
	// ParameterCount is the number of parameters, defaults included.
	ParameterCount() int
	// Call runs the function with args, as a call at tok: it counts the
	// step, checks the arguments and the call depth, and returns the value
	// or an *Error, like a call of a Function.
	Call(tok token.Token, args []Object) Object
}

// BuiltinFunction is the Go code behind a builtin. It returns an *Error
// to report a problem.
//
// args belongs to the caller and is only lent for the call: the bytecode VM
// passes a piece of its value stack, which it reuses as soon as the
// builtin returns. A builtin may keep the values in it, but must not keep
// the slice itself (or a slice of it), nor change it; to keep them all,
// copy them (append([]Object{}, args...)).
type BuiltinFunction func(args ...Object) Object

// Builtin is a function written in Go, like len or puts.
type Builtin struct {
	Name string
	Fn   BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin " + b.Name }

// ReturnValue wraps the value of a return statement while it travels up to
// the function call (or the program) that it leaves.
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// Break travels up from a break statement to the innermost loop.
type Break struct{}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "break" }

// Continue travels up from a continue statement to the innermost loop, which
// goes on with its next round.
type Continue struct{}

func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "continue" }

// Error is a runtime error. It stops the program as it travels up.
type Error struct {
	Message string
	// Line and Column are where the error happened, the same numbers that
	// are in Message. They are 0 for an error that has no position yet.
	Line, Column int
	// Source is the name of the source where the error happened (see
	// lexer.NewNamed), or "" when that source has no name. It is not part
	// of Message.
	Source string
	// Uncatchable marks the errors that protect the host: the step limit
	// and the stack overflows. A try statement lets them go through.
	Uncatchable bool
	// AtCaller marks an error that is reported where its file was called
	// from: error(message, {"at": "caller"}). The call into the file moves
	// it there (see evaluator.AtCaller).
	AtCaller bool
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return e.Message }

// Environment stores variables. Each function call and each block gets its
// own, enclosed by the one around it, so a name is looked up from the
// innermost scope outwards.
//
// The top scope of a file can also use modules (use "shapes"): each module
// is the environment its file ran in, and the names it defines are seen
// from the file as if they were there, alive: reading one reads the
// module's variable, and = changes it. A file's own names come first, then
// the modules it uses, the last used first, then the scopes around it.
// Names that start with "_" are the module's own and are not seen.
type Environment struct {
	store map[string]Object
	outer *Environment
	// version counts the writes to store, for Version.
	version uint64
	// uses are the modules this scope sees, in the order they were used;
	// users are the scopes that use this one, whose version a write here
	// changes too.
	uses, users []*Environment
}

// Version changes whenever a variable of this environment, of the ones
// around it or of the modules they use is defined or assigned. An engine
// that remembers what a name read (the VM's global reads) can trust it while
// the version stays.
func (e *Environment) Version() uint64 {
	var version uint64
	for env := e; env != nil; env = env.outer {
		version += env.version
	}
	return version
}

// NewEnvironment returns an empty top-level environment.
func NewEnvironment() *Environment {
	return &Environment{store: map[string]Object{}}
}

// OwnVersion is the part of Version that counts e's own changes, and
// whether that is all of it: true when no environment is around e, like a
// program's. Code that must not make calls can read it inline.
func (e *Environment) OwnVersion() (version uint64, all bool) {
	return e.version, e.outer == nil
}

// NewEnclosedEnvironment returns an empty environment inside outer.
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get looks up name in this environment, the modules it uses and the ones
// around it.
func (e *Environment) Get(name string) (Object, bool) {
	for env := e; env != nil; env = env.outer {
		if obj, ok := env.store[name]; ok {
			return obj, true
		}
		if module := env.moduleWith(name); module != nil {
			return module.store[name], true
		}
	}
	return nil, false
}

// Define creates name in this environment, hiding any variable with the
// same name in the ones around it or in a module it uses. This is what let
// does.
func (e *Environment) Define(name string, value Object) Object {
	e.store[name] = value
	e.changed()
	return value
}

// Assign changes the variable name in the innermost environment that has it
// (or a module it uses), and reports whether there was one. This is what =
// does.
func (e *Environment) Assign(name string, value Object) bool {
	for env := e; env != nil; env = env.outer {
		if _, ok := env.store[name]; ok {
			env.store[name] = value
			env.changed()
			return true
		}
		if module := env.moduleWith(name); module != nil {
			module.store[name] = value
			module.changed()
			return true
		}
	}
	return false
}

// Use makes the names of module, the top scope of a module's file, seen
// from this scope (see Environment). Using the same module twice changes
// nothing.
func (e *Environment) Use(module *Environment) {
	for _, used := range e.uses {
		if used == module {
			return
		}
	}
	e.uses = append(e.uses, module)
	module.users = append(module.users, e)
	e.changed()
}

// Names lists the names defined in this environment itself, not the ones
// around it or in its modules, in alphabetical order.
func (e *Environment) Names() []string {
	names := make([]string, 0, len(e.store))
	for name := range e.store {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Exported reports whether a module's name is seen by the files that use
// it: every name but the ones that start with "_".
func Exported(name string) bool {
	return name != "" && name[0] != '_'
}

// moduleWith returns the module, of the ones this scope uses, that defines
// name (the last used first), or nil.
func (e *Environment) moduleWith(name string) *Environment {
	if len(e.uses) == 0 || !Exported(name) {
		return nil
	}
	for i := len(e.uses) - 1; i >= 0; i-- {
		if _, ok := e.uses[i].store[name]; ok {
			return e.uses[i]
		}
	}
	return nil
}

// changed counts a write, here and in the scopes that use this one.
func (e *Environment) changed() {
	e.version++
	for _, user := range e.users {
		user.version++
	}
}
