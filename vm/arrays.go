package vm

import "lizardscript/object"

// Array literals of two and three elements (spans, pairs, small records)
// are the commonest, and get the array and its elements in one block of
// memory instead of two. The elements' capacity is their length, so a push
// moves them out of the block like it moves any full array.

type array2 struct {
	array    object.Array
	elements [2]object.Object
}

type array3 struct {
	array    object.Array
	elements [3]object.Object
}

// arrayOf makes the array of an array literal with these elements.
func arrayOf(values []object.Object) *object.Array {
	switch len(values) {
	case 2:
		block := &array2{elements: [2]object.Object{values[0], values[1]}}
		block.array.Elements = block.elements[:]
		return &block.array
	case 3:
		block := &array3{elements: [3]object.Object{values[0], values[1], values[2]}}
		block.array.Elements = block.elements[:]
		return &block.array
	}
	elements := make([]object.Object, len(values))
	copy(elements, values)
	return &object.Array{Elements: elements}
}
