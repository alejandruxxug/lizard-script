# 11. Numbers: floats, `math` and units

[← Previous: Strings and `text`](10-strings-and-text.md) · [Index](../README.md) · [Next: Colors and points →](12-colors-and-points.md) · [Español](../es/11-numbers-and-math.md)

This chapter goes deeper into numbers: the two kinds, how floats behave, the `math` library and the units `deg`, `rad`, `turn` and `px`.

## Integers and floats

An **integer** (`INTEGER`) is a whole number. A **float** (`FLOAT`) has a decimal point. A float needs digits on both sides of the point: `0.5`, not `.5`; `2.0`, not `2.`.

```
puts(type(2), type(2.0), type(0.5))
puts(2 == 2.0, 2 === 2.0)
```

**Output:**

```text
INTEGER FLOAT FLOAT
true false
```

The rules for mixing them:

- integer with integer gives an integer: `7 / 2` is `3`;
- if either side is a float, the result is a float: `7 / 2.0` is `3.5`.

```
puts(7 / 2, 7 / 2.0, 7 // 2.0, 7.5 % 2)
puts(3 * 2, 3 * 2.0)
```

**Output:**

```text
3 3.5 3.0 1.5
6 6.0
```

A float is always printed with a `.` (or an exponent), so you can tell it from an integer at a glance: `6.0`, not `6`.

## Floats are approximate

Floats are stored in binary, so many decimal numbers can't be represented exactly. The classic example:

```
puts(0.1 + 0.2)
puts(0.1 + 0.2 == 0.3)
```

**Output:**

```text
0.30000000000000004
false
```

This is not a lizard-script bug; every language with floats does it. To compare floats, check that they are close enough:

```
let a = 0.1 + 0.2
puts(math.abs(a - 0.3) < 0.000001)
```

**Output:**

```text
true
```

Very large and very small floats are printed with an exponent:

```
puts(1000000.0 * 1000000000000000.0, 0.0000001)
```

**Output:**

```text
1e+21 1e-07
```

You can also *write* an exponent: `1e3`, `2.5e-4` and `1E+6` are number literals. A number with an exponent is always a float, even when its value is whole:

```
puts(1e3, 2.5e-4, 1E+6)
puts(type(1e3), 1e3 == 1000)
puts(1e-7, 1.5e21)
```

**Output:**

```text
1000.0 0.00025 1000000.0
FLOAT true
1e-07 1.5e+21
```

The `e` must be followed by digits (with an optional `+` or `-` before them). A number too big for a float is an error:

```
puts(1e400)
```

**Output:**

```text
[line 1, column 6] syntax error: number 1e400 is too large
```

## Converting

```
puts(int(3.99), int(-3.99))
puts(float(3), float("0.25"))
puts(math.round(3.5), math.floor(3.99), math.ceil(3.01))
```

**Output:**

```text
3 -3
3.0 0.25
4 3 4
```

`int` cuts the decimals towards zero; `math.round`, `math.floor` and `math.ceil` round in different ways and give integers.

## Limits

Integers are 64-bit. Past the largest one, they wrap around to the most negative, as in Go:

```
puts(9223372036854775807 + 1)
```

**Output:**

```text
-9223372036854775808
```

Floats never become infinite or "not a number". An operation that would give one is an error:

```
puts(10.0 ** 400)
```

**Output:**

```text
[line 1, column 11] runtime error: the result is not a number
```

## The `math` library

`math` holds constants and functions, used with a dot. Angles are in **radians**.

| Name | Gives |
| --- | --- |
| `math.pi`, `math.tau` | π and 2π |
| `math.sqrt(x)` | square root |
| `math.abs(x)` | without the sign |
| `math.floor(x)`, `math.ceil(x)`, `math.round(x)` | integers, rounded down, up, or to the nearest |
| `math.min(a, b, ...)`, `math.max(a, b, ...)` | smallest or largest of two or more |
| `math.sin(x)`, `math.cos(x)`, `math.tan(x)` | trigonometry |
| `math.asin(x)`, `math.acos(x)`, `math.atan2(y, x)` | inverse trigonometry |
| `math.random()`, `math.random(a, b)` | random numbers |
| `math.seed(n)` | restart the random numbers |
| `math.dist(p, q)`, `math.angle(p, q)` | distance and angle between points (chapter 12) |

