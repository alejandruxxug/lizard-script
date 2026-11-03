# Exercises

[Index](../README.md) · [Cheat sheet](cheatsheet.md) · [Español](../es/exercises.md)

Twenty exercises, roughly in the order of the chapters. Try each one before opening the solution. Every solution was run, and the output shown is the real one. There are many correct answers; yours doesn't have to match.

---

### 1. Hello, you (chapters 1–2)

Store your name and your age in variables and print `My name is Liz and I am 3 years old.`, with your values, as one string built with `+`.

<details><summary>Solution</summary>

```
let name = "Liz"
let age = 3
puts("My name is " + name + " and I am " + str(age) + " years old.")
```

**Output:**

```text
My name is Liz and I am 3 years old.
```

`age` is a number, so it needs `str` before joining it with `+`.
</details>

### 2. Seconds (chapter 4)

How many seconds are there in a week? And how many whole weeks fit in 10,000,000 seconds, with how many seconds left over?

<details><summary>Solution</summary>

```
let week = 7 * 24 * 60 * 60
puts(week)
puts(10000000 // week, "weeks and", 10000000 % week, "seconds")
```

**Output:**

```text
604800
16 weeks and 323200 seconds
```
</details>

### 3. Predict the output (chapter 4)

Without running it, write down what each line prints. Then check.

```
puts(9 / 2, 9 / 2.0, -9 / 2, -9 // 2)
puts(2 ** 2 ** 3, -3 ** 2)
puts(1 == "1", 1 === "1", 0 == false, "" == 0)
```

<details><summary>Solution</summary>

```
puts(9 / 2, 9 / 2.0, -9 / 2, -9 // 2)
puts(2 ** 2 ** 3, -3 ** 2)
puts(1 == "1", 1 === "1", 0 == false, "" == 0)
```

**Output:**

```text
4 4.5 -4 -5
256 -9
true false true false
```

`2 ** 2 ** 3` is `2 ** 8`. `-3 ** 2` is `-(3 ** 2)`. `""` is not a number string, so it is never `==` to a number.
</details>

### 4. Fix the bugs (chapters 3 and 6)

This program should print the sum of 1 to 5, but it has two mistakes. Find them from the error messages.

```
let total = 0
for (i = 1; i <= 5; i += 1) {
  totl += i
}
puts(total)
```

**Output:**

```text
[line 2, column 6] runtime error: can't assign to i: it was never declared, use let i = ...
```

<details><summary>Solution</summary>

The `for` header needs `let i`. After fixing that, the next error would be `identifier not found: totl` (a typo).

```
let total = 0
for (let i = 1; i <= 5; i += 1) {
  total += i
}
puts(total)
```

**Output:**

```text
15
```
</details>

### 5. Grade (chapter 5)

Write `grade(score)` that returns `"A"` for 90 or more, `"B"` for 80 to 89, `"C"` for 70 to 79 and `"F"` otherwise.

<details><summary>Solution</summary>

```
function grade(score) {
  if (score >= 90) { "A" } elif (score >= 80) { "B" } elif (score >= 70) { "C" } else { "F" }
}
puts(grade(95), grade(85), grade(72), grade(10))
```

**Output:**

```text
A B C F
```

The `if` is the last statement, so its value is what the function returns.
</details>

### 6. Leap years (chapter 5)

A year is a leap year if it is divisible by 4, except years divisible by 100, unless they are also divisible by 400. Write `isLeap(year)`.

<details><summary>Solution</summary>

```
function isLeap(y) {
  (y % 4 == 0 && y % 100 != 0) || y % 400 == 0
}
puts(isLeap(2024), isLeap(1900), isLeap(2000), isLeap(2026))
```

**Output:**

```text
true false true false
```
</details>

### 7. Countdown with `while` (chapter 6)

Print `10 9 8 ... 1 lift off!` on **one** line.

<details><summary>Solution</summary>

```
let line = ""
let n = 10
while (n > 0) {
  line += str(n) + " "
  n -= 1
}
puts(line + "lift off!")
```

**Output:**

```text
10 9 8 7 6 5 4 3 2 1 lift off!
```
</details>

### 8. Sum of the odd numbers (chapter 6)

Using `continue`, add up the odd numbers from 1 to 99.

<details><summary>Solution</summary>

```
let sum = 0
for (let i = 1; i < 100; i += 1) {
  if (i % 2 == 0) { continue }
  sum += i
}
puts(sum)
```

**Output:**

```text
2500
```
</details>

### 9. First divisor (chapter 6)

Write `smallestDivisor(n)` that returns the smallest number greater than 1 that divides `n`, using a loop with `break`. Use it to write `isPrime(n)`.

<details><summary>Solution</summary>

```
function smallestDivisor(n) {
  let d = 2
  while (d * d <= n) {
    if (n % d == 0) { break }
    d += 1
  }
  if (d * d > n) { n } else { d }
}
function isPrime(n) { n > 1 && smallestDivisor(n) == n }
puts(smallestDivisor(91), smallestDivisor(13))
let primes = []
for (let i = 1; i <= 30; i += 1) {
  if (isPrime(i)) { push(primes, i) }
}
puts(primes)
```

**Output:**

```text
7 13
[2, 3, 5, 7, 11, 13, 17, 19, 23, 29]
```
</details>

### 10. Power, recursively (chapter 7)

Write `power(base, exp)` without using `**`, by recursion. `exp` is a whole number 0 or more.

<details><summary>Solution</summary>

```
function power(base, exp) {
  if (exp == 0) { return 1 }
  base * power(base, exp - 1)
}
puts(power(2, 10), power(3, 0), power(1.5, 2))
```

**Output:**

```text
1024 1 2.25
```
</details>

### 11. A greeting with a default (chapter 7)

