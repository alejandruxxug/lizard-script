# 18. Modules and tools

[← Previous: The Terrarium](17-terrarium.md) · [Index](../README.md) · [Next: lily2d →](19-lily2d.md) · [Español](../es/18-modules-and-tools.md)

A program soon outgrows one file. This chapter splits a program into **modules** with `use`, then shows the tools around a program of several files: `lizardscript run`, `lizardscript test`, `lizardscript build`, which makes one executable of it, and `lizardscript fmt` and `lizardscript check`, which tidy code and find mistakes in it before it runs. It ends with two modules of the Terrarium for programs that work with code and with other programs: `lizard` (the language itself, as data) and `io`.

The examples live in a folder called `garden`:

```text
garden/
  main.liz
  plants/
    flowers.liz
    watering.liz
```

## `use`: names from another file

`plants/flowers.liz` is an ordinary lizard-script file:

```
-~> plants/flowers.liz
let petals = 5

function flower(name) {
  return name + " with " + str(petals) + " petals"
}
```

`main.liz` uses it:

```
use "plants/flowers"

puts(flower("daisy"))
puts(petals)
```

Run the folder with `lizardscript run garden` (it runs `garden/main.liz`), or the file itself with `lizardscript garden/main.liz`.

**Output:**

```text
daisy with 5 petals
5
```

- `use "plants/flowers"` runs the file `plants/flowers.liz` and lets this file see every name it defines at its top level: `flower` and `petals`. The path uses `/` on every system, and `.liz` is added for you.
- The path is **relative to the file that says the `use`**, not to the folder you run the program from. A module can use another one next to itself: inside `plants/`, `use "watering"` is `plants/watering.liz`, and `use "../main_helpers"` goes one folder up.
- When the file isn't next to it, lizardscript looks in the folders of the environment variable `LIZARD_PATH` (separated by `:`, or `;` on Windows), so a library can live in one place for all your programs.
- A `use` goes at the **top level** of a file: not inside a function, an `if`, a loop or a layer.

### The names are alive

`plants/watering.liz` keeps a variable:

```
-~> plants/watering.liz
let water = 0
let _log = []

function pour(litres) {
  water += litres
  _log = _log + [litres]
  return water
}

function history() { return _log }
```

```
use "plants/watering"

pour(2)
pour(3)
puts(water)
water = 100
puts(pour(1))
puts(history())
let water = 7
puts(water, pour(1))
```

**Output:**

```text
5
101
[2, 3, 1]
7 102
```

- `water` is not a copy: reading it reads the module's own variable, so after two `pour`s it is `5`.
- `water = 100` changes the module's variable too (that is why the next `pour(1)` gives `101`). Every file that uses the module sees the same `water`.
- A `let` in your file makes a name of your own, which **hides** the module's: after `let water = 7`, `water` is yours, while `pour` still works on the module's (`102`).
- A module runs in a **scope of its own**: it doesn't see the variables of the files that use it, so a library can't break because your program happens to have a variable with the same name as one of its helpers.

### Private names

A name that starts with `_` stays inside its module:

```
use "plants/watering"
puts(_log)
```

**Output:**

```text
[line 2, column 6] runtime error: identifier not found: _log
```

The module's own functions use `_log` freely (`history()` gives it). Start the helpers of a library with `_` and only the names you mean to share are seen.

### `as`: a module in a hash

Two modules may have names that clash, or you may want to see where a name comes from. `use "..." as name` puts the module's names in a hash instead:

```
use "plants/flowers" as flowers
use "plants/watering" as w

puts(keys(flowers))
puts(flowers.flower("rose"))
w.pour(4)
puts(w.water, w.pour(1))
```

**Output:**

```text
["flower", "petals"]
rose with 5 petals
0 5
```

The hash is made **once, when the module has just run**. Its functions work on the module's live variables (`w.pour(1)` gives `5`), but a variable read through the hash is the value it had then: `w.water` is still `0`. Read state that changes through a function, or use the module without `as`.

### Once per program

However many files use a module, its file runs once:

```
-~> plants/loud.liz
puts("loading loud.liz")
let loud = true
```

```
-~> plants/garden_bed.liz
use "loud"
let bed = "ready"
```

```
use "plants/loud"
use "plants/garden_bed"
use "plants/loud" as again
puts(bed, loud, again.loud)
```

