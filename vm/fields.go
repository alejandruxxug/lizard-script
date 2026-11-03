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
func getField(field *code.Field, tok int, unit *code.Unit, left object.Value) (object.Value, *object.Error) {
	switch container := left.Ref().(type) {
	case *object.Hash:
		cache := &field.Cache
		if cache.Hash == container && cache.Version == container.Version() {
			return cache.Value, nil
		}
		value := nullValue
		if pair, ok := container.Pairs[field.Key]; ok {
			value = object.ValueOf(pair.Value)
		}
		*cache = code.FieldCache{Hash: container, Version: container.Version(), Value: value}
		return value, nil
	case *object.Point:
		if field.Coord >= 0 && field.Coord < len(container.Coords) {
			return object.ValueOf(container.Coords[field.Coord]), nil
		}
	}
	result := evaluator.Index(unit.Tokens[tok], left.Object(), field.Name)
	if err, ok := result.(*object.Error); ok {
		return object.Value{}, err
	}
	return object.ValueOf(result), nil
}
