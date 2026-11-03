# 15. Builtins reference

[← Previous: Errors, `try`/`catch` and `null`](14-errors.md) · [Index](../README.md) · [Next: Projects →](16-projects.md) · [Español](../es/15-builtins-reference.md)

Every builtin function, and every member of the `math`, `text`, `list` and `hash` libraries, with one example each. The **Result** column is the value of the example as the REPL shows it, so strings appear with quotes. When the example is an error, the result shows the message (without the `[line, column]` part).

Builtins are found **after** your variables, so `let len = 5` hides `len`. The same goes for `math`, `text`, `list`, `hash` and `null`.

## General

| Builtin | Does | Example | Result |
| --- | --- | --- | --- |
| `puts(a, b, ...)` | prints the values separated by spaces, then a line break; gives `null` | `puts("hi", 1)` | prints `hi 1` |
| `len(x)` | characters of a string, items of an array or keys of a hash | `len("lizard")` | `6` |
| `type(x)` | the name of the type | `type(1.5)` | `"FLOAT"` |
| `str(x)` | `x` as text, the way `puts` shows it | `str(3.0)` | `"3.0"` |
| `int(x)` | a whole-number string, a boolean, an integer or a float (cut towards zero) as an integer | `int("42")` | `42` |
| | | `int(-1.9)` | `-1` |
| | | `int("1.5")` | `runtime error: int: not a whole number: "1.5"` |
| `float(x)` | an integer, float or number string as a float | `float("2.5")` | `2.5` |
| `range(end)` | `[0, 1, ..., end - 1]` | `range(4)` | `[0, 1, 2, 3]` |
| `range(start, end)` | `[start, ..., end - 1]`; empty if `end <= start` | `range(2, 5)` | `[2, 3, 4]` |
| `error(message)` | stops with a runtime error with that message | `error("bad input")` | `runtime error: bad input` |
| `error(message, {"at": "caller"})` | the same, reported at the call into this file (for a library's checks) | `error("bad size", {"at": "caller"})` | `runtime error: bad size`, at the caller's line |
| `null` | the "no value" value | `type(null)` | `"NULL"` |

## Arrays and hashes

| Builtin | Does | Example | Result |
| --- | --- | --- | --- |
| `push(a, x)` | adds `x` at the end of `a`, changes `a` and returns it | `push([1, 2], 3)` | `[1, 2, 3]` |
| `pop(a)` | removes the last item and returns it; error if empty | `pop([1, 2, 3])` | `3` |
| | | `pop([])` | `runtime error: pop: the array is empty` |
| `first(a)` | the first item, or `null` | `first([7, 8])` | `7` |
| `last(a)` | the last item, or `null` | `last([7, 8])` | `8` |
| `rest(a)` | a new array without the first item, or `null` if empty | `rest([7, 8, 9])` | `[8, 9]` |
| `keys(h)` | the keys, in the order they were added | `keys({"b": 1, "a": 2})` | `["b", "a"]` |
| `values(h)` | the values, in the same order | `values({"b": 1, "a": 2})` | `[1, 2]` |
| `delete(h, key)` | removes `key`, says whether it was there | `delete({"a": 1}, "a")` | `true` |
| `same(a, b)` | whether `a` and `b` are the same array, hash or function (for other values, `===`) | `same([1], [1])` | `false` |

## Colors and points

| Builtin | Does | Example | Result |
| --- | --- | --- | --- |
| `rgb(r, g, b)` | a color, channels 0 to 255 (floats are rounded) | `rgb(255, 128, 0)` | `#ff8000` |
| `rgba(r, g, b, a)` | a color with alpha | `rgba(0, 0, 0, 128)` | `#00000080` |
| `hsl(h, s, l)` | a color from hue 0 to 360, saturation and lightness 0 to 100 | `hsl(210, 50, 40)` | `#336699` |
| `point(x, y)`, `point(x, y, z)` | the same as `(x, y)` and `(x, y, z)` | `point(3, 4)` | `(3, 4)` |

## `math`

| Member | Does | Example | Result |
| --- | --- | --- | --- |
| `math.pi` | π | `math.pi` | `3.141592653589793` |
| `math.tau` | 2π | `math.tau` | `6.283185307179586` |
| `math.sqrt(x)` | square root, as a float; error if negative | `math.sqrt(2)` | `1.4142135623730951` |
| `math.abs(x)` | without the sign, same type | `math.abs(-3.5)` | `3.5` |
| `math.floor(x)` | rounded down, as an integer | `math.floor(-1.5)` | `-2` |
| `math.ceil(x)` | rounded up, as an integer | `math.ceil(1.2)` | `2` |
| `math.round(x)` | to the nearest integer, halves away from zero | `math.round(2.5)` | `3` |
| `math.min(a, b, ...)` | the smallest of two or more numbers | `math.min(4, 1.5, 9)` | `1.5` |
| `math.max(a, b, ...)` | the largest of two or more numbers | `math.max(4, 1.5, 9)` | `9` |
| `math.sin(x)` | sine, `x` in radians | `math.sin(90deg)` | `1.0` |
| `math.cos(x)` | cosine | `math.cos(0)` | `1.0` |
| `math.tan(x)` | tangent | `math.tan(45deg)` | `0.9999999999999998` |
| `math.asin(x)` | the angle whose sine is `x` (-1 to 1) | `math.asin(1)` | `1.5707963267948966` |
| `math.acos(x)` | the angle whose cosine is `x` (-1 to 1) | `math.acos(1)` | `0.0` |
| `math.atan2(y, x)` | the angle of the point `(x, y)` | `math.atan2(1, 1) / 1deg` | `45.0` |
| `math.dist(p, q)` | distance between two points | `math.dist((0, 0), (3, 4))` | `5.0` |
| `math.angle(p, q)` | angle of the line from `p` to `q` (2D points) | `math.angle((0, 0), (0, 5)) / 1deg` | `90.0` |
| `math.random()` | a float from 0 up to 1 (not included) | `math.random()` | `0.6046602879796196` |
| `math.random(a, b)` | an integer from `a` to `b`, both included | `math.random(1, 6)` | `3` |
| `math.seed(n)` | restarts the random numbers from `n`; gives `null` | `math.seed(42)` | `null` |

The random examples show the first numbers of a fresh run: every run starts from the same seed.

## `text`

Indexes count characters. Where noted, a negative index counts from the end.

| Member | Does | Example | Result |
| --- | --- | --- | --- |
| `text.slice(s, start, end)` | characters from `start` up to `end` (negative counts from the end) | `text.slice("lizard", 1, 4)` | `"iza"` |
| `text.find(s, sub)` | index of the first `sub`, or `-1` | `text.find("banana", "an")` | `1` |
| `text.find(s, sub, from)` | the same, starting at `from` | `text.find("banana", "an", 2)` | `3` |
| `text.split(s, sep)` | the pieces between each `sep`; `""` splits into characters | `text.split("a,b,c", ",")` | `["a", "b", "c"]` |
| `text.join(list, sep)` | joins a list of strings with `sep` | `text.join(["a", "b"], "-")` | `"a-b"` |
| `text.replace(s, old, new)` | every `old` replaced by `new` | `text.replace("aXbX", "X", "-")` | `"a-b-"` |
| `text.trim(s)` | without spaces, tabs and line breaks at both ends | `text.trim("  hi  ")` | `"hi"` |
| `text.lines(s)` | the lines of `s` | `text.lines("a\nb")` | `["a", "b"]` |
| `text.starts(s, prefix)` | whether `s` starts with `prefix` | `text.starts("lizard", "liz")` | `true` |
| `text.ends(s, suffix)` | whether `s` ends with `suffix` | `text.ends("lizard", "ard")` | `true` |
| `text.upper(s)` | in uppercase | `text.upper("Liz")` | `"LIZ"` |
| `text.lower(s)` | in lowercase | `text.lower("Liz")` | `"liz"` |
| `text.code(s, i)` | the code of the character at `i`, or `null` | `text.code("A", 0)` | `65` |
| `text.char(code)` | the character with that code | `text.char(955)` | `"λ"` |
| `text.digest(s, algorithm)` | a fingerprint of the text as hex: `"sha256"` (the default) or `"fnv"` | `text.digest("abc", "fnv")` | `"e71fa2190541574b"` |
| `text.number(s)` | the number written in `s`, or `null` | `text.number("2.5")` | `2.5` |
| | | `text.number("12px")` | `null` |

## `list`

A negative index counts from the end.

| Member | Does | Example | Result |
| --- | --- | --- | --- |
| `list.slice(a, start, end)` | a new array with the items from `start` up to `end` | `list.slice([1, 2, 3, 4], 1, 3)` | `[2, 3]` |
| `list.insert(a, i, x)` | puts `x` before index `i`, changes `a` and returns it | `list.insert(["a", "c"], 1, "b")` | `["a", "b", "c"]` |
| `list.remove(a, i)` | takes out the item at `i`, changes `a` and returns it | `list.remove([1, 2, 3], -1)` | `[1, 2]` |
| `list.index(a, x)` | the index of the first item `=== x`, or `-1` | `list.index([5, 6, 7], 7)` | `2` |
| `list.reverse(a)` | a new array in the opposite order | `list.reverse([1, 2, 3])` | `[3, 2, 1]` |
| `list.sort(a)` | a new array, numbers by value or strings alphabetically | `list.sort([3, 1, 2])` | `[1, 2, 3]` |
| `list.sort(a, less)` | sorted by your function `less(x, y)`: does `x` go first? | `list.sort([3, 1, 2], function(x, y) { x > y })` | `[3, 2, 1]` |
| `list.copy(a)` | a new array with the same items | `list.copy([1, 2])` | `[1, 2]` |

## `hash`

None of these change the hash, except `hash.remove`.

| Member | Does | Example | Result |
| --- | --- | --- | --- |
| `hash.copy(h)` | a new hash with the same keys and values, in the same order | `hash.copy({"a": 1})` | `{"a": 1}` |
| `hash.keys(h)` | the keys, like `keys` | `hash.keys({"b": 1, "a": 2})` | `["b", "a"]` |
| `hash.values(h)` | the values, like `values` | `hash.values({"b": 1, "a": 2})` | `[1, 2]` |
| `hash.merge(a, b)` | a new hash with the keys of `a`, then the new ones of `b`; `b`'s values win | `hash.merge({"a": 1, "b": 2}, {"b": 3})` | `{"a": 1, "b": 3}` |
| `hash.has(h, key)` | whether `key` is there, even holding `null` | `hash.has({"a": null}, "a")` | `true` |
| | | `hash.has({}, [1])` | `runtime error: hash.has: unusable as hash key: ARRAY` |
| `hash.remove(h, key)` | removes `key`, says whether it was there, like `delete` | `hash.remove({"a": 1}, "b")` | `false` |

## Values with dot access

These aren't functions, but they're read with a dot too:

| Value | Members | Example | Result |
| --- | --- | --- | --- |
| hash | any string key | `{"name": "Liz"}.name` | `"Liz"` |
| color | `r`, `g`, `b`, `a` (0 to 255) | `#3fa34d.g` | `163` |
| point | `x`, `y`, and `z` on 3D points | `(1, 2, 3).z` | `3` |

[← Previous: Errors, `try`/`catch` and `null`](14-errors.md) · [Index](../README.md) · [Next: Projects →](16-projects.md)
