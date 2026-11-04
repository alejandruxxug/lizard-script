# LEP 0001: A statement ends at the end of its line

- **Author(s):** the lizard-script team
- **Status:** Final <!-- Draft · Accepted · Rejected · Deferred · Final -->
- **Created:** 2026-11-03
- **Decision:** Accepted by the developer 2026-11-03 (option 4, easiest to teach).
- **Discussion:** to be opened as a pull request

## Summary

A line that started with `-` (or `(` or `[`) used to go on with the line before it, so a forgotten `;` could silently turn two statements into one expression. Now **a statement ends at the end of its line. It goes on to the next line only when the line ends with an operator, a comma or an opening bracket, or while a `(`, `[` or hash `{` is still open.** This proposal listed five options; option 4 was chosen.

## Motivation

The parser has no required `;`, so it needs a rule to know where a statement ends. The rule before this LEP: it ended at a `;`, at a `}` or the end of the input, or at a new line, **unless** the next token could continue an expression. A token that could continue an expression was any infix operator (`-`, `+`, `*`, `==`, ...), `(` for a call and `[` for an index.

That made this program print `4`, not `5` and `-1`:

```
let a = 5
-1
puts(a)      -~> before this LEP: prints 4, the first two lines were let a = 5 - 1
```

A line that started with `(` or `[` was the same: a call or an index on the line before.

```
let f = function(x) { 7 }
let r = f
(1)          -~> before this LEP: r was f(1), which is 7, not the function f
```

The same rule was what let a long expression be split before its operator, which people did on purpose:

```
let total = first
  + second
  - discount
```

A learner who wrote a negative number on its own line, or started a line with `(` or `[`, got a wrong value or a confusing runtime error, with no syntax error to point at the line before.

## The change

Options, from the smallest change to the biggest:

