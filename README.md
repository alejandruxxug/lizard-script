# lizard-script 🦎

> The programming language so easy that even a lizard can understand it.

lizard-script is a small interpreted language written in Go, built phase by phase as the team project of the Compilers course at Universidad EIA.

## Team

- Alejandro Urrego Giraldo
- Maria José Sosa Fernández
- Samuel Buelvas Cabrales

## Status

| Phase | Component | Status |
| --- | --- | --- |
| 1 | Tokens + lexer | ✅ Done |
| 2 | Parser / AST | ✅ Done |
| 3 | Evaluator | ✅ Done |
| 4 | REPL | ✅ Done |

## A taste of the language

```
let five = 5;
let ten = 10;

function add(x, y) {
  return x + y;
}

let result = add(five, ten); -~> this is a comment

while (result > 0) {
  if (result > 10 && five != 0) {
    result = result // 2;
  } elif (result == 10) {
    result = result - 1;
  } else {
    break;
  }
}

let lizard = {"name": "Liz", "toes": [5, 5, 5, 5]};
```

## Documentation

- [docs/DESIGN.md](docs/DESIGN.md): the design document. It has the grammar in EBNF, the architecture, the design decisions, the known limitations and how we used AI.
- [docs/GRAMMAR.md](docs/GRAMMAR.md): the full grammar, with precedence and associativity.
- [docs/SEMANTICS.md](docs/SEMANTICS.md): what each construct does when it runs, such as truthiness, `==` vs `===`, `/` vs `//`, scopes and the builtins.
- [docs/TERRARIUM.md](docs/TERRARIUM.md): the Terrarium, the standard library: files, the system, time, JSON, images, the clipboard, and a window to draw in with its input. Examples are in `examples/terrarium/`.
- [docs/learn/](docs/learn/README.md): a teaching guide to the language, in English and Spanish, with exercises, projects and a cheat sheet. / Guía para aprender y enseñar el lenguaje, en inglés y español.

## Tokens

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

## Project structure

```
lizard-script/
├── token/     # token types, keywords and identifier lookup
├── lexer/     # turns source code into a stream of tokens (+ tests)
├── ast/       # the nodes of the abstract syntax tree
├── parser/    # turns tokens into an AST with a Pratt parser (+ tests)
├── object/    # the values of a running program and the environments that hold variables
├── evaluator/ # runs the AST by walking it, plus the builtins (+ tests)
├── repl/      # the interactive REPL, and running whole files (+ tests)
├── examples/  # one program per feature, and programs that show each kind of error
├── tests/     # valid and failing programs with their expected output
├── docs/      # design document, grammar and semantics
└── main.go    # the lizardscript command
```

## Getting started

### Install

You need [Go](https://go.dev/dl/) 1.27 or newer. Check it with `go version`. The project has no other dependencies.

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
