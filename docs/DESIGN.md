# lizard-script: design document

**Team:** Alejandro Urrego Giraldo, Maria José Sosa Fernández, Samuel Buelvas Cabrales. Compilers, Universidad EIA, 2026.

lizard-script is a small dynamically typed language with integers, booleans, strings, arrays, hashes and first-class functions. Full details are in [GRAMMAR.md](GRAMMAR.md) (syntax) and [SEMANTICS.md](SEMANTICS.md) (behavior).

## 1. Grammar (EBNF)

`{ x }` is zero or more, `[ x ]` is optional. `end` is a `;`, or nothing before a `}`, the end of the input or a line break.

```ebnf
program     = { statement } ;
statement   = "let" IDENT "=" expression end
            | "function" IDENT "(" [ params ] ")" block [ ";" ]
            | "return" [ expression ] end
            | "break" end
            | ( ifExpr | whileExpr | forExpr ) [ ";" ]
            | expression end
            | ";" ;
block       = "{" { statement } "}" ;
params      = IDENT { "," IDENT } ;

expression  = logicalOr [ "=" expression ] ;            (* right associative *)
logicalOr   = logicalAnd { "||" logicalAnd } ;
logicalAnd  = equality { "&&" equality } ;
equality    = comparison { ( "==" | "!=" | "===" | "!==" ) comparison } ;
comparison  = sum { ( "<" | ">" | "<=" | ">=" ) sum } ;
sum         = product { ( "+" | "-" ) product } ;
product     = prefix { ( "*" | "/" | "//" | "%" ) prefix } ;
prefix      = ( "-" | "!" ) prefix | power ;
power       = postfix [ "**" prefix ] ;                 (* right associative *)
postfix     = primary { "(" [ args ] ")" | "[" expression "]" } ;
primary     = IDENT | INT | STRING | "true" | "false" | "(" expression ")"
            | "[" [ args ] "]" | "{" [ pair { "," pair } [ "," ] ] "}"
            | "function" "(" [ params ] ")" block | ifExpr | whileExpr | forExpr ;
args        = expression { "," expression } ;
pair        = expression ":" expression ;
ifExpr      = "if" expression block { "elif" expression block } [ "else" block ] ;
whileExpr   = "while" expression block ;
forExpr     = "for" "(" [ statement ] ";" [ expression ] ";" [ expression ] ")" block ;

IDENT = ( letter | "_" ) { letter | digit | "_" } ;   INT = digit { digit } ;
STRING = '"' { any character but '"' } '"' ;        comment = "-~>" { any character but newline } ;
```

## 2. Architecture

```
source text ──> lexer ──> tokens ──> parser ──> AST ──> evaluator ──> values / output
                                                          │
                                       environments (scopes) + builtins
```

| Package | Responsibility |
| --- | --- |
| `token` | Token types, keywords, and each token's line and column. |
| `lexer` | Reads the text character by character and returns one token at a time. Always takes the longest operator (`===` before `==`). Unknown characters and unterminated strings become `ILLEGAL` tokens. |
| `ast` | The tree nodes. Each node keeps the token it comes from, so later stages know its line and column. |
| `parser` | A Pratt parser: one function per prefix form and one per infix operator, driven by a precedence table. It builds the AST and collects every syntax error with its position. |
| `object` | The runtime values (`Integer`, `String`, `Array`, `Hash`, `Function`, …) and `Environment`, a map of variables linked to the scope around it. |
| `evaluator` | `Eval(node, env)` walks the tree and returns a value. It also holds the builtins (`puts`, `len`, `str`, `int`, `push`, …). |
| `repl` | The interactive loop and `Run`, which runs a whole file. |
| `main.go` | `lizardscript` starts the REPL; `lizardscript file.liz` runs a file. |

**Errors.** Each kind of error is detected in its own stage and reported as `[line L, column C] <kind> error: <message>`:
- **Lexical:** the lexer turns a bad character into an `ILLEGAL` token, and the parser reports it as a lexical error.
- **Syntax:** the parser records the error, skips to the end of that statement and carries on, so it reports several independent errors at once. A program with any syntax error doesn't run at all.
- **Runtime:** the evaluator returns an `Error` value. It travels up the tree and stops the program, after any output the program had already printed.