**Output:**

```text
loading loud.liz
ready true true
```

`garden_bed.liz` uses `loud.liz` too, and `again` is a third use of it, but the line is printed once. Names are not passed on: `main.liz` sees `loud` because it uses `plants/loud` itself, not because `garden_bed` does.

### When something goes wrong

A module that isn't there:

```
use "plants/cactus"
```

**Output:**

```text
[line 1, column 1] runtime error: can't find the module "plants/cactus": there is no plants/cactus.liz
```

An error inside a module shows the module's file in front, so you know where to look:

```
-~> plants/thirsty.liz
function drink(cups) {
  return cups * "water"
}
```

```
use "plants/thirsty"
puts(drink(2))
puts(drink("two"))
```

**Output:**

```text
waterwater
plants/thirsty.liz: [line 2, column 15] runtime error: unknown operator: STRING * STRING
```

A `use` inside a function is a syntax error:

```
function setup() {
  use "plants/flowers"
}
```

**Output:**

```text
[line 2, column 3] syntax error: use must be at the top level of a file, not inside a block or a layer
```

### Modules that use each other

Two modules can use each other: a garden that waters its plants, and a watering can that counts the garden's plants.

```
-~> plants/garden.liz
let plants = ["fern"]
use "water"
function plant(name) {
  plants = plants + [name]
  return water(name)
}
```

```
-~> plants/water.liz
use "garden"
let first = plants[0]
function water(name) {
  return "watered " + name + ", " + str(len(plants)) + " plants now"
}
```

```
use "plants/garden"
use "plants/water"
puts(first)
puts(plant("cactus"))
```

**Output:**

```text
fern
watered cactus, 2 plants now
```

When `garden.liz` reaches `use "water"`, it is still loading. `water.liz` runs then, and its `use "garden"` doesn't start the garden again: it sees the garden **as it is so far**, alive. `plants` is already there (it is defined above the `use`), so `first` is `"fern"`; `plant` isn't yet, and appears when the garden goes on. The functions call each other later, when everything has loaded, so they find every name.

The one thing to watch: a line that runs **while the files load** can't use a name defined further down the other file.

```
-~> plants/early.liz
use "late"
let sun = 3
```

```
-~> plants/late.liz
use "early"
let light = sun * 2
```

```
use "plants/early"
```

**Output:**

```text
plants/late.liz: [line 2, column 13] runtime error: identifier not found: sun: plants/early.liz defines it further down, but it is still loading (use cycle: plants/early.liz uses plants/late.liz, which uses plants/early.liz)
```

Move `let sun = 3` above the `use "late"`, or read `sun` inside a function, which runs later.

### Standard modules

Some modules come with lizard-script itself: they are built into `lizardscript`, so every program can use them, but they are only loaded by the programs that ask. They are called **standard modules**. The first ones are the **flower** family of graphics libraries, and the 2D one is `flower/lily2d` (chapter 19 is all about it):

```
use "flower/lily2d" as lily2d

let picture = img.new(40, 20, #ffffff)
lily2d.circle(picture, (10, 10), 6, #2d6a8f)
puts(img.get(picture, 10, 10))
puts(lily2d.measure("hi"))
```

**Output:**

```text
#2d6a8f
(14, 13)
```

- `use` looks for a standard module **last**: after the folder of the file and the folders of `LIZARD_PATH`. So if your program has its own `flower/lily2d.liz`, that one wins.
- Name it with `as`. `use "flower/lily2d" as lily2d` gives the hash `lily2d`; any name works (`as g` gives `g.circle`).
- A standard module's file is called `std:/` and its path, and its errors say so. The message still starts with the function you called:

```
use "flower/lily2d" as lily2d

let picture = img.new(40, 20, #ffffff)
lily2d.circle(picture, (10, 10), "six", #2d6a8f)
```

**Output:**

```text
std:/flower/lily2d/check.liz: [line 7, column 8] runtime error: lily2d.circle: r must be a number, not STRING
```

The line is the module's, not yours: look for the call named in the message (`lily2d.circle`) in your program. The [Terrarium reference](../../TERRARIUM.md#standard-modules) lists the standard modules.

## `lizardscript test`: programs that check themselves

