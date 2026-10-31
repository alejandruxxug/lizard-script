# lizard-script 🦎

> The programming language so easy that even a lizard can understand it.

lizard-script is a small interpreted language written in Go, built phase by phase for my Languages & Compilers course.

## Status

| Phase | Component | Status |
| --- | --- | --- |
| 1 | Tokens + lexer | ✅ Done |
| 2 | Parser / AST | ✅ Done |
| 3 | Evaluator | ⏳ Coming soon |
| 4 | REPL | ⏳ Coming soon |

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

The full grammar is in [docs/GRAMMAR.md](docs/GRAMMAR.md).

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
├── token/   # token types, keywords and identifier lookup
├── lexer/   # turns source code into a stream of tokens (+ tests)
├── ast/     # the nodes of the abstract syntax tree
├── parser/  # turns tokens into an AST with a Pratt parser (+ tests)
└── docs/    # grammar of the language
```

## Getting started

You need Go 1.27 or newer.

```sh
git clone https://github.com/alejandruxxug/lizard-script.git
cd lizard-script
go test ./...
```

## Credits

This project follows the structure of the interpreter built in class ([sebastianzapatar/compi20262](https://github.com/sebastianzapatar/compi20262)), which is itself based on Thorsten Ball's *Writing an Interpreter in Go*.
