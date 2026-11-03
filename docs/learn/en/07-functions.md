# 7. Functions

[← Previous: Loops](06-loops.md) · [Index](../README.md) · [Next: Arrays →](08-arrays.md) · [Español](../es/07-functions.md)

A function is a piece of code with a name that you can run whenever you need it, with different inputs each time. In lizard-script functions are values, just like numbers and strings: you can store them in variables, pass them to other functions and return them.

## Defining and calling

```
function add(a, b) {
  return a + b
}
puts(add(2, 3))
puts(add(10, -4))
```

**Output:**

```text
5
6
```

`a` and `b` are the **parameters**; `2` and `3` are the **arguments** of that call. `return` gives the result back and ends the function.

## The last value is returned

Without `return`, a function gives the value of its last statement. Both styles are common:

```
function square(x) {
  x * x
}
puts(square(9))
```

**Output:**

```text
81
```

A bare `return` (or a function whose last statement has no value) gives `null`:

```
function greet(name) {
  puts("hi,", name)
}
let result = greet("Liz")
puts(result)
```

**Output:**

```text
hi, Liz
null
```

`return` can also end a function early:

```
function sign(n) {
  if (n < 0) { return "negative" }
  if (n == 0) { return "zero" }
  "positive"
}
puts(sign(-5), sign(0), sign(8))
```

**Output:**

```text
negative zero positive
```

## Functions are values

`function name(...) { }` is the same as `let name = function(...) { }`. The second form, a **function literal**, is useful when you want a function without giving it a name:

```
let greet = function(name) { "Hello, " + name + "!" }
puts(greet("Liz"))
puts(type(greet))
```

**Output:**

```text
Hello, Liz!
FUNCTION
```

## Wrong number of arguments

A call must give exactly one argument per parameter (unless the parameter has a default, below):

```
function add(a, b) { a + b }
add(1)
```

**Output:**

```text
[line 2, column 4] runtime error: wrong number of arguments: want 2, got 1
```

Calling something that isn't a function is also an error:

```
let n = 5
n(1)
```

**Output:**

```text
[line 2, column 2] runtime error: not a function: INTEGER
```

## Default values

A parameter can have a default value, used when the call leaves it out. Parameters with defaults go last:

```
function greet(name, greeting = "Hello") {
  greeting + ", " + name + "!"
}
puts(greet("Liz"))
puts(greet("Liz", "Good morning"))
```

**Output:**

```text
Hello, Liz!
Good morning, Liz!
```

A default is worked out at each call, after the parameters before it, so it can use them:

```
function label(name, size = 12, caption = name + " (" + str(size) + ")") {
  caption
}
puts(label("tree"))
puts(label("tree", 20))
puts(label("tree", 20, "big tree"))
```

**Output:**

```text
tree (12)
tree (20)
big tree
```

A default `[]` is a new empty array at each call, not one shared between calls:

```
function collect(x, into = []) {
  push(into, x)
}
puts(collect(1), collect(2))
```

**Output:**

```text
[1] [2]
```

A parameter without a default can't come after one that has one:

```
function f(a = 1, b) { a + b }
```

**Output:**

```text
[line 1, column 19] syntax error: parameter b needs a default value, it comes after one that has it
```

## Recursion

A function can call itself. Every recursive function needs a **base case** that stops it:

```
function factorial(n) {
  if (n <= 1) {
    return 1
  }
  n * factorial(n - 1)
}
puts(factorial(5), factorial(10))

function fib(n) {
  if (n < 2) { return n }
  fib(n - 1) + fib(n - 2)
}
puts(fib(20))
```

**Output:**

```text
120 3628800
6765
```

Without a base case, the calls pile up until the interpreter stops them:

```
function forever(n) {
  forever(n + 1)
}
forever(0)
```

**Output:**

```text
[line 2, column 10] runtime error: stack overflow: more than 10000 nested calls
```

## Closures

A function remembers the scope where it was **created**, and can keep using and changing its variables even after that scope has finished. Such a function is called a **closure**.

