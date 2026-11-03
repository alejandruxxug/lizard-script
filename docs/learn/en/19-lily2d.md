# 19. lily2d: 2D graphics

[← Previous: Modules and tools](18-modules-and-tools.md) · [Index](../README.md) · [Cheat sheet →](cheatsheet.md) · [Español](../es/19-lily2d.md)

In chapter 17 you painted with `img.span` and `img.fill`: one row, one rectangle at a time. That is enough for a circle, but not for a smooth curve, a star with a stroke, a gradient sky or a pen line that follows the pressure. **lily2d** is a library that does all of that. It is a standard module (chapter 18) of the **flower** family of graphics libraries, written in lizard-script itself.

Every function is in the [lily2d reference](../../flower/LILY2D.md). This chapter shows the ideas, one step at a time.

## Your first picture

```
use "flower/lily2d" as lily2d

let picture = img.new(200, 120, #f4efe6)
let sun = lily2d.circle(picture, (60, 60), 40, #f2a65a)
puts(sun.kind, sun.fill, len(sun.path))
puts(img.get(picture, 60, 60), img.get(picture, 150, 60))
img.save(picture, "sun.png")
```

**Output:**

```text
circle #f2a65a 6
#f2a65a #f4efe6
```

- `lily2d.circle(picture, (60, 60), 40, #f2a65a)` paints a circle on `picture`: its center, its radius and its color. Open `sun.png` to see it. Its edge is **anti-aliased**: the pixels the edge cuts are mixed with the paper, so it looks smooth instead of jagged.
- The first argument is the **target**. Like the Terrarium's drawing functions, lily2d takes an image, a window or a surface, so the same code draws on any of them.
- Every call also **returns the shape**, a hash. `sun.kind` is `"circle"`, `sun.fill` is its paint, and `sun.path` is its outline as a list of commands (a circle is 4 curves, a start and a close: 6).

> **Try it:** draw three circles that overlap, with colors that have alpha (`rgba(224, 96, 122, 160)`). Which one is on top?

## Fill, stroke and width

The last argument is the **style**. A color alone fills the shape. A hash says more:

```
use "flower/lily2d" as lily2d

let picture = img.new(200, 120, #ffffff)
lily2d.rect(picture, (10, 10), (80, 50), {"fill": #8fb996, "stroke": #1a1418, "width": 4}, 10)
lily2d.line(picture, (10, 90), (190, 90), {"stroke": #2d6a8f, "width": 6, "dash": [12, 6], "cap": "butt"})
puts(img.get(picture, 50, 35), img.get(picture, 10, 35), img.get(picture, 15, 90), img.get(picture, 25, 90))
```

**Output:**

```text
#8fb996 #1a1418 #2d6a8f #ffffff
```

- `fill` is the inside, `stroke` the outline, `width` how thick the outline is. The last argument of `rect`, `10`, rounds its corners.
- `dash` cuts a stroke into pieces: 12 pixels on, 6 off. The pixel at x = 25 falls in a gap, so it is still white.
- `cap` is how an open line ends (`"round"`, `"butt"`, `"square"`) and `join` how its corners look (`"round"`, `"miter"`, `"bevel"`).
- For a line, a color alone is the stroke: `lily2d.line(picture, a, b, #000)` is a black line 1 pixel wide.

The other style keys are `opacity` (0 to 1), `blend` (`"multiply"`, `"screen"` and the other modes of chapter 17) and `rule`; the reference has the whole table.

## Paths: curves through points

`lily2d.path` takes a list of points and draws a smooth outline **through** them. Where the outline turns sharply it keeps a corner: a point where it turns by more than `corner` degrees (55 unless you say) stays sharp.

```
use "flower/lily2d" as lily2d

let corners = [(20, 20), (180, 20), (180, 100), (20, 100)]
let box = lily2d.path(null, corners, #000)
let pillow = lily2d.path(null, corners, {"fill": #000, "corner": 120})
puts(box.path[1][0], pillow.path[1][0])
let frame = lily2d.path(null, corners, {"fill": #000, "holes": [[(60, 40), (140, 40), (140, 80), (60, 80)]]})
puts(lily2d.contains(frame, (30, 30)), lily2d.contains(frame, (100, 60)))
```

**Output:**

```text
L C
true false
```

- A `null` target only **makes** the shape, without painting it anywhere. That's handy for shapes you will move, test or draw later.
- The four corners of a rectangle turn by 90 degrees, more than 55, so `box` is straight lines (`"L"`). With `"corner": 120` none of them is sharp, so `pillow` is curves (`"C"`): a pillow.
- `holes` cuts shapes out: `frame` is a picture frame. `lily2d.contains(shape, point)` says whether the shape paints that point, so the middle of the frame is `false`.

You can also write the commands yourself, like in SVG: `["M", p]` moves, `["L", p]` is a line, `["C", c1, c2, p]` a cubic Bézier curve, `["Q", c, p]` a quadratic one and `["Z"]` closes.

> **Try it:** draw a heart with two `"C"` commands from the bottom point up to the middle of the top and back down.

