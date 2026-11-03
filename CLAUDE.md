# lizard-script

A small interpreted language in Go, the team project of the Compilers course at Universidad EIA. Phases 1 to 4 (lexer, parser, evaluator, REPL) follow the course's reference interpreter.

## Read first

- `docs/GRAMMAR.md`: syntax and precedence
- `docs/SEMANTICS.md`: what each construct does
- `docs/DESIGN.md`: architecture, design decisions, limitations, and the extensions log

## Rules

- **The language grows by adding, never by replacing.** The Go foundation (lexer, parser, AST, evaluator) stays, and every extension after commit `cc9dc0d` gets a row in `docs/DESIGN.md` → *7. Extensions beyond the course design*: what was added, why, and what it adds to the original tokens, AST, evaluator or builtins. Update GRAMMAR.md and SEMANTICS.md in the same change, and `4. Known limitations` when a limitation goes away.
- **Existing programs keep working.** The existing tests and `.expected` files stay green unchanged. If an extension would change what an existing program does, stop: it's a breaking change and needs the team's agreement first.
- Every change is tested the way the repo already does it: table-driven unit tests per package, plus programs in `tests/valid` or `tests/errors` with their `.expected` output. `go test ./...` must be green.
- Match the existing style: plain-word comments, small functions, errors with `[line L, column C]`.
