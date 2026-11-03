package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
)

// Dot access (a.b) with a name known when compiling: the field's hash key
// is computed once, by the compiler, instead of at every read, a read of
// the same hash as last time, unchanged since, gives the value it found
// then, and a point's coordinate is read directly. Anything else goes to
// the evaluator's index, with the same result and errors.
func getField(field *code.Field, tok int, unit *code.Unit, left object.Object) object.Object {
	switch left := left.(type) {
	case *object.Hash:
		cache := &field.Cache
		if cache.Hash == left && cache.Version == left.Version() {
			return cache.Value
		}
		var value object.Object = evaluator.NULL
		if pair, ok := left.Pairs[field.Key]; ok {
			value = pair.Value
		}
		*cache = code.FieldCache{Hash: left, Version: left.Version(), Value: value}
		return value
	case *object.Point:
		if field.Coord >= 0 && field.Coord < len(left.Coords) {
			return left.Coords[field.Coord]
		}
	}
	return evaluator.Index(unit.Tokens[tok], left, field.Name)
}
