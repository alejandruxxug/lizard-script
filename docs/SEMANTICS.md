# lizard-script semantics

What a program does when it runs. The syntax is in [GRAMMAR.md](GRAMMAR.md).

A program can run on two engines: the tree-walking evaluator (`evaluator.Eval`, the default) and the bytecode VM (`vm.Run`, `lizardscript --vm`). Everything on this page holds for both: they give the same values, output, errors with the same positions, steps and hook calls, and functions of one can be called from the other through the same environment. The one difference is a limit, see Functions.

## Values

| Type | Examples | Notes |
| --- | --- | --- |
| `INTEGER` | `5`, `-12` | 64-bit signed. Arithmetic wraps around on overflow, like Go. |
| `FLOAT` | `1.5`, `-0.25`, `45deg`, `1e3`, `2.5e-4` | 64-bit decimal number. Never infinite or NaN. |
| `COLOR` | `#3fa34d`, `rgb(63, 163, 77)` | Four channels, each an integer 0 to 255. Can't be changed or used as a hash key. |
| `POINT` | `(1, 2)`, `(0.5, 2, 3)` | Two or three coordinates, each an integer or a float that keeps its type. Can't be changed or used as a hash key. |
| `BOOLEAN` | `true`, `false` | |
| `STRING` | `"Liz"` | Length and indexes count characters, not bytes. Strings can't be changed. `"5" * 2` is `"55"`: a string times an integer repeats it. |
| `NULL` | | What an `if` without `else`, a loop, `let`, `puts` and a missing index produce. It can't be written directly. |
| `ARRAY` | `[1, "a", true]` | |
| `HASH` | `{"name": "Liz", 1: true}` | Keys must be integers, booleans or strings, not floats. Keys keep the order they were added in. |
| `FUNCTION` | `function(x) { x }` | |
| `BUILTIN` | `len`, `puts` | Functions written in Go, see below. |

Arrays and hashes are shared, not copied. After `let b = a`, a change made through `b` shows through `a` too, and a function that receives an array can change it.

An array or hash can even contain itself, like after `push(a, a)`. When it is shown, the inner copy is written as `[...]` or `{...}`: `a` shows as `[1, [...]]`. Comparing such values works too.

### Truthiness

Conditions accept any value. `false`, `null`, `0`, `0.0`, `""`, `[]` and `{}` count as false. Everything else counts as true.

## Variables and scopes

- `let x = ...` creates `x` in the current scope. If an outer scope already has an `x`, it is hidden, not changed. Writing `let` twice in the same scope replaces the old value.
- `x = ...` changes the closest `x` that already exists, so a function can change a variable of the scope it was created in. Assigning to a name that was never declared is an error.
- `x += e`, `x -= e`, `x *= e` and `x /= e` mean `x = x + e` and so on, with the same rules: `x` must exist, it is read before `e` runs, and the operator follows the rules of the types (`s += "a"` joins strings, `list += [1]` joins arrays, `s *= 2` repeats a string). They also work on items: `a[i] += e` and `h.n += e` evaluate `a`, `i` and `h` once, then read the item, apply the operator and store the result, so a missing item (which reads as `null`) is a type error, not a silent zero. The value of the expression is the new value.
- Each block (`{ ... }` of an `if`, a loop round or a function call) is a new scope. Variables created inside it are gone after it ends.
- The `let` in a `for` header lives in a scope of its own that wraps the whole loop. Each round of the body gets a fresh scope inside it.
- A name is looked up from the innermost scope outwards, and then among the builtins. That means a variable can hide a builtin: `let len = 5`.

## Operators

### Arithmetic

On integers, or on integers and floats. Two integers always give an integer, so `1 / 2` is `0`. If either side is a float, the other is converted to a float and the result is a float, except for comparisons.

| Operator | Meaning |
| --- | --- |
| `+` `-` `*` | As usual. `+` also joins two strings or two arrays into a new one. `*` repeats a string: `"ab" * 3` is `"ababab"`. |
| `/` | Division, rounding towards zero: `-7 / 2` is `-3`. |
| `//` | Division, rounding down: `-7 // 2` is `-4`. |
| `%` | The remainder that goes with `//`, so it has the sign of the right side: `-7 % 3` is `2`. `a == (a // b) * b + a % b` always holds. |
| `**` | Power. A negative exponent is an error, because the result wouldn't be an integer. `0 ** 0` is `1`. |
| `-x` | Negation, for integers and floats. |

