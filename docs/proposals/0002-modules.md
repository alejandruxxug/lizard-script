# LEP 0002: Modules with `use`

- **Author(s):** the lizard-script team
- **Status:** Final <!-- Draft · Accepted · Rejected · Deferred · Final -->
- **Created:** 2026-11-03
- **Decision:** Accepted 2026-11-03, with live names, `as` for a hash, and `_` for private names.
- **Discussion:** to be opened as a pull request

## Summary

`use "draw/shapes"` runs the file `draw/shapes.liz` (next to the file that says it, or in a folder of the module path) once per run, in a top scope of its own, and lets the file see every name the module defined, alive: reading one reads the module's variable and `=` changes it. `use "draw/shapes" as shapes` gives a hash of those names instead. Names that start with `_` stay inside the module. A use is only allowed at the top level of a file, cycles are an error (allowed since Amendment 2), and a module's errors carry its file name.

## Motivation

A program of more than one file had no way to say so. Every host that wanted libraries loaded the files itself, in an order it kept by hand, into one environment, where every top-level name of every file could clash with every other:

```
-~> today: the host runs geometry.liz, then shapes.liz, then the program,
-~> all in one environment, and the program has no line that says so.
puts(area(square(3)))      -~> where do area and square come from?
```

A tool that bundles a program into one file can't know which files it needs, and an error in a library is reported at a line of a file nobody can name.

## The change

```
use "draw/shapes"             -~> every name of draw/shapes.liz
use "draw/colors" as colors   -~> a hash: colors.red, colors.mix(a, b)
puts(square(3), colors.names)
```