## Text

```
use "flower/lily2d" as lily2d

let picture = img.new(120, 40, #ffffff)
let hello = lily2d.text(picture, (60, 20), "hello", {"fill": #1a1418, "size": 2, "align": "center", "valign": "middle"})
puts(lily2d.measure("hello", 2), lily2d.bounds(hello))
```

**Output:**

```text
(70, 26) [25.0, 11.0, 68.0, 18.0]
```

- The text is drawn in a 7x13 bitmap font, at `size` 1 to 8 (each pixel of the font becomes `size` by `size`). `align` and `valign` say where `pos` is: here, the middle of the text.
- `lily2d.measure` is the box the font gives the text; `lily2d.bounds` is the box of the pixels it really painted (`[x, y, w, h]`), which is smaller, since letters don't fill their whole cell.

## Gradients

A gradient goes where a color goes:

```
use "flower/lily2d" as lily2d

let picture = img.new(200, 60, #ffffff)
let sky = lily2d.linear([#2b1d3f, #e0607a, #f6c453])
lily2d.rect(picture, (0, 0), (200, 60), sky)
puts(img.get(picture, 100, 0), img.get(picture, 100, 30), img.get(picture, 100, 59))
let bands = lily2d.linear([#000000, #ffffff], 0deg, 4)
lily2d.rect(picture, (0, 0), (200, 60), bands)
puts(img.get(picture, 10, 30), img.get(picture, 60, 30), img.get(picture, 110, 30), img.get(picture, 190, 30))
```

**Output:**

```text
#2e1e40 #e06279 #f6c254
#000000 #555555 #aaaaaa #ffffff
```

- `lily2d.linear(stops)` goes from the first color to the last **across the shape it paints**: top to bottom (`90deg`) unless you give an angle. So the same `sky` fits a small button or the whole window.
- `0deg` goes left to right, and the third argument, `4`, cuts the gradient into 4 flat bands.
- `lily2d.radial(stops)` goes out from the middle in circles. `lily2d.linear_at(from, to, stops)` and `lily2d.radial_at(center, r, stops)` are pinned to the picture in pixels instead, so several shapes can share one gradient, like one sunset behind many hills.
- `lily2d.hard(color)` as a stop makes a sharp edge instead of a smooth blend.

## A pen that feels the pressure

`lily2d.brush(target, points, pressures, color, pen)` draws a smooth stroke whose width follows a pressure, a number from 0 to 1 for each point. The pressure comes from the window's events (`e.pen.pressure`, chapter 17), and the **pen** decides how it looks:

```
use "flower/lily2d" as lily2d

let ink = lily2d.brush(null, [(10, 50), (60, 20), (110, 50)], [0.1, 1.0, 0.1], #1a1418, lily2d.pens.ink)
let marker = lily2d.brush(null, [(10, 50), (60, 20), (110, 50)], [0.1, 1.0, 0.1], #1a1418, lily2d.pens.marker)
puts(ink.widths)
puts(marker.widths, marker.fill)
puts(keys(lily2d.pens))
```

**Output:**

```text
[2.777809742251543, 6.670711017102133, 2.9631859934349043]
[10.0, 10.0, 10.0] #1a14188c
["ink", "pencil", "marker", "brush", "calligraphy"]
```

- The ink pen gets thicker where you press harder (it also smooths the pressures a little, so the line is steady). The marker keeps one width and is see-through: its fill has alpha `8c`.
- A pen is a hash, so you can write your own: `{"name": "felt", "min": 4, "max": 6, ...}`. The reference lists its keys, and the **pressure block** that says, for example, that a trackpad needs a firmer press.

Run `lizardscript examples/lily2d/doodle.liz` to draw with all five pens; keys 1 to 5 change the pen.

## Moving shapes

A **matrix** moves, turns or resizes a shape. `lily2d.transform(shape, m)` gives a moved copy:

```
use "flower/lily2d" as lily2d

let leaf = lily2d.ellipse(null, (100, 60), (10, 30), #3fa34d)
let turned = lily2d.transform(leaf, lily2d.rotate(90deg, (100, 100)))
puts(lily2d.bounds(leaf))
puts(lily2d.bounds(turned))
let m = lily2d.combine(lily2d.scale(2), lily2d.translate((5, 0)))
puts(lily2d.apply(m, (10, 10)))
```

**Output:**

```text
[90.0, 30.0, 20.0, 60.0]
[110.0, 90.0, 60.0, 20.0]
(25.0, 20.0)
```

- `lily2d.rotate(90deg, (100, 100))` turns a quarter turn clockwise round the point `(100, 100)`: the tall leaf above it becomes a wide leaf to its right.
- `lily2d.combine(first, second)` does one, then the other: scale by 2, then move 5 to the right, so `(10, 10)` goes to `(25, 20)`.
- `lily2d.draw(target, shape)` paints a shape you made before. A flower is eight copies of one petal, each turned 45 degrees more: see `examples/lily2d/scene.liz`.

## Scenes: paint only what changed

