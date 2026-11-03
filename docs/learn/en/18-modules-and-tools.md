# 18. Modules and tools

[← Previous: The Terrarium](17-terrarium.md) · [Index](../README.md) · [Cheat sheet →](cheatsheet.md) · [Español](../es/18-modules-and-tools.md)

A program soon outgrows one file. This chapter splits a program into **modules** with `use`, then shows the tools around a program of several files: `lizardscript run`, `lizardscript test` and `lizardscript build`, which makes one executable of it. It ends with two modules of the Terrarium for programs that work with code and with other programs: `lizard` (the language itself, as data) and `io`.

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

Two modules that use each other can never finish loading, so that is an error too. With `plants/a.liz` saying `use "b"` and `plants/b.liz` saying `use "a"`:

**Output:**

```text
plants/b.liz: [line 1, column 1] runtime error: use cycle: plants/a.liz uses plants/b.liz, which uses plants/a.liz
```

Move what both need into a third module that both use.

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
- `--vm` makes the program run on the bytecode VM.
- Every module the program uses must be inside its folder.

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
[{"kind": "syntax", "message": "expected identifier, got \"=\"", "line": 1, "column": 5}]
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

## Summary

- `use "path"` runs a module once per program, in its own scope, and lets the file see its names, alive: `=` changes the module's variable, a `let` hides it. Names that start with `_` are private.
- `use "path" as name` gives a hash of the module's names, made when it has run.
- Paths are relative to the file that says the `use`, then the folders of `LIZARD_PATH`. A `use` goes at the top level of a file.
- `lizardscript test dir` runs every program with a `.expected` file and compares; `--update`, `--vm`, `--bench`.
- `lizardscript build dir -o file` makes one executable with the modules and the files of `bundle.txt`; find your files with `os.program_dir()`. `--os` and `--arch` build for other systems.
- `lizard.tokens`, `lizard.parse`, `lizard.run` and `lizard.docs` read, check and run code; `io` reads lines and writes to the output and the error output.

[← Previous: The Terrarium](17-terrarium.md) · [Index](../README.md) · [Cheat sheet →](cheatsheet.md)
