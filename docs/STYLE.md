# lizard-script style

How lizard-script code is laid out. `lizardscript fmt` writes it this way (package `format`), so nobody has to: the same program always comes out the same, whoever wrote it.

```text
$ lizardscript fmt file.liz         prints the formatted code
$ lizardscript fmt -w src           rewrites every .liz in src and the folders inside it
$ lizardscript fmt --check src      lists the files that are not formatted, exit code 1 if any
```

A file with a syntax error is left alone: fmt prints the errors and ends with exit code 2.

## The rules

- **Indentation**: two spaces for each block, bracket and layer body. The next lines of a statement that goes on (after an operator at the end of a line) get two more. A closing bracket goes back to the indentation of the line that opened it. Tabs become spaces.

  ```
  function total(list) {
    let sum = 0
    for (let i = 0; i < len(list); i += 1) {
      sum += list[i] *
        2
    }
    return sum
  }
  ```

- **Spaces**: one around binary operators (`a + b`, `x = 1`, `b = 2` in a parameter list), one after a comma, a `;` or a `:`. None inside brackets (`f(1, 2)`, `[1, 2]`, `{"a": 1}`), around a `.`, after a prefix `-` or `!`, or before the `(` of a call or of a function's parameters: `function(a, b)`, `f(x)[0].name`. A block on one line has a space inside its braces, `{ return 1 }`; an empty one is `{}`.
- **Braces**: a block's `{` goes on the line of what it belongs to, `if (a) {` and not `{` on the next line.
- **else, elif and catch**: on the line of the `}` before them, `} else {`, `} elif (b) {`, `} catch (e) {`, when that block was written over several lines. After a block written on one line they stay on the next line, so a chain of short blocks stays one per line:

  ```
  if (n < 0) { "negative" }
  elif (n == 0) { "zero" }
  else { "positive" }
  ```

- **Hashes**: a hash written on one line stays on one line. A hash that spans lines has one pair per line, and its `}` on a line of its own:

  ```
  let pond = {
    "frogs": 2,
    "lilies": 3
  }
  ```

- **Blank lines** are kept, but never two in a row, and never at the start or the end of the file.
- **Comments**: a comment on a line of its own is indented like the code around it. The comments at the end of consecutive lines start in one column, one space after the longest code among them.

  ```
  let width = 320 -~> pixels
  let h = 200     -~> pixels

  let scale = 2 -~> a blank line starts another group
  ```

- **Layers**: a `layer` starts its own line, and its body is indented two spaces more, like a block.
- **Line ends**: no spaces at the end of a line, `\n` line breaks, and one line break at the end of the file.

## What fmt never does

- **Change what the program means.** It only changes the spaces between tokens, and the line breaks the parser reads as a space: inside brackets and hashes, before a block's `{`, before an `else`, `elif` or `catch` after a `}`, before a `layer` after a `;`. Every other line break stays where it is, so a statement still ends at the end of its line ([LEP 0001](proposals/0001-statement-ends-at-newline.md)), and an operator that continues a line stays at its end. Before giving its result, fmt checks that the tokens and the syntax tree are the ones of the input; if they weren't (a bug), it leaves the file alone.
- **Reflow long lines.** A long line stays long; split it yourself after an operator or a comma.
- **Touch strings.** A string, even one over several lines, is written exactly as it was, and so is a number with a unit (`45deg`) or an escape (`"\u{1F98E}"`).
- **Add or take away semicolons.**
