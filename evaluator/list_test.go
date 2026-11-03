package evaluator

import "testing"

func TestListFunctions(t *testing.T) {
	ResetMath()
	checkAll(t, []expectation{
		{`list.slice([1, 2, 3, 4], 1, 3)`, "[2, 3]"},
		{`list.slice([1, 2, 3, 4], 0, 99)`, "[1, 2, 3, 4]"},
		{`list.slice([1, 2, 3, 4], -2, 4)`, "[3, 4]"},
		{`list.slice([1, 2, 3, 4], -99, -3)`, "[1]"},
		{`list.slice([1, 2, 3, 4], 3, 1)`, "[]"},
		{`list.slice([], 0, 1)`, "[]"},
		{`let a = [1, 2]; let b = list.slice(a, 0, 2); push(b, 3); a`, "[1, 2]"},

		{`list.insert([1, 3], 1, 2)`, "[1, 2, 3]"},
		{`list.insert([1, 2], 0, 0)`, "[0, 1, 2]"},
		{`list.insert([1, 2], 2, 3)`, "[1, 2, 3]"},
		{`list.insert([], 0, "a")`, `["a"]`},
		{`list.insert([1, 2], -1, 9)`, "[1, 9, 2]"},
		{`let a = [1]; list.insert(a, 1, 2); a`, "[1, 2]"},

		{`list.remove([1, 2, 3], 1)`, "[1, 3]"},
		{`list.remove([1, 2, 3], 0)`, "[2, 3]"},
		{`list.remove([1, 2, 3], -1)`, "[1, 2]"},
		{`list.remove([1], 0)`, "[]"},
		{`let a = [1, 2]; list.remove(a, 0); a`, "[2]"},

		{`list.index([5, 6, 7], 6)`, "1"},
		{`list.index([5, 6, 6], 6)`, "1"},
		{`list.index([5, 6, 7], 9)`, "-1"},
		{`list.index([], 1)`, "-1"},
		{`list.index([1, 2], 1.0)`, "-1"},
		{`list.index(["a", "b"], "b")`, "1"},
		{`list.index([[1], [2]], [2])`, "1"},

		{`list.reverse([1, 2, 3])`, "[3, 2, 1]"},
		{`list.reverse([])`, "[]"},
		{`let a = [1, 2]; list.reverse(a); a`, "[1, 2]"},

		{`list.copy([1, [2]])`, "[1, [2]]"},
		{`let a = [1]; let b = list.copy(a); push(b, 2); a`, "[1]"},
		{`let a = [[1]]; let b = list.copy(a); push(b[0], 2); a`, "[[1, 2]]"},

		{`list.sort([3, 1, 2])`, "[1, 2, 3]"},
		{`list.sort([2.5, 1, 2])`, "[1, 2, 2.5]"},
		{`list.sort(["b", "é", "a", ""])`, `["", "a", "b", "é"]`},
		{`list.sort([])`, "[]"},
		{`list.sort([1])`, "[1]"},
		{`let a = [2, 1]; list.sort(a); a`, "[2, 1]"},
		{`list.sort([3, 1, 2], function(x, y) { x > y })`, "[3, 2, 1]"},
		{`list.sort(["bb", "a", "ccc"], function(x, y) { len(x) > len(y) })`, `["ccc", "bb", "a"]`},
		{`list.sort([{"n": 2}, {"n": 1}], function(x, y) { x.n < y.n })`, `[{"n": 1}, {"n": 2}]`},
		// Equal elements keep their order.
		{`list.sort([[1, "a"], [0, "b"], [1, "c"]], function(x, y) { x[0] < y[0] })`, `[[0, "b"], [1, "a"], [1, "c"]]`},
		{`let calls = 0; list.sort([2, 1], function(x, y) { calls = calls + 1; x < y }); calls > 0`, "true"},

		{`let list = 1; list`, "1"},
		{`len(keys(list))`, "7"},
	})
}

func TestListErrors(t *testing.T) {
	ResetMath()
	checkErrors(t, []expectation{
		{`list.slice([1], 0)`, "[line 1, column 11] runtime error: list.slice: wrong number of arguments: want 3, got 2"},
		{`list.slice("a", 0, 1)`, "[line 1, column 11] runtime error: list.slice: argument 1 must be ARRAY, got STRING"},
		{`list.slice([1], "a", 1)`, "[line 1, column 11] runtime error: list.slice: argument 2 must be INTEGER, got STRING"},
		{`list.insert([1], 2, 0)`, "[line 1, column 12] runtime error: list.insert: index out of range: 2, the list has 1 elements"},
		{`list.insert([1], -3, 0)`, "[line 1, column 12] runtime error: list.insert: index out of range: -3, the list has 1 elements"},
		{`list.remove([1], 1)`, "[line 1, column 12] runtime error: list.remove: index out of range: 1, the list has 1 elements"},
		{`list.remove([], 0)`, "[line 1, column 12] runtime error: list.remove: index out of range: 0, the list has 0 elements"},
		{`list.remove([1], 0.5)`, "[line 1, column 12] runtime error: list.remove: argument 2 must be INTEGER, got FLOAT"},
		{`list.index([1])`, "[line 1, column 11] runtime error: list.index: wrong number of arguments: want 2, got 1"},
		{`list.reverse(1)`, "[line 1, column 13] runtime error: list.reverse: argument 1 must be ARRAY, got INTEGER"},
		{`list.copy()`, "[line 1, column 10] runtime error: list.copy: wrong number of arguments: want 1, got 0"},
		{`list.sort([1, "a"])`, "[line 1, column 10] runtime error: list.sort: without a comparison function the elements must be all numbers or all strings"},
		{`list.sort([true, false])`, "[line 1, column 10] runtime error: list.sort: without a comparison function the elements must be all numbers or all strings"},
		{`list.sort([1], 5)`, "[line 1, column 10] runtime error: list.sort: argument 2 must be a function, got INTEGER"},
		{`list.sort([1], function(x) { x })`, "[line 1, column 10] runtime error: list.sort: the comparison function must take 2 parameters, it takes 1"},
		{`list.sort([1, 2], function(x, y) { nope })`, "[line 1, column 10] runtime error: list.sort: [line 1, column 36] runtime error: identifier not found: nope"},
		{`list.sort([1, 2, 3], 1, 2)`, "[line 1, column 10] runtime error: list.sort: wrong number of arguments: want 1 to 2, got 3"},
	})
}