With a float on either side:

- `/` is real division: `1 / 2.0` is `0.5`.
- `//` rounds down and gives a float: `-7.5 // 2` is `-4.0`.
- `%` is the remainder that goes with `//`, with the sign of the right side: `-7.5 % 2` is `0.5`, and `a == (a // b) * b + a % b` still holds.
- `**` is the power: `2.0 ** -1` is `0.5`, `4 ** 0.5` is `2.0`. Two integers with a negative exponent are still the error.

Dividing by zero, with any of `/`, `//` or `%`, is an error, for `0` and `0.0`. A float result that would be infinite or not a number (`10.0 ** 400`, `0.0 ** -1`, a product that overflows) is the runtime error `the result is not a number`, so lizard-script never has Infinity or NaN.

### Printing strings inside arrays and hashes

`puts("hi")` and `str("hi")` write the text as it is. A string inside an array or hash is written as code, in quotes with escapes, so the output can be pasted back into a program and `["1"]` is not mistaken for `[1]`: `puts(["a", "say \"hi\"", "two\nlines"])` writes `["a", "say \"hi\"", "two\nlines"]`. The escapes are `\"`, `\\`, `\n`, `\t`, `\r`, and `\u{...}` for other control characters; any other character is written as it is. The REPL shows a string result in quotes too, with only `"` and `\` escaped, so a string with a new line in it still takes several lines. `str(["a"])` is the string `["a"]` (with its quotes), so the REPL shows it as `"[\"a\"]"`.

### Printing floats

`puts` and `str` write the shortest text that reads back as the same float, and always with a `.` or an exponent, so a float never looks like an integer: `1.0`, `0.5`, `0.1`, `0.30000000000000004`. Numbers from `1e21` up and below `0.000001` use an exponent: `1e+21`, `1e-07`.

### Comparison

`<`, `>`, `<=` and `>=` compare two numbers by value (an integer and a float work too: `1.5 < 2`), or two strings in dictionary order (byte by byte). Comparing different types is an error.

Byte by byte is not alphabetical order: every uppercase letter is before every lowercase one (`"B" < "a"` is true), `"10" < "9"` is true, and a letter with an accent comes after `z` (`"é" < "z"` is false), because its bytes are higher. To sort names the way a person would, compare `text.lower(a)` and `text.lower(b)` (and know that accents still come last).

### Equality

- `===` is exact: the two values have the same type and the same value. `!==` is its opposite.
- `==` is loose: it works like `===`, but values of different types are first converted to numbers when possible. `true` is `1`, `false` is `0`, and a string that is a whole number is that number. So `1 == true` and `"5" == 5`, but `1 !== true`. Integers and floats are compared by value: `1 == 1.0` is true, and a string with a number such as `"1.5"` equals the float `1.5`. Only plain decimal strings count: `"1e3"`, `"inf"` and `"1."` are not numbers. `===` also needs the same type, so `1 === 1.0` is false. `!=` is its opposite.
- Values that can't be converted are never `==` to a value of another type: `"five" == 5` is `false`, and `null == 0` is `false`.
- Arrays and hashes are equal when their contents are, using the same kind of equality for the elements: `[1, "2"] == [1, 2]` but `[1, "2"] !== [1, 2]`. To ask whether two names hold the very same array or hash (one that a change through either name changes), use `same(a, b)`, see Builtins.
- Functions and builtins are only equal to themselves.

Equality never fails, whatever the types.

### Logic

`&&` and `||` stop early: the right side only runs when the left one doesn't decide the result. `!x` is `true` when `x` counts as false. The three of them always produce `true` or `false`.

## Control flow

- `if` produces the value of the block that ran, or `null` if none did. `elif` and `else if` are the same: one more condition, checked only if the ones before were false.
- `while` and `for` produce `null`. `break` leaves the innermost loop. `continue` skips the rest of the current round of the innermost loop and goes on with the next one: in a `for` the update part still runs, in a `while` the condition is checked again. Both can sit inside blocks, layers and `try` statements of the loop, and both are syntax errors outside a loop (or in a function defined inside one).
- `try { ... } catch (e) { ... }` runs the first block. If a runtime error stops it (any error: a failed operation, a builtin, `error(...)`, a function called from the block), the rest of the block is skipped and the second block runs, with `e` holding a hash with `"message"` (the text of the error without the `[line, column] runtime error:` start), `"line"` and `"column"` (where it happened, integers). The name is a variable of the catch block only, and `catch { ... }` has none. Each block has its own scope. The statement has the value of the block that ran, so it can be the last statement of a function. `return`, `break` and `continue` in either block work as if the try was not there. An error inside the catch block is not caught by the same try, but an outer try catches it.
- The errors that protect a host are never caught: the step limit (`the program ran too long`) and the stack overflows, also when they happen inside a builtin that called a function, like `list.sort`. They go up through every `try` and stop the program.
- A block, and a function body, produces the value of its last statement.

## Functions

- Calling a function with a different number of arguments than it has parameters is an error, except for parameters with a default value.
- A parameter can have a default: `function f(a, b = a * 2)`. When the call gives fewer arguments, each missing parameter gets its default, worked out at that call, in the scope of the function, from left to right, so it can use the parameters before it (not the ones after) and the variables around the function. `[]` as a default is a new list at each call. An error in a default is an error of the call. A call that leaves out a parameter without a default, or gives more arguments than parameters, is the error `wrong number of arguments: want 1 to 2, got 0` (`want 1, got 0` when no parameter has a default).
- A function remembers the scope it was created in (it is a closure), and sees the changes made to it later.
- A function without `return` gives the value of its last statement. A bare `return` gives `null`, and so does a `return` at the end of its line: a statement ends at the end of its line ([GRAMMAR.md](GRAMMAR.md#where-a-statement-ends)), so the value must start on the line of the `return`.
- `return` outside of a function ends the program. Its value is the value of the program, which the REPL shows.
- More than 10000 nested calls is a stack overflow error. So is a smaller number of calls whose bodies are nested deeply enough that, together, they would use too much of the interpreter's memory: `stack overflow: the calls and the code inside them nest too deeply`. The VM doesn't need that second limit, so there only the 10000 calls count.

## Indexing

- `array[i]` and `string[i]` take an integer, starting at 0. An index outside the array or string gives `null`, and that includes a negative one: `[1, 2, 3][-1]` is `null`, not the last element. (The `text` and `list` functions are different: there a negative index counts from the end, so `text.slice("hello", -3, 5)` is `"llo"`. Use `a[len(a) - 1]` for the last element with `[]`.)
- `hash[key]` gives `null` when the key isn't there. The key must be a valid key type. Keys are compared exactly, so `1` and `"1"` are different keys.
- `array[i] = x` changes an element that already exists. An index outside of the array is an error, so use `push` to add elements.
- `hash[key] = x` adds or replaces a key.
- `c.r`, `c.g`, `c.b`, `c.a` read the channels of a color (integers), and `p.x`, `p.y`, `p.z` the coordinates of a point (`z` only exists on a point with three). Any other name is an error, and so is assigning: `c.r = 1` is `colors can't be changed` and `p.x = 1` is `points can't be changed`, like strings. Make a new value instead.
- `a.name` is exactly `a["name"]`, for reading and for assigning, so on a hash it gives the value or `null`, and on anything else it gives the same error as indexing it with a string.

## Units

A number directly followed by `deg`, `rad`, `turn` or `px` is read by the lexer: `45deg` is the float `0.785...` (radians), `1.5rad` is `1.5`, `0.25turn` is `0.25 * 2π`, and `10px` is just `10` (an integer stays an integer, `10.5px` is a float). Letters right after a number that are not exactly one of these words are not a unit: `5kg` is the number `5` and the name `kg`, which is a syntax error (`unknown unit "kg"`).

## Colors

`#rgb`, `#rrggbb` and `#rrggbbaa` are colors; `#rgb` repeats each digit (`#f80` is `#ff8800`) and the alpha is 255 when it isn't written. A color is always truthy. `==` and `===` compare the four channels, and a color is never equal to a value of another type (`#fff == "#ffffff"` is false). There is no arithmetic on colors.

