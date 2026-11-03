# lizard-script semantics

What a program does when it runs. The syntax is in [GRAMMAR.md](GRAMMAR.md).

## Values

| Type | Examples | Notes |
| --- | --- | --- |
| `INTEGER` | `5`, `-12` | 64-bit signed. Arithmetic wraps around on overflow, like Go. |
| `FLOAT` | `1.5`, `-0.25`, `45deg` | 64-bit decimal number. Never infinite or NaN. |
| `BOOLEAN` | `true`, `false` | |
| `STRING` | `"Liz"` | Length and indexes count characters, not bytes. Strings can't be changed. |
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

### Printing floats

`puts` and `str` write the shortest text that reads back as the same float, and always with a `.` or an exponent, so a float never looks like an integer: `1.0`, `0.5`, `0.1`, `0.30000000000000004`. Numbers from `1e21` up and below `0.000001` use an exponent: `1e+21`, `1e-07`.

### Comparison

`<`, `>`, `<=` and `>=` compare two numbers by value (an integer and a float work too: `1.5 < 2`), or two strings in dictionary order (byte by byte). Comparing different types is an error.

### Equality

- `===` is exact: the two values have the same type and the same value. `!==` is its opposite.
- `==` is loose: it works like `===`, but values of different types are first converted to numbers when possible. `true` is `1`, `false` is `0`, and a string that is a whole number is that number. So `1 == true` and `"5" == 5`, but `1 !== true`. Integers and floats are compared by value: `1 == 1.0` is true, and a string with a number such as `"1.5"` equals the float `1.5`. Only plain decimal strings count: `"1e3"`, `"inf"` and `"1."` are not numbers. `===` also needs the same type, so `1 === 1.0` is false. `!=` is its opposite.
- Values that can't be converted are never `==` to a value of another type: `"five" == 5` is `false`, and `null == 0` is `false`.
- Arrays and hashes are equal when their contents are, using the same kind of equality for the elements: `[1, "2"] == [1, 2]` but `[1, "2"] !== [1, 2]`.
- Functions and builtins are only equal to themselves.

Equality never fails, whatever the types.

### Logic

`&&` and `||` stop early: the right side only runs when the left one doesn't decide the result. `!x` is `true` when `x` counts as false. The three of them always produce `true` or `false`.

## Control flow

- `if` produces the value of the block that ran, or `null` if none did.
- `while` and `for` produce `null`. `break` leaves the innermost loop.
- A block, and a function body, produces the value of its last statement.

## Functions

- Calling a function with a different number of arguments than it has parameters is an error.
- A function remembers the scope it was created in (it is a closure), and sees the changes made to it later.
- A function without `return` gives the value of its last statement. A bare `return` gives `null`.
- `return` outside of a function ends the program. Its value is the value of the program, which the REPL shows.
- More than 10000 nested calls is a stack overflow error. So is a smaller number of calls whose bodies are nested deeply enough that, together, they would use too much of the interpreter's memory: `stack overflow: the calls and the code inside them nest too deeply`.

## Indexing

- `array[i]` and `string[i]` take an integer, starting at 0. An index outside the array or string gives `null`.
- `hash[key]` gives `null` when the key isn't there. The key must be a valid key type. Keys are compared exactly, so `1` and `"1"` are different keys.
- `array[i] = x` changes an element that already exists. An index outside of the array is an error, so use `push` to add elements.
- `hash[key] = x` adds or replaces a key.
- `a.name` is exactly `a["name"]`, for reading and for assigning, so on a hash it gives the value or `null`, and on anything else it gives the same error as indexing it with a string.

## Units

A number directly followed by `deg`, `rad`, `turn` or `px` is read by the lexer: `45deg` is the float `0.785...` (radians), `1.5rad` is `1.5`, `0.25turn` is `0.25 * 2π`, and `10px` is just `10` (an integer stays an integer, `10.5px` is a float). Letters right after a number that are not exactly one of these words are not a unit: `5kg` is the number `5` and the name `kg`, which is a syntax error (`unknown unit "kg"`).

## Strings

In a string literal, `\"` is `"`, `\\` is `\`, `\n` is a line break and `\t` is a tab. The value of the string holds the character, not the two source characters, so `len("a\nb")` is 3.

## Builtins

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
| `type(x)` | The name of the type of `x`, like `"INTEGER"`. |
| `str(x)` | `x` as a string, the way `puts` shows it. |
| `int(x)` | A string with a whole number, a boolean, an integer or a float as an integer. A float loses its decimals towards zero: `int(-1.9)` is `-1`. `int("1.5")` is an error. |
| `float(x)` | An integer, a float or a string with a number (`"2"`, `"-0.25"`) as a float. |
| `range(end)`, `range(start, end)` | `[start, start + 1, ..., end - 1]`, with `start` 0 by default. Empty if `end <= start`. |

## The math library

`math` is a hash of builtins and constants, used with dot access. Angles are in radians; the units `deg`, `rad` and `turn` make them easy to write. Every function takes integers or floats, and a wrong type or number of arguments is an error like in any builtin.

| Name | Does |
| --- | --- |
| `math.pi`, `math.tau` | π and 2π, as floats. |
| `math.sin(x)`, `math.cos(x)`, `math.tan(x)` | As a float. |
| `math.atan2(y, x)` | The angle of the point `(x, y)`, as a float. |
| `math.sqrt(x)` | The square root, as a float. A negative `x` is an error. |
| `math.abs(x)` | Without the sign. An integer for an integer, a float for a float. |
| `math.floor(x)`, `math.ceil(x)`, `math.round(x)` | An integer. `round` goes away from zero on halves: `math.round(2.5)` is `3`, `math.round(-2.5)` is `-3`. An integer is returned as it is, and a float too big for an integer is an error. |
| `math.min(a, b, ...)`, `math.max(a, b, ...)` | The smallest or largest of at least two numbers, compared by value. The result is that argument itself, so `math.min(1, 2.5)` is the integer `1`; on a tie the first one. |
| `math.random()` | A float from 0 up to but not including 1. |
| `math.random(a, b)` | An integer from `a` to `b`, both included. An error if `b < a`. |
| `math.seed(n)` | Restarts `random` from the integer `n`. Returns `null`. |

`random` is deterministic: every run starts from the same default seed, so the same program makes the same numbers (and the same picture). `math.seed(n)` picks another sequence, and the same `n` always gives the same one. A host that runs programs one after another calls `evaluator.ResetMath()` before each run; the REPL and `Run` do it themselves, once per session or file.

`math` is found after variables and the other builtins, so `let math = 1` hides it. The hash is made new for each run: a program can write `math.pi = 3`, but only that run sees it.

## Errors

A runtime error stops the program. Its message has the position of the operator, name or call where it happened:

```
[line 2, column 8] runtime error: type mismatch: INTEGER + BOOLEAN
```

Syntax errors are found before the program starts, so a program with one doesn't run at all.

## Known limitations

- By default an infinite loop runs forever. In the REPL, stop it with Ctrl+C. A host that embeds the interpreter can limit it, see below.

## Step limit (for hosts)

`evaluator.MaxSteps` is the most steps a run can take. Its default, 0, means no limit. A step is one round of a `while` or `for` loop, or one function call (builtins included). The host sets `MaxSteps` and calls `evaluator.ResetSteps()` before each run; the REPL and `Run` reset it themselves. Past the limit the program stops with a runtime error at the loop or call:

```
[line 1, column 1] runtime error: the program ran too long (more than 1000 steps)
```
