package evaluator

import "lizardscript/object"

// hashLibraryHash is the value of the name hash in the current run. Like
// mathHash, it is made on first use and dropped by ResetMath.
var hashLibraryHash *object.Hash

// hashLibrary returns the value of the name hash, a hash of builtins for
// copying, merging and asking hashes. A variable called hash hides it.
func hashLibrary() *object.Hash {
	if hashLibraryHash != nil {
		return hashLibraryHash
	}
	hash := object.NewHash()
	for name, fn := range map[string]object.BuiltinFunction{
		"copy":   hashCopy,
		"keys":   hashKeys,
		"values": hashValues,
		"merge":  hashMerge,
		"has":    hashHas,
		"remove": hashRemove,
	} {
		hash.Set(&object.String{Value: name}, &object.Builtin{Name: "hash." + name, Fn: fn})
	}
	hashLibraryHash = hash
	return hash
}

// hashArg reads argument i as a hash.
func hashArg(args []object.Object, i int) (*object.Hash, *object.Error) {
	h, ok := args[i].(*object.Hash)
	if !ok {
		return nil, argError("argument %d must be HASH, got %s", i+1, args[i].Type())
	}
	return h, nil
}

// hashAndKey reads a hash and a key that can be one.
func hashAndKey(args []object.Object) (*object.Hash, object.Hashable, *object.Error) {
	if err := checkCount(args, 2, 2); err != nil {
		return nil, nil, err
	}
	h, err := hashArg(args, 0)
	if err != nil {
		return nil, nil, err
	}
	key, ok := args[1].(object.Hashable)
	if !ok {
		return nil, nil, argError("unusable as hash key: %s", args[1].Type())
	}
	return h, key, nil
}

// onlyHash reads the one argument of a builtin as a hash.
func onlyHash(args []object.Object) (*object.Hash, *object.Error) {
	if err := checkCount(args, 1, 1); err != nil {
		return nil, err
	}
	return hashArg(args, 0)
}

// hash.copy(h) is a new hash with the same keys and values, in the same
// order (the values themselves are shared).
func hashCopy(args ...object.Object) object.Object {
	h, err := onlyHash(args)
	if err != nil {
		return err
	}
	return h.Copy()
}

// hash.keys(h) is an array with the keys, in order, like keys(h).
func hashKeys(args ...object.Object) object.Object {
	h, err := onlyHash(args)
	if err != nil {
		return err
	}
	keys := make([]object.Object, 0, h.Len())
	h.Each(func(pair object.HashPair) { keys = append(keys, pair.Key) })
	return &object.Array{Elements: keys}
}

// hash.values(h) is an array with the values, in the order of the keys,
// like values(h).
func hashValues(args ...object.Object) object.Object {
	h, err := onlyHash(args)
	if err != nil {
		return err
	}
	values := make([]object.Object, 0, h.Len())
	h.Each(func(pair object.HashPair) { values = append(values, pair.Value) })
	return &object.Array{Elements: values}
}

// hash.merge(a, b) is a new hash with the keys of a and then the new keys
// of b; where both have a key, the value of b wins and the key keeps its
// place in a. Neither a nor b changes.
func hashMerge(args ...object.Object) object.Object {
	if err := checkCount(args, 2, 2); err != nil {
		return err
	}
	a, err := hashArg(args, 0)
	if err != nil {
		return err
	}
	b, err := hashArg(args, 1)
	if err != nil {
		return err
	}
	merged := a.Copy()
	b.Each(func(pair object.HashPair) { merged.Set(pair.Key.(object.Hashable), pair.Value) })
	return merged
}

// hash.has(h, key) reports whether key is in h, even with the value null.
func hashHas(args ...object.Object) object.Object {
	h, key, err := hashAndKey(args)
	if err != nil {
		return err
	}
	_, ok := h.Get(key)
	return nativeBoolToBooleanObject(ok)
}

// hash.remove(h, key) takes key out of h, and returns whether it was there,
// like delete(h, key).
func hashRemove(args ...object.Object) object.Object {
	h, key, err := hashAndKey(args)
	if err != nil {
		return err
	}
	return nativeBoolToBooleanObject(h.Delete(key))
}
