# 16. Projects

[← Previous: Builtins reference](15-builtins-reference.md) · [Index](../README.md) · [Cheat sheet →](cheatsheet.md) · [Español](../es/16-projects.md)

Four small programs that put the chapters together. Each one is built in steps: read a step, try to write the next one yourself, then compare. Save each program in a file and run it with `./lizardscript file.liz`.

## Project 1: FizzBuzz

**Goal:** print the numbers from 1 to 15, but write `Fizz` for multiples of 3, `Buzz` for multiples of 5 and `FizzBuzz` for multiples of both.

**Uses:** `for`, `%`, `if`/`elif`/`else` (chapters 4 to 6).

The trick is the order of the checks: a multiple of 15 is also a multiple of 3, so it must be checked **first**.

```
for (let i = 1; i <= 15; i += 1) {
  if (i % 15 == 0) {
    puts("FizzBuzz")
  } elif (i % 3 == 0) {
    puts("Fizz")
  } elif (i % 5 == 0) {
    puts("Buzz")
  } else {
    puts(i)
  }
}
```

**Output:**

```text
1
2
Fizz
4
Buzz
Fizz
7
8
Fizz
Buzz
11
Fizz
13
14
FizzBuzz
```

**Step 2:** turn it into a function that **returns** the word instead of printing it, so it can be reused and tested. Then build the whole line with `text.join`:

```
function fizzbuzz(n) {
  if (n % 15 == 0) { return "FizzBuzz" }
  if (n % 3 == 0) { return "Fizz" }
  if (n % 5 == 0) { return "Buzz" }
  str(n)
}

let words = []
for (let i = 1; i <= 15; i += 1) {
  push(words, fizzbuzz(i))
}
puts(text.join(words, " "))
```

**Output:**

```text
1 2 Fizz 4 Buzz Fizz 7 8 Fizz Buzz 11 Fizz 13 14 FizzBuzz
```

**Go further:** let the caller choose the rules with a hash, like `{3: "Fizz", 5: "Buzz", 7: "Bazz"}`.

## Project 2: A word counter

**Goal:** given a text, count how many times each word appears and show the most common ones.

**Uses:** strings and `text` (chapter 10), hashes (chapter 9), `list.sort` with a function (chapter 8).

**Step 1: split the text into clean words.** Lowercase everything, turn punctuation into spaces, split on spaces and drop the empty pieces:

```
function words(s) {
  let clean = text.lower(s)
  let marks = [",", ".", "!", "?", ";", ":", "\n"]
  for (let i = 0; i < len(marks); i += 1) {
    clean = text.replace(clean, marks[i], " ")
  }
  let out = []
  let pieces = text.split(clean, " ")
  for (let i = 0; i < len(pieces); i += 1) {
    if (pieces[i] != "") { push(out, pieces[i]) }
  }
  out
}
puts(words("The lizard sees the sun. The sun, the sun!"))
```

**Output:**

```text
["the", "lizard", "sees", "the", "sun", "the", "sun", "the", "sun"]
```

**Step 2: count them in a hash.** A missing key reads as `null`, so the first time a word appears we set it to 1:

```
function count(ws) {
  let counts = {}
  for (let i = 0; i < len(ws); i += 1) {
    let w = ws[i]
    if (counts[w] === null) {
      counts[w] = 1
    } else {
      counts[w] += 1
    }
  }
  counts
}
puts(count(["sun", "lizard", "sun"]))
```

**Output:**

```text
{"sun": 2, "lizard": 1}
```

**Step 3: sort by count and print the top.** Hashes have no order of their own besides insertion, so we turn the hash into an array of `{"word", "n"}` records and sort that:

