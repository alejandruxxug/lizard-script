package evaluator

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"sort"
)

// listHash is the value of the name list in the current run. Like mathHash,
// it is made on first use and dropped by ResetMath.
var listHash *object.Hash

// listLibrary returns the value of the name list, a hash of builtins for
// cutting and reordering arrays. A variable called list hides it.
func listLibrary() *object.Hash {
	if listHash != nil {
		return listHash
	}
	hash := object.NewHash()
	setLibrary(hash, "list.", map[string]object.BuiltinFunction{
		"slice":   listSlice,
		"insert":  listInsert,
		"remove":  listRemove,
		"index":   listIndex,
		"reverse": listReverse,
		"sort":    listSort,
		"copy":    listCopy,
	})
	listHash = hash
	return hash
}

// newArray makes an array that owns a copy of elements.
func newArray(elements []object.Object) *object.Array {
	return &object.Array{Elements: append([]object.Object{}, elements...)}
}

// list.slice(a, start, end) is a new array with the elements from start up
// to end, clamped like text.slice.
func listSlice(args ...object.Object) object.Object {
	if err := checkCount(args, 3, 3); err != nil {
		return err
	}
	a, err := arrayArg(args, 0)
	if err != nil {
		return err
	}
	from, err := integerArg(args, 1)
	if err != nil {
		return err
	}
	to, err := integerArg(args, 2)
	if err != nil {
		return err
	}
	start, end := clampIndex(from, len(a.Elements)), clampIndex(to, len(a.Elements))
	if end < start {
		end = start
	}
	return newArray(a.Elements[start:end])
}

// elementIndex reads argument 1 as an index into a, where a negative one
// counts from the end. allowEnd says whether len(a) is valid (it is for
// insert, where it means "at the end").
func elementIndex(a *object.Array, args []object.Object, allowEnd bool) (int, *object.Error) {
	n, err := integerArg(args, 1)
	if err != nil {
		return 0, err
	}
	length := int64(len(a.Elements))
	i := n
	if i < 0 {
		i += length
	}
	if i < 0 || i > length || (i == length && !allowEnd) {
		return 0, argError("index out of range: %d, the list has %d elements", n, length)
	}
	return int(i), nil
}

// list.insert(a, i, x) puts x before the element at index i, changes a in
// place and returns it. The index can be len(a) to add at the end.
func listInsert(args ...object.Object) object.Object {
	if err := checkCount(args, 3, 3); err != nil {
		return err
	}
	a, err := arrayArg(args, 0)
	if err != nil {
		return err
	}
	i, err := elementIndex(a, args, true)
	if err != nil {
		return err
	}
	a.Elements = append(a.Elements, nil)
	copy(a.Elements[i+1:], a.Elements[i:])
	a.Elements[i] = args[2]
	return a
}

// list.remove(a, i) takes out the element at index i, changes a in place and
// returns it. An index that doesn't exist is an error.
func listRemove(args ...object.Object) object.Object {
	if err := checkCount(args, 2, 2); err != nil {
		return err
	}
	a, err := arrayArg(args, 0)
	if err != nil {
		return err
	}
	i, err := elementIndex(a, args, false)
	if err != nil {
		return err
	}
	a.Elements = append(a.Elements[:i], a.Elements[i+1:]...)
	return a
}

// list.index(a, x) is the index of the first element that is === x, or -1.
func listIndex(args ...object.Object) object.Object {
	if err := checkCount(args, 2, 2); err != nil {
		return err
	}
	a, err := arrayArg(args, 0)
	if err != nil {
		return err
	}
	for i, element := range a.Elements {
		if strictEquals(element, args[1]) {
			return &object.Integer{Value: int64(i)}
		}
	}
	return &object.Integer{Value: -1}
}

// list.reverse(a) is a new array in the opposite order.
func listReverse(args ...object.Object) object.Object {
	if err := checkCount(args, 1, 1); err != nil {
		return err
	}
	a, err := arrayArg(args, 0)
	if err != nil {
		return err
	}
	out := newArray(a.Elements)
	for i, j := 0, len(out.Elements)-1; i < j; i, j = i+1, j-1 {
		out.Elements[i], out.Elements[j] = out.Elements[j], out.Elements[i]
	}
	return out
}

// list.copy(a) is a new array with the same elements (not copies of them).
func listCopy(args ...object.Object) object.Object {
	if err := checkCount(args, 1, 1); err != nil {
		return err
	}
	a, err := arrayArg(args, 0)
	if err != nil {
		return err
	}
	return newArray(a.Elements)
}

// list.sort(a, [less]) is a new array in order, keeping the order of equal
// elements. Without less the elements must be all numbers or all strings.
// less(x, y) is a lizard function that says whether x goes before y.
func listSort(args ...object.Object) object.Object {
	if err := checkCount(args, 1, 2); err != nil {
		return err
	}
	a, err := arrayArg(args, 0)
	if err != nil {
		return err
	}
	out := newArray(a.Elements)

	if len(args) == 2 {
		return sortWith(out, args[1])
	}
	return sortNatural(out)
}

// sortNatural sorts numbers by value or strings by their characters.
func sortNatural(out *object.Array) object.Object {
	allNumbers, allStrings := true, true
	for _, element := range out.Elements {
		allNumbers = allNumbers && isNumber(element)
		_, isString := element.(*object.String)
		allStrings = allStrings && isString
	}
	switch {
	case len(out.Elements) < 2:
	case allNumbers:
		sort.SliceStable(out.Elements, func(i, j int) bool {
			return numberLess(out.Elements[i], out.Elements[j])
		})
	case allStrings:
		sort.SliceStable(out.Elements, func(i, j int) bool {
			return out.Elements[i].(*object.String).Value < out.Elements[j].(*object.String).Value
		})
	default:
		return argError("without a comparison function the elements must be all numbers or all strings")
	}
	return out
}

// numberLess compares two numbers by value, exactly when both are integers.
func numberLess(x, y object.Object) bool {
	xi, xok := x.(*object.Integer)
	yi, yok := y.(*object.Integer)
	if xok && yok {
		return xi.Value < yi.Value
	}
	return toFloat(x) < toFloat(y)
}

// sortWith sorts using a lizard function. An error inside it stops the sort
// and is the result.
func sortWith(out *object.Array, less object.Object) object.Object {
	var parameters int
	switch function := less.(type) {
	case *object.Function:
		parameters = len(function.Parameters)
	case object.Callable:
		parameters = function.ParameterCount()
	default:
		return argError("argument 2 must be a function, got %s", less.Type())
	}
	if parameters != 2 {
		return argError("the comparison function must take 2 parameters, it takes %d", parameters)
	}

	var failure object.Object
	sort.SliceStable(out.Elements, func(i, j int) bool {
		if failure != nil {
			return false
		}
		result := applyFunction(token.Token{}, less, []object.Object{out.Elements[i], out.Elements[j]})
		if isError(result) {
			failure = result
			return false
		}
		return isTruthy(result)
	})
	if failure != nil {
		return failure
	}
	return out
}
