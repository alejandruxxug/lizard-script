# 5. Conditionals

[← Previous: Operators](04-operators.md) · [Index](../README.md) · [Next: Loops →](06-loops.md) · [Español](../es/05-conditionals.md)

A conditional lets a program choose what to do. lizard-script has `if`, `elif` (short for "else if") and `else`.

## `if`, `elif`, `else`

```
let temperature = 27

if (temperature < 15) {
  puts("too cold, the lizard hides")
} elif (temperature < 30) {
  puts("nice, the lizard sunbathes")
} else {
  puts("too hot, the lizard looks for shade")
}
```

**Output:**

```text
nice, the lizard sunbathes
```

The conditions are checked from top to bottom, and only the **first** block whose condition is true runs. `elif` and `else` are optional, and you can have as many `elif`s as you need.

The braces `{ }` are always required, even for a single statement. The parentheses around the condition are optional, so `if temperature < 15 { ... }` also works; this guide always writes them because it's easier to read.

## Common mistake: `else if`

Many languages write `else if`. In lizard-script, `else` must be followed by `{`:

```
let x = 3
if (x > 5) {
  puts("big")
} else if (x > 1) {
  puts("medium")
}
```

**Output:**

```text
[line 4, column 8] syntax error: expected "{", got "if"
```

Write `elif` instead:

```
let x = 3
if (x > 5) {
  puts("big")
} elif (x > 1) {
  puts("medium")
}
```

**Output:**

```text
medium
```

## Nesting

A block can contain another `if`:

```
function describe(temperature) {
  if (temperature < 30) {
    if (temperature >= 25) {
      puts(temperature, "-> perfect")
    } else {
      puts(temperature, "-> fine")
    }
  } else {
    puts(temperature, "-> too hot")
  }
}
describe(20)
describe(27)
describe(35)
```

**Output:**

```text
20 -> fine
27 -> perfect
35 -> too hot
```

Often you can flatten nested `if`s with `&&`: `if (temperature >= 25 && temperature < 30)`.

## Any value can be a condition

Remember truthiness from chapter 2: `false`, `null`, `0`, `0.0`, `""`, `[]` and `{}` count as false.

```
let name = ""
if (name) {
  puts("hello,", name)
} else {
  puts("you didn't tell me your name")
}

let items = []
if (!items) {
  puts("the list is empty")
}
```

**Output:**

```text
you didn't tell me your name
the list is empty
```

## `if` gives a value

An `if` is also an expression: its value is the value of the block that ran. That's handy to pick between two values:

```
let toes = 4
let label = if (toes == 5) { "complete" } else { "missing toes" }
puts("the foot is", label)
```

**Output:**

```text
the foot is missing toes
```

If no block runs (an `if` with no `else` whose condition is false), the value is `null`:

```
let x = if (false) { 1 }
puts(x)
```

**Output:**

```text
null
```

## Conditionals in the REPL

In the REPL, after an `if` block closes, the prompt stays at `🦎 ..` for one more line, in case the next line starts with `elif` or `else`. Press Enter on an empty line to run the `if` alone:

```text
🦎 >> let x = 7
🦎 >> if (x > 5) {
🦎 ..   puts("big")
🦎 .. }
🦎 .. 
big
🦎 >> if (x > 10) { puts("huge") }
🦎 .. else { puts("not huge") }
not huge
🦎 >> exit
```

> **Try it.** Write a program that, given a variable `n`, prints `"negative"`, `"zero"` or `"positive"`. Then turn it into an expression: `let sign = if (...) { ... } elif ...`.

## Summary

- `if (cond) { } elif (cond) { } else { }`: braces always, parentheses optional.
- Use `elif`, not `else if`.
- Any value works as a condition.
- An `if` has a value: the one of the block that ran, or `null`.

[← Previous: Operators](04-operators.md) · [Index](../README.md) · [Next: Loops →](06-loops.md)
