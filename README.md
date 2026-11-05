<h1 align="center">lizard-script 🦎</h1>

<p align="center"><i>The programming language so easy that even a lizard can understand it.</i></p>

<p align="center">
  <a href="https://github.com/alejandruxxug/lizard-script/releases/latest"><img src="https://img.shields.io/github/v/release/alejandruxxug/lizard-script?label=download&color=3fa34d" alt="Download"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-e0a526" alt="MIT license"></a>
  <img src="https://img.shields.io/badge/written%20in-Go-00add8" alt="Written in Go">
  <img src="https://img.shields.io/badge/macOS%20·%20Windows%20·%20Linux-1d1d1f" alt="macOS, Windows, Linux">
  <a href="docs/learn/README.md"><img src="https://img.shields.io/badge/guide-English%20·%20Español-6f5bd8" alt="Guide in English and Spanish"></a>
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#a-taste-of-the-language">Taste</a> ·
  <a href="#whats-inside">What's inside</a> ·
  <a href="#speed">Speed</a> ·
  <a href="docs/learn/README.md">Learn it</a> ·
  <a href="#built-with-lizard-script">Built with it</a>
</p>

lizard-script is a small, friendly language with a big standard library. It started as the team project of the Compilers course at Universidad EIA, following the interpreter built in class, and grew from there: a bytecode VM that keeps up with Python, a standard library called **the Terrarium** that opens windows and reads pen pressure, a formatter, a checker, a language server for VS Code and Zed, and a guide to teach it in English and Spanish.

## A taste of the language

```
-~> lizards count their toes
let liz = {"name": "Liz", "color": #3fa34d, "home": (12, 4), "toes": [5, 5, 5, 5]}

function total(list) {
  let sum = 0
  for (let i = 0; i < len(list); i += 1) {
    sum += list[i]
  }
  return sum
}

let friends = list.sort(["iguana", "gecko", "chameleon"], function(a, b) { len(a) < len(b) })
puts(liz.name + " has " + str(total(liz.toes)) + " toes and lives at " + str(liz.home))
puts(liz.home + (1, 2), friends)
```

```
$ lizardscript taste.liz
Liz has 20 toes and lives at (12, 4)
(13, 6) ["gecko", "iguana", "chameleon"]
```

Colors (`#3fa34d`) and points (`(12, 4)`) are values of their own, `-~>` starts a comment (it looks like a lizard), and a statement ends at the end of its line.

## What's inside

| | |
|---|---|
| **Two engines, one language** | a tree-walking evaluator and a bytecode VM (`--vm`) that give the same output for every program, checked by a differential fuzzer |
| **The Terrarium** | the standard library: files, the system, time, JSON, images, the clipboard, windows with drawing surfaces and blend modes, input, pen pressure and tilt, file dialogs, background tasks. See [TERRARIUM.md](docs/TERRARIUM.md) |
| **Modules** | `use "shapes"` and `use "flower/lily2d" as lily`, with the 2D and 3D graphics libraries of the **flower** family |
| **Tools** | `lizardscript fmt`, `check`, `test`, `build` (one executable for macOS, Windows or Linux, with an icon), `drive` (automate a running window) and `lsp` |
| **Editors** | a language server, VS Code and Zed extensions, Tree-sitter and TextMate grammars: [editors/](editors/README.md) |
| **Learn it** | a teaching guide with exercises, projects and a cheat sheet, in [English](docs/learn/en/) and [Spanish](docs/learn/es/) |
| **Changes in the open** | language changes start as a Lizard Enhancement Proposal: [docs/proposals/](docs/proposals/README.md) |

## Install

