# 14. Errors, `try`/`catch` and `null`

[← Previous: Layers](13-layers.md) · [Index](../README.md) · [Next: Builtins reference →](15-builtins-reference.md) · [Español](../es/14-errors.md)

Programs go wrong. This chapter is about understanding error messages, raising your own errors with `error()`, recovering from them with `try`/`catch`, and using `null` for "no value".

## The three kinds of error

Every message has the same shape: `[line L, column C] kind error: message`.

**Lexical errors** are characters the language doesn't know. They are found before anything runs:

```
let price = 10 @ 2
```

**Output:**

```text
[line 1, column 16] lexical error: unexpected character "@"
```

**Syntax errors** are pieces in the wrong order. They are also found before anything runs, and the parser reports all of them at once: one error for each mistake, while the lines after a mistake are still checked:

```
puts("this line never runs")
let = 5
let total = (1 + 2
if (true) { break }
```

**Output:**

```text
[line 2, column 5] syntax error: let needs a name before the "=", like let x = 1
[line 3, column 19] syntax error: the "(" at line 3, column 13 is never closed
[line 4, column 13] syntax error: break outside of a loop
```

A missing `)`, `]` or `}` is reported right where it is missing, and the message says where the bracket was opened. A line gets at most one error: a mistake often breaks the rest of its line, so a second mistake on the same line shows up once the first is fixed.

### Messages that help

Many syntax errors say what you probably meant:

```
fucntion double(x) {
  x * 2
}
let n = 3
if (n = 3) {
  puts("three")
} elseif (n > 3) {
  puts("more")
}
let pet = {"name" = "Liz"}
puts("unclosed"
let age = 2
```

**Output:**

```text
[line 1, column 1] syntax error: "fucntion" is not a keyword; did you mean "function"?
[line 5, column 7] syntax error: "=" in a condition gives n a value; did you mean "=="?
[line 7, column 3] syntax error: "elseif" is not a keyword; did you mean "elif"?
[line 10, column 19] syntax error: a hash puts ":" between a key and its value, not "="
[line 11, column 16] syntax error: the "(" at line 11, column 5 is never closed
```

| Message starts with | Usually means |
| --- | --- |
| `"x" is not a keyword; did you mean ...` | a misspelled keyword, or one from another language (`elseif`, `var`, `def`) |
| `"=" in a condition gives x a value` | `=` where `==` was meant: `if (x == 1)` |
| `the "(" at line L, column C is never closed` | a missing `)` (or `]`, `}`) for the bracket at that place |
| `block is never closed` | a missing `}`; the position is the `{` that lacks it |
| `this "}" has nothing to close` | one `}` (or `)`, `]`) too many |
| `let needs a name before the "="` | `let = 5` instead of `let x = 5` |
| `"if" is a reserved word, it can't be a name` | a keyword used as a variable or parameter name |
| `a hash puts ":" between a key and its value` | `{"a" = 1}` instead of `{"a": 1}` |
| `a let gives its value with "=", not ":"` | `let x: 1` instead of `let x = 1` |
| `unterminated string: this " is never closed` | a string without its closing `"`; the position is where it starts |

**Runtime errors** happen while the program runs. Everything before them has already happened:

```
let toes = 20
puts("toes per foot:", toes / 4)
puts("toes per wing:", toes / 0)
puts("this line never runs")
```

**Output:**

```text
toes per foot: 5
[line 3, column 29] runtime error: division by zero
```

The column points at the operator, name or call that failed. Here, column 29 is the `/` of `toes / 0`.

### Common runtime errors

| Message starts with | Usually means |
| --- | --- |
| `identifier not found: x` | a typo, or a variable used outside its scope |
| `can't assign to x: it was never declared` | `x = ...` without `let x` first |
| `type mismatch: STRING + INTEGER` | mixing types; convert with `str`, `int`, `float` |
| `wrong number of arguments` | a call with too many or too few arguments |
| `not a function: INTEGER` | calling something that isn't a function |
| `index out of range` | `a[i] = x` past the end; use `push` |
| `division by zero` | `/`, `//` or `%` by `0` or `0.0` |
| `stack overflow` | recursion without a base case |

## Raising your own errors: `error()`

`error(message)` stops the program with your message, at the position of the call. Use it when a function gets an input it can't handle:

```
function divide(a, b) {
  if (b == 0) {
    error("can't divide " + str(a) + " by zero")
  }
  a / b
}
puts(divide(10, 2))
puts(divide(1, 0))
```

**Output:**

```text
5
[line 3, column 10] runtime error: can't divide 1 by zero
```

## Recovering: `try` / `catch`