Write `greet(name, punctuation = "!")` that returns `"Hello, Liz!"`, and `greet("Liz", "?")` gives `"Hello, Liz?"`.

<details><summary>Solution</summary>

```
function greet(name, punctuation = "!") {
  "Hello, " + name + punctuation
}
puts(greet("Liz"), greet("Liz", "?"))
```

**Output:**

```text
Hello, Liz! Hello, Liz?
```
</details>

### 12. Compose (chapter 7)

Write `compose(f, g)` that returns a new function computing `f(g(x))`.

<details><summary>Solution</summary>

```
function compose(f, g) {
  function(x) { f(g(x)) }
}
let addTen = function(x) { x + 10 }
let square = function(x) { x * x }
puts(compose(addTen, square)(3), compose(square, addTen)(3))
```

**Output:**

```text
19 169
```
</details>

### 13. A counter with steps (chapter 7)

Write `makeCounter(step = 1)` that returns a function; each call adds `step` and returns the new count.

<details><summary>Solution</summary>

```
function makeCounter(step = 1) {
  let count = 0
  function() {
    count += step
  }
}
let byOne = makeCounter()
let byFive = makeCounter(5)
byOne()
byFive()
puts(byOne(), byFive(), byFive())
```

**Output:**

```text
2 10 15
```

`count += step` is an expression whose value is the new count, so it is also the result of the function.
</details>

### 14. Maximum and average (chapter 8)

Write `maxOf(xs)` (without `list.sort`) and `average(xs)`, which must return a float.

<details><summary>Solution</summary>

```
function maxOf(xs) {
  let best = xs[0]
  for (let i = 1; i < len(xs); i += 1) {
    if (xs[i] > best) { best = xs[i] }
  }
  best
}
function average(xs) {
  let total = 0
  for (let i = 0; i < len(xs); i += 1) { total += xs[i] }
  float(total) / len(xs)
}
let xs = [4, 8, 15, 16, 23, 42]
puts(maxOf(xs), average(xs))
```

**Output:**

```text
42 18.0
```
</details>

### 15. Shared or copied? (chapter 8)

Predict the output, then check.

```
let a = [1, 2]
let b = a
let c = list.copy(a)
push(b, 3)
push(c, 4)
puts(a, b, c)
```

<details><summary>Solution</summary>

```
let a = [1, 2]
let b = a
let c = list.copy(a)
push(b, 3)
push(c, 4)
puts(a, b, c)
```

**Output:**

```text
[1, 2, 3] [1, 2, 3] [1, 2, 4]
```

`b` is the same array as `a`; `c` is a separate copy.
</details>

### 16. Counting words (chapter 9)

Write `countWords(words)` that takes an array of strings and returns a hash with how many times each one appears.

<details><summary>Solution</summary>

```
function countWords(words) {
  let counts = {}
  for (let i = 0; i < len(words); i += 1) {
    let w = words[i]
    if (counts[w] === null) { counts[w] = 0 }
    counts[w] += 1
  }
  counts
}
puts(countWords(["a", "b", "a", "c", "a"]))
```

**Output:**

```text
{"a": 3, "b": 1, "c": 1}
```
</details>

### 17. Capitalize (chapter 10)

Write `capitalize(s)` that makes the first letter uppercase and the rest lowercase. Then `titleCase(s)` that does it for every word.

<details><summary>Solution</summary>

```
function capitalize(s) {
  text.upper(text.slice(s, 0, 1)) + text.lower(text.slice(s, 1, len(s)))
}
function titleCase(s) {
  let words = text.split(s, " ")
  let out = []
  for (let i = 0; i < len(words); i += 1) { push(out, capitalize(words[i])) }
  text.join(out, " ")
}
puts(capitalize("lIZARD"))
puts(titleCase("the lizard and the sun"))
```

**Output:**

```text
Lizard
The Lizard And The Sun
```
</details>

### 18. Temperature table (chapter 11)

Print Celsius from 0 to 100 in steps of 25, with Fahrenheit (`c * 9 / 5 + 32`) as a float rounded to one decimal.

<details><summary>Solution</summary>

```
for (let c = 0; c <= 100; c += 25) {
  let f = c * 9 / 5.0 + 32
  puts(c, "C =", math.round(f * 10) / 10.0, "F")
}
```

**Output:**

```text
0 C = 32.0 F
25 C = 77.0 F
50 C = 122.0 F
75 C = 167.0 F
100 C = 212.0 F
```

Dividing by `5.0` makes it a float. To round to one decimal, multiply by 10, round, and divide by `10.0`.
</details>

### 19. Midpoint and a lighter color (chapter 12)

Write `midpoint(p, q)` and `lighter(c)`, which moves every channel of a color halfway towards 255.

<details><summary>Solution</summary>

```
function midpoint(p, q) { (p + q) / 2.0 }
function lighter(c) {
  rgb((c.r + 255) / 2.0, (c.g + 255) / 2.0, (c.b + 255) / 2.0)
}
puts(midpoint((0, 0), (10, 5)))
puts(lighter(#3fa34d), lighter(#000000))
```

**Output:**

```text
(5.0, 2.5)
#9fd1a6 #808080
```

`rgb` rounds float channels to the nearest integer.
</details>

### 20. Safe division (chapter 14)

Write `safeDivide(a, b)` that returns `a / b`, or the string `"undefined"` when the division fails, using `try`/`catch`.

<details><summary>Solution</summary>

```
function safeDivide(a, b) {
  try { a / b } catch { "undefined" }
}
puts(safeDivide(10, 2), safeDivide(1, 0), safeDivide(1.0, 0.0))
```

**Output:**

```text
5 undefined undefined
```

The `try` is the last statement of the function, so its value is the result.
</details>

---

[Index](../README.md) · [Cheat sheet](cheatsheet.md)
