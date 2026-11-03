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
            | "continue" end
            | "try" block "catch" [ "(" IDENT ")" ] block [ ";" ]
            | ( ifExpr | whileExpr | forExpr ) [ ";" ]
            | expression end
            | ";" ;
block       = "{" { statement } "}" ;
params      = param { "," param } ;
param       = IDENT [ "=" expression ] ;

expression  = logicalOr [ ( "=" | "+=" | "-=" | "*=" | "/=" ) expression ] ;   (* right associative *)
logicalOr   = logicalAnd { "||" logicalAnd } ;
logicalAnd  = equality { "&&" equality } ;
equality    = comparison { ( "==" | "!=" | "===" | "!==" ) comparison } ;
comparison  = sum { ( "<" | ">" | "<=" | ">=" ) sum } ;
sum         = product { ( "+" | "-" ) product } ;
product     = prefix { ( "*" | "/" | "//" | "%" ) prefix } ;
prefix      = ( "-" | "!" ) prefix | power ;
power       = postfix [ "**" prefix ] ;                 (* right associative *)
postfix     = primary { "(" [ args ] ")" | "[" expression "]" } ;
primary     = IDENT | INT | FLOAT | STRING | COLOR | "true" | "false" | "(" expression ")"
            | "(" expression "," expression [ "," expression ] ")"
            | "[" [ args ] "]" | "{" [ pair { "," pair } [ "," ] ] "}"
            | "function" "(" [ params ] ")" block | ifExpr | whileExpr | forExpr ;
args        = expression { "," expression } ;
pair        = expression ":" expression ;
ifExpr      = "if" expression block { "elif" expression block } [ "else" block ] ;
whileExpr   = "while" expression block ;
forExpr     = "for" "(" [ statement ] ";" [ expression ] ";" [ expression ] ")" block ;

IDENT = ( letter | "_" ) { letter | digit | "_" } ;   INT = digit { digit } ;   FLOAT = digit { digit } "." digit { digit } ;   unit = "deg" | "rad" | "turn" | "px" ;
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
| `code` | The bytecode: opcodes, their encoding, and the compiled unit with its tables. |
| `compiler` | Turns the AST into bytecode for the VM, with the evaluator's rules for scopes and signals. |
| `vm` | `vm.Run(program, env)`: a stack VM that runs the bytecode, a second, faster engine with the same results. |
| `repl` | The interactive loop and `Run`, which runs a whole file (`RunVM` runs it on the VM). Both install the Terrarium first (the REPL without `window` and `pen`). |
| `terrarium` | The standard library: the modules `fs`, `os`, `time`, `json`, `img`, `clip`, and in `terrarium/window` and `terrarium/pen`, `window` and `pen`. See below and [TERRARIUM.md](TERRARIUM.md). |
| `main.go` | `lizardscript` starts the REPL; `lizardscript file.liz [args]` runs a file; `lizardscript --vm file.liz [args]` runs it on the VM. `os.args()` gives the args. |

**Errors.** Each kind of error is detected in its own stage and reported as `[line L, column C] <kind> error: <message>`:
- **Lexical:** the lexer turns a bad character into an `ILLEGAL` token, and the parser reports it as a lexical error.
- **Syntax:** the parser records the error, skips to the end of that statement and carries on, so it reports several independent errors at once. A program with any syntax error doesn't run at all.
- **Runtime:** the evaluator returns an `Error` value. It travels up the tree and stops the program, after any output the program had already printed.

**Control flow.** `return`, `break` and errors are values that travel up through `Eval`:
- a `ReturnValue` stops at the function call it belongs to;
- a `Break` stops at the innermost loop;
- an `Error` goes all the way up.

### The second engine: bytecode compiler and VM

`vm.Run(program, env)` has the contract of `evaluator.Eval(program, env)` and gives the same results: the same value, output, error messages and positions, steps and hook calls. It is several times faster on loops and calls (no environment map per scope, no allocation for small integers), for hosts that run a lot of code. The evaluator stays the default for the REPL and `Run`.

```
AST ──> compiler ──> bytecode (code.Unit) ──> vm ──> values / output
                                                │
                    frames + slots + cells, the host's environment, and the evaluator's operators and builtins
```