`try { ... } catch (e) { ... }` runs the first block. If a runtime error happens inside it (also deep inside a function it called), the rest of the block is skipped and the `catch` block runs instead:

```
function divide(a, b) {
  if (b == 0) { error("can't divide by zero") }
  a / b
}
try {
  puts(divide(10, 2))
  puts(divide(1, 0))
  puts("not reached")
} catch (e) {
  puts("caught:", e.message)
}
puts("the program goes on")
```

**Output:**

```text
5
caught: can't divide by zero
the program goes on
```

`e` is a hash with the `message`, and the `line` and `column` where the error happened:

```
try {
  let x = nope
} catch (e) {
  puts(e)
  puts("line", e.line, "column", e.column)
}
```

**Output:**

```text
{"message": "identifier not found: nope", "line": 2, "column": 11}
line 2 column 11
```

The name is optional: `catch { ... }` when you don't need the details.

### `try` as the last statement of a function

`try` is a **statement**, not an expression, so `let x = try { ... }` doesn't work:

```
let n = try { int("12") } catch { 0 }
```

**Output:**

```text
[line 1, column 13] syntax error: expected end of statement, got "{"
```

But a `try` has the value of the block that ran, so as the last statement of a function it becomes the function's result. That's the usual pattern:

```
function toNumber(s, fallback = 0) {
  try { int(s) } catch { fallback }
}
puts(toNumber("12"), toNumber("twelve"), toNumber("twelve", -1))
```

**Output:**

```text
12 0 -1
```

### Errors inside `catch`

An error inside the `catch` block isn't caught by the same `try`, but an outer `try` catches it. That's how you add information and pass an error on:

```
function load(name) {
  try {
    error("file missing")
  } catch (e) {
    error("couldn't load " + name + ": " + e.message)
  }
}
try {
  load("song.txt")
} catch (e) {
  puts(e.message)
}
```

**Output:**

```text
couldn't load song.txt: file missing
```

### What `try` can't catch

A stack overflow (and the step limit some hosts set) is never caught, so a broken program can't hide it:

```
function forever(n) { forever(n + 1) }
try {
  forever(0)
} catch (e) {
  puts("caught it")
}
```

**Output:**

```text
[line 1, column 30] runtime error: stack overflow: more than 10000 nested calls
```

Syntax errors aren't caught either: they stop the program before it starts.

### In the REPL

In the REPL, a `try` block runs as soon as its line is complete, so write `} catch (e) {` on the **same** line as the closing `}` of the `try`.

## `null`: "no value"

`null` is the value for "nothing here". You get it from a missing index or key, an `if` without `else`, a function without a result, and you can write it yourself:

```
let h = {"a": 1}
puts(h.b, [1, 2][9], type(null))
puts(null == null, null == 0, null == false)
if (null) { puts("never") } else { puts("null counts as false") }
```

**Output:**

```text
null null NULL
true false false
null counts as false
```

### `null` or `error`?

Two ways for a function to say "I couldn't do it":

- **return `null`** when "not found" is a normal answer the caller should check, like `text.number("abc")` or looking up a missing key;
- **call `error`** when it's a real mistake that shouldn't go unnoticed, like dividing by zero.

```
function findPet(pets, name) {
  for (let i = 0; i < len(pets); i += 1) {
    if (pets[i].name == name) { return pets[i] }
  }
  null
}
let pets = [{"name": "Liz", "age": 3}]
let found = findPet(pets, "Rex")
if (found === null) {
  puts("no pet called Rex")
}
```

**Output:**

```text
no pet called Rex
```

> **Common mistake: using a `null` result.** If you forget to check, the error appears later, far from the cause:
>
> ```
> let pets = {"Liz": 3}
> let age = pets["Rex"]
> puts(age + 1)
> ```
>
> **Output:**
>
> ```text
> [line 3, column 10] runtime error: type mismatch: NULL + INTEGER
> ```
>
> `NULL + INTEGER` almost always means "something I expected to exist wasn't there".

> **Try it.** Write `safeDivide(a, b)` that returns `a / b`, or the string `"undefined"` when `b` is `0`, using `try`/`catch` (not an `if`).

## Summary

- Lexical and syntax errors stop the program before it starts; runtime errors stop it where they happen.
- `error(message)` raises your own runtime error.
- `try { } catch (e) { }` recovers; `e.message`, `e.line`, `e.column`.
- `try` is a statement; use it as the last statement of a function to get a value.
- Stack overflows can't be caught.
- `null` means "no value": check for it with `=== null`.

[← Previous: Layers](13-layers.md) · [Index](../README.md) · [Next: Builtins reference →](15-builtins-reference.md)
