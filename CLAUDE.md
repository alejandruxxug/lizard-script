# lizard-script

A small language in Go, the team project of the Compilers course at Universidad EIA. Phases 1 to 4 (lexer, parser, evaluator, REPL) follow the course's reference interpreter; since then the language has grown well beyond it: a bytecode VM, a standard library (the Terrarium) and a teaching guide.

## Read first

- `docs/GRAMMAR.md`: syntax and precedence
- `docs/SEMANTICS.md`: what each construct does
- `docs/DESIGN.md`: architecture, design decisions, limitations, and the extensions log (§7)
- `docs/TERRARIUM.md`: the standard library
- `docs/learn/`: the guide for learning the language (English and Spanish)

## Rules

- **The language may change.** Growing by adding is still the first choice, but replacing or redesigning a part is allowed when it makes the language better. Every change after commit `cc9dc0d` gets a row in `docs/DESIGN.md` → *7. Extensions beyond the course design*: what changed, why, and what it touches (tokens, AST, evaluator, VM, builtins, Terrarium). Update GRAMMAR.md and SEMANTICS.md in the same change, and `4. Known limitations` when a limitation goes away.
- **Know when a change breaks programs.** If a change alters what an existing program does, say so in the commit and in §7, update the affected `.expected` files in the same change (`go test ./tests -update`, then read the diff), and give the team a heads-up. Never let tests go red to land something.
- **Two engines, one language.** The tree-walker (`evaluator`) and the bytecode VM (`compiler` + `vm`) must give the same output for every program. A language change goes into both, and every test program runs on both.
- **The Terrarium** (`terrarium/`) is where programs reach the outside world: files, OS, time, images, windows, input, pen pressure. Modules have plain names (`fs`, `img`, `window`…). A new OS-facing feature becomes a Terrarium module, not a special case in the interpreter. Modules that don't need speed are written in lizard-script itself.
- **Keep the guide true.** A change users can see updates `docs/learn/` in both languages, with examples that were actually run.
- Every change is tested the way the repo already does it: table-driven unit tests per package, plus programs in `tests/valid` or `tests/errors` with their `.expected` output. `go test ./...` and `go vet ./...` must be green, and the code must build for macOS, Windows and Linux.
- Match the existing style: plain-word comments, small functions, errors with `[line L, column C]`.