A color is shown as `#rrggbb` in lowercase, or `#rrggbbaa` when the alpha is not 255, so `puts(#3FA34D)` writes `#3fa34d` and `str(#fff)` is `"#ffffff"`.

| Builtin | Does |
| --- | --- |
| `rgb(r, g, b)`, `rgba(r, g, b, a)` | A color from channels. Each is an integer or a float, rounded to the nearest integer (halves away from zero), and must end up from 0 to 255; otherwise it is an error. |
| `hsl(h, s, l)` | A color from a hue in degrees (0 to 360, so `360` is the same as `0`) and a saturation and lightness in percent (0 to 100). Anything outside is an error. Channels are rounded: `hsl(210, 50, 40)` is `#336699`. |

## Points

`(x, y)` and `(x, y, z)` are points; `point(x, y)` and `point(x, y, z)` make the same value. Every coordinate must be an integer or a float, and keeps its type: `(1, 2.5)` has an integer `x` and a float `y`. A point is always truthy, even `(0, 0)`.

| Expression | Result |
| --- | --- |
| `p + q`, `p - q` | A point, coordinate by coordinate. Both need the same number of coordinates; mixing a 2D and a 3D point is an error. |
| `p * n`, `n * p`, `p / n` | Every coordinate times or divided by the number `n`. |
| `-p` | Every coordinate negated. |

