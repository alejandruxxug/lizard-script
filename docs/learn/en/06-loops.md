# 6. Loops

[← Previous: Conditionals](05-conditionals.md) · [Index](../README.md) · [Next: Functions →](07-functions.md) · [Español](../es/06-loops.md)

A loop repeats a block. lizard-script has two: `while` and `for`, plus `break` and `continue` to steer them.

## `while`

`while (condition) { body }` runs the body again and again, as long as the condition is true. The condition is checked **before** each round:

```
let countdown = 3
while (countdown > 0) {
  puts("countdown:", countdown)
  countdown -= 1
}
puts("lift off!")
```

**Output:**

```text
countdown: 3
countdown: 2
countdown: 1
lift off!
```

If the condition is false from the start, the body never runs.

## `for`

`for` puts the three parts of a counting loop in one line: **start**; **condition**; **update**.

```
for (let i = 1; i <= 5; i += 1) {
  puts("round", i)
}
```

**Output:**

```text
round 1
round 2
round 3
round 4
round 5
```

It runs like this: do the start once; check the condition; if it's true, run the body, then the update, and check again.

```
let total = 0
for (let i = 1; i <= 100; i += 1) {
  total += i
}
puts("1 + 2 + ... + 100 =", total)
```

**Output:**

```text
1 + 2 + ... + 100 = 5050
```

The parentheses are required in a `for`, and any of the three parts can be left empty: `for (;;) { ... }` loops forever (until a `break`).

> **Common mistake: forgetting `let` in the `for` header.**
>
> ```
> for (i = 0; i < 3; i += 1) {
>   puts(i)
> }
> ```
>
> **Output:**
>
> ```text
> [line 1, column 6] runtime error: can't assign to i: it was never declared, use let i = ...
> ```
>
> `i = 0` changes a variable, and there is no `i` yet. Write `let i = 0`.

The variable made in the header exists only inside the loop. After the loop, `i` is gone:

```
for (let i = 0; i < 3; i += 1) { }
puts(i)
```

**Output:**

```text
[line 2, column 6] runtime error: identifier not found: i
```

## Going through an array

The usual way is a counter from `0` to `len(list) - 1`:

```
let pets = ["Liz", "Iggy", "Rex"]
for (let i = 0; i < len(pets); i += 1) {
  puts(i, pets[i])
}
```

**Output:**

```text
0 Liz
1 Iggy
2 Rex
```

`range(n)` builds the array `[0, 1, ..., n - 1]`, which is useful for small counts:

```
puts(range(5))
puts(range(2, 6))
```

**Output:**

```text
[0, 1, 2, 3, 4]
[2, 3, 4, 5]
```

## `break`: leave the loop

`break` stops the innermost loop right away:

```
let n = 1
while (true) {
  if (n * n > 50) {
    break
  }
  n += 1
}
puts("the first n with n * n > 50 is", n)
```

**Output:**

```text
the first n with n * n > 50 is 8
```

`break` outside a loop is a syntax error, found before the program runs:

```
puts("hello")
break
```

**Output:**

```text
[line 2, column 1] syntax error: break outside of a loop
```

## `continue`: skip to the next round

`continue` skips the rest of the current round. In a `for`, the update still runs, so the counter keeps moving:

```
for (let i = 1; i <= 10; i += 1) {
  if (i % 3 == 0) {
    continue
  }
  puts(i)
}
```

**Output:**

```text
1
2
4
5
7
8
10
```

> **Careful with `continue` in a `while`.** `continue` jumps straight back to the condition. If the update is after the `continue`, it never happens and the loop never ends. Put the update first:
>
> ```
> let i = 0
> while (i < 6) {
>   i += 1
>   if (i % 2 == 0) { continue }
>   puts(i)
> }
> ```
>
> **Output:**
>
> ```text
> 1
> 3
> 5
> ```

`continue` is only special inside a loop. Outside one, it's an ordinary name, so the error is a bit surprising:

```
if (true) { continue }
```

**Output:**

```text
[line 1, column 13] runtime error: identifier not found: continue
```

## Nested loops

A loop inside a loop: the inner one runs completely for each round of the outer one. `break` and `continue` only affect the innermost loop.

```
for (let row = 1; row <= 3; row += 1) {
  let line = ""
  for (let col = 1; col <= 4; col += 1) {
    line += str(row * col) + "\t"
  }
  puts(line)
}
```

**Output:**

```text
1	2	3	4	
2	4	6	8	
3	6	9	12	
```

## Loops have no value

An `if` gives a value, but `while` and `for` always give `null`. To build a result, collect it in a variable:

```
let squares = []
for (let i = 1; i <= 5; i += 1) {
  push(squares, i * i)
}
puts(squares)
```

**Output:**

```text
[1, 4, 9, 16, 25]
```

## Infinite loops

A loop whose condition never becomes false runs forever. In the REPL or in a terminal, stop it with **Ctrl+C**. If a program seems frozen, check that the loop's variable really changes in every round.

> **Try it.** Print the multiplication table of 7 (`7 x 1 = 7` to `7 x 10 = 70`). Then print only the lines whose result is even, using `continue`.

## Summary

- `while (cond) { }` checks before each round.
- `for (let i = 0; i < n; i += 1) { }`: start, condition, update. Don't forget `let`.
- `break` leaves the innermost loop; `continue` skips to its next round.
- In a `while`, update before any `continue`.
- Loops give `null`; collect results with `push`.

[← Previous: Conditionals](05-conditionals.md) · [Index](../README.md) · [Next: Functions →](07-functions.md)