Grammar (see [GRAMMAR.md](../GRAMMAR.md#use)): `useStatement = "use" STRING [ "as" IDENT ] end`, read only when the identifier `use` is followed on the same line by a string, and only at the top level of a file.

Meaning (see [SEMANTICS.md](../SEMANTICS.md#modules)):

1. **Finding the file.** The path is written with `/`; `.liz` is added when missing. It is looked for next to the file that says the `use` (its source name), then in each folder of the module path (`$LIZARD_PATH` for the `lizardscript` command). Not found is the error `can't find the module "x": there is no a/x.liz or b/x.liz`.
2. **Once per run, in its own scope.** The first use of a file in a run parses it and runs it, on the program's engine, in a new top scope, with the host's prelude around it (the whole Terrarium for `lizardscript`). It doesn't see the program's variables. Later uses of the same file reuse it.
3. **Live names.** `use "x"` adds the module's scope to the file's top scope as a module it uses. A name is looked up in the file's own names first, then in its modules (the last used first), then outward. Reading `count` reads the module's variable; `count = 10` changes it for everyone. A file's own `let` hides a module's name.
4. **`as`.** `use "x" as m` defines `m`, a hash of the module's names made when the module has just run, the same hash for every `use ... as`. Its functions work on the module's live variables.
5. **Private names.** A name that starts with `_` is not seen from outside, in either form.
6. **Not passed on.** What a module itself uses is not seen by the files that use it.
7. **Errors.** A syntax error in a module fails the use with all its messages. A runtime error keeps the module's position and source name; the command line shows `draw/shapes.liz: [line 3, column 5] runtime error: ...`. Using a module that is still loading is `use cycle: a.liz uses b.liz, which uses a.liz` (changed by Amendment 2: a cycle now loads).

```
-~> counter.liz
let count = 0
function bump() { count += 1; return count }

-~> main.liz
use "counter"
bump()
count = 10
puts(bump(), count)   -~> 11 11
```

## What breaks

Nothing. `use` followed by a string on the same line was a syntax error (`expected end of statement`), and `use` and `as` are ordinary names everywhere else: `let use = 1`, `use("x")`, `use + 1` and `let as = 2` work as before. No `.expected` file changed; the program runners of `tests/` now name each program with its path (so a test can use a module next to it), which changes no output.

## Implementation

- `ast`: `UseStatement{Token, Path, Name}`. Also, separately, every node gets a `Span` with its start and end (needed by the tools that read programs, and by `lizard.parse`).
- `parser`: `startsUse` and `parseUseStatement`, the top-level check.
- `object`: `Environment.Use(module)`, lookups and assignments through the modules a scope uses, `Names()`, `Exported(name)`; a write in a module changes the `Version()` of its users, so the VM's remembered global reads stay right.
- `evaluator`: `modules.go` (`Use`, `FindModule`, `ResetModules`, `ReadModule`, `ModulePath`, `ModulePrelude`).
- `code`, `compiler`, `vm`: `OpUse node`, which calls `evaluator.Use` with the VM as the engine for the module. A use is at the top level, whose scope is the globals read by name, so the compiler needs nothing else.
- `repl`: `RunFile(name, ...)`, module-aware error printing; the tools `lizardscript build` (which follows the uses to bundle a program) and `lizardscript test`.

## Alternatives

- **Copy the names (like Python's `from x import *`).** Simpler, but `count = 10` in the file would change only its copy, silently, and a library that keeps state in a variable (a cache, a setting another file changes) would break with no error. Live names keep the meaning programs had when every file ran in one environment.
- **Only `use "x" as x`.** Clear about where names come from, but every call becomes `shapes.circle(...)`, which reads worse in drawing code, the main use of the language.
- **A `use` anywhere, even in functions.** The VM decides at compile time which names are a function's locals; names that appear at run time could only live at the top level. And imports in the middle of code are hard to follow.
- **Load a file every time it is used.** A module used by ten files would run ten times, with ten copies of its state.
- **A new keyword.** Would break any program with a variable called `use`.

## Open questions

- A `use` inside a `try` is impossible (a use is top level only), so a program can't recover from a missing module; a host can check with `evaluator.FindModule` first.
- The `as` hash doesn't follow the module's variables after loading; a live namespace value could be added later without breaking this.

## Amendment 1 (2026-11-03): standard modules

Big libraries that come with the language (the **flower** graphics family: `flower/lily2d`, and `flower/lily3d`) are **standard modules**: lizard-script files built into `lizardscript`, found by `use` after the places above. Rule 1 becomes: next to the file, then the module path, then the standard modules.

```
use "flower/lily2d" as lily2d
lily2d.circle(picture, (50, 50), 20, #f2a65a)
```

- **Where they are.** `evaluator.StandardModules`, a file system the host sets (the Terrarium embeds `terrarium/std/`). `use "flower/lily2d"` reads `flower/lily2d.liz` from it; a folder in the path is a folder there, like on disk.
- **Their names.** A standard module's source name is `std:/` and its path, `std:/flower/lily2d.liz`. So its errors say where they come from, and a `use` inside it finds the files next to it by rule 1 (`use "lily2d/raster"` in `std:/flower/lily2d.liz` is `std:/flower/lily2d/raster.liz`); `..` can climb inside the standard modules but not out of them. Two ways of writing the same path load it once.
- **The program's files win.** A file `flower/lily2d.liz` next to the program, or in the module path, is used instead, so adding a standard module never changes a program that already had a file of that name. The not-found message still lists only the files on disk.
- **The name in the program is the program's choice.** `use` binds nothing new for a standard module: it is an ordinary module, so `use "flower/lily2d" as lily2d` is how a program names it (any name works after `as`), and a plain `use "flower/lily2d"` makes its functions names of the file, as for any module. We considered binding the last part of the path (`lily2d`) by default, but then a plain `use` of a standard module would mean something a plain `use` of a file doesn't, and one rule is easier to teach than two.
- **Tools.** `lizardscript build` leaves standard modules out of the bundle (every `lizardscript` has them); `evaluator.IsStandardModule(path)` tells their source names.

What breaks: nothing. A `use` that found a file before finds the same file; one that failed with `can't find the module` may now find a standard module of that path, which no program could rely on failing.

## Amendment 2 (2026-11-03): use cycles

Modules can use each other in a cycle, the way Python allows it: a use of a module that is still loading gets it **as it is so far** instead of the error `use cycle: ...`. Rule 7's cycle error goes away.

### Why

Programs of many files call each other. An editor's input code calls the painting, the painting asks the layout where things are, and the layout reads the app's state, which the input changes: thirteen files of one real editor form one cycle. Each call happens when the program runs, long after every file has loaded, so the order of loading doesn't matter to them. With the error, the only ways out were one huge file, or passing every function around as an argument; both are worse than the cycle. Live names (rule 3) already make the partly loaded module work: the file sees the module's own scope, so the names that appear after the use are seen as soon as they are defined.

### The rule

```
-~> input.liz
let pressed = 0
use "paint"                 -~> paint.liz runs now, and uses input.liz back
function press() { pressed += 1; return repaint() }

-~> paint.liz
use "input"                 -~> input.liz is loading: this gets it as it is
let seen = pressed          -~> 0: defined before input.liz's use of paint
function repaint() { return "painted after " + str(pressed) }

-~> main.liz
use "input"
puts(press())               -~> painted after 1
```

1. The first use of a file in a run loads it (rule 2). A use of it while it is loading, from a file it uses directly or through others, doesn't run it again: it makes the file see the module's scope, alive, with the names it has defined so far. The rest appear as the module goes on.
2. `use "a" as m` while `a.liz` is loading gives the module's hash with the names defined so far; it is the same hash every `use ... as` of the module gets, and it is filled with every name when `a.liz` has run. So `m.f()` in a function works; `m.f` read at load time, before `a.liz` defines `f`, is `null`.
3. A file that uses itself already sees its names: that use does nothing.
4. A module that fails stops the run as before; the files of the cycle that were loading fail with it.

### The pitfall: a name used at load time before its module defines it

Only the names used *while the cycle loads* can be missing: a top-level line of `paint.liz` that reads a name `input.liz` defines after its `use "paint"`, or a function called at that time that does. That is the error

```
paint.liz: [line 3, column 12] runtime error: identifier not found: late: input.liz defines it further down, but it is still loading (use cycle: input.liz uses paint.liz, which uses input.liz)
```

The message is precise: the loader adds the explanation only when a module of the cycle that is still loading defines that name at its top level (or in a layer) further down its file and hasn't yet; any other unknown name stays a plain `identifier not found`. The fix is to move the use below the definitions the other file needs at load time, or to read the name inside a function, which runs later.

### How `check` should warn about it

`check` already follows `use` to know a module's names. It should also follow the cycle: for each file of a cycle, the names each other file of it may read **at load time** are the ones defined above the line where the loading of the cycle reaches it. A top-level statement (not a function body) that reads a name of a module of its cycle that the module defines only below its `use` that leads back is a warning at that name: `late is defined in input.liz below its use of paint.liz, so it doesn't exist yet when this line runs`. Names read in function bodies are not warned about, since a function usually runs after loading; a function called at load time is the one case `check` can't tell without following the calls, and is left to the error at run time. This is not built yet.

### What breaks

A program with a use cycle stopped with the error `use cycle: ...`; it now runs. No program can have relied on that error to work (a `use` can't be inside a `try`), but the tests of the error change on purpose: in `engine/evaluator/modules_test.go`, the rows `cycle` and `uses itself` of `TestUse` (now `null`, the value of a use) and the cycle row of `TestUseErrorsCarryTheModule` (now the not-loaded-yet error of a cycle that reads a name too early). The VM's `TestModulesOnBothEngines` keeps its `use "cycle"` program, which now gives `null` on both engines. No `.expected` file changed.

### Also in this amendment

- **Parsed once.** A module file is parsed again only when its text changed: the programs are kept by path and text across runs (`ResetModules` keeps them), and the VM keeps the code it compiled for each program. The code still runs once per run, so a run's state is its own.
- **Errors at the caller.** `error(message, {"at": "caller"})` reports an error where the program called into the file that raised it (the standard modules count as one file), so a library's checks of its arguments point at the program's line; lily2d and lily3d use it. See [SEMANTICS.md](../SEMANTICS.md#errors-at-the-caller).