```
puts(math.pi, math.sqrt(16), math.abs(-7))
puts(math.min(3, 1.5, 8), math.max(3, 1.5, 8))
puts(math.round(2.5), math.round(-2.5), math.floor(-1.5))
```

**Output:**

```text
3.141592653589793 4.0 7
1.5 8
3 -3 -2
```

`math.round` rounds halves away from zero. `math.min` and `math.max` give back the argument itself, so `math.min(1, 2.5)` is the integer `1`.

Some inputs make no sense and are errors:

```
puts(math.sqrt(-1))
```

**Output:**

```text
[line 1, column 15] runtime error: math.sqrt: can't take the square root of a negative number: -1
```

## Units: `deg`, `rad`, `turn`, `px`

Thinking in radians is hard, so lizard-script lets you write a unit right after a number, **without a space**:

| Unit | Means | Example |
| --- | --- | --- |
| `deg` | degrees, converted to radians | `90deg` is `1.5707...` |
| `rad` | radians, as they are | `1.5rad` is `1.5` |
| `turn` | full turns, converted to radians | `0.25turn` is `90deg` |
| `px` | pixels: the number stays as it is | `10px` is `10` |

```
puts(180deg, 180deg == math.pi)
puts(0.25turn == 90deg)
puts(math.sin(90deg), math.cos(0deg))
puts(10px, type(10px), 10.5px)
```

**Output:**

```text
3.141592653589793 true
true
1.0 1.0
10 INTEGER 10.5
```

To turn radians back into degrees, divide by `1deg`:

```
let angle = math.atan2(1, 1)
puts(angle, angle / 1deg)
```

**Output:**

```text
0.7853981633974483 45.0
```

Only these four words are units. Anything else right after a number is an error:

```
let weight = 5kg
```

**Output:**

```text
[line 1, column 14] syntax error: unknown unit "kg"
```

It is reported the same way inside a call, an array or a hash:

```
puts(5kg)
```

**Output:**

```text
[line 1, column 6] syntax error: unknown unit "kg"
```

## Random numbers

`math.random()` gives a float from 0 up to (not including) 1; `math.random(a, b)` an integer from `a` to `b`, both included.

lizard-script's random numbers are **repeatable**: every run starts from the same seed, so the same program prints the same numbers every time. That makes programs easy to test. `math.seed(n)` picks a different sequence; the same `n` always gives the same one.

```
math.seed(7)
let first = [math.random(1, 6), math.random(1, 6), math.random(1, 6)]
math.seed(7)
let again = [math.random(1, 6), math.random(1, 6), math.random(1, 6)]
puts(first, again, first == again)
```

**Output:**

```text
[2, 1, 1] [2, 1, 1] true
```

## Example: points on a circle

```
let radius = 100px
for (let i = 0; i < 4; i += 1) {
  let angle = i * 90deg
  let x = math.round(radius * math.cos(angle))
  let y = math.round(radius * math.sin(angle))
  puts(i * 90, "degrees ->", x, y)
}
```

**Output:**

```text
0 degrees -> 100 0
90 degrees -> 0 100
180 degrees -> -100 0
270 degrees -> 0 -100
```

> **Try it.** Write `hypotenuse(a, b)` with `math.sqrt`. Then write `toFahrenheit(c)` that gives a float (`c * 9 / 5 + 32`). What happens with `toFahrenheit(37)` if you forget the float?

## Summary

- `7 / 2` is `3`; `7 / 2.0` is `3.5`. A float is printed with a `.`.
- Floats are approximate: compare with a small tolerance.
- `1e3`, `2.5e-4` are float literals; no Infinity or NaN.
- `math.*` for roots, rounding, trigonometry and random numbers.
- `45deg`, `0.5turn`, `1.2rad` are radians; `10px` is just `10`.
- Random numbers repeat from run to run; `math.seed(n)` changes them.

[← Previous: Strings and `text`](10-strings-and-text.md) · [Index](../README.md) · [Next: Colors and points →](12-colors-and-points.md)