The structure follows the sequel of the course book (*Writing a Compiler in Go*), but lizard's rules win where they differ from Monkey's:
- **Resolved at compile time.** Each name a scope declares (lets, parameters, the catch name) gets a slot in its function's frame. The evaluator looks names up at run time in maps, so the compiler decides each read with the same answer: inside a function, a `let` has run before every read after it in its scope and not before any read earlier in it (each block, loop round and call starts its scope fresh), so an earlier read means the scope outside. A closure only knows whether the `let` had run when it was made: if not, its read checks the variable at run time and falls back to the next scope out while it is still unset. A local that a closure uses is a cell, new each time its scope starts, so closures share it by reference and each loop round gets its own.
- **Resolved at run time.** The program's own scope is the host's `env`, read and written by name, so globals added between runs are found, then the builtins, `null`, `math`, `text` and `list`.
- **Signals.** `return`, `break` and `continue` are jumps that close the layers and trys they leave. An `if` or loop used as a value stops them and gets them as its value, and a statement whose value is one of those values acts like it, exactly as in the evaluator.
- **Interop points.** Everything but the fast paths (integer arithmetic and comparisons, array indexing) calls the evaluator's own code through `evaluator/engines.go` (`Infix`, `Index`, `SetIndex`, `ApplyFunction`, `Builtin`, …), so messages can't drift apart. A VM closure is an `object.Callable`: it is a `FUNCTION`, prints its source, and the evaluator and `list.sort` call it; the VM calls the evaluator's functions and the builtins through `evaluator.ApplyFunction`. The step counter, `MaxSteps`, `Output`, the hooks and the call depth (10000 calls of both engines together) are shared.

### The Terrarium: the standard library

The Terrarium is how a program reaches the world outside it: files, the system, time, images, the clipboard, and a window with its input and pen pressure. With it a whole app can be written in lizard-script; the Go code only reaches the system, and what to draw, every pixel of it, is decided by the program. The full reference is [TERRARIUM.md](TERRARIUM.md).

```
host ──> terrarium.Install(env) ──> env: fs os time json img clip (+ window pen) ──> evaluator or vm
                │
     Go modules (one file each) + lizard-script modules (terrarium/lib/*.liz, run into env)
```

- **Modules are values, not syntax.** Each module is a HASH of builtins that `Install` defines in the program's environment, so `fs.read(path)` is plain dot access and a call, both engines run it unchanged, and a program's own `let fs = ...` hides it like any variable. They are made new for each install, so a change to one doesn't reach the next run.
- **Lizard-script first.** A module that needs no system access and no speed is written in lizard-script and embedded (`json` is). It keeps its helpers inside one function that returns the module's hash, so a program's names never meet them.
- **Opaque values.** An image or a window is an `object.Native{Kind, Value}`: type `NATIVE`, shown as `<image 640x480>`, equal only to itself. Only the builtins that know its kind look inside.
- **Drawing targets.** Images and windows are drawn on the same way: `size`, `clear`, `fill`, `span`, `spans` and `blit` are one implementation (`terrarium/internal/canvas`) present in both modules, and each takes either target, so a drawing library can take either one. A new target implements `pixels.Target`.
- **Optional window.** `window` and `pen` need Ebitengine, so they live in their own package, which adds itself to `Install` from its `init` when a host imports it. A host with no screen calls `InstallCore` or doesn't import it, and doesn't link a window. `window.run` calls the program's `tick` through `evaluator.ApplyFunction`, like `list.sort` calls its comparison, so it works with functions of either engine.
- **System differences** live in files with build tags (`sys_darwin.go`, `pen_windows.go`, ...). Nothing needs a C compiler, so the CLI builds for macOS, Windows and Linux from any of them.

## 3. Design decisions

- **A Pratt parser instead of one function per grammar level.** Precedence and associativity live in one table. Right associativity (`=`, `**`) comes from parsing the right side with one level less.
- **Desugaring.** `function f(x) {...}` builds the same tree as `let f = function(x) {...};`. An `elif` is an `if` inside the previous `else`. This keeps the AST and the evaluator small.
- **`if`, `while` and `for` are expressions.** An `if` gives the value of the branch that ran, so `let x = if (a) { 1 } else { 2 }` works. A loop gives `null`.
- **Block scopes.** Every block and every function call gets a new `Environment` linked to the one around it:
  - `let` creates a variable in the current scope, hiding outer ones with the same name;
  - `=` changes the closest existing variable, and assigning to an undeclared name is an error, which catches typos.
