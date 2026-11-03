# 12. Colors and points

[← Previous: Numbers, `math` and units](11-numbers-and-math.md) · [Index](../README.md) · [Next: Layers →](13-layers.md) · [Español](../es/12-colors-and-points.md)

lizard-script has two values made for drawing-style programs: **colors** and **points**. They are ordinary values, so you can store them in variables, arrays and hashes, and pass them to functions.

## Colors

A color is written like in CSS or a design tool: `#` and then 3, 6 or 8 hexadecimal digits.

| Form | Example | Means |
| --- | --- | --- |
| `#rgb` | `#f80` | short form, each digit doubled: `#ff8800` |
| `#rrggbb` | `#3fa34d` | red, green, blue from `00` to `ff` |
| `#rrggbbaa` | `#00000080` | plus alpha (opacity): `00` is invisible, `ff` is solid |

```
let leaf = #3fa34d
puts(leaf, type(leaf))
puts(#f80, #3FA34D, #00000080)
```

**Output:**

```text
#3fa34d COLOR
#ff8800 #3fa34d #00000080
```

Colors are always printed in lowercase, and the alpha is only shown when it isn't `ff`.

### Reading the channels

`c.r`, `c.g`, `c.b` and `c.a` give the four channels, as integers from 0 to 255:

```
let leaf = #3fa34d
puts(leaf.r, leaf.g, leaf.b, leaf.a)
```

**Output:**

```text
63 163 77 255
```

### Making colors from numbers

| Builtin | Makes |
| --- | --- |
| `rgb(r, g, b)` | a color from red, green and blue, 0 to 255 |
| `rgba(r, g, b, a)` | the same, with alpha |
| `hsl(h, s, l)` | a color from a hue in degrees (0 to 360), saturation and lightness in percent (0 to 100) |

```
puts(rgb(255, 128, 0))
puts(rgba(0, 0, 0, 128))
puts(hsl(0, 100, 50), hsl(120, 100, 25), hsl(210, 50, 40))
```

**Output:**

```text
#ff8000
#00000080
#ff0000 #008000 #336699
```

`hsl` is handy for making a range of related colors: keep `s` and `l` and change the hue.

```
for (let i = 0; i < 6; i += 1) {
  puts(hsl(i * 60, 80, 50))
}
```

**Output:**

```text
#e61919
#e6e619
#19e619
#19e6e6
#1919e6
#e619e6
```

Values outside the range are an error:

```
puts(rgb(300, 0, 0))
```

**Output:**

```text
[line 1, column 9] runtime error: rgb: red must be from 0 to 255, got 300
```

### Colors can't be changed

Like strings, colors are fixed. To get a variation, make a new color:

```
let leaf = #3fa34d
leaf.a = 100
```

**Output:**

```text
[line 2, column 5] runtime error: colors can't be changed
```

```
function withAlpha(c, alpha) {
  rgba(c.r, c.g, c.b, alpha)
}
puts(withAlpha(#3fa34d, 100))
```

**Output:**

```text
#3fa34d64
```

There is no arithmetic on colors (`#fff + #000` is an error). `==` compares the channels, and a color is never equal to a string:

```
puts(#fff == #ffffff, #fff == "#ffffff")
```

**Output:**

```text
true false
```

## Points

A point is two or three numbers between parentheses: `(x, y)` or `(x, y, z)`. The comma is what makes it a point; `(5)` is just the number `5` in parentheses.

```
let a = (10, 20)
let b = (40, 60)
puts(a, type(a))
puts(a.x, a.y)
puts((1, 2, 3).z)
puts(point(3, 4))
```

**Output:**

```text
(10, 20) POINT
10 20
3
(3, 4)
```

Each coordinate keeps its own type, so `(0.5, 2)` has a float `x` and an integer `y`.

### Arithmetic on points

| Expression | Gives |
| --- | --- |
| `p + q`, `p - q` | coordinate by coordinate |
| `p * n`, `n * p`, `p / n` | every coordinate times or divided by the number `n` |
| `-p` | every coordinate negated |

```
let a = (10, 20)
let b = (40, 60)
puts(a + b, b - a)
puts(a * 2, b / 2, -a)
puts((3, 4) / 2, (3, 4) / 2.0)
```

**Output:**

```text
(50, 80) (30, 40)
(20, 40) (20, 30) (-10, -20)
(1, 2) (1.5, 2.0)
```

The usual number rules apply to each coordinate: `(3, 4) / 2` rounds like `3 / 2`. Points with a different number of coordinates can't be mixed, and a point plus a number makes no sense:

```
puts((1, 2) + (1, 2, 3))
```

**Output:**

```text
[line 1, column 13] runtime error: can't mix points of 2 and 3 coordinates: POINT + POINT
```

```
puts((1, 2) + 1)
```

**Output:**

```text
[line 1, column 13] runtime error: type mismatch: POINT + INTEGER
```

Points can't be changed either (`p.x = 3` is the error `points can't be changed`); make a new one: `(p.x + 3, p.y)`.

### Distance and angle

```
let a = (0, 0)
let b = (3, 4)
puts(math.dist(a, b))
puts(math.angle(a, (10, 10)) / 1deg)
```

**Output:**

```text
5.0
45.0
```

`math.angle(p, q)` is the direction from `p` to `q`, in radians, measured from the positive x direction.

### A point is one argument

Inside a call, `f((1, 2))` passes **one** point, while `f(1, 2)` passes two numbers:

```
function show(p) {
  puts("x:", p.x, "y:", p.y)
}
show((1, 2))
show(1, 2)
```

**Output:**

```text
x: 1 y: 2
[line 5, column 5] runtime error: wrong number of arguments: want 1, got 2
```

## Example: a star of points

Points, units and `math` together. Each point of a 5-pointed star is a rotation around the center:

```
let center = (100, 100)
let radius = 50
for (let i = 0; i < 5; i += 1) {
  let angle = i * 72deg - 90deg
  let offset = (math.round(radius * math.cos(angle)), math.round(radius * math.sin(angle)))
  puts(i, center + offset)
}
```

**Output:**

```text
0 (100, 50)
1 (148, 85)
2 (129, 140)
3 (71, 140)
4 (52, 85)
```

> **Try it.** Write `midpoint(p, q)` that gives the point halfway between two points. Then write `lighter(c)` that gives a color with every channel moved halfway towards 255.

## Summary

- Colors: `#rgb`, `#rrggbb`, `#rrggbbaa`, or `rgb`, `rgba`, `hsl`. Read `c.r c.g c.b c.a`.
- Points: `(x, y)` or `(x, y, z)`, or `point(...)`. Read `p.x p.y p.z`.
- `+`, `-` between points; `*`, `/` with a number.
- Both are fixed: make new ones instead of changing them.
- `math.dist` and `math.angle` measure between points.

[← Previous: Numbers, `math` and units](11-numbers-and-math.md) · [Index](../README.md) · [Next: Layers →](13-layers.md)
