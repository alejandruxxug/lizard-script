# 3. Variables and scope

[← Previous: Values and types](02-values-and-types.md) · [Index](../README.md) · [Next: Operators →](04-operators.md) · [Español](../es/03-variables-and-scope.md)

A variable is a name for a value. In lizard-script there are two different actions, and keeping them apart is the key to this chapter:

- `let name = value` **creates** a variable.
- `name = value` **changes** a variable that already exists.

## Creating and changing

```
let name = "Liz"
let toes = 20
puts(name, "has", toes, "toes")

toes = toes - 1
puts("after an accident,", name, "has", toes, "toes")
```

**Output:**

```text
Liz has 20 toes
after an accident, Liz has 19 toes
```

A variable can hold any type, and can later hold a value of another type:

```
let thing = 5
thing = "five"
puts(thing)
```

**Output:**

```text
five
```

Names start with a letter or `_`, followed by letters, digits or `_`. Upper and lower case are different: `Name` and `name` are two variables. Only plain ASCII letters are allowed, so `café` is not a valid name (it's fine inside a string).

## Common mistake: changing a variable that was never created

```
total = 10
```

**Output:**

```text
[line 1, column 1] runtime error: can't assign to total: it was never declared, use let total = ...
```

The message tells you the fix. Requiring `let` first catches typos: if you misspell a name when you assign to it, you get an error instead of a silent new variable.

```
let count = 0
cuont = count + 1
```

**Output:**

```text
[line 2, column 1] runtime error: can't assign to cuont: it was never declared, use let cuont = ...
```

Reading a name that doesn't exist is also an error:

```
puts(score)
```

**Output:**

```text
[line 1, column 6] runtime error: identifier not found: score
```

## `let` twice

Writing `let` again for the same name, in the same place, simply replaces it:

```
let toes = 20
let toes = "many"
puts(toes)
```

**Output:**

```text
many
```

## Assignment is an expression

`=` gives back the value it stored, so you can chain it:

```
let a = 0
let b = 0
a = b = 7
puts(a, b)
```

**Output:**

```text
7 7
```

## Scope: where a variable lives

Every block `{ ... }` (the body of an `if`, of a loop, of a function) is a new **scope**. A variable created with `let` inside a block exists only inside it:

```
if (true) {
  let secret = 42
  puts("inside:", secret)
}
puts("outside:", secret)
```

**Output:**

```text
inside: 42
[line 5, column 18] runtime error: identifier not found: secret
```

A block can **read and change** the variables around it, as long as it uses `=` and not `let`:

```
let counter = 0
function increment() {
  counter = counter + 1
}
increment()
increment()
puts("counter:", counter)
```

**Output:**

```text
counter: 2
```

## Shadowing: the same name in two scopes

If a block uses `let` with a name that already exists outside, it creates a **new** variable that hides the outer one until the block ends. The outer one is not changed:

```
let x = "global"

function show() {
  let x = "local"
  puts("inside show, x is", x)
}
show()
puts("after show, x is still", x)

if (true) {
  let x = "in the if"
  puts("inside the if, x is", x)
}
puts("after the if, x is", x)
```

**Output:**

```text
inside show, x is local
after show, x is still global
inside the if, x is in the if
after the if, x is global
```

Function parameters are local too:

```
let value = 100
function double(value) {
  value * 2
}
puts(double(5), value)
```

**Output:**

```text
10 100
```

The `let` in a `for` header lives only inside the loop (chapter 6):

```
let i = "outside"
for (let i = 0; i < 2; i = i + 1) {
  puts("in the loop, i is", i)
}
puts("after the loop, i is", i)
```

**Output:**

```text
in the loop, i is 0
in the loop, i is 1
after the loop, i is outside
```

**Try it.** Predict what this prints, then run it:

```
let n = 1
if (true) {
  n = 2
  let n = 3
  n = 4
}
puts(n)
```

<details><summary>Answer</summary>

```
let n = 1
if (true) {
  n = 2
  let n = 3
  n = 4
}
puts(n)
```

**Output:**

```text
2
```

`n = 2` changes the outer `n`. Then `let n = 3` creates a new `n` inside the block, so `n = 4` changes that one, which disappears at the `}`.
</details>

## You can even hide a builtin

Names are looked up from the innermost scope outwards, and the builtins come last. So a variable can hide a builtin:

```
let len = 5
puts(len)
```

**Output:**

```text
5
```

After that line, `len("abc")` would fail, because `len` is now a number. Avoid builtin names for your variables (`len`, `type`, `str`, `list`, `text`, `math`...).

## Common mistake: a line that starts with `-`

A line break ends a statement only when the statement can't go on. A line that starts with an operator, `(` or `[` **continues** the previous line:

```
let x = 10
-1
puts(x)
```

**Output:**

```text
9
```

That was `let x = 10 - 1`. If you really want a separate statement, end the first line with `;`. Two values side by side on the same line, on the other hand, are an error:

```
let x = 5 6
```

**Output:**

```text
[line 1, column 11] syntax error: expected end of statement, got "6"
```

## Summary

- `let` creates, `=` changes. Assigning to a name that was never created is an error.
- Every `{ }` block is a scope; `let` inside it stays inside.
- Without `let`, a block changes the outer variable.
- A line starting with an operator, `(` or `[` continues the line before.

[← Previous: Values and types](02-values-and-types.md) · [Index](../README.md) · [Next: Operators →](04-operators.md)
