# 13. Layers

[← Previous: Colors and points](12-colors-and-points.md) · [Index](../README.md) · [Next: Errors, `try`/`catch` and `null` →](14-errors.md) · [Español](../es/13-layers.md)

A **layer** gives a name to a group of statements. It is the only part of lizard-script that uses **indentation** instead of braces.

## Writing a layer

`layer` and a name, alone on a line. The **body** is every following line that is indented more than the word `layer`:

```
let picture = []

layer sky
  push(picture, "sun")
  push(picture, "cloud")

layer ground
  push(picture, "grass")

puts(picture)
```

**Output:**

```text
["sun", "cloud", "grass"]
```

The first line that is back at the column of `layer` (here `layer ground`, then `puts`) ends the body. The name can be a word, or a string when it has spaces:

```
layer "tall grass"
  puts("swaying")
```

**Output:**

```text
swaying
```

## Layers nest

A `layer` inside a body is a child layer. Each body is measured from the column of its own `layer` word:

```
let picture = []
layer sky
  push(picture, "sun")
  layer clouds
    push(picture, "cloud 1")
    push(picture, "cloud 2")
  push(picture, "birds")
puts(picture)
```

**Output:**

```text
["sun", "cloud 1", "cloud 2", "birds"]
```

`"birds"` is back at the column of `layer clouds`, so it belongs to `sky`, not to `clouds`.

## What a layer does when it runs

A layer runs its body **once**, right where it is, as a block with its own scope. So it works just like the code would without the `layer` line, except that `let` inside stays inside:

```
let count = 0
layer counting
  let step = 5
  count += step
puts(count)
puts(step)
```

**Output:**

```text
5
[line 6, column 6] runtime error: identifier not found: step
```

`count += step` changed the outer variable; `step` didn't survive the layer.

## Why layers exist

On its own, a layer changes nothing in what a program computes. Layers are **names for the program that runs your code**. An editor or drawing tool that embeds lizard-script can be told when each layer starts and ends, so it can tag everything made inside, for example to show "sky" and "ground" as separate groups that can be hidden or selected. In a plain `./lizardscript` run, they are a tidy way to organize long programs into named sections.

## Layers inside functions and loops

Layers work inside braces too, measured from their own `layer` word. A `return` inside a layer returns from the function around it:

```
function draw(n) {
  layer body
    puts("drawing", n)
    if (n > 1) { return "big" }
  "small"
}
puts(draw(1))
puts(draw(2))
```

**Output:**

```text
drawing 1
small
drawing 2
big
```

## Rules and mistakes

The name must be alone on its line (a `-~>` comment is allowed):

```
layer sky blue
  puts("x")
```

**Output:**

```text
[line 1, column 11] syntax error: a layer is "layer name" alone on its line, got identifier "blue" after the name
```

`layer` is not a reserved word: you can still have a variable called `layer`. It only starts a layer when it's followed by a name and nothing else:

```
let layer = 1
puts(layer + 1)
```

**Output:**

```text
2
```

Indent with spaces, two per level is the usual style. A tab counts as one column, so mixing tabs and spaces makes the body hard to predict.

> **Layers don't work in the REPL.** The REPL runs one line at a time, so a `layer` typed there has an empty body and the indented lines run on their own. Use layers in files.

> **Try it.** Write a program with layers `background`, `house` (with child layers `walls` and `roof`) and `garden`, where each layer pushes the names of its parts to an array. Print the array at the end.

## Summary

- `layer name` alone on a line, then an indented body.
- The body ends at the first line back at the column of `layer`.
- Layers nest, have their own scope, and run once.
- They are names for host programs; in a plain run they organize the code.
- Files only, not the REPL.

[← Previous: Colors and points](12-colors-and-points.md) · [Index](../README.md) · [Next: Errors, `try`/`catch` and `null` →](14-errors.md)
