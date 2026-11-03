# LEP 0002: Modules with `use`

- **Author(s):** the lizard-script team
- **Status:** Final <!-- Draft · Accepted · Rejected · Deferred · Final -->
- **Created:** 2026-11-03
- **Decision:** Accepted 2026-11-03, with live names, `as` for a hash, and `_` for private names.
- **Discussion:** to be opened as a pull request

## Summary

`use "draw/shapes"` runs the file `draw/shapes.liz` (next to the file that says it, or in a folder of the module path) once per run, in a top scope of its own, and lets the file see every name the module defined, alive: reading one reads the module's variable and `=` changes it. `use "draw/shapes" as shapes` gives a hash of those names instead. Names that start with `_` stay inside the module. A use is only allowed at the top level of a file, cycles are an error, and a module's errors carry its file name.

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

Grammar (see [GRAMMAR.md](../docs/GRAMMAR.md#use)): `useStatement = "use" STRING [ "as" IDENT ] end`, read only when the identifier `use` is followed on the same line by a string, and only at the top level of a file.

Meaning (see [SEMANTICS.md](../docs/SEMANTICS.md#modules)):

1. **Finding the file.** The path is written with `/`; `.liz` is added when missing. It is looked for next to the file that says the `use` (its source name), then in each folder of the module path (`$LIZARD_PATH` for the `lizardscript` command). Not found is the error `can't find the module "x": there is no a/x.liz or b/x.liz`.
2. **Once per run, in its own scope.** The first use of a file in a run parses it and runs it, on the program's engine, in a new top scope, with the host's prelude around it (the whole Terrarium for `lizardscript`). It doesn't see the program's variables. Later uses of the same file reuse it.
3. **Live names.** `use "x"` adds the module's scope to the file's top scope as a module it uses. A name is looked up in the file's own names first, then in its modules (the last used first), then outward. Reading `count` reads the module's variable; `count = 10` changes it for everyone. A file's own `let` hides a module's name.
4. **`as`.** `use "x" as m` defines `m`, a hash of the module's names made when the module has just run, the same hash for every `use ... as`. Its functions work on the module's live variables.
5. **Private names.** A name that starts with `_` is not seen from outside, in either form.
6. **Not passed on.** What a module itself uses is not seen by the files that use it.
7. **Errors.** A syntax error in a module fails the use with all its messages. A runtime error keeps the module's position and source name; the command line shows `draw/shapes.liz: [line 3, column 5] runtime error: ...`. Using a module that is still loading is `use cycle: a.liz uses b.liz, which uses a.liz`.

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