A test is a program with a `.expected` file next to it that holds exactly what it should print (errors included). Put them in a folder:

```text
garden/tests/
  flowers.liz        use "../plants/flowers"
                     puts(flower("tulip"))
  flowers.expected   tulip with 5 petals
  watering.liz       use "../plants/watering"
                     puts(pour(2), pour(2))
  watering.expected  2 4
```

```text
$ lizardscript test garden/tests
ok    garden/tests/flowers.liz (1.3 ms)
ok    garden/tests/watering.liz (0.8 ms)
2 passed, 0 failed
```

- It runs every `.liz` of the folder (and the folders inside it) that has a `.expected` file; a `.liz` without one, like a module the tests share, is not a test.
- When the output differs, it shows both and the command ends with exit code 1, so a script or a CI job notices. With `watering.expected` saying `2 5`:

```text
FAIL  garden/tests/watering.liz
--- expected:
2 5
--- got:
2 4
1 passed, 1 failed
```

- `--update` writes what each program prints into its `.expected` file. Use it for a new test, and read the file before you keep it.
- `--vm` runs the tests on the bytecode VM, which must give the same output.
- A file ending in `_bench.liz` is a benchmark: with `--bench` it runs too and its time is shown (`bench garden/tests/count_bench.liz 27.0 ms`). Each run takes its own time, so yours will differ.

## `lizardscript build`: one file to share

`lizardscript build garden -o garden-app` makes **one executable file** of the program: a copy of `lizardscript` with `main.liz`, every module it uses, and the data files you list, inside it. Whoever gets the file runs it, with nothing installed.

The data files go in `garden/bundle.txt`, one path or pattern per line (a folder takes everything in it):

```text
-~> the files the program reads
notes
```

A program finds its own files from `os.program_dir()`, the folder of its main file:

```
use "plants/flowers"

puts(flower("daisy"))
puts(fs.read(fs.join(os.program_dir(), "notes/care.txt")))
```

```text
$ lizardscript build garden -o garden-app
built garden-app with main.liz, notes/care.txt, plants/flowers.liz
$ ./garden-app
daisy with 5 petals
Water on Mondays.

```

- Inside the built file, `os.program_dir()` is `"bundle:"` and its files are at `bundle:/notes/care.txt` and so on: `fs.read`, `fs.list`, `fs.exists` and `fs.info` read them, but nothing can change them. The same program works built or not, as long as it finds its files through `os.program_dir()`.
- `--os` and `--arch` build for another system: `lizardscript build garden -o garden.exe --os windows --arch amd64`, or `--os linux --arch arm64`. For that, lizardscript builds itself for that system with Go, so it needs Go and its own source (run it inside the lizard-script folder, or set `LIZARD_SRC`); `--runtime file` gives it a lizardscript already built for that system instead.
- `--icon icon.png` gives the program an icon: a square PNG of 256 pixels or more (1024 is best), and `--icon-small small.png` a hand-drawn one for 16 pixels. On Windows the icon goes inside the `.exe`. On Linux, `icon.png`, `icon-256.png` and a `.desktop` template are written next to the program. On macOS the output is an app folder, so name it `.app`: `lizardscript build garden -o Garden.app --icon icon.png` makes `Garden.app` (with its icon, and signed ad hoc when `codesign` is installed). The program's files are inside the app, so open it with `open Garden.app`. In the program, `window.open(w, h, title, {"icon": path})` sets the window's icon.
- `--vm` makes the program run on the bytecode VM.
- Every module the program uses must be inside its folder.

## `lizardscript fmt`: one way to lay out code

`lizardscript fmt` lays out code one way: the same program comes out the same, whoever wrote it, so nobody has to argue about spaces. Here is `pond.liz`, written in a hurry:

```
let frogs = [ "ana","leo" ]
function count(list)
{
    let total=0
    for(let i=0;i<len(list);i+=1){ total += 1 }
    return total
}
if (count(frogs) > 1) {
puts("a crowd") -~> more than one
}
else { puts("alone") } -~> just one
let pond = {"frogs": frogs, "lilies": 3,
            "deep": true}



puts(pond.lilies)
```