An animation could clear the window and draw every shape in every frame, but most shapes don't move. A **scene** keeps the shapes and, each time you draw it, paints again only around the ones that changed:

```
use "flower/lily2d" as lily2d

let pond = lily2d.scene({"background": #2d6a8f})
let frog = lily2d.circle(pond, (40, 40), 10, #3fa34d)
lily2d.circle(pond, (150, 80), 20, #f4b6c2)
let picture = img.new(200, 120)
puts(pond.draw(picture))
puts(pond.draw(picture))
frog = pond.replace(frog, lily2d.transform(frog, lily2d.translate((20, 0))))
puts(pond.draw(picture))
puts(same(pond.at((60, 40)), frog), pond.at((5, 5)))
```

**Output:**

```text
[[0, 0, 200, 120]]
[]
[[29, 29, 42, 22]]
true null
```

- With a scene as the target, a drawing call **adds** the shape to the scene instead of painting it.
- `pond.draw(picture)` returns the rectangles it painted. The first time it is everything; with no changes it is nothing; after the frog moves it is one small box round where the frog was and where it is now.
- `pond.replace(old, new)` keeps the new shape at the same depth (in front of the same shapes). Keep what it returns: the scene knows a shape by being the very same value.
- `pond.at(point)` is the frontmost shape under a point, for clicks.

In a window, call `pond.draw(w)` in `tick`; the scene also tells the window which rectangles changed, so only they go to the screen. `lizardscript examples/lily2d/scene.liz` shows how much of the window each frame paints.

## Saving as SVG

`lily2d.svg(shapes, size, background)` writes the shapes as an SVG file, which any browser or vector editor opens:

```
use "flower/lily2d" as lily2d

let dot = lily2d.circle(null, (10, 10), 5, #f2a65a)
puts(lily2d.svg([dot], (20, 20)))
```

**Output:**

```text
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20">
  <path d="M 15 10 C 15 12.76 12.76 15 10 15 C 7.24 15 5 12.76 5 10 C 5 7.24 7.24 5 10 5 C 12.76 5 15 7.24 15 10 Z" fill="#f2a65a"/>
</svg>

```

Save it with `fs.write("dot.svg", lily2d.svg(...))`. A scene's shapes are `pond.shapes()`.

## Common mistakes

**A position as two numbers.** Points are one value: `(50, 50)`, not `50, 50`.

```
use "flower/lily2d" as lily2d

let picture = img.new(100, 100)
lily2d.circle(picture, 50, 50, 20, #fff)
```

**Output:**

```text
[line 4, column 14] runtime error: wrong number of arguments: want 4, got 5
```

**A style key that doesn't exist.** The color of a fill is `"fill"`, and the message lists every key:

```
use "flower/lily2d" as lily2d

let picture = img.new(100, 100)
lily2d.circle(picture, (50, 50), 20, {"color": #fff})
```

**Output:**

```text
std:/flower/lily2d/check.liz: [line 7, column 8] runtime error: lily2d.circle: the style has no key color; it can have fill, stroke, width, join, cap, miter, dash, dash_offset, rule, opacity, blend
```

The position is inside lily2d (chapter 18): the call to look for is the one the message names, `lily2d.circle`.

**Forgetting what `replace` returns.** After a replace, the old shape is not in the scene any more:

```
use "flower/lily2d" as lily2d

let pond = lily2d.scene()
let frog = lily2d.circle(pond, (40, 40), 10, #3fa34d)
let moved = lily2d.transform(frog, lily2d.translate((5, 0)))
pond.replace(frog, moved)
pond.remove(frog)
```

**Output:**

```text
std:/flower/lily2d/check.liz: [line 7, column 8] runtime error: lily2d.scene.remove: this shape is not in the scene
```

Write `frog = pond.replace(frog, moved)`.

**A plain `use`.** `use "flower/lily2d"` without `as` works, but it makes every lily2d function a name of your file, and one of them is `text`, which hides the `text` library (`text.join` stops working). Use `as lily2d`.

## Summary

- `use "flower/lily2d" as lily2d`, then `lily2d.circle(target, center, r, style)` and the other shapes: `ellipse`, `rect`, `line`, `polyline`, `polygon`, `path`, `arc`, `star`, `text`, `brush`, `group`.
- The target is an image, a window or a surface (paint now), a scene (keep it) or `null` (just make it). Every call returns the shape.
- A style is a color, a gradient, or a hash: `fill`, `stroke`, `width`, `join`, `cap`, `dash`, `opacity`, `blend`, ...
- `lily2d.linear` and `lily2d.radial` fit each shape; the `_at` versions are pinned in pixels.
- `lily2d.brush` with a pen turns pressures into a stroke.
- `lily2d.transform(shape, matrix)` moves shapes; `rotate`, `scale`, `translate`, `combine`.
- `lily2d.contains` and `scene.at` find what is under a point; `lily2d.svg` exports.
- A scene repaints only what changed: `add`, `replace`, `remove`, `draw`.

[← Previous: Modules and tools](18-modules-and-tools.md) · [Index](../README.md) · [Cheat sheet →](cheatsheet.md)
