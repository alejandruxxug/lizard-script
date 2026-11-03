package evaluator

import "testing"

func TestHashFunctions(t *testing.T) {
	ResetMath()
	checkAll(t, []expectation{
		{`hash.copy({"b": 1, "a": [2]})`, `{"b": 1, "a": [2]}`},
		{`hash.copy({})`, "{}"},
		{`let h = {"a": 1}; let c = hash.copy(h); c.b = 2; h`, `{"a": 1}`},
		{`let h = {"a": 1}; let c = hash.copy(h); c.a = 5; [h.a, c.a]`, "[1, 5]"},
		{`let h = {"a": [1]}; let c = hash.copy(h); push(c.a, 2); h`, `{"a": [1, 2]}`},
		{`let h = {"a": 1, "b": 2}; let c = hash.copy(h); delete(c, "a"); c.a = 3; [keys(h), keys(c)]`, `[["a", "b"], ["b", "a"]]`},
		{`let h = {1: "x", true: "y"}; hash.copy(h) == h`, "true"},

		{`hash.keys({"z": 1, 2: 0, true: 3})`, `["z", 2, true]`},
		{`hash.keys({})`, "[]"},
		{`hash.values({"z": 1, "a": null})`, "[1, null]"},
		{`let h = {"a": 1}; hash.keys(h) == keys(h) && hash.values(h) == values(h)`, "true"},

		{`hash.merge({"a": 1, "b": 2}, {"b": 3, "c": 4})`, `{"a": 1, "b": 3, "c": 4}`},
		{`hash.merge({}, {"x": 1})`, `{"x": 1}`},
		{`hash.merge({"x": 1}, {})`, `{"x": 1}`},
		{`let a = {"k": 1}; let b = {"k": 2}; hash.merge(a, b); [a.k, b.k]`, "[1, 2]"},
		{`let a = {"k": 1}; let m = hash.merge(a, a); [m == a, same(m, a)]`, "[true, false]"},

		{`hash.has({"a": null}, "a")`, "true"},
		{`hash.has({"a": 1}, "b")`, "false"},
		{`hash.has({1: 1}, "1")`, "false"},
		{`hash.has({true: 1}, true)`, "true"},

		{`let h = {"a": 1, "b": 2}; [hash.remove(h, "a"), h]`, `[true, {"b": 2}]`},
		{`hash.remove({"a": 1}, "z")`, "false"},
		{`let h = {"a": 1}; hash.remove(h, "a"); hash.remove(h, "a")`, "false"},

		{`let hash = 1; hash`, "1"},
		{`len(keys(hash))`, "6"},
	})
}

func TestHashErrors(t *testing.T) {
	ResetMath()
	checkErrors(t, []expectation{
		{`hash.copy([1])`, "[line 1, column 10] runtime error: hash.copy: argument 1 must be HASH, got ARRAY"},
		{`hash.copy()`, "[line 1, column 10] runtime error: hash.copy: wrong number of arguments: want 1, got 0"},
		{`hash.keys(1)`, "[line 1, column 10] runtime error: hash.keys: argument 1 must be HASH, got INTEGER"},
		{`hash.values({}, {})`, "[line 1, column 12] runtime error: hash.values: wrong number of arguments: want 1, got 2"},
		{`hash.merge({}, [1])`, "[line 1, column 11] runtime error: hash.merge: argument 2 must be HASH, got ARRAY"},
		{`hash.merge({})`, "[line 1, column 11] runtime error: hash.merge: wrong number of arguments: want 2, got 1"},
		{`hash.has({}, [1])`, "[line 1, column 9] runtime error: hash.has: unusable as hash key: ARRAY"},
		{`hash.has("a", "a")`, "[line 1, column 9] runtime error: hash.has: argument 1 must be HASH, got STRING"},
		{`hash.remove({}, 1.5)`, "[line 1, column 12] runtime error: hash.remove: unusable as hash key: FLOAT"},
		{`hash.remove({})`, "[line 1, column 12] runtime error: hash.remove: wrong number of arguments: want 2, got 1"},
	})
}

// same tells apart two containers with equal contents, and is === for the
// values that have no identity.
func TestSame(t *testing.T) {
	ResetMath()
	checkAll(t, []expectation{
		{`let a = [1]; same(a, a)`, "true"},
		{`let a = [1]; let b = a; push(b, 2); same(a, b)`, "true"},
		{`same([1], [1])`, "false"},
		{`same([], [])`, "false"},
		{`let a = [1]; same(a, list.copy(a))`, "false"},
		{`let h = {"a": 1}; same(h, h)`, "true"},
		{`same({"a": 1}, {"a": 1})`, "false"},
		{`let h = {"a": 1}; same(h, hash.copy(h))`, "false"},
		{`let h = {"a": 1}; [same(h, hash.copy(h)), h == hash.copy(h), h === hash.copy(h)]`, "[false, true, true]"},
		{`let h = {"in": [1]}; same(h.in, hash.copy(h).in)`, "true"},
		{`let f = function() { 1 }; same(f, f)`, "true"},
		{`same(function() { 1 }, function() { 1 })`, "false"},
		{`function make() { function() { 1 } }; same(make(), make())`, "false"},
		{`same(len, len)`, "true"},
		{`same(len, puts)`, "false"},
		{`same(math.sin, math.sin)`, "true"},
		{`same([1], {"a": 1})`, "false"},

		{`same(1, 1)`, "true"},
		{`same(1, 1.0)`, "false"},
		{`same(100000, 99999 + 1)`, "true"},
		{`same(0.5, 0.25 * 2)`, "true"},
		{`same("ab", "a" + "b")`, "true"},
		{`same("1", 1)`, "false"},
		{`same(true, true)`, "true"},
		{`same(true, 1)`, "false"},
		{`same(null, null)`, "true"},
		{`same(#ff0000, rgb(255, 0, 0))`, "true"},
		{`same((1, 2), (1, 2))`, "true"},
		{`same((1, 2), (1.0, 2))`, "false"},
		{`same([1], null)`, "false"},
	})
	checkErrors(t, []expectation{
		{`same(1)`, "[line 1, column 5] runtime error: same: wrong number of arguments: want 2, got 1"},
	})
}