Each coordinate follows the rules of numbers above: integers stay integers (`(3, 4) / 2` is `(1, 2)`, rounding towards zero), a float makes that coordinate a float (`(3, 4) / 2.0` is `(1.5, 2.0)`), dividing by zero is an error, and a coordinate that would be infinite or not a number is `the result is not a number`. Any other operator or operand (`p * q`, `p + 1`, `1 + p`, `2 / p`, `p // 2`, `p < q`) is the usual type error.

`==` and `===` compare the coordinates with the equality of numbers, so `(1, 2) == (1.0, 2)` is true and `(1, 2) === (1.0, 2)` is false. Points with a different number of coordinates are never equal.

A point is shown as `(1, 2)` or `(0.5, 2)`, each coordinate the way `puts` writes it.

`math.dist(p, q)` is the distance between two points with the same number of coordinates, as a float. `math.angle(p, q)` is the angle in radians of the line from `p` to `q`, measured from the positive x direction like `math.atan2`, as a float; it is only for points with two coordinates.

## Strings

In a string literal, `\"` is `"`, `\\` is `\`, `\n` is a line break, `\t` is a tab and `\r` is a carriage return. `\uXXXX` and `\u{X...}` are the character with that code point (`"\u{1F98E}"` is one character, so its `len` is 1); surrogate halves and values above 10FFFF are lexical errors. The value of the string holds the character, not the two source characters, so `len("a\nb")` is 3. Indexing (`s[i]`), `len` and the `text` helpers that count characters (`slice`, `find`, `code`) are fast on long strings: a string remembers its characters after the first use, so a loop over every index of a 200 KB string is linear, not quadratic.

## Layers

`layer name` followed by an indented body (see [GRAMMAR.md](GRAMMAR.md) for how the body is measured) runs the body once, in a new scope like any other block: a `let` inside stays inside, while outer variables can be read and assigned. A layer is a statement and its value is `null`. A `return`, a `break` or an error inside the body goes on up exactly as it would from any block, so `return` in a layer ends the function around it. Layers have no effect of their own on a program; they are names for hosts, see the hooks below.

## Builtins

`null` can be written in code. It is the same `null` that functions without a value give, and it is found after variables and builtins, so `let null = 5` still works. `type(null)` is `"NULL"`; it is false in conditions and equal only to `null`.

| Builtin | Does |
| --- | --- |
| `len(x)` | The number of characters of a string, or of elements of an array or hash. |
| `puts(a, b, ...)` | Writes its arguments separated by spaces, then a new line. Returns `null`. |
| `push(list, x)` | Adds `x` at the end of `list` and returns `list`. |
| `pop(list)` | Removes the last element of `list` and returns it. An error if `list` is empty. |
| `first(list)`, `last(list)` | The first or last element, or `null` if `list` is empty. |
| `rest(list)` | A new array with every element but the first, or `null` if `list` is empty. |
| `keys(hash)`, `values(hash)` | Arrays with the keys or the values, in the order the keys were added. |
| `delete(hash, key)` | Removes `key`, and returns whether it was there. |
| `same(a, b)` | Whether `a` and `b` are the same value: the same array, hash, function, builtin or native, not two with equal contents (`same([1], [1])` is `false`, while `[1] === [1]` is `true`). Numbers, strings, booleans, colors, points and `null` have no identity of their own, so for them it is `===`: `same(1, 1)` is `true`, `same(1, 1.0)` is `false`. |
| `type(x)` | The name of the type of `x`, like `"INTEGER"`. |
| `str(x)` | `x` as a string, the way `puts` shows it. |
| `int(x)` | A string with a whole number, a boolean, an integer or a float as an integer. A float loses its decimals towards zero: `int(-1.9)` is `-1`. `int("1.5")` is an error. |
| `error(message)` | Stops the program with the runtime error `[line L, column C] runtime error: message`, positioned at the call. A string is used as it is; any other value is turned into text like `str` does. |
| `error(message, options)` | The same, with a hash of options. `{"at": "caller"}` reports the error where the program called into this file instead (see [Errors at the caller](#errors-at-the-caller)); `{"at": "here"}` is the default. Any other key or value is an error of `error` itself. A second argument that is not a hash is still `wrong number of arguments: want 1, got 2`. |
| `float(x)` | An integer, a float or a string with a number (`"2"`, `"-0.25"`) as a float. |
| `range(end)`, `range(start, end)` | `[start, start + 1, ..., end - 1]`, with `start` 0 by default. Empty if `end <= start`. |

## The math library

`math` is a hash of builtins and constants, used with dot access. Angles are in radians; the units `deg`, `rad` and `turn` make them easy to write. Every function takes integers or floats, and a wrong type or number of arguments is an error like in any builtin.

| Name | Does |
| --- | --- |
| `math.pi`, `math.tau` | π and 2π, as floats. |
| `math.sin(x)`, `math.cos(x)`, `math.tan(x)` | As a float. |
| `math.atan2(y, x)` | The angle of the point `(x, y)`, as a float. |
| `math.asin(x)`, `math.acos(x)` | The angle in radians whose sine or cosine is `x`, as a float. `x` must be from -1 to 1, otherwise it is an error like `sqrt(-1)`. |
| `math.sqrt(x)` | The square root, as a float. A negative `x` is an error. |
| `math.abs(x)` | Without the sign. An integer for an integer, a float for a float. |
| `math.floor(x)`, `math.ceil(x)`, `math.round(x)` | An integer. `round` goes away from zero on halves: `math.round(2.5)` is `3`, `math.round(-2.5)` is `-3`. An integer is returned as it is, and a float too big for an integer is an error. |
| `math.min(a, b, ...)`, `math.max(a, b, ...)` | The smallest or largest of at least two numbers, compared by value. The result is that argument itself, so `math.min(1, 2.5)` is the integer `1`; on a tie the first one. |
| `math.random()` | A float from 0 up to but not including 1. |
| `math.random(a, b)` | An integer from `a` to `b`, both included. An error if `b < a`. |
| `math.dist(p, q)`, `math.angle(p, q)` | On points, see Points above. |
| `math.seed(n)` | Restarts `random` from the integer `n`. Returns `null`. |

`random` is deterministic: every run starts from the same default seed, so the same program makes the same numbers (and the same picture). `math.seed(n)` picks another sequence, and the same `n` always gives the same one. A host that runs programs one after another calls `evaluator.ResetMath()` before each run; the REPL and `Run` do it themselves, once per session or file.

`math` is found after variables and the other builtins, so `let math = 1` hides it. The hash is made new for each run: a program can write `math.pi = 3`, but only that run sees it.

## The text library

`text` is a hash of builtins like `math`, for cutting and joining strings. Indexes count characters, not bytes, so `"é"` is one character. A wrong type or number of arguments is an error like in any builtin.

| Name | Does |
| --- | --- |
| `text.slice(s, start, end)` | The characters from `start` up to `end`. Both are clamped to the string, and a negative one counts from the end (`-1` is the last character's end). `""` if `end` is not after `start`. |
| `text.find(s, sub, [from])` | The index of the first `sub` at or after `from` (0 by default; negative counts from the end, clamped), or `-1`. An empty `sub` is found at `from`. |
| `text.split(s, sep)` | An array of the pieces between each `sep`. An empty `sep` gives the characters. |
| `text.join(list, sep)` | The strings of `list` joined with `sep`. Every element must be a string. |
| `text.replace(s, old, new)` | Every `old` replaced by `new`. An empty `old` is an error. |
| `text.trim(s)` | Without the white space at both ends. |
| `text.lines(s)` | `s` cut at every `"\n"` or `"\r\n"`. Empty lines stay, and a trailing line break gives a last empty line, so `text.join(text.lines(s), "\n")` gives `s` back (with `"\n"` line breaks). |
| `text.starts(s, prefix)`, `text.ends(s, suffix)` | Booleans. |
| `text.upper(s)`, `text.lower(s)` | Case changed. |
| `text.code(s, i)` | The character code of the character at index `i` (negative counts from the end), or `null` if there is none. |
| `text.char(code)` | The one-character string with that code. An error for a code outside Unicode or a surrogate. |
| `text.digest(s, algorithm = "sha256")` | The hash of the text's bytes (UTF-8) as lowercase hex: `"sha256"` (64 digits, for telling contents apart) or `"fnv"` (FNV-1a of 64 bits, 16 digits: fast, but easy to collide on purpose). The same text gives the same digest on every machine. `text.digest("abc", "fnv")` is `"e71fa2190541574b"`. Any other algorithm is an error. |
| `text.number(s)` | The number written in `s` (optional white space, sign, digits, fraction, exponent): an INTEGER when it has no fraction or exponent and fits, else a FLOAT. `null` if `s` isn't a number (`"12px"`, `""`, `"Inf"`, `"0x10"`), never an error for a string. |

## The list library

`list` is a hash of builtins for arrays, used like `math` and `text`. Indexes are integers; a negative one counts from the end.

| Name | Does |
| --- | --- |
| `list.slice(a, start, end)` | A new array with the elements from `start` up to `end`, clamped like `text.slice`. |
| `list.insert(a, i, x)` | Puts `x` before index `i` (`len(a)` adds at the end), changes `a` in place and returns it, like `push`. An index outside `0` to `len(a)` is an error. |
| `list.remove(a, i)` | Takes out the element at `i`, changes `a` in place and returns `a`. An index that doesn't exist is an error. |
| `list.index(a, x)` | The index of the first element that is `=== x`, or `-1`. |
| `list.reverse(a)` | A new array in the opposite order. |
| `list.sort(a, [less])` | A new array in order, keeping the order of equal elements. Without `less` the elements must be all numbers (by value) or all strings (by characters). `less(x, y)` is a lizard function with two parameters that says whether `x` goes before `y`. An error inside it stops the sort and is reported with the position inside `less`. |
| `list.copy(a)` | A new array with the same elements (the elements themselves are shared). |

## The hash library

`hash` is a hash of builtins for whole hashes, used like `list`. Keys follow the rules of indexing: a key that can't be one (a float, an array, ...) is the error `unusable as hash key`. Only `hash.remove` changes a hash.

| Name | Does |
| --- | --- |
| `hash.copy(h)` | A new hash with the same keys and values, in the same order. The values themselves are shared, like `list.copy`. |
| `hash.keys(h)`, `hash.values(h)` | The same arrays as `keys(h)` and `values(h)`. |
| `hash.merge(a, b)` | A new hash with the keys of `a` in their order, then the keys of `b` that `a` doesn't have, in theirs. Where both have a key, the value is `b`'s. |
| `hash.has(h, key)` | Whether `key` is in `h`, also when its value is `null` (both read as `null` with `h[key]`). |
| `hash.remove(h, key)` | Removes `key`, and returns whether it was there, like `delete`. |

`text`, `list` and `hash` are found after variables and the other builtins, so `let text = 1` hides the library. They are made new for each run, like `math`, and `evaluator.ResetMath()` resets them too.

## Modules

`use "draw/shapes"` loads a **module**: the file `draw/shapes.liz`, found next to the file that says the `use` (its source name, see below), then in each folder of the module path (`evaluator.ModulePath`; the `lizardscript` command takes it from `$LIZARD_PATH`, folders separated like `PATH`), then among the **standard modules** (`evaluator.StandardModules`, libraries built into `lizardscript`, like `flower/lily2d`; see [TERRARIUM.md](TERRARIUM.md#standard-modules)). A standard module's source name starts with `std:/` (`std:/flower/lily2d.liz`), so a `use` inside it finds the files next to it the same way, and a path can't climb out of them with `..`. The not-found message lists only the files on disk, as before. A path that starts with `/` (or a drive) is used as it is. ([LEP 0002](proposals/0002-modules.md))

- **Once per run.** The first use of a module in a run runs its file on the same engine as the program, in a **top scope of its own**: it doesn't see the program's variables, or those of the files that use it. Around that scope there is what the host gives every module (`evaluator.ModulePrelude`: `lizardscript` gives the whole Terrarium, the same as the program has), and then the builtins and libraries. Every later use of the same file (however its path is written) in the same run reuses it: its code doesn't run again. `evaluator.ResetModules()` forgets them; the REPL and `Run` call it at the start. A file is parsed only when its text changed since it was last parsed (the parsed programs are kept by path and text across runs, and the VM keeps the code it compiled for each), so a host that runs programs one after another with the same libraries doesn't parse or compile them again; their code still runs once per run.
- **`use "x"` lets the file see every name the module defined at its top level, alive.** Reading one reads the module's own variable, so a module's function that changes it is seen at once; `=` (and `+=` ...) on it changes the module's variable, for every file that uses the module. A file's own names come first: a `let` of the same name in the file, before or after the `use`, hides the module's. Between modules, the last one used comes first. Names that start with `_` are the module's own and are not seen (`identifier not found: _made`). A module's names are not passed on: what a module itself uses is not seen by the files that use it.
- **`use "x" as m` defines `m`**, a hash from each of those names to its value, in alphabetical order (`keys(m)`). The hash is filled when the module has just run, and every `use ... as` of the module gets that same hash: its functions work on the module's live variables, but a variable read through it (`m.count`) is the value it had then. Read state that changes through a function.
- A use's value is `null`. Both forms of the same module share the one run of its file.
- **Errors.** A file that is not found is the runtime error `can't find the module "x": there is no a/x.liz or b/x.liz` at the use. A module whose file has syntax errors makes the use fail with all of them, one per line. A runtime error while the module runs stops the program like any error: it keeps the module's position and source name (so the `lizardscript` command shows `draw/shapes.liz: [line 3, column 5] runtime error: ...`, the module's path from the program's folder in front of each line), and the next use of that module tries again.
- **Cycles.** Modules can use each other: `a.liz` uses `b.liz`, which uses `a.liz` (the way Python allows it). A use of a module that is still loading doesn't run it again: it gets the module **as it is so far**, its live scope, so the names it defined before the use that started the cycle are there, and the ones further down appear as it goes on. Functions that call each other across the files work, since they look the names up when they run. The pitfall is a name used *while the cycle loads* (at the top level, or in a function called then) before its module defines it: that is `identifier not found: late: a.liz defines it further down, but it is still loading (use cycle: a.liz uses b.liz, which uses a.liz)`. A `use "a" as m` during the cycle gets the module's hash with the names so far, filled with the rest when `a.liz` has run. A file that uses itself already sees its names: the use does nothing. ([LEP 0002, Amendment 2](proposals/0002-modules.md#amendment-2-2026-11-03-use-cycles))
- Hosts: `evaluator.Use(node, env, run)` runs a use statement in `env`, a file's top scope; `evaluator.FindModule(node)` finds the file without running it; `evaluator.ReadModule` reads files (`lizardscript` reads a built program's own files with it, see [TERRARIUM.md](TERRARIUM.md#building-a-program)). The live names are `object.Environment.Use(module)`: lookups and assignments go through the scope's own names, then its modules', then the scope around it, and a write in a module changes `Version()` of the scopes that use it.

## The Terrarium

A program run by `lizardscript file.liz` (or `repl.Run`, on either engine) has the Terrarium, the standard library, installed: the names `fs`, `os`, `time`, `json`, `img`, `clip`, `io`, `lizard`, `window` and `pen` are modules, hashes of builtins used with dot access like `math`. The REPL has all of them but `window` and `pen`. Unlike `math`, `text`, `list` and `hash`, they are ordinary variables of the program's top scope, so `let fs = 1` replaces the module for the rest of the program. They are made new for each run. Their errors are runtime errors that start with the builtin's name (`fs.read: can't read notes.txt: no such file or directory`) and a `try` catches them. Images and windows are values of type `NATIVE`, equal only to themselves. Every function is in [TERRARIUM.md](TERRARIUM.md).

## Hooks (for hosts)

A host that embeds the interpreter can watch a program run. Both hooks are package variables of `evaluator`, `nil` by default, and cost one `nil` check per statement while they are `nil`. The REPL and `Run` never set them.

| Hook | Called |
| --- | --- |
| `evaluator.OnStatement func(node ast.Statement)` | Right before every statement is evaluated: in the program, in blocks (if, loops, functions), and in layer bodies. A layer statement itself is announced first, then its body's statements. `node.TokenLiteral()` and the statement's token tell where in the source it is. |
| `evaluator.OnLayer func(name string, enter bool, tok token.Token)` | With `enter` true when a layer's body starts, and with `enter` false when it ends, also when it ends by an error, a `return` or a `break`. `name` is the layer's name without quotes, and `tok` the `layer` word (its line and column). Layers nest, so exits come in the reverse order of the enters. |

A host that tags what a program makes keeps a stack of names in `OnLayer` and reads the current top whenever a builtin it provides is called.

`evaluator.HookFunctions` (`true` by default) says whether `OnStatement` is called for the statements of function bodies too. With `false` it is called only for the statements outside every function: the program's and a module's, with their blocks, loops and layers, but not the ones that run inside a call (a host that tags a shape with the line of the statement that drew it wants the line of the call, not a line of the helper function). The run state keeps it (see *Run state*), and `lizard.run`'s option `hook_functions` sets it.

### Source names (for hosts)

A host that loads several sources (libraries and the user's program) into one environment can name each one: `lexer.NewNamed(input, name)` is `lexer.New(input)` whose tokens all carry `Source: name` (`lexer.New` leaves it `""`). The name is copied into the tree, so the token of any statement a hook receives (`LetStatement.Token.Source`, and so on) tells which source it came from, even when a function defined in `"lib.liz"` runs because of a call in `"user"`. A runtime error gets the same in `object.Error.Source`, next to `Line` and `Column`: those are positions inside that source. Names never appear in messages, so output is the same with or without them. The name is also where a `use` looks for modules, and `repl.RunFile(name, ...)` (what `lizardscript file.liz` calls) names the program with its path; only an error from another source than the program, a module's, is shown with that source in front.

### Run state (for hosts)

A host that runs a program inside another one, like a sandbox, keeps the outer program's state with `state := evaluator.SaveRunState()` (where puts writes, the step limit and count, the hooks and `HookFunctions`, the random numbers, the libraries, the modules and the module prelude), starts the inner run with `evaluator.StartRun()` (no steps, the default seed, new libraries, no modules), and puts it all back with `state.Restore()`. `state.Outside(f)` runs `f` as the outer program in the middle of the inner run, for hooks. The Terrarium's `lizard.run` is built this way.

## Errors

A runtime error stops the program. Its message has the position of the operator, name or call where it happened:

```
[line 2, column 8] runtime error: type mismatch: INTEGER + BOOLEAN
```

A `try` statement can catch it and read the message, line and column (see Control flow). `object.Error` also has `Line` and `Column` fields with the same position, so a host can tell where an error was raised (for example by `error(...)` inside a library function) and show it at another line, and an `Uncatchable` flag, true for the step limit and stack overflow errors, which a `try` lets through.

Syntax errors are found before the program starts, so a program with one doesn't run at all. All of them are reported at once, one for each mistake (see [GRAMMAR.md](GRAMMAR.md#syntax-errors)).

### Errors at the caller

`error(message, {"at": "caller"})` is for a library's checks of its arguments: the mistake is in the program that called the library, so the error is reported at the program's call, not at the line of the library that found it.

```
-~> checks.liz
function half(v) {
  if (type(v) != "INTEGER") { error("half: want a number", {"at": "caller"}) }
  return v / 2
}

-~> main.liz
use "checks"
half("four")       -~> [line 2, column 5] runtime error: half: want a number
```

The error keeps its own position while it goes out of calls made from the file that raised it (a helper of the library that does the checks), and takes the position and source name of the first call made from another file: the call into that file. The standard modules count as one file for this, since a standard library is often several files that call each other, so a check inside `flower/lily2d` is reported at the program's call of `lily2d`. A `try` inside the raising file sees the error where it was raised; one in the program sees it moved. In a program of one file it stays where it was raised. Both engines move it the same way (hosts: `object.Error.AtCaller`, `evaluator.AtCaller`).

## Known limitations

- By default an infinite loop runs forever. In the REPL, stop it with Ctrl+C. A host that embeds the interpreter can limit it, see below.

## Step limit (for hosts)

`evaluator.MaxSteps` is the most steps a run can take. Its default, 0, means no limit. A step is one round of a `while` or `for` loop, or one function call (builtins included). The host sets `MaxSteps` and calls `evaluator.ResetSteps()` before each run; the REPL and `Run` reset it themselves. Steps are only counted while `MaxSteps` is above 0. Past the limit the program stops with a runtime error at the loop or call:

```
[line 1, column 1] runtime error: the program ran too long (more than 1000 steps)
```

A host that runs one program from inside another (a user's drawing from inside its own UI code) reads and restores the count: `evaluator.Steps()` returns the steps counted since the last reset, and `evaluator.SetSteps(n)` sets the count to `n`. Save `Steps()`, call `ResetSteps()` for the inner run, then `SetSteps(saved)` so the outer program carries on from where it was, still under the same limit.