```
function words(s) {
  let clean = text.lower(s)
  let marks = [",", ".", "!", "?", ";", ":", "\n"]
  for (let i = 0; i < len(marks); i += 1) {
    clean = text.replace(clean, marks[i], " ")
  }
  let out = []
  let pieces = text.split(clean, " ")
  for (let i = 0; i < len(pieces); i += 1) {
    if (pieces[i] != "") { push(out, pieces[i]) }
  }
  out
}

function count(ws) {
  let counts = {}
  for (let i = 0; i < len(ws); i += 1) {
    let w = ws[i]
    if (counts[w] === null) { counts[w] = 1 } else { counts[w] += 1 }
  }
  counts
}

function top(counts, n) {
  let rows = []
  let ks = keys(counts)
  for (let i = 0; i < len(ks); i += 1) {
    push(rows, {"word": ks[i], "n": counts[ks[i]]})
  }
  -~> more first; on a tie, alphabetical
  let sorted = list.sort(rows, function(a, b) {
    if (a.n != b.n) { a.n > b.n } else { a.word < b.word }
  })
  list.slice(sorted, 0, n)
}

let poem = "The lizard sees the sun.
The sun sees the lizard.
And the lizard, warm, sleeps in the sun!"

let ws = words(poem)
puts("words:", len(ws))
let best = top(count(ws), 3)
for (let i = 0; i < len(best); i += 1) {
  let row = best[i]
  puts(row.word + ":", "*" * row.n, row.n)
}
```

**Output:**

```text
words: 18
the: ****** 6
lizard: *** 3
sun: *** 3
```

**Go further:** ignore short words like "the", "and", "in" with a list of stop words; or show each word's share as a percentage (watch out for integer division!).

## Project 3: A tiny bank account with closures

**Goal:** an account whose balance can only change through `deposit` and `withdraw`, which refuse bad amounts.

**Uses:** closures (chapter 7), hashes of functions (chapter 9), `error` and `try`/`catch` (chapter 14).

The balance is a local variable of `makeAccount`. The functions in the returned hash are closures that share it, and nothing else in the program can reach it.

```
function makeAccount(owner, balance = 0) {
  let history = []

  function deposit(amount) {
    if (amount <= 0) { error("deposit must be positive, got " + str(amount)) }
    balance += amount
    push(history, "+" + str(amount))
    balance
  }

  function withdraw(amount) {
    if (amount > balance) {
      error(owner + " can't withdraw " + str(amount) + ", balance is " + str(balance))
    }
    balance -= amount
    push(history, "-" + str(amount))
    balance
  }

  {
    "owner": owner,
    "deposit": deposit,
    "withdraw": withdraw,
    "balance": function() { balance },
    "history": function() { list.copy(history) }
  }
}

let liz = makeAccount("Liz", 100)
let iggy = makeAccount("Iggy")

liz.deposit(50)
liz.withdraw(30)
iggy.deposit(10)
puts(liz.owner, liz.balance(), liz.history())
puts(iggy.owner, iggy.balance(), iggy.history())

try {
  iggy.withdraw(500)
} catch (e) {
  puts("refused:", e.message)
}
try {
  liz.deposit(-5)
} catch (e) {
  puts("refused:", e.message)
}
puts("Liz still has", liz.balance())
```

**Output:**

```text
Liz 120 ["+50", "-30"]
Iggy 10 ["+10"]
refused: Iggy can't withdraw 500, balance is 10
refused: deposit must be positive, got -5
Liz still has 120
```

Notice:

- `liz` and `iggy` each have their own `balance` and `history`, because each call to `makeAccount` creates new ones.
- `history` returns a **copy**, so a caller can't change the real history by pushing into it.
- A failed `withdraw` calls `error` **before** changing anything, so the balance stays right.

**Step 2: a transfer.** It must not lose money if the second half fails. Since `withdraw` checks first, we do it first:

```
function makeAccount(owner, balance = 0) {
  {
    "owner": owner,
    "balance": function() { balance },
    "deposit": function(amount) { balance += amount },
    "withdraw": function(amount) {
      if (amount > balance) { error(owner + " has only " + str(balance)) }
      balance -= amount
    }
  }
}

function transfer(from, to, amount) {
  from.withdraw(amount)
  to.deposit(amount)
}

let liz = makeAccount("Liz", 100)
let iggy = makeAccount("Iggy", 5)
transfer(liz, iggy, 40)
puts(liz.balance(), iggy.balance())
try {
  transfer(iggy, liz, 1000)
} catch (e) {
  puts("transfer failed:", e.message)
}
puts(liz.balance(), iggy.balance())
```

**Output:**

```text
60 45
transfer failed: Iggy has only 45
60 45
```

