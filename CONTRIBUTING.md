# Contributing to lizard-script 🦎

Thanks for wanting to help! lizard-script is open source under the [MIT license](LICENSE): fork it, change it, build your own things with it. This page is about sending changes back.

## Kinds of changes

| You want to… | Do this |
|---|---|
| Fix a bug | Open an issue with the program, what it printed and what you expected (the **Bug** template), or send a pull request with the fix and a test. |
| Improve the docs or the guide | Send a pull request. Guide changes go in both `docs/learn/en` and `docs/learn/es`; every output shown must come from running the code. |
| Add to the Terrarium (a new module or function) | Open an issue first so we agree on the name and shape, then a pull request. See "Adding a module" in [docs/TERRARIUM.md](docs/TERRARIUM.md). |
| **Change the language** (syntax, semantics, a new builtin) | Write a **proposal** (a LEP): see [docs/proposals/](docs/proposals/README.md). |

## Building and testing

```sh
go build          # the lizardscript command
go test ./...     # every package, every program in tests/, on both engines
go vet ./...
```

Before a pull request, all three must pass, and the code must build for macOS, Windows and Linux (`GOOS=windows go build`, `GOOS=linux go build`).

## How the code is organized

Read [docs/DESIGN.md](docs/DESIGN.md) first: the packages, the two engines (the tree-walking `evaluator` and the bytecode `compiler` + `vm`) and why things are the way they are. The rules the project follows are in [CLAUDE.md](CLAUDE.md) (it's written for AI assistants, but it's the shortest summary for humans too). The important ones:

- **Two engines, one language.** A language change goes into the evaluator and the VM, and every test program runs on both.
- **Tests the way the repo does them:** table-driven unit tests per package, plus programs in `tests/valid` or `tests/errors` with their `.expected` output.
- **Clean code:** one concept per file, small functions, plain-word comments that explain the why, errors with `[line L, column C]`. Someone new should be able to read any package top to bottom.
- **Breaking changes are allowed but never silent:** say so in the pull request and in DESIGN.md §7, and update the affected `.expected` files in the same change.

## Pull requests

- One topic per pull request, with a short description of what and why.
- Add a row to DESIGN.md §7 for anything users can see, and update GRAMMAR.md / SEMANTICS.md / the guide as needed.
- Be kind in reviews. Everyone here is learning, the lizard included.
