# lizard-script semantics

What a program does when it runs. The syntax is in [GRAMMAR.md](GRAMMAR.md).

## Values

| Type | Examples | Notes |
| --- | --- | --- |
| `INTEGER` | `5`, `-12` | 64-bit signed. Arithmetic wraps around on overflow, like Go. |
| `BOOLEAN` | `true`, `false` | |
| `STRING` | `"Liz"` | Length and indexes count characters, not bytes. Strings can't be changed. |
| `NULL` | | What an `if` without `else`, a loop, `let`, `puts` and a missing index produce. It can't be written directly. |
| `ARRAY` | `[1, "a", true]` | |
| `HASH` | `{"name": "Liz", 1: true}` | Keys must be integers, booleans or strings. Keys keep the order they were added in. |
| `FUNCTION` | `function(x) { x }` | |
| `BUILTIN` | `len`, `puts` | Functions written in Go, see below. |

Arrays and hashes are shared, not copied. After `let b = a`, a change made through `b` shows through `a` too, and a function that receives an array can change it.

### Truthiness

Conditions accept any value. `false`, `null`, `0`, `""`, `[]` and `{}` count as false. Everything else counts as true.

## Variables and scopes

- `let x = ...` creates `x` in the current scope. If an outer scope already has an `x`, it is hidden, not changed. Writing `let` twice in the same scope replaces the old value.
- `x = ...` changes the closest `x` that already exists, so a function can change a variable of the scope it was created in. Assigning to a name that was never declared is an error.
- Each block (`{ ... }` of an `if`, a loop round or a function call) is a new scope. Variables created inside it are gone after it ends.
- The `let` in a `for` header lives in a scope of its own that wraps the whole loop. Each round of the body gets a fresh scope inside it.
- A name is looked up from the innermost scope outwards, and then among the builtins. That means a variable can hide a builtin: `let len = 5`.

## Operators

### Arithmetic

Only on integers, except where noted.

| Operator | Meaning |
| --- | --- |
| `+` `-` `*` | As usual. `+` also joins two strings or two arrays into a new one. `*` repeats a string: `"ab" * 3` is `"ababab"`. |
| `/` | Division, rounding towards zero: `-7 / 2` is `-3`. |
| `//` | Division, rounding down: `-7 // 2` is `-4`. |
| `%` | The remainder that goes with `//`, so it has the sign of the right side: `-7 % 3` is `2`. `a == (a // b) * b + a % b` always holds. |
| `**` | Power. A negative exponent is an error, because the result wouldn't be an integer. `0 ** 0` is `1`. |
| `-x` | Negation, only for integers. |

Dividing by zero, with any of `/`, `//` or `%`, is an error.

### Comparison

`<`, `>`, `<=` and `>=` compare two integers, or two strings in dictionary order (byte by byte). Comparing different types is an error.

### Equality

- `===` is exact: the two values have the same type and the same value. `!==` is its opposite.
- `==` is loose: it works like `===`, but values of different types are first converted to numbers when possible. `true` is `1`, `false` is `0`, and a string that is a whole number is that number. So `1 == true` and `"5" == 5`, but `1 !== true`. `!=` is its opposite.
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
- More than 10000 nested calls is a stack overflow error.

## Indexing

- `array[i]` and `string[i]` take an integer, starting at 0. An index outside the array or string gives `null`.
- `hash[key]` gives `null` when the key isn't there. The key must be a valid key type. Keys are compared exactly, so `1` and `"1"` are different keys.
- `array[i] = x` changes an element that already exists. An index outside of the array is an error, so use `push` to add elements.
- `hash[key] = x` adds or replaces a key.

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
| `int(x)` | A string with a whole number, a boolean or an integer, as an integer. |
| `range(end)`, `range(start, end)` | `[start, start + 1, ..., end - 1]`, with `start` 0 by default. Empty if `end <= start`. |

## Errors

A runtime error stops the program. Its message has the position of the operator, name or call where it happened:

```
[line 2, column 8] runtime error: type mismatch: INTEGER + BOOLEAN
```

Syntax errors are found before the program starts, so a program with one doesn't run at all.

## Known limitations

- An array or hash that contains itself, like after `push(a, a)`, can't be shown or compared: trying it crashes the interpreter.
- An infinite loop runs forever. In the REPL, stop it with Ctrl+C.