- **Errors can be caught, the host's limits cannot.** `try`/`catch` catches any runtime error, but the step limit and the stack overflows are marked `Uncatchable` on the error and go through every `try`: they exist to protect the host, and a program that could swallow them could loop forever in a `catch`.
- **Contextual words, no new keywords.** `try`, `catch` and `continue` are plain identifiers that the parser reads as statements only in the one shape that was invalid before (see section 7), so no program loses a variable by the name.
- **Closures.** A function value keeps the environment where it was created, so it can read and change those variables later.
- **Indentation for layers, braces for code.** Code blocks are closed by braces, and only `layer` bodies are closed by indentation, measured by the column of the first token of each statement with the positions the parser already has (no INDENT or DEDENT tokens). Mixing the two works: a layer inside a function body is measured from the `layer` word, not from the brace, and a tab counts as one column, so a file should not mix tabs and spaces in the same layer.
- **Two equalities.** `===` requires the same type and value. `==` first converts booleans and numeric strings to numbers, so `"5" == 5`. Arrays and hashes compare by content.
- **Integer arithmetic stays exact.** Floats (section 7) only appear when a float is written or produced by a float function, and any operation between two integers is the integer operation it always was.
  - `/` rounds towards zero and `//` rounds down. `%` pairs with `//`, so `a == (a // b) * b + a % b` always holds.
  - `**` with a negative exponent is an error, since the result wouldn't be an integer.
- **Floats never hold Infinity or NaN.** Any operation that would produce one (`10.0 ** 400`, `0.0 ** -1`, `math.sqrt(-1)`) is a runtime error, so a float is always a real number that prints and compares normally.
- **Truthiness.** `false`, `null`, `0`, `""`, `[]` and `{}` are false. `&&` and `||` short-circuit and always give a boolean.
- **Shared data.** Arrays and hashes are references: `let b = a` shares the same array, and functions can change their arguments.
- **Hash keys and order.** Keys keep their insertion order, so output is predictable. Only integers, booleans and strings can be keys (not floats, colors or points), and `1` and `"1"` are different keys.
- **No crashes on adversarial input.**
  - Printing and comparing values use an explicit stack instead of recursion, and detect values that contain themselves (shown as `[...]`).
  - The parser rejects programs nested more than 10000 levels.
  - The evaluator limits call depth (10000 calls) and total depth, so it reports a stack overflow instead of overflowing Go's own stack.
- **The REPL keeps reading lines while the input is unfinished**, like an open bracket or a trailing operator.

## 4. Known limitations

- Integer overflow wraps around, as in Go. Floats are 64-bit and never infinite or NaN (an operation that would produce one is an error).
- After a syntax error the parser skips to the next `;` or `}`. Without semicolons, a second error on a later line of the same block may show up only after fixing the first.
- In the REPL, an `if` without `else` runs once the next line shows it is complete, or after an empty line.
- Nesting is limited to 10000 levels in the source, and recursion to 10000 calls.
- A layer is closed by indentation, so the REPL, which reads one line at a time, runs a `layer` typed there as an empty layer.
- In the REPL, a `try` block is run as soon as its line is complete, so the `catch` must start on the same line as the closing `}` (`} catch (e) {`); in a file it can be on the next one.
- `try` is a statement, not an expression: `let x = try { ... } catch { ... }` is a syntax error. Its value is only seen as the last statement of a function or block.
- A lone `continue` statement inside a loop is a `continue`, so a variable of that name used as a lone statement there (which did nothing) no longer does. Outside a loop it is the plain name, as before.
- Indexing a string is O(1) only after its first use, which costs one pass over the text (and memory for the characters of non-ASCII text); a `String` object caches that and is not safe to share between goroutines.
- An emoji written as a surrogate pair (`"\uD83D\uDE00"`) is an error: write the code point (`"\u{1F600}"`).
- An infinite loop runs forever unless the host sets `evaluator.MaxSteps`; in the REPL, stop it with Ctrl+C.
- The VM has no Go recursion for lizard calls, so the evaluator's second stack limit (`the calls and the code inside them nest too deeply`, for up to 10000 calls whose bodies nest very deeply) never happens on it; only the 10000-call limit does. No test program reaches that limit.