```text
$ lizardscript fmt pond.liz
let frogs = ["ana", "leo"]
function count(list) {
  let total = 0
  for (let i = 0; i < len(list); i += 1) { total += 1 }
  return total
}
if (count(frogs) > 1) {
  puts("a crowd")        -~> more than one
} else { puts("alone") } -~> just one
let pond = {
  "frogs": frogs,
  "lilies": 3,
  "deep": true
}

puts(pond.lilies)
```

- Two spaces for each block, a space around each operator and after each comma, the `{` of a block on the line of its `if` or `function`, `} else {` on one line, a hash that spans lines with one pair per line, the comments at the end of lines one below the other, and never two blank lines in a row. [STYLE.md](../../STYLE.md) has every rule.
- It prints the result. `-w` writes it back into the files, and `--check` only lists the files that would change and ends with exit code 1 if there is any, for a script that checks a project. A folder means every `.liz` in it and in the folders inside it: `lizardscript fmt -w garden`.
- It only moves spaces, and line breaks where the parser reads a line break as a space, so the program does exactly what it did before. It doesn't split a long line for you, and a short block stays on its line (`if (a) { 1 }`, then `else { 2 }` on the next one, stays like that).
- A file with a syntax error is left as it is: fmt shows the error and ends with exit code 2.

## `lizardscript check`: mistakes before it runs

`lizardscript check` reads a program without running it and tells you what will go wrong. `bugs.liz`:

```
let frogs = ["ana", "leo"]

function area(w, h) {
  let unit = "cm"
  return w * h
}

puts(area(3))
puts(len(forgs))
puts(math.sine(1))
if (len(frogs) == 2) { total = 5 }
```

```text
$ lizardscript check bugs.liz
bugs.liz: [line 4, column 7] warning: unit is never used (unused)
    hint: start its name with _ if that is on purpose
bugs.liz: [line 8, column 6] error: wrong number of arguments to area: want 2, got 1 (arguments)
bugs.liz: [line 9, column 10] error: identifier not found: forgs (unknown-name)
    hint: did you mean `frogs`?
bugs.liz: [line 10, column 6] error: math has no sine (member)
    hint: did you mean `math.sin`?
bugs.liz: [line 11, column 24] error: can't assign to total: it was never declared, use let total = ... (assign)
```