Download `lizardscript` for your system from the [latest release](https://github.com/alejandruxxug/lizard-script/releases/latest) (macOS Apple Silicon and Intel, Windows, Linux), unpack it and put it on your PATH. On macOS the first run may need `xattr -d com.apple.quarantine lizardscript`, since the binary isn't notarized.

Or build it from source with [Go](https://go.dev/dl/) 1.27 or newer:

```sh
git clone https://github.com/alejandruxxug/lizard-script.git
cd lizard-script
go build              # makes ./lizardscript
./lizardscript examples/05_functions.liz
```

More in [Getting started](#getting-started) below.

## Speed

The VM against CPython 3.14 and Lua 5.5 on an M4 (best of 5, startup included; the full table and how to run it are in [bench/](bench/README.md)):

| Benchmark | lizard VM | Python | Lua |
|---|---|---|---|
| loop | 0.080 s | 0.230 s | 0.053 s |
| closures | 0.019 s | 0.046 s | 0.014 s |
| fib | 0.013 s | 0.028 s | 0.007 s |
| strings | 0.015 s | 0.024 s | 0.205 s |
| hashes | 0.068 s | 0.049 s | 0.025 s |

lizard is faster than Python on loops, closures, recursion, floats and strings, and slower on hashes; Lua is still the one to beat.

## Built with lizard-script

<a href="https://github.com/alejandruxxug/linkz"><img src="https://raw.githubusercontent.com/alejandruxxug/linkz/main/docs/Readme/artist-mode.png" width="560" alt="Linkz, a vector art app written in lizard-script"></a>

**[Linkz](https://github.com/alejandruxxug/linkz)** (*lizard inkz*) is a vector art app where the drawing is code, written entirely in lizard-script: about 60,000 lines of it, with the editor, the rasterizer, an image tracer and an MCP server, shipped with `lizardscript build`.

## Documentation

- [docs/DESIGN.md](docs/DESIGN.md): the design document. It has the grammar in EBNF, the architecture, the design decisions, the known limitations and how we used AI.
- [docs/GRAMMAR.md](docs/GRAMMAR.md): the full grammar, with precedence and associativity.
- [docs/SEMANTICS.md](docs/SEMANTICS.md): what each construct does when it runs, such as truthiness, `==` vs `===`, `/` vs `//`, scopes and the builtins.
- [docs/TERRARIUM.md](docs/TERRARIUM.md): the Terrarium, the standard library. Examples are in `examples/terrarium/`.
- [editors/](editors/README.md): editor support.
- [docs/learn/](docs/learn/README.md): a teaching guide to the language, in English and Spanish. / Guía para aprender y enseñar el lenguaje, en inglés y español.

## Team

- Alejandro Urrego Giraldo
- Maria José Sosa Fernández
- Samuel Buelvas Cabrales

## Language reference

### Tokens

**Keywords:** `let`, `function`, `return`, `if`, `elif`, `else`, `for`, `while`, `break`, `true`, `false`

**Operators**

| Kind | Operators |
| --- | --- |
| Arithmetic | `+` `-` `*` `/` `//` (integer division) `**` (power) `%` (modulo) |
| Comparison | `<` `>` `<=` `>=` `==` `!=` `===` (exact) `!==` (not exact) |
| Logical | `&&` (and) `\|\|` (or) `!` (not) |
| Other | `=` (assign) |

**Delimiters:** `,` `;` `:` `(` `)` `{` `}` `[` `]`

**Literals:** identifiers (a letter or `_`, then letters, digits or `_`), integers and strings between double quotes.

**Comments:** `-~>` starts a comment that runs to the end of the line. It looks like a lizard's body, and `//` is already integer division.

The lexer always takes the longest operator it can, so `====` lexes as `===` followed by `=`.

**Lines:** a statement ends at the end of its line. It goes on to the next line only when the line ends with an operator, a comma or an opening bracket, or while a `(`, `[` or hash is still open; `;` separates statements on one line. Programs written before this rule can be checked with `go run ./tools/checknewlines file.liz...`.

## Project structure

```
lizard-script/
├── main.go       # the lizardscript command
├── lang/         # the front end
│   ├── token/    #   token types, keywords and identifier lookup
│   ├── lexer/    #   turns source code into a stream of tokens (+ tests)
│   ├── ast/      #   the nodes of the abstract syntax tree
│   └── parser/   #   turns tokens into an AST with a Pratt parser (+ tests)
├── engine/       # the two ways to run a program
│   ├── object/   #   the values of a running program and the environments that hold variables
│   ├── evaluator/#   runs the AST by walking it, plus the builtins (+ tests)
│   ├── code/     #   the bytecode: opcodes and how instructions are encoded
│   ├── compiler/ #   turns the AST into bytecode (+ tests)
│   └── vm/       #   runs the bytecode on a stack machine (+ tests)
├── terrarium/    # the standard library: fs, os, time, json, img, clip, window, pen
├── tools/        # what the command does besides running
│   ├── repl/     #   the interactive REPL, and running whole files (+ tests)
│   ├── testrunner/ # lizardscript test
│   ├── build/    #   lizardscript build: a program and the runtime in one file
│   ├── bundle/   #   the file a built program carries
│   ├── format/   #   lizardscript fmt
│   ├── check/    #   lizardscript check
│   ├── lsp/      #   the language server, lizardscript lsp (+ tests)
│   ├── checknewlines/ # lists lines whose meaning changed with LEP 0001
│   └── ebitencopy/    # makes third_party/ebiten
├── editors/      # the VS Code and Zed extensions and the syntax grammars
├── docs/         # design, grammar, semantics, the Terrarium, the guide (docs/learn) and the proposals (docs/proposals: LEPs)
├── examples/     # one program per feature, and programs that show each kind of error
├── tests/        # valid and failing programs with their expected output
├── bench/        # the same programs in lizard-script, Python and Lua
├── fuzz/         # random programs through both engines
└── third_party/  # a copy of Ebitengine with its Metal device made on the first window (see its README)
```

## Getting started

### Install

You need [Go](https://go.dev/dl/) 1.27 or newer. Check it with `go version`. The first build downloads the Go modules the Terrarium's window uses (Ebitengine itself is in `third_party/ebiten`, with one change, see [its README](third_party/README.md)); on Linux the window also needs the X11/OpenGL development packages (see [Ebitengine's install guide](https://ebitengine.org/en/documents/install.html)).

```sh
git clone https://github.com/alejandruxxug/lizard-script.git
cd lizard-script
```

### Compile

```sh
go build
```

This creates the `lizardscript` command (`lizardscript.exe` on Windows). You can also skip this step and use `go run .` instead of `./lizardscript` below.

### Run

The interpreter reads programs in two ways.

**From a file:** pass the path of the program.

```sh
./lizardscript examples/05_functions.liz
go run . examples/05_functions.liz
```

The program's output goes to standard output. A program with an error stops with a message like `[line 2, column 8] runtime error: division by zero` on standard error, and exits with code 1.

**From the console (REPL):** run it with no arguments.

```
$ go run .
🦎 >> let toes = [5, 5]
🦎 >> push(toes, 4)
[5, 5, 4]
🦎 >> "Liz" * 2
"LizLiz"
🦎 >> exit
```

- **Variables:** they live on between inputs.
- **Several lines:** an input with an open `(`, `[`, `{` or string keeps reading lines until it is closed.
- **`if`:** after an `if`, the REPL waits for one more line in case it starts with `elif` or `else`. Press Enter on an empty line to run the `if` alone.
- **Leaving:** type `exit`, or end the input with Ctrl+D (Ctrl+Z then Enter on Windows).

### Test

```sh
go test ./...
```

This one command runs everything:
- the unit tests of each component;
- the programs in [tests/](tests/), each compared with its expected output;
- every program in [examples/](examples/).

## Examples

| Program | Shows |
| --- | --- |
| [01_variables.liz](examples/01_variables.liz) | declaring, reading and assigning variables |
| [02_expressions.liz](examples/02_expressions.liz) | arithmetic, comparison and logical operators, precedence and associativity |
| [03_conditionals.liz](examples/03_conditionals.liz) | `if`, `elif`, `else`, nested conditionals, `if` as a value |
| [04_loops.liz](examples/04_loops.liz) | `while`, `for`, `break`, nested loops |
| [05_functions.liz](examples/05_functions.liz) | definition, parameters, return values, recursion |
| [06_scopes.liz](examples/06_scopes.liz) | global and local variables, the same name in different scopes |
| [07_data_structures.liz](examples/07_data_structures.liz) | arrays and hashes |
| [08_builtins.liz](examples/08_builtins.liz) | `puts`, `len`, `type`, conversion with `str` and `int`, and the other builtins |
| [09_extensions.liz](examples/09_extensions.liz) | closures, short-circuit evaluation, higher-order functions |
| [10_floats_and_math.liz](examples/10_floats_and_math.liz) | floats, the `math` library, angle units |
| [11_colors_and_points.liz](examples/11_colors_and_points.liz) | color literals, `rgb`/`hsl`, points and their arithmetic |
| [12_layers.liz](examples/12_layers.liz) | `layer` blocks and the `text` and `list` helpers |
| [13_defaults_try_continue.liz](examples/13_defaults_try_continue.liz) | default parameters, `try` / `catch`, `continue`, `+=` and friends |
| [fizzbuzz.liz](examples/fizzbuzz.liz) | a complete small program |
| [errors/lexical.liz](examples/errors/lexical.liz) | a lexical error |
| [errors/syntax.liz](examples/errors/syntax.liz) | several syntax errors reported at once |
| [errors/runtime.liz](examples/errors/runtime.liz) | a runtime error after some output |

## Tests

[tests/](tests/) has two folders of programs, and each `program.liz` comes with a `program.expected` file holding its exact output:
- `valid/` has 16 programs that must run without errors.
- `errors/` has 31 programs that must fail: lexical errors (`lexical_*`), syntax errors (`syntax_*`) and runtime errors (`runtime_*`).

To add a test, write the `.liz` file and generate its `.expected` file with `go test ./tests -update`. Then read that file to check the output is really right before committing it.

## Credits

This project follows the structure of the interpreter built in class, the `evaluator` branch of [sebastianzapatar/compi8am](https://github.com/sebastianzapatar/compi8am/tree/evaluator), which is itself based on Thorsten Ball's *Writing an Interpreter in Go*.

## Contributing

lizard-script is open source, and anyone is welcome to tweak it, build on it and propose changes. Bug fixes, docs and Terrarium additions go through normal pull requests; changes to the language itself start as a **Lizard Enhancement Proposal**. See [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/proposals/](docs/proposals/README.md).

## License

[MIT](LICENSE) © 2026 Alejandro Urrego Giraldo, Maria José Sosa Fernández, Samuel Buelvas Cabrales, and the lizard-script contributors.
