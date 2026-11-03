# 1. First steps

[Index](../README.md) · [Next: Values and types →](02-values-and-types.md) · [Español](../es/01-first-steps.md)

In this chapter you will start the interpreter, type your first lines, run a program from a file and learn how to write comments.

## Two ways to run code

lizard-script runs code in two ways:

- **The REPL** (Read, Eval, Print, Loop): you type a line, it runs, and you see the result right away. Great for trying things.
- **A file**: you write a whole program in a file ending in `.liz` and run it. Great for anything longer than a few lines.

If you haven't built the interpreter yet, see [How to install and run](../README.md#how-to-install-and-run). From now on, the command is called `./lizardscript` (on Windows, `lizardscript.exe`).

## The REPL

Run the interpreter with no arguments:

```sh
./lizardscript
```

You get a lizard prompt, `🦎 >>`. Type an expression and press Enter. The REPL shows its value:

```text
🦎 >> 1 + 2
3
🦎 >> "Liz" * 2
"LizLiz"
🦎 >> let toes = 20
🦎 >> toes - 1
19
🦎 >> exit
```

A few things to notice:

- `"Liz" * 2` shows `"LizLiz"` **with quotes**. The REPL shows values the way you would write them in code, so you can tell the string `"5"` from the number `5`.
- `let toes = 20` shows nothing. Creating a variable has no value to show.
- Variables live on between lines: `toes` still exists on the next line.
- `exit` leaves. Ctrl+D (Ctrl+Z then Enter on Windows) also works.

If a line is unfinished, like an open `(`, `[` or `{`, the REPL changes the prompt to `🦎 ..` and keeps reading until you close it:

```text
🦎 >> function double(n) {
🦎 ..   n * 2
🦎 .. }
🦎 >> double(21)
42
🦎 >> exit
```

> **Try it.** Start the REPL and compute how many seconds there are in a day (`24 * 60 * 60`).

## Printing with `puts`

`puts` writes its arguments separated by spaces and then starts a new line. Strings are written without quotes:

```
puts("Hello, lizard!")
puts("I have", 4, "legs and", 1, "tail")
puts()
puts("puts() alone wrote an empty line")
```

**Output:**

```text
Hello, lizard!
I have 4 legs and 1 tail

puts() alone wrote an empty line
```

`puts` accepts any number of arguments and any kind of value. In the REPL, `puts` prints its text and then the REPL shows nothing else, because `puts` itself gives `null` (no value).

## Running a file

Create a file called `hello.liz` with this content:

```
puts("Hello from a file")
puts("2 + 2 is", 2 + 2)
```

**Output:**

```text
Hello from a file
2 + 2 is 4
```

and run it:

```sh
./lizardscript hello.liz
```

A file does **not** show the value of each line like the REPL does: only what you print with `puts` appears.

## Comments

A comment starts with `-~>` and runs to the end of the line. It looks like a little lizard running to the right. The interpreter skips it.

```
-~> This whole line is a comment.
puts("code runs")   -~> and this part is a comment too
-~> puts("this doesn't run")
```

**Output:**

```text
code runs
```

Why not `//` like in many languages? Because in lizard-script `//` is an operator: integer division (chapter 4). Using it for a comment is a common mistake:

```
puts("hi") // a comment
```

**Output:**

```text
[line 1, column 17] syntax error: expected end of statement, got identifier "comment"
```

The interpreter read `// a comment` as "divide by `a`" and then got confused by `comment`. And `#` doesn't work either, because `#` starts a color (chapter 12):

```
puts("hi")
# a comment
```

**Output:**

```text
[line 2, column 1] lexical error: unexpected character "#"
```

## Semicolons and lines

A statement ends at the end of its line, so semicolons are optional. You can use them to put several statements on one line:

```
let a = 1; let b = 2; puts(a + b)
let c = 3
puts(c)
```

**Output:**

```text
3
3
```

## Reading error messages

Every error says **where** it happened (line and column) and **what kind** it is:

- **lexical error**: a character the language doesn't know, like `@`;
- **syntax error**: the pieces are in the wrong order, like a missing `)`;
- **runtime error**: the program was fine but something went wrong while running, like dividing by zero.

```
puts("before the mistake")
puts(10 / 0)
puts("never printed")
```

**Output:**

```text
before the mistake
[line 2, column 9] runtime error: division by zero
```

The first line ran, then the program stopped at line 2, column 9 (the `/`). With a syntax error, nothing runs at all, not even the first line, because the whole program is checked before it starts:

```
puts("this line never runs")
let total = (1 + 2
```

**Output:**

```text
[line 3, column 1] syntax error: expected ")", got end of input
```

Chapter 14 is all about errors.

## Summary

- `./lizardscript` opens the REPL; `./lizardscript file.liz` runs a file.
- `puts(a, b, ...)` prints values separated by spaces.
- Comments start with `-~>`. Not `//` and not `#`.
- `;` is optional at the end of a line.
- Errors look like `[line L, column C] kind error: message`.

[Index](../README.md) · [Next: Values and types →](02-values-and-types.md)