- An **error** is something the program would hit when it runs, or a syntax error that stops it from running at all: a name that is defined nowhere, `=` to a name that has no `let`, a call with the wrong number of arguments, a function a module doesn't have, a `use` whose file isn't there, a `return` whose value is on the next line.
- A **warning** is probably a mistake: a variable or a parameter that is never used (start its name with `_` to say it's on purpose), a `let` that hides a variable of a scope around it, code after a `return`, `break` or `continue` that never runs, `==` between two literals, `x = x`, a `use` none of whose names are used.
- The word in parentheses names the check. It ends with exit code 1 when it finds an error, so a script can stop there; warnings alone end with 0. `--json` writes the findings as JSON, for an editor.
- It knows when a `let` has run. A function may use a name whose `let` comes further down, because the function runs later; reading a name in the same function before its `let` is a mistake: `identifier not found: x: its let, at line 4, comes later`.
- It reads the modules a program uses, so it knows their names: `use "plants/flowers"` and then `flowr("rose")` is an unknown name with the hint `` did you mean `flower`? ``.
- No error doesn't mean the program can't fail: check doesn't run it, so a wrong type or a missing file shows up only when it runs.

A program that runs inside another one (an app that runs its users' sketches, say) uses names the app gives it, which check can't know about:

```text
$ lizardscript check sketches
sketches/sky.liz: [line 1, column 1] error: identifier not found: canvas (unknown-name)
sketches/sky.liz: [line 1, column 13] error: identifier not found: brush (unknown-name)
    hint: did you mean `push`?
$ lizardscript check --names canvas,brush sketches
```

`--names` lists them for one run. To say it once for a whole folder, put a `lizard-check.liz` in it: one hash, which check reads and never runs, and which counts for the files of that folder and every folder inside it (the nearest one wins):

```
-~> The names the painting app gives every program of this folder.
{"names": ["canvas", "brush"]}
```

The editor (`lizardscript lsp`) reads it too.

## Editors: `lizardscript lsp`

`lizardscript lsp` is a language server: an editor starts it and asks it about the code while you write. It shows what `lizardscript check` finds, underlined, as you type; formats the file with `lizardscript fmt` (on save, if you like); completes names, with what each one is (after `math.` the functions of `math`, after `lily2d.` the ones of the module, after `config.` the keys of your hash); shows a function's signature and its doc comment when the mouse is over its name, and what a literal is worth (`90deg` is `1.5708 rad`); jumps to where a name is declared, also in another file; renames a name everywhere it means the same thing; and puts a color picker on every color.

```
-~> The area of a rectangle.
-~> w: its width
-~> h: its height
-~> example: area(3, 4)
function area(w, h = 1) {
  return w * h
}
```

The comment above the function is what the editor shows for `area`, read the way `lizard.docs` reads it: the first lines are the summary, a `name: ...` line explains a parameter, and `example:` is an example. Writing them is how your functions explain themselves in every file that uses them.

- **VS Code**: the extension is in `editors/vscode`. `npm install` and `npm run package` there make `lizard-script.vsix`; install it with `code --install-extension lizard-script.vsix`. Its commands run the file (the ▷ button), run it on the VM, format it and check the whole folder.
- **Zed**: the extension is in `editors/zed`. In Zed, **zed: install dev extension** and pick that folder.
- Both need the `lizardscript` command on your `PATH` (or its full path in their settings). Any other editor with an LSP client (Neovim, Helix, ...) can start `lizardscript lsp` too: [editors/README.md](../../../editors/README.md) has the settings.

## `lizard`: code as data

The `lizard` module reads lizard-script code the way the language does, so an editor, a checker or a teaching tool can be written in lizard-script.

### Tokens

```
let code = "let size = 10 * 2 -~> big"
for (let i = 0; i < len(lizard.tokens(code)); i += 1) {
  let t = lizard.tokens(code)[i]
  puts(t.kind, t.text, t.column)
}
```

**Output:**

```text
keyword let 1
name size 5
operator = 10
number 10 12
operator * 15
number 2 17
comment -~> big 19
```

Each token also has its `type` (like `"IDENT"`), its `value`, and where it ends (`end_line`, `end_column`). The `kind` is what a syntax highlighter colors by.

A program that walks the tokens of a big file asks for them as columns: one hash of lists, the same keys, one element per token. It skips making a hash for every token, which is about four times faster on a file of 200,000 tokens:

```
let columns = lizard.tokens("let size = 10 * 2", {"columns": true})
puts(columns.kind)
puts(columns.text[1], columns.column[1], len(columns.type))
```

**Output:**

```text
["keyword", "name", "operator", "number", "operator", "number"]
size 5 6
```

### The syntax tree

```
let tree = lizard.parse("let size = 10 * 2")
let let_node = tree.program.statements[0]
puts(let_node.kind, let_node.name.value)
let product = let_node.value
puts(product.kind, product.operator, product.left.value, product.right.value)
puts(product.column, product.end_column)
puts(lizard.parse("let = 1").errors)
```

**Output:**

```text
let size
infix * 10 2
12 18
[{"kind": "syntax", "message": "let needs a name before the \"=\", like let x = 1", "line": 1, "column": 5}]
```

Every node is a hash with its `kind`, its fields and its positions: `10 * 2` goes from column 12 to just before column 18. [TERRARIUM.md](../../TERRARIUM.md#lizard) lists every kind of node.

### Running code in a sandbox

`lizard.run(code, options)` runs code in a fresh, empty scope: it sees only the builtins and what you give it in `globals`, and whatever it does, your program goes on as before.

```
let user_code = "let area = w * h\nputs(\"area:\", area)"
let r = lizard.run(user_code, {"globals": {"w": 3, "h": 4}})
puts(r.output)
puts(r.globals.area, r.error)

let forever = lizard.run("while (true) {}", {"steps": 1000})
puts(forever.error.message)

let broken = lizard.run("let x = 1\nx + \"a\"", {"name": "sketch"})
puts(broken.error)

let lines = []
lizard.run("let a = 1\nlet b = a + 1", {"on_statement": function(line, column, source) { lines = lines + [line] }})
puts(lines)
```

**Output:**

```text
area: 12

12 null
the program ran too long (more than 1000 steps)
{"kind": "runtime", "message": "type mismatch: INTEGER + STRING", "line": 2, "column": 3, "source": "sketch"}
[1, 2]
```

- What the code prints is kept in `output` instead of being printed, and its top-level names are in `globals` after the run.
- `steps` stops code that would run forever. An error doesn't stop your program: it comes back in `error`.
- `on_statement` is called before each statement runs, and `on_layer` when a layer starts and ends: an editor uses them to know which line made what.
- The Terrarium is not there unless you pass it: `{"globals": {"fs": fs}}` gives the code `fs`.

A library the code should use alive goes in `use`, not `globals`. A name given in `globals` is a copy: changing it changes only the code's copy. A module in `use` is the module itself, as if the code started with `use "plants/settings"`:

```
-~> plants/settings.liz
let pot_size = 2
function pot() { return "a pot of size " + str(pot_size) }
```

```
let r = lizard.run("pot_size = 5\npot()", {"name": "sketch.liz", "use": ["plants/settings"]})
puts(r.value)

let lines = []
lizard.run("use \"plants/settings\"\nlet p = pot()", {"name": "sketch.liz", "hook_source": "sketch.liz",
  "on_statement": function(line, column, source) { lines = lines + [line] }})
puts(lines)
```

**Output:**

```text
a pot of size 5
[1, 2]
```

- The modules are found from the code's `name`, like its own `use` lines.
- `hook_source` calls `on_statement` only for the statements of that source: the code's own two lines, not the line inside `pot`. An editor that runs a drawing on every change saves a call per library statement.
- Running the same libraries again and again is cheap: a module file is parsed (and compiled, on the VM) once while it doesn't change.

Two more things an editor needs: every mistake of a text that doesn't parse, and the statements of the code itself, not of the functions it calls:

```
let typos = lizard.run("let = 1\nlet ok = 2\nlet y = 5kg", {"name": "sketch"})
puts(typos.error.line, len(typos.errors))
for (let i = 0; i < len(typos.errors); i += 1) {
  puts(typos.errors[i].line, typos.errors[i].message)
}

let lines = []
let code = "function twice(n) {\n  return n * 2\n}\nlet a = twice(1)\nlet b = twice(a)"
lizard.run(code, {"hook_functions": false, "on_statement": function(line, _column, _source) { lines = lines + [line] }})
puts(lines)
```

**Output:**

```text
1 2
1 let needs a name before the "=", like let x = 1
3 unknown unit "kg"
[1, 4, 5]
```

- `errors` lists every syntax error, in order; `error` is the first of them. When the code ran and failed, `errors` has that one error, and when it worked, none.
- With `"hook_functions": false`, `on_statement` is called only for the statements outside every function: line 2, inside `twice`, runs twice but is never reported. The statements in a layer still are.

### Checking code

`lizard.check(code)` finds mistakes in code the way `lizardscript check` does, and gives them as hashes, for an editor that underlines them:

```
let found = lizard.check("let size = 10\nputs(siz * 2)")
puts(len(found))
puts(found[0].severity, found[0].message)
puts(found[0].hint, found[0].line, found[0].column)
puts(lizard.check("draw(canvas)", "", ["draw", "canvas"]))
```

**Output:**

```text
1
error identifier not found: siz
did you mean `size`? 2 6
[]
```

Each finding also has its `rule` and where it ends (`end_line`, `end_column`). The code knows the names of the Terrarium; the third argument lists more names the code will have when it runs, like the ones an editor gives the programs it runs.

### Doc comments

`lizard.docs(code)` reads the `-~>` lines right above each function:

```
let library = "-~> A flower with its petals.
-~> name: what it is called
-~> example: flower(\"daisy\")
function flower(name, petals = 5) { name }"
let docs = lizard.docs(library)
puts(docs.flower.summary)
puts(docs.flower.params)
puts(docs.flower.example)
```

**Output:**

```text
A flower with its petals.
[{"name": "name", "default": null, "doc": "what it is called"}, {"name": "petals", "default": "5", "doc": null}]
flower("daisy")
```

## `io`: talking to another program

`io.read_line()` reads one line of the program's input (`null` when it ends), `io.write(text)` prints without adding a line break, and `io.error(text)` writes to the error output, which stays apart from the output another program reads:

```
let line = io.read_line()
while (line != null) {
  io.write(text.upper(line) + "!\n")
  line = io.read_line()
}
io.error("done\n")
```

```text
$ printf 'hello\nlizards\n' | lizardscript shout.liz
HELLO!
LIZARDS!
done
```

`done` went to the error output: with `2>/dev/null` at the end of the command, only the two lines are left.

## `task`: work in the background

A program with a window draws a frame many times a second, so a computation that takes a second freezes it. `task.start(module, function, args)` runs a function of a module in the background instead, in a fresh interpreter of its own, and gives back a handle at once. The module is found like a `use`, from the file that calls `task.start`:

```
-~> count.liz: how many numbers below n are prime, slow on purpose.
function primes_below(n) {
  let count = 0
  for (let k = 2; k < n; k += 1) {
    let prime = true
    for (let d = 2; d * d <= k; d += 1) {
      if (k % d == 0) {
        prime = false
        break
      }
    }
    if (prime) {
      count += 1
    }
  }
  puts("counted up to " + str(n))
  count
}
```

```
let small = task.start("count", "primes_below", [1000])
let big = task.start("count", "primes_below", [200000])
puts(small.state())
puts(small.result(), big.result())
puts(big.state(), big.done())

let endless = task.start("count", "primes_below", [1000000000])
endless.cancel()
puts(endless.state())
try {
  endless.result()
} catch (e) {
  puts(e["message"])
}
try {
  task.start("count", "primes_below", [function() { 1 }])
} catch (e) {
  puts(e["message"])
}
```

```text
running
[task] counted up to 1000
[task] counted up to 200000
168 17984
done true
cancelled
task.result: the task was cancelled
task.start: args[0] is a FUNCTION, and only data crosses to and from a task (numbers, strings, booleans, null, lists, hashes, points and colors)
```

- `t.result()` waits for the task and gives what its function returned. `t.done()` says whether it has ended without waiting, which is what a window's `tick` asks every frame; `t.state()` is `"running"`, `"done"`, `"failed"` or `"cancelled"`.
- `t.cancel()` stops it at once, even in the middle of a loop. Its `result()` is then an error.
- An error inside the task comes out of `t.result()`, with the file and line where it happened.
- Nothing is shared: the arguments and the result are copied, so a task that changes a hash changes its own copy. Only data goes across (numbers, strings, booleans, `null`, lists, hashes, points and colors); a function is an error that says where it was. A task that needs an image gets its path and loads it.
- Inside a task there is no `window`, `pen` or `clip` (they are the error `not available inside a task`), and what it prints goes to the error output with `[task] ` in front, so the order of those lines next to the program's own can change from run to run.
- Each task is a process of its own, so several run at the same time on the machine's processors; starting one costs about as much as starting `lizardscript`, so it is for work that takes a while. When the program ends, its tasks end too.
- In a built program, a module that only `task.start` names must also be used somewhere (`use "count" as count`), so `lizardscript build` takes it.

## Summary

- `use "path"` runs a module once per program, in its own scope, and lets the file see its names, alive: `=` changes the module's variable, a `let` hides it. Names that start with `_` are private.
- `use "path" as name` gives a hash of the module's names, made when it has run.
- Paths are relative to the file that says the `use`, then the folders of `LIZARD_PATH`, then the standard modules (`flower/lily2d`). A `use` goes at the top level of a file.
- `lizardscript test dir` runs every program with a `.expected` file and compares; `--update`, `--vm`, `--bench`.
- `lizardscript build dir -o file` makes one executable with the modules and the files of `bundle.txt`; find your files with `os.program_dir()`. `--os` and `--arch` build for other systems.
- `lizardscript fmt` lays code out one way (`-w` writes it, `--check` lists what would change); `lizardscript check` finds errors and warnings without running the program (`--json` for editors; `--names` or a `lizard-check.liz` for the names a host gives).
- `lizardscript lsp` is the language server: the VS Code and Zed extensions of `editors/` start it, for diagnostics, formatting, completion, hover, go to definition and a color picker.
- `lizard.tokens`, `lizard.parse`, `lizard.check`, `lizard.run` and `lizard.docs` read, check and run code; `io` reads lines and writes to the output and the error output.
- `task.start(module, function, args)` runs a function in the background, in a process of its own; `done`, `state`, `result` and `cancel` follow it, and only data crosses.

[← Previous: The Terrarium](17-terrarium.md) · [Index](../README.md) · [Next: lily2d →](19-lily2d.md)
