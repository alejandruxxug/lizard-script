// Package object defines the values that exist while a lizard-script program
// runs, and the environments that store variables.
package object

import (
	"fmt"
	"hash/fnv"
	"lizardscript/ast"
	"strings"
)

// ObjectType names the type of a value. It is what the type() builtin
// returns and what error messages show.
type ObjectType string

const (
	INTEGER_OBJ  = "INTEGER"
	BOOLEAN_OBJ  = "BOOLEAN"
	STRING_OBJ   = "STRING"
	NULL_OBJ     = "NULL"
	ARRAY_OBJ    = "ARRAY"
	HASH_OBJ     = "HASH"
	FUNCTION_OBJ = "FUNCTION"
	BUILTIN_OBJ  = "BUILTIN"

	// Internal values, never stored in a variable.
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	BREAK_OBJ        = "BREAK"
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

// Boolean is true or false. There are only two of them, see the evaluator.
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

// String is a text. Its length and indexes count characters, not bytes.
type String struct {
	Value string
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
func (a *Array) Inspect() string {
	parts := []string{}
	for _, e := range a.Elements {
		parts = append(parts, inspectNested(e))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

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

func (s *String) HashKey() HashKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))
	return HashKey{Type: s.Type(), Value: h.Sum64()}
}

// HashPair keeps the original key next to the value, so the hash can show
// and return its keys.
type HashPair struct {
	Key   Object
	Value Object
}

// Hash is a dictionary. Like arrays, hashes are shared, not copied. Order
// keeps the keys in the order they were first added.
type Hash struct {
	Pairs map[HashKey]HashPair
	Order []HashKey
}

// NewHash returns an empty hash.
func NewHash() *Hash {
	return &Hash{Pairs: map[HashKey]HashPair{}}
}

// Set adds or replaces the value of key.
func (h *Hash) Set(key Hashable, value Object) {
	hashKey := key.HashKey()
	if _, ok := h.Pairs[hashKey]; !ok {
		h.Order = append(h.Order, hashKey)
	}
	h.Pairs[hashKey] = HashPair{Key: key.(Object), Value: value}
}

// Delete removes key, and reports whether it was there.
func (h *Hash) Delete(key Hashable) bool {
	hashKey := key.HashKey()
	if _, ok := h.Pairs[hashKey]; !ok {
		return false
	}
	delete(h.Pairs, hashKey)
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
func (h *Hash) Inspect() string {
	parts := []string{}
	h.Each(func(pair HashPair) {
		parts = append(parts, inspectNested(pair.Key)+": "+inspectNested(pair.Value))
	})
	return "{" + strings.Join(parts, ", ") + "}"
}

// inspectNested shows strings inside arrays and hashes with their quotes,
// so ["1"] and [1] look different.
func inspectNested(o Object) string {
	if s, ok := o.(*String); ok {
		return "\"" + s.Value + "\""
	}
	return o.Inspect()
}

// Function is a function value. Env is the environment where it was
// created, which is how it remembers the variables around it (a closure).
type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}
	return "function(" + strings.Join(params, ", ") + ") " + f.Body.String()
}

// BuiltinFunction is the Go code behind a builtin. It returns an *Error
// to report a problem.
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

// Error is a runtime error. It stops the program as it travels up.
type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return e.Message }

// Environment stores variables. Each function call and each block gets its
// own, enclosed by the one around it, so a name is looked up from the
// innermost scope outwards.
type Environment struct {
	store map[string]Object
	outer *Environment
}

// NewEnvironment returns an empty top-level environment.
func NewEnvironment() *Environment {
	return &Environment{store: map[string]Object{}}
}

// NewEnclosedEnvironment returns an empty environment inside outer.
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get looks up name in this environment and the ones around it.
func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return obj, ok
}

// Define creates name in this environment, hiding any variable with the
// same name in the ones around it. This is what let does.
func (e *Environment) Define(name string, value Object) Object {
	e.store[name] = value
	return value
}

// Assign changes the variable name in the innermost environment that has it,
// and reports whether there was one. This is what = does.
func (e *Environment) Assign(name string, value Object) bool {
	for env := e; env != nil; env = env.outer {
		if _, ok := env.store[name]; ok {
			env.store[name] = value
			return true
		}
	}
	return false
}