**Go further:** add an interest function that multiplies the balance by `1.05` (what happens to the type of `balance`?), or a limit on how many withdrawals per account.

## Project 4: A drawing in the terminal

**Goal:** describe a small scene with colors, points and layers, then draw it as text.

**Uses:** colors and points (chapter 12), `math` and units (chapter 11), layers (chapter 13), arrays and hashes.

**Step 1: shapes are hashes.** Each shape has a kind, a position, a color and the name of the group it belongs to. In a plain run the program isn't told which layer it is in, so we keep the group name in a variable that each layer sets.

**Step 2: a canvas.** The canvas is an array of rows, each row an array of characters. `plot` puts a character at a point, ignoring points outside.

**Step 3: draw each shape.** A circle is drawn by walking around it with `math.cos` and `math.sin`; a line by stepping from one point to the other.

```
let W = 32
let H = 12
let shapes = []
let group = ""

function add(shape) {
  shape.group = group
  push(shapes, shape)
}
function circle(center, r, color, mark) {
  add({"kind": "circle", "at": center, "r": r, "color": color, "mark": mark})
}
function line(from, to, color, mark) {
  add({"kind": "line", "at": from, "to": to, "color": color, "mark": mark})
}

-~> The scene. Layers name the groups; the indented lines belong to them.
layer sky
  group = "sky"
  circle((25, 3), 2, #ffcc00, "o")
layer ground
  group = "ground"
  line((0, 10), (31, 10), #3fa34d, "=")
  layer tree
    group = "tree"
    line((7, 9), (7, 7), #8b5a2b, "|")
    circle((7, 4), 2, hsl(120, 60, 30), "*")

-~> The canvas.
let canvas = []
for (let y = 0; y < H; y += 1) {
  let row = []
  for (let x = 0; x < W; x += 1) { push(row, " ") }
  push(canvas, row)
}

function plot(p, mark) {
  let x = math.round(p.x)
  let y = math.round(p.y)
  if (x >= 0 && x < W && y >= 0 && y < H) {
    canvas[y][x] = mark
  }
}

function draw(s) {
  if (s.kind == "circle") {
    for (let i = 0; i < 24; i += 1) {
      let a = i * 1turn / 24
      plot(s.at + (2 * s.r * math.cos(a), s.r * math.sin(a)), s.mark)
    }
  } elif (s.kind == "line") {
    let steps = math.max(math.abs(s.to.x - s.at.x), math.abs(s.to.y - s.at.y))
    for (let i = 0; i <= steps; i += 1) {
      plot(s.at + (s.to - s.at) * (float(i) / steps), s.mark)
    }
  }
}

for (let i = 0; i < len(shapes); i += 1) { draw(shapes[i]) }

for (let y = 0; y < H; y += 1) {
  puts(text.join(canvas[y], ""))
}

-~> A legend: one line per shape.
for (let i = 0; i < len(shapes); i += 1) {
  let s = shapes[i]
  puts(s.mark, s.kind, "in", s.group, "at", s.at, "color", s.color)
}
```

**Output:**

```text
                                
                       ooooo    
     *****           oo     oo  
   **     **         o       o  
   *       *         oo     oo  
   **     **           ooooo    
     *****                      
       |                        
       |                        
       |                        
================================
                                
o circle in sky at (25, 3) color #ffcc00
= line in ground at (0, 10) color #3fa34d
| line in tree at (7, 9) color #8b5a2b
* circle in tree at (7, 4) color #1f7a1f
```

A few things to notice:

- Circles are drawn twice as wide as they are tall (`2 * s.r` in x) because a character in the terminal is about twice as tall as it is wide.
- `s.at + (s.to - s.at) * t` is the classic formula for a point a fraction `t` of the way along a line. It works directly on points.
- `float(i) / steps` matters: with two integers, `i / steps` would be `0` for every step but the last.
- The `tree` layer is inside `ground`, so it is drawn after the ground line and on top of it.

**Go further:** add a `rect(corner, size, color, mark)` shape; make the sun's rays with eight `line`s at `i * 45deg`; or sort `shapes` with `list.sort` so that a `"z"` key decides what is drawn on top.

[← Previous: Builtins reference](15-builtins-reference.md) · [Index](../README.md) · [Cheat sheet →](cheatsheet.md)