**Control flow.** `return`, `break` and errors are values that travel up through `Eval`:
- a `ReturnValue` stops at the function call it belongs to;
- a `Break` stops at the innermost loop;
- an `Error` goes all the way up.

## 3. Design decisions

- **A Pratt parser instead of one function per grammar level.** Precedence and associativity live in one table. Right associativity (`=`, `**`) comes from parsing the right side with one level less.
- **Desugaring.** `function f(x) {...}` builds the same tree as `let f = function(x) {...};`. An `elif` is an `if` inside the previous `else`. This keeps the AST and the evaluator small.
- **`if`, `while` and `for` are expressions.** An `if` gives the value of the branch that ran, so `let x = if (a) { 1 } else { 2 }` works. A loop gives `null`.
- **Block scopes.** Every block and every function call gets a new `Environment` linked to the one around it:
  - `let` creates a variable in the current scope, hiding outer ones with the same name;
  - `=` changes the closest existing variable, and assigning to an undeclared name is an error, which catches typos.
- **Closures.** A function value keeps the environment where it was created, so it can read and change those variables later.
- **Two equalities.** `===` requires the same type and value. `==` first converts booleans and numeric strings to numbers, so `"5" == 5`. Arrays and hashes compare by content.
- **Integer arithmetic only.**
  - `/` rounds towards zero and `//` rounds down. `%` pairs with `//`, so `a == (a // b) * b + a % b` always holds.
  - `**` with a negative exponent is an error, since the result wouldn't be an integer.
- **Truthiness.** `false`, `null`, `0`, `""`, `[]` and `{}` are false. `&&` and `||` short-circuit and always give a boolean.
- **Shared data.** Arrays and hashes are references: `let b = a` shares the same array, and functions can change their arguments.
- **Hash keys and order.** Keys keep their insertion order, so output is predictable. Only integers, booleans and strings can be keys, and `1` and `"1"` are different keys.
- **No crashes on adversarial input.**
  - Printing and comparing values use an explicit stack instead of recursion, and detect values that contain themselves (shown as `[...]`).
  - The parser rejects programs nested more than 10000 levels.
  - The evaluator limits call depth (10000 calls) and total depth, so it reports a stack overflow instead of overflowing Go's own stack.
- **The REPL keeps reading lines while the input is unfinished**, like an open bracket or a trailing operator.

## 4. Known limitations

- Integers only, with no floating point; overflow wraps around, as in Go.
- Strings have no escape sequences, so a string can't contain `"`.
- After a syntax error the parser skips to the next `;` or `}`. Without semicolons, a second error on a later line of the same block may show up only after fixing the first.
- In the REPL, an `if` without `else` runs once the next line shows it is complete, or after an empty line.
- Nesting is limited to 10000 levels in the source, and recursion to 10000 calls.
- An infinite loop runs forever; in the REPL, stop it with Ctrl+C.

## 5. Testing

`go test ./...` runs everything:
- unit tests for each package (lexer, parser, evaluator, REPL);
- the programs in `tests/`: 16 valid programs and 31 that must fail with a lexical, syntax or runtime error, each compared with its `.expected` output;
- every program in `examples/`.

## 6. Use of AI

We used Claude Code (Anthropic's Claude Opus 5.5) as a programming assistant for:
- writing the `object` and `evaluator` packages (phase 3);
- the REPL and the `lizardscript` command (phase 4);
- later fixes: values that contain themselves, `elif`/`else` on a new line in the REPL, and the nesting and stack limits;
- the `tests/` folder, the examples and a first draft of this document.

**To complete by the team:** state here whether AI was also used in phases 1 and 2 (tokens, lexer, AST, parser) and for what.

How we verified the generated code:
- Every behavior has unit tests, and the output of each program in `tests/` was checked by hand before saving it as the expected result.
- The runner was confirmed to fail when an expected output is wrong.
- For the stack limits, the frame sizes of the evaluator functions were measured. The limits were tested with programs that try every recursive path, right at the limit and past it.

## 7. Extensions beyond the course design

Phases 1 to 4 follow the course's reference interpreter, and that foundation stays: the Go lexer, parser, AST and evaluator. Everything after commit `cc9dc0d` is added on top of it, so programs that worked before keep working the same way. Each extension is recorded here: what was added, why, and what it adds to the original tokens, AST, evaluator or builtins.

| Date | Extension | Why | What it adds to the course design |
| --- | --- | --- | --- |