## 5. Testing

`go test ./...` runs everything:
- unit tests for each package (lexer, parser, evaluator, REPL);
- the programs in `tests/`: 29 valid programs and 50 that must fail with a lexical, syntax or runtime error, each compared with its `.expected` output;
- every program in `examples/`;
- the VM: every program of `tests/` and `examples/` also runs on it with the same `.expected` output, and a differential test runs each of the about 1400 programs in the evaluator's unit tests (valid and failing), plus hand-written corners of scopes, signals, try, layers and steps, on both engines, comparing output, value, error, step count and hook calls.

## 6. Use of AI

We used Claude Code (Anthropic's Claude Opus 5.5) as a programming assistant for:
- writing the `object` and `evaluator` packages (phase 3);
- the REPL and the `lizardscript` command (phase 4);
- later fixes: values that contain themselves, `elif`/`else` on a new line in the REPL, and the nesting and stack limits;
- the `tests/` folder, the examples and a first draft of this document;
- the extensions of section 7 (string escapes, dot access, the step limit, floats, `math`, units, colors and points), written with Claude Code.

**To complete by the team:** state here whether AI was also used in phases 1 and 2 (tokens, lexer, AST, parser) and for what.

How we verified the generated code:
- Every behavior has unit tests, and the output of each program in `tests/` was checked by hand before saving it as the expected result.
- The runner was confirmed to fail when an expected output is wrong.
- For the stack limits, the frame sizes of the evaluator functions were measured. The limits were tested with programs that try every recursive path, right at the limit and past it.

## 7. Extensions beyond the course design

Phases 1 to 4 follow the course's reference interpreter, and that foundation stays: the Go lexer, parser, AST and evaluator. Everything after commit `cc9dc0d` is added on top of it, so programs that worked before keep working the same way. Each extension is recorded here: what was added, why, and what it adds to the original tokens, AST, evaluator or builtins.

| Date | Extension | Why | What it adds to the course design |
| --- | --- | --- | --- |
| 2026-11-02 | String escapes | A string could not hold `"` or a line break on purpose. | Lexer only: inside a string, `\"`, `\\`, `\n` and `\t` become one character in the STRING literal; any other `\x` is an ILLEGAL token reported as `unknown escape \x in string`. No new token or AST node. No existing program or test had a backslash in a string. |
| 2026-11-02 | Dot access `a.b` | Reading and writing hash keys with `a["b"]` is noisy. | New token `DOT`, and a parser rule at `INDEX` precedence that builds the same `IndexExpression` with a string key as `a["b"]`, so the evaluator is untouched. A name must follow the dot (`expected a name after "."`). |
| 2026-11-02 | Step limit | A host that runs programs on every keystroke needs a way to stop `while (true) {}`. | Evaluator: `MaxSteps` (0 = no limit, the default) and `ResetSteps()`; one step per loop round and per function or builtin call; past the limit the runtime error `the program ran too long (more than N steps)`. The REPL and `Run` call `ResetSteps()`. No change to tokens, AST or builtins. |
| 2026-11-02 | Floats | Curves and rotations need decimal numbers. | New token `FLOAT` (digits `.` digits; `.5` and `1.` are not floats, `1.x` is still an integer and a dot), AST node `FloatLiteral`, value `Float` with type name `FLOAT`. INTEGER with INTEGER is exactly as before (`1 / 2` is `0`); a float on either side makes the operation float: `/` is real division, `//` floors and `%` pairs with it, `**` uses `math.Pow`, comparisons work by value, `1 == 1.0` is true and `1 === 1.0` false. Division or `%` by zero is the existing error, and a result that would be Infinity or NaN is the runtime error `the result is not a number`. Builtins: new `float(x)`; `int` also takes a float (towards zero). A float is not a hash key. |
| 2026-11-02 | `math` library | Curves, waves and rotations need `sin`, `sqrt`, `random` and friends. | A name `math` that the evaluator resolves after variables and builtins, so a variable called `math` hides it. Its value is a HASH of builtins (`sin cos tan atan2 sqrt abs floor ceil round min max random seed`) and the floats `pi` and `tau`, used with dot access (`math.sin(x)`). The hash is made once per run, so changing it (`math.pi = 3`) does not reach the next run. `random` is deterministic: a fixed default seed, restarted by `evaluator.ResetMath()` (called by the REPL and `Run`, and by hosts before each run next to `ResetSteps`), changed with `math.seed(n)`. No new token or AST node. |
| 2026-11-02 | Number units | Angles and sizes are easier to read as `45deg` or `10px`. | Lexer: a number directly followed by a whole-word unit. `deg`, `rad` and `turn` become a FLOAT in radians, already converted in the token (`45deg` is `0.785...`); `px` keeps the number, integer or float. Any other word after a number lexes as before (number, then identifier), and the parser words that syntax error as `unknown unit "kg"` when the name touches the number. Before, `45deg` was a syntax error, and `5x` still is, so no working program changes. |
| 2026-11-02 | Colors | Drawing code reads better with `#3fa34d` than with three numbers. | New token `COLOR`, and the lexer takes the whole word after a `#`: 3, 6 or 8 hex digits in any case, anything else is an ILLEGAL token worded `bad color #12` (a lone `#` stays `unexpected character "#"`). AST node `ColorLiteral` (the parser decodes the digits; `#rgb` repeats each digit). Value `Color` with four 0 to 255 channels, type name `COLOR`, shown as `#rrggbb` or `#rrggbbaa` when alpha is not 255. Read with dot access `c.r c.g c.b c.a`; assigning is `colors can't be changed`. `==` and `===` compare the channels. Not a hash key. Builtins: `rgb`, `rgba`, `hsl`. `#` was an unexpected character before, so no working program changes. |
| 2026-11-02 | Points | Positions are the main data of a drawing, and `(120, 80)` reads like one. | No new token. The parser reads `(a, b)` and `(a, b, c)` inside parentheses as AST node `PointLiteral`; `(x)` is the grouped expression it always was, while `()` and four or more coordinates are the syntax error `a point has two or three coordinates` (both were syntax errors before). Value `Point` with type name `POINT` holding two or three integers or floats, each keeping its type. Operators `p + q`, `p - q`, `p * n`, `n * p`, `p / n` and `-p` apply the number rules of floats per coordinate (so the zero-division and `the result is not a number` errors apply); mixing 2D and 3D points is an error. Dot access reads `x y z`, assigning is `points can't be changed`; `==` and `===` compare coordinates. Not a hash key. Builtin `point`, and in the `math` hash `dist` and `angle`. |
| 2026-11-02 | `layer` blocks | A drawing is made of named groups, and a program should be able to say which statement made what. | No new token and no reserved word: the parser treats an identifier `layer` followed on the same line by a name (identifier or string) and nothing else as a layer statement. AST node `LayerStatement{Token, Name, Body *BlockStatement}`, whose body is every following statement that starts on a later line in a greater column than `layer` (measured with the token positions; see GRAMMAR.md). `String()` prints `layer name` and the body indented. The evaluator runs the body in a new enclosed scope and gives `null`; `return`, `break` and errors pass through. `layer sun foo` was a syntax error and still is, with a clearer message; `layer` anywhere else is the identifier it was. |
| 2026-11-02 | Host hooks | A host (an editor) needs to know which statement and layer is running to tag what a program makes. | Evaluator only: package variables `OnStatement func(ast.Statement)`, called before every statement of programs, blocks, layer bodies and function bodies, and `OnLayer func(name string, enter bool, tok token.Token)`, called when a layer body starts and when it ends (also on error, return or break). Both are `nil` by default, so a host that sets none sees no change and the cost is a `nil` check per statement. The REPL and `Run` leave them `nil`. |
| 2026-11-02 | `text` and `list` libraries | An editor written in lizard needs to cut, join, search and sort text and arrays. | Two names, `text` and `list`, resolved after variables and builtins like `math`; each is a HASH of builtins (`text.slice find split join replace trim lines starts ends upper lower code char number`, `list.slice insert remove index reverse sort copy`) used with dot access, made once per run and dropped by `ResetMath()`. Text indexes count characters. `list.sort` takes an optional lizard function as the comparison, called through the normal function-call path (so it counts toward the step limit). No new token or AST node. |
| 2026-11-02 | `null` value | `null` could only come out of things; writing it was "identifier not found". | The name `null` is resolved by the evaluator after variables and builtins (and before the libraries) to the one NULL object, so `let null = 5` keeps working. `type(null)` is `NULL`; equality and truthiness are as before. No new token or AST node. |
| 2026-11-02 | `error(message)` and error positions | A library written in lizard needs to report misuse, and a host needs to show it at the user's line. | Builtin `error(message)`: a runtime error whose text is the message (non-strings as `str` shows them), at the position of the call, without the `error:` builtin prefix other builtin errors get. `object.Error` gains `Line` and `Column` ints (filled by `newError`, 0 otherwise); `Message` is unchanged. |
| 2026-11-02 | `Steps()` and `SetSteps(n)` | A host that runs a user's program (with a step limit) from inside its own program needs to save the step count, run the inner one after `ResetSteps()`, and put the count back. | Evaluator: exported `Steps()` (steps since the last reset) and `SetSteps(n)` (restores a saved count). Nothing else changes: no token, AST or builtin, and the limit works as before. |
| 2026-11-02 | Default parameter values | A library function like `t(key, values = null)` should not need every caller to pass every argument. | Parser: a parameter may be followed by `=` and an expression (a syntax error before); a parameter without a default after one with a default is the syntax error `parameter b needs a default value, it comes after one that has it`. AST: `FunctionLiteral.Defaults` (one entry per parameter, nil when none have one) and the same field on `object.Function`; both `String`/`Inspect` print `b = 2`. Evaluator: at each call, missing arguments are filled left to right by evaluating their default in the call's own scope, so `b = a * 2` sees `a`; fewer arguments than the parameters without a default, or more than all parameters, stays the `wrong number of arguments` error (`want 1 to 2, got 0` when defaults exist, the old text otherwise). No new token. |
| 2026-11-02 | `try` / `catch` | A library or an editor must survive a failing user function and show its message instead of stopping. | No new token or reserved word. The parser reads a statement that starts with the identifier `try` followed on the same line by `{` as a try statement (`try {` was a syntax error before, so nothing that worked changes; `try` as a variable, a call or a condition keeps working, and `catch` is read as a word only right after the try block, where it is required). AST `TryStatement{Body, Name, Catch}`; `catch (e)` name optional. Evaluator: a runtime error from the body runs the catch block in a new scope with `e` as the HASH `{"message", "line", "column"}` (message without the position prefix); `return`, `break` and `continue` pass through; the value is the one of the block that ran. `object.Error` gains `Uncatchable`, set on the step limit and the stack overflows (and kept when a builtin like `list.sort` re-reports an error from the function it called), which a try never catches. |
| 2026-11-02 | `continue` | Skipping a round with a nested `if` is noisy, and `break` already existed. | No new token or keyword: a new keyword would break a program with a variable called `continue`, so the parser reads a statement made only of the identifier `continue` (followed by `;`, `}`, the end of the input or a line that does not go on with an operator, `(` or `[`) as a continue; `continue + 1`, `continue = 1`, `continue(1)`, `f(continue)` and `let continue = 1` keep their meaning. It is only read that way inside a loop body of the same function; outside one (or in a function inside a loop) it stays the identifier it was, so it reads a variable or gives `identifier not found`. The only changed meaning is a lone `continue` statement inside a loop, which used to be a variable read with no visible effect. AST `ContinueStatement`; value `object.Continue` (type `CONTINUE`) that goes up to the innermost loop, which goes on with its next round (in a `for` the update still runs). |
| 2026-11-02 | Compound assignment | `count = count + 1` and `list[i] = list[i] + 1` repeat the target and, for `a[f()]`, would run it twice. | Lexer: tokens `+=`, `-=`, `*=`, `/=` (`PLUS_ASSIGN` and kin), taken before `+ - * /`; each was a syntax error before (an operator followed by `=`), and `**=`, `//=`, `%=` still are. Parser: they are infix at `ASSIGN` precedence, right associative, with the same targets as `=`. AST: `AssignExpression` gains `Operator` (`""` for `=`, otherwise `+ - * /`). Evaluator: the target's container and index are evaluated once, the current value is read first (an unknown name is `identifier not found`, a missing item reads as `null`), the right side is evaluated, the operator is applied by the same code as `x + e` (the infix evaluation was split in two to share it), and the result is stored. |
| 2026-11-02 | `math.asin` and `math.acos` | Angles from a ratio (a slope, a dot product) need the inverses of `sin` and `cos`. | Two more builtins in the `math` hash (`math.asin(x)`, `math.acos(x)`, radians, as floats); an `x` outside -1 to 1 is an argument error like `math.sqrt(-1)`. No new token or AST node. |
| 2026-11-03 | `\r` and `\u` escapes | Text with carriage returns or characters the editor can't type needs to be written in a literal. | Lexer only: `\r`, `\uXXXX` (exactly 4 hex digits) and `\u{X...}` (1 to 6) become one character in the STRING literal; a malformed escape, a surrogate half (D800 to DFFF) or a value above 10FFFF is an ILLEGAL token whose literal is the escape, worded by the parser as `invalid unicode escape ...: reason`. They were `unknown escape` errors before, so no working program changes (one lexer test case that used `\r` as the unknown escape now uses `\q`). |
| 2026-11-03 | Cached string characters | `s[i]` and `len(s)` converted the whole string each time, so a loop over a long string was quadratic. | `object.String` keeps unexported cache fields filled on first use (ASCII text needs no copy) and gains `Len()`, `Runes()` and `RuneAt(i)`; `Value` and the printed output are unchanged. The evaluator's `s[i]`, `len` and `text.slice`, `find` and `code` use them. |
| 2026-11-03 | Source names | A host loads libraries and the user's program into one environment and must tell which source a statement or error came from. | `lexer.NewNamed(input, name)` (`New` is unchanged, name `""`); `token.Token.Source`; `object.Error.Source`, filled by `newError` from the token. Names are not in any message. The parser copies the source onto the `let` token it builds for `function name() {}`. |
| 2026-11-03 | Bytecode compiler and VM | A host that runs a lot of code (redrawing on every change) needs a faster engine than walking the tree. | New packages `code` (opcodes, encoding, compiled unit), `compiler` (AST to bytecode) and `vm` (`vm.Run(program, env)`, same contract and results as `evaluator.Eval`; `vm.RunUnit` for a program compiled once), `repl.RunVM` and `lizardscript --vm file.liz`. The evaluator stays the default and is untouched in meaning. It gains `evaluator/engines.go`, exports that wrap what Eval already does (`ApplyFunction`, `Infix`, `Prefix`, `Index`, `SetIndex`, `IsTruthy`, `Builtin`, `CountStep`, `CallDepth`/`SetCallDepth`, `ErrorHash`, `NewError`, `MaxCallDepth`); `object.Callable`, an interface for function values of another engine, which `applyFunction` calls before "not a function" and `list.sort` accepts as a comparison. No token, AST node or builtin changes. See section 2. |
| 2026-11-03 | The Terrarium, a standard library | A whole app (a paint program, a game) should be writable in lizard-script alone, with files, images, a window and pen input, and no host of its own. | New package `terrarium` (`Install`, `InstallCore`, `Register`, `Args`) with the modules `fs`, `os`, `time`, `img`, `clip` in Go and `json` in lizard-script (embedded), and the packages `terrarium/window` (Ebitengine, only to open the window, read input and show the buffer) and `terrarium/pen` (pressure on macOS and Windows Ink). New value `object.Native` (type `NATIVE`) for images and windows. `repl.Run` and `RunVM` install the whole Terrarium, the REPL all but `window` and `pen`; `lizardscript file.liz args...` passes the args to `os.args()`. No token, AST node, evaluator or VM change. A program that used one of the names `fs`, `os`, `time`, `json`, `img`, `clip`, `window` or `pen` without defining it got `identifier not found` before and now gets the module; a program that defines one hides the module, so no working program changes. New dependencies: Ebitengine, purego, golang.org/x/sys. See section 2 and TERRARIUM.md. |
