# lily2d: 2D vector graphics

`flower/lily2d` draws anti-aliased 2D shapes into an image, a window or a surface: circles, rectangles with round corners, smooth paths with holes, Bézier curves, arcs, stars, text, and strokes with joins, caps and dashes; painted with colors, linear and radial gradients, opacity and blend modes. It also has pressure brushes with five pens, 2D matrices, hit tests, SVG export, and a retained scene that paints again only what changed. It is a [standard module](../TERRARIUM.md#standard-modules) of the [flower](README.md) family, written in lizard-script on top of the Terrarium's [drawing targets](../TERRARIUM.md#drawing-targets).

```
use "flower/lily2d" as lily2d

let w = window.open(600, 400, "lily")
window.clear(w, #f4efe6)
lily2d.circle(w, (300, 200), 50, #f2a65a)
lily2d.path(w, [(10, 10), (200, 40), (120, 180)], {"fill": lily2d.linear([#f2a65a, #3b2a5c], 90deg), "stroke": #1a1418, "width": 3})
window.run(w, function(events) { true })
```

The examples in `examples/lily2d/` show all of it: `gallery.liz` (every shape), `poster.liz` (gradients), `doodle.liz` (a pressure doodle pad on a scene) and `scene.liz` (an animated scene). Each one takes `--save file.png` to draw itself on a headless window and save the picture.

- [Using it](#using-it)
- [Targets and shapes](#targets-and-shapes)
- [Styles](#styles)
- [Shapes](#shapes)
- [Paint: gradients](#paint-gradients)
- [Brushes and pens](#brushes-and-pens)
- [Transforms](#transforms)
- [Geometry and hit tests](#geometry-and-hit-tests)
- [Spans and SVG](#spans-and-svg)
- [Scenes](#scenes)
- [How the pixels are made](#how-the-pixels-are-made)
- [Limits](#limits)

## Using it

`use "flower/lily2d" as lily2d` gives the hash `lily2d` with every function below. The module is found after the program's own folder and `$LIZARD_PATH`, so a file `flower/lily2d.liz` of the program's own would win. Any name works after `as` (`as g` gives `g.circle`).

A plain `use "flower/lily2d"` works too, and makes every name below a name of the file (`circle(w, ...)`), but one of them is `text`, which then hides the `text` library in that file. Use `as`.

## Targets and shapes

Every drawing function takes a **target** first:

| target | what happens |
| --- | --- |
| an image, a window or a surface | the shape is painted on it now |
| a [scene](#scenes) | the shape is added to the scene, drawn by `scene.draw` |
| `null` | the shape is only made, to draw later, move, test or export |

and returns the **shape**, a hash. Its keys are `kind` (`"circle"`, `"rect"`, `"path"`, `"text"`, `"brush"`, `"group"`, ...), `path`, the outline as absolute commands (`[["M", p], ["L", p], ["C", c1, c2, p], ["Z"]]`), and every key of its [style](#styles). A group is `{"kind": "group", "shapes": [...], "opacity", "blend"}`. Changing a key (`dot.fill = #ff0000`) changes how the shape is drawn next time; in a scene, tell it with `scene.changed(dot)`.

```
let dot = lily2d.circle(null, (20, 20), 8, #2d6a8f)
lily2d.draw(picture, dot)
lily2d.draw(picture, lily2d.transform(dot, lily2d.translate((30, 0))))
```

Positions are points and sizes are in pixels from the top left, y down. Angles are radians, and the units `deg` and `turn` write them: `90deg`, `0.25turn`. They turn clockwise on the screen, from "to the right".

## Styles

Every drawing call takes a **style**: a paint alone (a color or a [gradient](#paint-gradients)), or a hash of these keys:

| key | default | what it is |
| --- | --- | --- |
| `fill` | `null` | the paint inside the shape |
| `stroke` | `null` | the paint of its outline |
| `width` | `1` | the outline's thickness in pixels |
| `join` | `"round"` | the outline's corners: `"round"`, `"miter"` (a sharp point) or `"bevel"` (cut flat) |
| `cap` | `"round"` | the ends of an open outline: `"round"`, `"butt"` (ends at the point) or `"square"` (goes on by half the width) |
| `miter` | `4` | how long a miter corner may get, in half widths, before it is cut to a bevel |
| `dash` | `null` | a dash pattern, lengths on and off: `[8, 4]`, `[16, 6, 2, 6]`; an odd list is repeated to make it even; `[0, 12]` with round caps is dots |
| `dash_offset` | `0` | how far into the pattern the outline starts |
| `rule` | `"nonzero"` | which parts of a path that crosses itself are inside: `"nonzero"` or `"evenodd"` |
| `opacity` | `1` | 0 to 1, how see-through the whole shape is |
| `blend` | `"normal"` | how it mixes with what is below: `"normal"`, `"multiply"`, `"screen"`, `"overlay"`, `"darken"`, `"lighten"` or `"add"` (the Terrarium's [blend modes](../TERRARIUM.md#blend-modes)) |

A paint alone fills a closed shape (circle, rect, polygon, path, star, text) and strokes a line (line, polyline, arc), 1 pixel wide. A hash needs a `fill` or a `stroke` (or both: the fill is painted first). Some calls take a few keys of their own in the same hash, listed with them. An unknown key is an error that lists the ones there are, so a typo doesn't go unnoticed:

```
lily2d.circle(w, (1, 1), 4, {"fil": #fff})
-~> lily2d.circle: the style has no key fil; it can have fill, stroke, width, join, cap, miter, dash, dash_offset, rule, opacity, blend
```

An opacity below 1 on a shape that has both a fill and a stroke, or any blend mode other than `"normal"`, paints the shape in a [layer](../TERRARIUM.md#push_layertarget) of its own and mixes it in, so the fill doesn't show through the stroke. Otherwise the opacity goes into the colors.

## Shapes

Every error starts with the function's name and says what was wanted: `lily2d.circle: center must be a point like (10, 20), not STRING`.

### lily2d.circle(target, center, r, style)
A circle of radius `r` round `center`.
```
lily2d.circle(w, (300, 200), 50, #f2a65a)
```

### lily2d.ellipse(target, center, radii, style)
An ellipse; `radii` is a point, half its width and half its height.
```
lily2d.ellipse(w, (210, 60), (60, 32), {"fill": #8fb996, "stroke": #1a1418, "width": 2})
```

### lily2d.rect(target, pos, size, style, radius = 0)
A rectangle with its top left at `pos` and `size` a point `(w, h)`; with a `radius` above 0 its corners are round (the radius is cut to half the shorter side).
```
lily2d.rect(w, (300, 20), (120, 80), {"fill": #2d6a8f, "stroke": #1a1418, "width": 3}, 18)
```

### lily2d.line(target, a, b, style)
A straight line from `a` to `b`.
```
lily2d.line(w, (20, 160), (120, 220), {"stroke": #1a1418, "width": 6, "cap": "butt"})
```

### lily2d.polyline(target, points, style)
Straight segments through `points` (a list of at least one point), not closed.
```
lily2d.polyline(w, [(150, 220), (180, 160), (210, 220)], {"stroke": #3b2a5c, "width": 8, "join": "miter"})
```

### lily2d.polygon(target, points, style)
A closed shape with straight sides.
```
lily2d.polygon(w, [(500, 20), (550, 100), (450, 100)], #c0392b)
```

### lily2d.path(target, points, style)
A path, in one of two ways.

**Through points.** `points` is a list of points and the outline goes through every one of them smoothly (Catmull-Rom curves). A point where the outline turns by more than `corner` degrees stays a sharp corner, and the piece between two sharp corners is straight, so a traced picture keeps both its curves and its corners. The style hash takes four keys more:

| key | default | what it is |
| --- | --- | --- |
| `closed` | `true` | join the last point to the first |
| `smooth` | `true` | `false` makes every side straight, like a polygon |
| `corner` | `55` | the turn, in degrees, from which a point stays sharp (`180`: never) |
| `holes` | `[]` | a list of point lists, each cut out of the shape (needs `closed`) |

```
lily2d.path(w, [(300, 200), (330, 150), (380, 170), (420, 150), (410, 220)], {"fill": lily2d.linear([#f2a65a, #e0607a])})
lily2d.path(w, [(20, 220), (80, 180), (140, 240)], {"stroke": #3b2a5c, "width": 5, "closed": false})
lily2d.path(w, square, {"fill": #8fb996, "smooth": false, "holes": [window_points]})
```

**From commands.** `points` is a list of commands, each a list that starts with its letter:

| command | what it does |
| --- | --- |
| `["M", p]` | starts a new piece at `p` |
| `["L", p]` | a straight line to `p` |
| `["C", c1, c2, p]` | a cubic Bézier curve to `p`, pulled by `c1` and `c2` |
| `["Q", c, p]` | a quadratic Bézier curve to `p`, pulled by `c` |
| `["Z"]` | closes the piece back to its start |

The first command is an `"M"`. A path of several pieces with `"rule": "evenodd"` has holes where pieces overlap.
```
lily2d.path(w, [["M", (450, 220)], ["C", (450, 140), (550, 140), (550, 220)], ["Q", (500, 180), (450, 220)], ["Z"]], {"fill": #8fb996})
```

### lily2d.arc(target, center, r, from, to, style)
Part of the circle round `center`, from the angle `from` to the angle `to`. A paint alone strokes the curve; with a `fill` it is a pie slice, closed through the center.
```
lily2d.arc(w, (630, 190), 40, -60deg, 220deg, {"stroke": #2d6a8f, "width": 10})
lily2d.arc(w, (400, 340), 40, -90deg, 30deg, {"fill": #c0392b})
```

### lily2d.star(target, center, count, outer, inner, style)
A star of `count` tips (2 or more) at `outer` pixels from `center`, with its dents at `inner`; the first tip points straight up.
```
lily2d.star(w, (630, 62), 5, 46, 19, {"fill": #f6c453, "stroke": #1a1418, "width": 2, "join": "miter"})
```

### lily2d.text(target, pos, text, style)
Text in the 7x13 bitmap font (ASCII and Latin-1; a character it lacks is a hollow box), `"\n"` starting a new line. It is a path of the font's pixels, so it takes every style key, a gradient fill, a transform and SVG export. The style hash takes three keys more:

| key | default | what it is |
| --- | --- | --- |
| `size` | `1` | 1 to 8: each pixel of the font is `size` by `size` |
| `align` | `"left"` | where `pos` is across each line: `"left"`, `"center"` or `"right"` |
| `valign` | `"top"` | where `pos` is up and down: `"top"`, `"middle"`, `"bottom"` or `"baseline"` |

At a whole-number `pos` the text is crisp, pixel for pixel.
```
lily2d.text(w, (300, 20), "hello", {"fill": #1a1418, "size": 2, "align": "center"})
```

### lily2d.measure(text, size = 1)
The size of `text` in the font, as a point: the longest line's width and the height of all the lines.
```
puts(lily2d.measure("hello"), lily2d.measure("hi\nthere", 2))   -~> (35, 13) (70, 52)
```

### lily2d.group(target, shapes, style = {})
Draws `shapes` (made with a `null` target) together and mixes them onto what is below with one `opacity` and `blend`, the only two keys its style has. A group at half opacity doesn't show its own overlaps; a group in `"multiply"` darkens what is below like ink. Groups nest.
```
let a = lily2d.circle(null, (500, 320), 30, #2d6a8f)
let b = lily2d.circle(null, (530, 320), 30, #e0607a)
lily2d.group(w, [a, b], {"blend": "multiply", "opacity": 0.85})
```

### lily2d.draw(target, shape)
Draws a shape again, or a list of shapes back to front, or adds them to a scene. It returns what it was given.
```
lily2d.draw(w, lily2d.transform(petal, lily2d.rotate(45deg, (120, 120))))
```

## Paint: gradients

A paint is a color or a gradient. A gradient is made by one of the four functions below; its `stops` are colors (spread evenly), `[t, color]` pairs with `t` from 0 to 1 in order, or `lily2d.hard(color)`. `steps` is 0 for a smooth gradient or 2 to 64 for that many flat bands. Before the first stop and after the last the color stays. The Terrarium works out each pixel's color ([gradient spans](../TERRARIUM.md#span_gradienttarget-y-x0-x1-gradient)).

Two of them **fit each shape**: their numbers are fractions of the box of the shape they paint, so one value works on any shape. The other two are **pinned** to the picture, in pixels, and a shape moved with `lily2d.transform` takes them with it.

### lily2d.linear(stops, angle = 90deg, steps = 0)
A linear gradient across each shape's box. `angle` is the way the colors go: `0deg` left to right, `90deg` top to bottom (the default), `180deg` right to left, `270deg` bottom to top. The first stop is at the box's start corner and the last at the opposite one.
```
lily2d.rect(w, (0, 0), (420, 560), lily2d.linear([#2b1d3f, #e0607a, #f6c453]))
lily2d.rect(w, (140, 10), (120, 70), lily2d.linear([#f2a65a, #e0607a, #3b2a5c], 45deg, 5))
```

### lily2d.linear_at(from, to, stops, steps = 0)
A linear gradient pinned to the picture: the first stop at the point `from`, the last at `to`. Errors when they are the same point.
```
lily2d.rect(w, (220, 210), (120, 60), lily2d.linear_at((220, 0), (340, 0), [#f4d35e, #3d1c4f]))
```

### lily2d.radial(stops, center = (0.5, 0.5), r = 0.5, steps = 0)
A radial gradient that fits each shape: circles round `center`, in fractions of the box (`(0, 0)` its top left, `(1, 1)` its bottom right), with the last stop at `r` times the box's longer side. The rings are circles on a long box too.
```
lily2d.circle(w, (70, 140), 50, lily2d.radial([#fff3c4, #e0607a], (0.3, 0.3), 0.8))
```

### lily2d.radial_at(center, r, stops, steps = 0)
A radial gradient pinned to the picture, round the point `center`, the last stop at `r` pixels (above 0).
```
lily2d.circle(w, (210, 300), 110, lily2d.radial_at((210, 300), 110, [#fff3c4, #f6c453, #e0607a], 6))
```

### lily2d.hard(color)
A hard stop: its color starts right where it is, with no blend from the stop before. `[#fff, #333, lily2d.hard(#c00), #000]` is a smooth white to grey, a sharp step to red, then a smooth red to black. `[t, color, "hard"]` puts the edge at a chosen `t`.

## Brushes and pens

### lily2d.brush(target, points, pressures, color, pen = lily2d.pens.ink, source = null)
A smooth stroke through `points` whose width and opacity follow `pressures`, one number from 0 to 1 for each point, the way the `pen` styles them. `source` is the device the pressures came from, `"mouse"`, `"trackpad"` or `"tablet"` (a window event's `e.pen.source`), for the pen's settings for that device; `null` uses none. The shape keeps `points`, `pressures`, and the `widths` and `opacities` the pen gave, so the same raw stroke can be drawn again with another pen.
```
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "move" && drawing) {
      push(points, e.pos)
      push(pressures, e.pen.pressure)
    }
  }
  img.clear(w, #f4efe6)
  if (len(points) > 0) {
    lily2d.brush(w, points, pressures, #1a1418, lily2d.pens.brush, "trackpad")
  }
})
```
`examples/lily2d/doodle.liz` is a whole doodle pad.

### lily2d.pens
The five pens: `ink` (crisp, width from pressure, tapered ends), `pencil` (thin, opacity from pressure, a grainy edge), `marker` (one width, see-through, flat ends), `brush` (wide range, soft taper, needs a firm press) and `calligraphy` (a flat nib at 45 degrees: thin along it, thick across). A pen is a hash, so a program can change one (`lily2d.pens.ink.max = 14`) or write its own:

| key | what it is |
| --- | --- |
| `name` | what to call it |
| `min`, `max` | the width in pixels at pressure 0 and at 1 (the same for a constant width) |
| `taper` | 0 to 0.5: the share of the stroke's length, at each end, where it narrows to a tip |
| `opacity` | `[low, high]`: the opacity at pressure 0 and at 1 |
| `nib` | `null`, or the angle of a flat nib: the width then follows the direction of the stroke |
| `grain` | 0, or the roughness of the edge in pixels (the same every time) |
| `tilt` | optional `{"nib": gain}`: a tablet's tilt turns the nib |
| `pressure` | the pressure block (below), optional |

The **pressure block** says how a raw pressure becomes the look; every key is optional:

| key | default | what it does |
| --- | --- | --- |
| `curve` | `[[0, 0], [1, 1]]` | points `[raw, effective]`; the effective pressure is read off the line through them |
| `width` | `[0, 1]` | how far along `min` to `max` the width goes at effective pressure 0 and at 1 |
| `opacity` | `[0, 1]` | the same for the `opacity` pair |
| `smoothing` | `0` | 0 to 0.95: blends each pressure with its neighbours first |
| `speed` | `0` | 0 to 1: how much a fast stroke (points far apart) thins |
| `floor` | `0` | 0 to 0.9: pressures under it count as 0, the rest is stretched to 0 to 1 |
| `nib_weight` | `0` | for a nib, how much pressure also scales its width |
| `inputs` | `{}` | overrides for a device: `{"trackpad": {"floor": 0.15}}` |

```
let felt = {"name": "felt", "min": 4, "max": 6, "taper": 0.1, "opacity": [0.8, 0.8], "nib": null, "grain": 0}
lily2d.brush(w, points, pressures, #2d6a8f, felt)
```

## Transforms

A **matrix** is a list of six numbers `[a, b, c, d, e, f]`, the ones of SVG's `matrix()`: it takes `(x, y)` to `(a*x + c*y + e, b*x + d*y + f)`.

| function | the matrix of |
| --- | --- |
| `lily2d.identity()` | no change, `[1.0, 0.0, 0.0, 1.0, 0.0, 0.0]` |
| `lily2d.translate(offset)` | a move by the point `offset` |
| `lily2d.rotate(angle, around = (0, 0))` | a turn by `angle`, clockwise on the screen, round `around` |
| `lily2d.scale(k, around = (0, 0))` | a resize by `k`, a number or a point (x and y apart), from `around` |
| `lily2d.combine(first, second)` | doing `first`, then `second` |
| `lily2d.invert(m)` | undoing `m` (an error for a matrix that squashes everything flat) |

### lily2d.apply(m, p)
The point `p` moved by `m`.
```
let m = lily2d.combine(lily2d.rotate(90deg), lily2d.translate((10, 0)))
puts(lily2d.apply(m, (1, 0)))   -~> (10.0, 1.0)
```

### lily2d.transform(shape, m)
A copy of `shape` moved by `m`: its path, its pinned gradients, and its stroke width and dashes (scaled by how much `m` stretches lengths on average, the square root of its area factor). A gradient that fits the shape fits the moved one. A group moves every shape in it. Other keys (a brush's `points`) stay as they were.
```
for (let i = 0; i < 8; i += 1) {
  lily2d.draw(w, lily2d.transform(petal, lily2d.rotate(i * 45deg, (120, 120))))
}
```

## Geometry and hit tests

### lily2d.bounds(shape)
The box the shape paints, `[x, y, w, h]` (the same form as the Terrarium's rectangles), its stroke included (a miter join or a square cap can reach further, and the box allows for it), or `null` for a shape with no points.
```
puts(lily2d.bounds(lily2d.circle(null, (50, 50), 20, {"fill": #000, "stroke": #f00, "width": 4})))   -~> [28.0, 28.0, 44.0, 44.0]
```

### lily2d.contains(shape, p)
Whether the shape paints the point `p`: inside its fill (by its `rule`), or within half the width of its stroke. A group asks its shapes. It is how a program finds what was clicked.
```
if (lily2d.contains(button, e.pos)) { pressed() }
```

### lily2d.flatten(shape, tolerance = 0.25)
The outline of a shape (or a path) as straight lines, within `tolerance` pixels of its curves: a list of `{"points": [...], "closed": true or false}`, one for each piece.

### lily2d.bezier(p0, c1, c2, p3, t)
The point at `t` (0 to 1) on the cubic Bézier from `p0` to `p3` pulled by `c1` and `c2`.
```
puts(lily2d.bezier((0, 0), (0, 100), (100, 100), (100, 0), 0.5))   -~> (50.0, 75.0)
```

### lily2d.nearest(shape, p)
The closest spot to `p` on the outline of a shape (or a path): `{"point", "dist", "command": the index of the path command it is on, "t": 0 to 1 along that command}`, or `null` for an empty path.
```
puts(lily2d.nearest(lily2d.line(null, (0, 0), (100, 0), #000), (30, 12)))
-~> {"point": (30.0, 0.0), "dist": 12.0, "command": 1, "t": 0.3}
```

### lily2d.simplify(points, tolerance)
The points without the ones that bend the line by less than `tolerance` pixels (Ramer-Douglas-Peucker); the first and last stay. A stroke recorded from a mouse gets much lighter.
```
puts(lily2d.simplify([(0, 0), (10, 0.2), (20, -0.1), (30, 0), (30, 30)], 1))   -~> [(0, 0), (30, 0), (30, 30)]
```

### lily2d.smooth(points, closed = false, corner = 55)
The path commands of the smooth outline through `points` that `lily2d.path` draws, sharp where it turns by more than `corner` degrees.

## Spans and SVG

### lily2d.spans(shape)
The spans of a shape, the rows the Terrarium paints: `{"fill": spans or null, "stroke": spans or null}`, each span `[y, x0, x1]` (row `y` from `x0` up to `x1`). They can be painted with `img.spans` or `img.spans_gradient`, and moved with their offset, as many times as needed. A group has no spans of its own.
```
puts(lily2d.spans(lily2d.rect(null, (1, 1), (2.5, 2), #000)))   -~> {"fill": [[1, 1.0, 3.5], [2, 1.0, 3.5]], "stroke": null}
```

### lily2d.svg(shapes, size, background = null)
The SVG text of `shapes`, back to front, on a picture of `size`, a point, with a `background` paint or none. Each shape is a `<path>` with its fill, stroke, joins, caps, dashes, rule, opacity and blend mode; a group with an opacity or a blend mode is a `<g>`; gradients go in `<defs>`, one for each different one (a gradient that fits a shape is written as the pinned one it is for that shape).
```
fs.write("art.svg", lily2d.svg(scene.shapes(), (600, 400), #ffffff))
```

## Scenes

A scene is lily2d's retained mode: it keeps a list of shapes, back to front, and draws them on a target. After the first draw, it paints again **only the regions that changed**: the boxes of the shapes that were added, removed or changed since, merged into a few regions that don't overlap. Each region is clipped, cleared to the background and painted with only the shapes that touch it, which a grid of buckets finds without looking at the others. On a window or a surface the scene also tells the window which rectangles changed ([`window.dirty`](../TERRARIUM.md#windowdirtytarget-rect)), so only they reach the screen. An app gets fast partial repaints by keeping its shapes in a scene.

### lily2d.scene(options = {})
A new, empty scene. The only option is `background`: a paint for every region it paints, or `null` (the default) to clear them to transparent. A background that fits shapes fits the whole target.
```
let pad = lily2d.scene({"background": #f4efe6})
lily2d.circle(pad, (20, 20), 8, #2d6a8f)        -~> a drawing call adds to the scene
```

A scene is a hash of functions, called with a dot:

| method | what it does |
| --- | --- |
| `s.add(shape)` | puts `shape` in front of the others and returns it (an error if it is in the scene already) |
| `s.remove(shape)` | takes it out (an error if it is not in the scene) |
| `s.replace(old, shape)` | puts `shape` where `old` was, in front of the same shapes, and returns it |
| `s.changed(shape)` | says `shape` was changed in place (a key set on it) |
| `s.clear()` | takes every shape out |
| `s.shapes()` | the shapes, back to front |
| `s.at(p)` | the frontmost shape that paints the point `p` (see `lily2d.contains`), or `null` |
| `s.draw(target)` | paints what changed and returns the rectangles it painted, `[x, y, w, h]` each |
| `s.invalidate()` | the next draw paints everything |

`s.draw` paints everything the first time, on a new target, when the target's size changed, and after `s.invalidate()`. A shape is found by being the very same value (`same`), so keep what `add` and `replace` return:

```
let ball = pad.add(lily2d.circle(null, (20, 20), 8, #c0392b))
window.run(w, function(events) {
  ball = pad.replace(ball, lily2d.transform(ball, lily2d.translate((2, 0))))
  pad.draw(w)      -~> repaints two small boxes, not the window
})
```

The scene owns its target: it expects the pixels it painted to still be there. After drawing something else on the target, call `s.invalidate()`. `s.draw` clears the target's clip when it ends.

## How the pixels are made

- **Scanline fill.** A shape's path is flattened into straight lines (within a quarter pixel of its curves), and a scanline filler walks the rows: each row is sampled at its middle, `y + 0.5`, and gives spans whose ends are fractional. The Terrarium covers the end pixels in part, so the left and right edges of a shape are soft; the top and bottom ones are not, since a row is either in or out. Both rules, `nonzero` and `evenodd`, are exact.
- **Strokes** are the union of a quad for each segment, a piece at each corner and one at each end, joined by the nonzero rule, so a stroke that crosses itself has no seams. A round join is a fan whose step keeps it within the same quarter pixel.
- **Kept spans.** The spans of a shape are kept by its path and how it is filled and stroked (not by its paint), so a shape drawn again, or a recolored one, is not rasterized again. The cache keeps what the last paints used and grows when a big scene needs it.
- **The same picture on both engines.** lily2d is lizard-script and runs the same on the tree-walking evaluator and the VM: the golden tests compare both with the same PNGs. The pictures were also compared with an independent implementation of the same rasterizer that paints with premultiplied 8-bit colors: every pixel is the same or differs by 1 in one channel, at soft edges, where the two round a partly covered pixel's mix differently (the Terrarium mixes in floats and rounds once).

## Limits

- **Speed.** It is lizard-script all the way to the spans: a few hundred shapes rasterize in a second or two on the VM (`lizardscript test --vm --bench tests/lily2d`), and a shape that doesn't change costs only its painting after that. A scene is the way to animate: a frame costs what changed. The tree-walking evaluator is several times slower.
- **Errors show the module's line.** An error in a wrong argument is raised inside the module, so when it is not caught the command line shows it at the module's file, `std:/flower/lily2d/check.liz: [line 7, column 8] runtime error: lily2d.circle: center must be a point ...`; the message names the call, but not the program's line.
- **One font.** Text is the 7x13 bitmap font, at whole sizes.
- **Radial gradients under a stretch.** A pinned radial gradient moved by a matrix that stretches x and y differently stays round (its radius takes the average stretch).