```
function makeCounter() {
  let count = 0
  function() {
    count += 1
    count
  }
}
let a = makeCounter()
let b = makeCounter()
a()
a()
puts("a:", a(), "b:", b())
```

**Output:**

```text
a: 3 b: 1
```

Each call to `makeCounter` creates a new `count`, so `a` and `b` count separately. Nobody else can touch `count`: it is private to the counter.

A function that builds functions:

```
function adder(x) {
  function(y) { x + y }
}
let addTen = adder(10)
let addOne = adder(1)
puts(addTen(5), addOne(5))
```

**Output:**

```text
15 6
```

## Higher-order functions

A function that takes or returns another function is called *higher-order*. Here are three classics you can write yourself:

```
function map(xs, f) {
  let out = []
  for (let i = 0; i < len(xs); i += 1) { push(out, f(xs[i])) }
  out
}
function filter(xs, keep) {
  let out = []
  for (let i = 0; i < len(xs); i += 1) {
    if (keep(xs[i])) { push(out, xs[i]) }
  }
  out
}
function reduce(xs, f, start) {
  let acc = start
  for (let i = 0; i < len(xs); i += 1) { acc = f(acc, xs[i]) }
  acc
}

let evens = filter(range(1, 11), function(n) { n % 2 == 0 })
puts(evens)
puts(map(evens, function(n) { n * n }))
puts(reduce(evens, function(a, b) { a + b }, 0))
```

**Output:**

```text
[2, 4, 6, 8, 10]
[4, 16, 36, 64, 100]
30
```

> **Try it.** Write `compose(f, g)` that returns a function computing `f(g(x))`. Test it with `compose(addTen, square)(3)`, which should be `19`.

## Where does the time go?

When a program is slow, measure before you change it. Run it with `--profile` (it runs on the faster bytecode engine) and, after the program's own output, lizard-script prints every function with how many times it was called and the time spent in it. **total** is the time from the call to its return; **self** leaves out the time of the functions it called. Anonymous functions show as `function@line`. The times change from run to run; the calls don't.

Save this as `slow.liz` and run `lizardscript --profile slow.liz`:

```
function fib(n) {
  if (n < 2) { return n }
  fib(n - 1) + fib(n - 2)
}
let square = function(x) { x * x }
let total = 0
for (let i = 0; i < 1000; i += 1) { total += square(i) + len(str(i)) }
puts(fib(20), total)
```

**Output** (the profile goes to the error output, after the program's):

```text
6765 332836390
profile: 2.77ms

functions by self time
        self        total      calls  name
     1.878ms      1.878ms      21891  fib (slow.liz:1)
     0.300ms      2.297ms          1  program (slow.liz:1)
     0.023ms      0.023ms       1000  square (slow.liz:5)

functions by total time
        self        total      calls  name
     0.300ms      2.297ms          1  program (slow.liz:1)
     1.878ms      1.878ms      21891  fib (slow.liz:1)
     0.023ms      0.023ms       1000  square (slow.liz:5)

builtins by self time
        self        total      calls  name
     0.067ms      0.067ms       1000  str
     0.023ms      0.023ms       1000  len
     0.006ms      0.006ms          1  puts
```

`fib` is called 21891 times to compute `fib(20)`: almost all the time is there, and it is the function to make faster (it computes the same values again and again). `square` was called 1000 times but costs nothing.

> **Try it.** Write a `fib` that keeps the values it has computed in a hash, and profile it again. How many calls are left?

## Summary

- `function name(params) { body }` or `let name = function(params) { body }`.
- The last value is returned; `return` returns early; nothing returned means `null`.
- Defaults: `function f(a, b = 2)`. Defaults go last.
- A recursive function needs a base case.
- A closure remembers the variables of the scope where it was created.
- `lizardscript --profile file.liz` shows where the time goes: calls, self and total time of each function.

[← Previous: Loops](06-loops.md) · [Index](../README.md) · [Next: Arrays →](08-arrays.md)