1. **Do nothing, document it.** Keep the rule and teach it (the guide's "common mistakes" already does). Nothing breaks.
2. **Only `-` is special.** A line that starts with `-` ends the previous statement (so it is a negative number), while every other operator still continues. `-` is the one operator that is also a prefix, so it is the only one that can start a statement on its own. A multi-line sum would have to put the `-` at the end of the line, not at the start.
3. **Prefix-capable tokens end the statement** (`-`, `!`, `(` and `[`): a line that starts with one of them is a new statement. Splitting before `+`, `*`, `==`, `&&`, `.` and other binary-only operators still works.
4. **A statement always ends at the end of its line**, unless the line ends with an operator, a comma or an open bracket (the way the REPL already decides that an input is unfinished). Splitting before an operator no longer works; people split after it. **This is the one that was chosen.**
5. **Require `;` or a new line**, and make `a\n+ b` a syntax error that tells the author to join the lines or move the operator to the end of the previous line.

The rule as built (option 4, with the error message of option 5):

- A line goes on when it ends with a binary operator (`+ - * / // % ** == != === !== < > <= >= && ||`), `=` or a compound assignment (`+= -= *= /=`), `!`, `.`, `,`, `(`, `[` or `{`. A comment at the end of the line doesn't count.
- Inside `(`, `[` and a hash's `{` a line break is just a space, so multi-line calls, arrays, points, hashes and conditions in parentheses keep working. A hash is treated like the other brackets because it only ever holds expressions, never statements. The `{` of a block starts statements again, even inside brackets (`f(function() { ... })`).
- A line that starts with an operator that only goes between two values, right after a finished line, is a syntax error that says how to fix it. The line is then read as part of the one before, so the parse stays in step and later mistakes are still found.
- `return` takes a value only when it starts on the line of the `return`.
- A statement that isn't finished yet still reads the next line: `let x` then `= 1`, `if (a)` then `{`, and `else`, `elif` or `catch` on the line after a `}`.
- `;` still separates statements on one line.

```
let x = 10
-1           -~> a new statement: x is 10
let total = first +
  second -
  discount   -~> the operators at the end of the lines keep it going
let t = first
  + second   -~> [line 2, column 3] syntax error: a line can't start with +; put the + at the end of the previous line
puts(1,
  2)          -~> inside brackets a new line is just a space
```

The REPL follows the same rule when it decides whether an input is unfinished: it shares `lexer.ContinuesLine` with the parser.

## How other languages do it

| Language | Rule | `a` then a line `+ b` | `a` then a line `-1` |
|---|---|---|---|
| JavaScript | A new line is whitespace; a `;` is inserted only where the next token can't continue (with exceptions such as after `return`). | one expression, `a + b` | one expression, `a - 1` |
| Lua | A new line is whitespace; statements need no separator at all. | one expression | one expression |
| Python | A new line ends the statement, except inside `(`, `[` and `{` or after a `\`. | two statements (`+ b` is a unary plus) | two statements |
| Go | The lexer inserts a `;` at a line end after an identifier, a literal, `)`, `]`, `}`, `return` and a few more, so an operator must end the line. | two statements (`+ b` is an error as a statement) | two statements |
| Ruby | A new line ends the statement, unless the line ends with an operator, a comma or an open bracket. | two statements (`+ b` is a unary plus) | two statements |
| Kotlin | A new line ends the statement, but a line that starts with `.`, `?.`, `&&` or `\|\|` continues the one before. | two statements | two statements |
| Swift | Whether an operator is binary or prefix depends on the whitespace around it, and a line break counts as whitespace. | one expression (space on both sides of `+`) | two statements (`-1` touches its operand, so it is a prefix) |
| **lizard-script** | Ends at the end of the line, unless the line ends with an operator, a comma or an opening bracket, or a `(`, `[` or hash is open. | error with a fix-it message | two statements |

lizard-script is closest to Ruby and Go, with Python's brackets: it ends where Ruby ends, but reports a leading binary operator instead of reading it as a prefix, and keeps going inside brackets as Python does.

## What breaks

Measured with the implementation, `go test ./...` and `tools/checknewlines`:

- A line that starts with an operator to continue the line before (`+ b`, `&& c`, `.name`, `= 5`) is now a syntax error.
- A line that starts with `-`, `(` or `[` after a finished line is now its own statement.
- A `return` whose value starts on the next line now returns `null`.

In this repository no `.liz` program, example, Terrarium library or `.expected` file changed. Three Go test strings did (`hanoi` in `parser_test.go`, `try\n+ 1` and `continue\n- 1` in the evaluator and parser tests); each now has the operator at the end of the line and tests the same thing. The guide's examples of the old surprise were rewritten.

To move a program, run `go run ./tools/checknewlines file.liz...`. It lists every line whose meaning changed; put the operator at the end of the line before, or join the two lines, or wrap the expression in parentheses.

## Implementation

- Token: `LineStart`, true for the first token of a line (a line break inside a multi-line string doesn't count).
- Lexer: `ContinuesLine(type)`, the list of tokens that let a line go on, used by the parser and the REPL.
- Parser: a stack of open brackets (a `{` is a block until `parseHashLiteral` marks it as a hash); `lineEnds()` is the rule in one place; `parseExpression`'s infix loop, `return` and `continue` use it; `endStatement` compares `LineStart` instead of line numbers. `LineChanges()` lists the lines whose meaning changed, for the migration tool.
- No AST node changes, so the evaluator, the compiler and the VM do not change: both engines read the same tree.

## Alternatives

- **Do nothing** (option 1) was a fair answer: the rule was simple, and most languages without required semicolons (JavaScript, Go for some tokens) have a corner like this.
- **A formatter or a linter warning** for a line that starts with `-`, without changing the language, so the surprise is reported and no program breaks.
- Options 2 and 3 break less but leave two rules to learn ("these operators continue, those don't"). Option 4 is one sentence.

## Open questions (answered)

- *Is a warning (no breakage) enough for learners?* The developer chose a rule over a warning: a learner reads one sentence instead of meeting the surprise.
- *If `-` becomes special (option 2), should `+` be too?* Not needed: with option 4 no operator continues from the start of a line.
- *Should the REPL, which reads one line at a time, behave the same as a file?* Yes. It already waited for more after an operator, a comma or an open bracket; it now uses the same function as the parser.
