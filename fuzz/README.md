# Fuzz tests

Go native fuzz tests (`go test -fuzz`) for the parser and the two engines.

| Target | What it gives | What must hold |
| --- | --- | --- |
| `FuzzParser` | random text, starting from the programs of the repo | the lexer and parser never panic or hang (`ParseTimeout`), every error starts with `[line L, column C]` and that place is inside the input, every token starts and ends inside the input |
| `FuzzTokenSoup` | random sequences of real tokens and spaces (`vocabulary`) | the same as `FuzzParser`, deeper into the parser |
| `FuzzEngines` | programs made by the generator from random bytes | the generated program parses, and the evaluator and the VM agree on it |
| `FuzzEnginesText` | mutated programs of the repo | when it parses, the evaluator and the VM agree on it |
| `FuzzFormat` | mutated programs of the repo | a program that parses can be formatted (`format.Source`), formatting twice gives the same text, and the formatted program does the same on the evaluator: output, value, steps and hook calls, without the positions, which move with the layout |
| `FuzzFormatGenerated` | generated programs | the same as `FuzzFormat` |

"Agree" is the contract of `engine/vm/differential_test.go`: with a step limit of 20000, the same output, the same value (or the same error message, line, column and catchability), the same number of steps and the same hook calls (`OnStatement`, `OnLayer`), both without and with the hooks. A panic of an engine is a failure too.

## Running

A plain `go test ./...` runs the seed corpus of every target, a few hundred generated programs (`TestGeneratedPrograms`), the whole corpus on both engines (`TestCorpusAgrees`), the formatter on the corpus and on generated programs (`TestCorpusFormats`) and the bugs found so far (`TestKnownBugs`). To fuzz, one target at a time:

```sh
cd fuzz
go test -run XXX -fuzz '^FuzzEngines$' -fuzztime 15m
go test -run XXX -fuzz '^FuzzEnginesText$' -fuzztime 15m
go test -run XXX -fuzz '^FuzzParser$' -fuzztime 15m
go test -run XXX -fuzz '^FuzzTokenSoup$' -fuzztime 15m
go test -run XXX -fuzz '^FuzzFormat$' -fuzztime 15m
go test -run XXX -fuzz '^FuzzFormatGenerated$' -fuzztime 15m
```

Add `-parallel N` to use fewer cores. `-run XXX` skips the ordinary tests.

When a generated program makes the engines differ, the failure is shrunk first (`Shrink`: it removes statements, unwraps blocks and replaces expressions by a part of them or by `0` while the engines still differ), so the message shows the minimal program and what each engine gave, then the program as generated. `FuzzEnginesText` shrinks by lines (`ShrinkLines`). Go writes the raw input to `testdata/fuzz/<target>/`; a plain `go test` runs it again until it is fixed or removed.

The seed corpus (`corpus.go`) is every `.liz` file of `tests/` and `examples/` and every code block of the guide in `docs/learn` fenced with a bare ` ``` ` (the `text` and `sh` blocks are output and shell).

## Bugs found

Each bug found goes into `testdata/bugs/<what it shows>.liz`, as its minimal program. `TestKnownBugs` checks every file there with the parser checks and the engine comparison, so it fails until the bug is fixed and is a regression test after. Its entry in `knownBugs` (`engines.go`) makes the fuzzers skip the inputs that would show the same bug again, so they go on to new ones; remove the entry when the bug is fixed.

## Adding a generator feature

The generator (`generator.go`) reads one byte for each choice (`pick`, `chance`), and option 0 is always the simplest one, so every input ends in a finished program. It builds a tree of `Node`s, which is what the shrinker takes apart:

- `ex(format, kids...)` is an expression and `st(format, kids...)` a statement, with a `%s` in `format` for each kid. `block(stmts...)` is `{ }` with each statement on its own line, `seq(stmts...)` is statements that go together, and a `Layer` node is `layer name` with its statements indented under it. Write everything through these, never a `\n` inside a format: the indentation must follow the nesting, since layers are measured by columns.
- Types: the generator remembers each variable's type (`typ`) in `scopes`, and `expr(t, depth)` makes an expression of type `t`. A new kind of expression goes into the `case` of its type (`intExpr`, `floatExpr`, `strExpr`, `boolExpr`, or the switch in `expr`), a new literal into `literal`, a new type into the `typ` list (before `tFunc`, which is not a value type for `==`) with its cases.
- A new kind of statement goes into `statement`'s switch. Use `g.body(n)` for a block in a new scope, `g.loopBody` for the body of a loop (so `break` and `continue` may appear), and `g.functionBody` for a function (it resets the loops and allows `return`).
- Keep programs well-typed-ish: a statement that fails on purpose goes in `mistake`, which mostly runs inside a `try`. Too many errors make programs end early and test less.
- After the change, `go test ./fuzz` checks that the generated programs still parse and are the same for the same bytes.
