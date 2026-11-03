# 2. Values and types

[← Previous: First steps](01-first-steps.md) · [Index](../README.md) · [Next: Variables and scope →](03-variables-and-scope.md) · [Español](../es/02-values-and-types.md)

Everything a program works with is a **value**, and every value has a **type**. lizard-script is *dynamically typed*: you never write types down, but each value knows its own, and the builtin `type(x)` tells you which one it is.

## The types at a glance

```
puts(type(42))
puts(type(3.5))
puts(type("Liz"))
puts(type(true))
puts(type(null))
puts(type([1, 2, 3]))
puts(type({"name": "Liz"}))
puts(type(function(x) { x }))
puts(type(len))
puts(type(#3fa34d))
puts(type((10, 20)))
```

**Output:**

```text
INTEGER
FLOAT
STRING
BOOLEAN
NULL
ARRAY
HASH
FUNCTION
BUILTIN
COLOR
POINT
```

| Type | Examples | Chapter |
| --- | --- | --- |
| `INTEGER` | `0`, `42`, `-7` | 4, 11 |
| `FLOAT` | `3.5`, `0.1`, `45deg` | 11 |
| `STRING` | `"Liz"`, `""` | 10 |
| `BOOLEAN` | `true`, `false` | 4, 5 |
| `NULL` | `null` | 14 |
| `ARRAY` | `[1, "a", true]` | 8 |
| `HASH` | `{"name": "Liz"}` | 9 |
| `FUNCTION` | `function(x) { x * 2 }` | 7 |
| `BUILTIN` | `puts`, `len` | 15 |
| `COLOR` | `#3fa34d` | 12 |
| `POINT` | `(10, 20)` | 12 |

## Numbers

There are two kinds of numbers. **Integers** have no decimal point; **floats** do. The difference matters, especially for division:

```
puts(7 / 2)
puts(7.0 / 2)
puts(type(7), type(7.0))
```

**Output:**

```text
3
3.5
INTEGER FLOAT
```

Two integers always give an integer. Chapter 4 and chapter 11 go into the details.

## Strings

Text goes between **double quotes**. Single quotes are not part of the language.

```
puts("Hello")
puts("")
puts(len("lizard"))
```

**Output:**

```text
Hello

6
```

(The second line printed an empty string, so it looks like an empty line.)

## Booleans and `null`

`true` and `false` are the two booleans. They come out of comparisons:

```
puts(3 > 2, 3 == 4)
```

**Output:**

```text
true false
```

`null` means "no value". Many things give `null`: a function that returns nothing, reading an index that doesn't exist, `puts` itself.

```
let nothing = null
puts(nothing, type(nothing))
puts([1, 2, 3][10])
```

**Output:**

```text
null NULL
null
```

## Truthiness

Conditions (`if`, `while`, `&&`, `||`, `!`) accept **any** value, not just booleans. These count as false:

`false`, `null`, `0`, `0.0`, `""`, `[]`, `{}`

Everything else counts as true. `!x` turns any value into the opposite boolean, so `!!x` tells you how a value counts:

```
puts(!!0, !!"", !![], !!{}, !!null)
puts(!!1, !!"a", !![0], !!-1)
```

**Output:**

```text
false false false false false
true true true true
```

## Converting between types

| Builtin | Turns into | Example |
| --- | --- | --- |
| `str(x)` | a string | `str(42)` is `"42"` |
| `int(x)` | an integer | `int("42")` is `42`, `int(2.9)` is `2` |
| `float(x)` | a float | `float(3)` is `3.0` |

```
puts(str(42) + "!")
puts(int("41") + 1)
puts(int(2.9), int(-1.9), int(true))
puts(float(3), float("2.5"))
```

**Output:**

```text
42!
42
2 -1 1
3.0 2.5
```

`int` only accepts a string that holds a whole number:

```
puts(int("1.5"))
```

**Output:**

```text
[line 1, column 9] runtime error: int: not a whole number: "1.5"
```

> **Common mistake: joining a string and a number.** `+` doesn't convert for you.
>
> ```
> let age = 3
> puts("Liz is " + age + " years old")
> ```
>
> **Output:**
>
> ```text
> [line 2, column 16] runtime error: type mismatch: STRING + INTEGER
> ```
>
> Use `str`, or let `puts` add the spaces:
>
> ```
> let age = 3
> puts("Liz is " + str(age) + " years old")
> puts("Liz is", age, "years old")
> ```
>
> **Output:**
>
> ```text
> Liz is 3 years old
> Liz is 3 years old
> ```

## How values are shown

`puts` and `str` write strings without quotes. Inside an array or a hash, strings keep their quotes so you can see where each one ends:

```
puts("Liz")
puts(["Liz", "Iggy"])
puts({"name": "Liz"})
```

**Output:**

```text
Liz
["Liz", "Iggy"]
{"name": "Liz"}
```

> **Try it.** Guess the type of each value, then check with `type`: `"5"`, `5`, `5.0`, `5 > 4`, `[5]`, `(5, 5)`.

## Summary

- Every value has a type; `type(x)` gives its name.
- Two integers make an integer: `7 / 2` is `3`. Write `7.0` to get `3.5`.
- `false`, `null`, `0`, `0.0`, `""`, `[]` and `{}` count as false.
- `str`, `int` and `float` convert. `+` never converts a number to a string for you.

[← Previous: First steps](01-first-steps.md) · [Index](../README.md) · [Next: Variables and scope →](03-variables-and-scope.md)
