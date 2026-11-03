# lizard-script cheat sheet

[Index](../README.md) · [Exercises →](exercises.md) · [Español](../es/cheatsheet.md)

## Syntax

| What | How |
| --- | --- |
| comment | `-~> to the end of the line` |
| create a variable | `let x = 5` |
| change a variable | `x = 6`, `x += 1`, `-=`, `*=`, `/=` (no `**=`, `//=`, `%=`) |
| statement end | a new line or `;` (a line starting with an operator, `(` or `[` continues the one before) |
| condition | `if (a) { } elif (b) { } else { }` (not `else if`) |
| `if` as a value | `let s = if (n > 0) { "+" } else { "-" }` |
| while loop | `while (cond) { }` |
| for loop | `for (let i = 0; i < n; i += 1) { }` (don't forget `let`) |
| leave / skip | `break`, `continue` (inside loops) |
| function | `function f(a, b = 2) { a + b }` or `let f = function(a) { a }` |
| return | the last value, or `return x`; nothing means `null` |
| errors | `error("message")`, `try { } catch (e) { e.message }` (a statement) |
| layer | `layer name` alone on a line, then an indented body |

## Values

| Type | Literals | Notes |
| --- | --- | --- |
| `INTEGER` | `42`, `-7`, `10px` | 64-bit |
| `FLOAT` | `3.5`, `2.0`, `90deg`, `0.5turn`, `1rad` | no `.5`, no `1e3` |
| `STRING` | `"Liz"`, `"a\n\t\"\\"`, `"\u{1F98E}"` | double quotes only; can't be changed |
| `BOOLEAN` | `true`, `false` | |
| `NULL` | `null` | missing index or key, no result |
| `ARRAY` | `[1, "a", [2]]` | shared, index from 0 |
| `HASH` | `{"name": "Liz", 1: true}` | keys: string, integer, boolean; shared |
| `FUNCTION` | `function(x) { x }` | closures |
| `COLOR` | `#f80`, `#3fa34d`, `#00000080` | `c.r c.g c.b c.a`; can't be changed |
| `POINT` | `(1, 2)`, `(1, 2, 3)` | `p.x p.y p.z`; `+ -` points, `* /` numbers |

**False values:** `false`, `null`, `0`, `0.0`, `""`, `[]`, `{}`. Everything else is true.

## Operators, weakest to strongest

`=` `+=` … · `||` · `&&` · `==` `!=` `===` `!==` · `<` `>` `<=` `>=` · `+` `-` · `*` `/` `//` `%` · `-x` `!x` · `**` · `f(x)` `a[i]` `a.b`

```
puts(7 / 2, 7 / 2.0, -7 / 2, -7 // 2, -7 % 3)
puts(2 ** 3 ** 2, -2 ** 2)
puts(5 == "5", 5 === "5", 1 == 1.0, 1 === 1.0)
puts("ab" + "c", "ab" * 2, [1] + [2])
puts(0 || "", 5 && "x")
```

**Output:**

```text
3 3.5 -3 -4 2
512 -4
true false true false
abc abab [1, 2]
false true
```

## Builtins

| Group | Names |
| --- | --- |
| general | `puts len type str int float range error` |
| arrays, hashes | `push pop first last rest keys values delete` |
| colors, points | `rgb rgba hsl point` |
| `math.` | `pi tau sqrt abs floor ceil round min max sin cos tan asin acos atan2 dist angle random seed` |
| `text.` | `slice find split join replace trim lines starts ends upper lower code char number` |
| `list.` | `slice insert remove index reverse sort copy` |

## The Terrarium (chapter 17)

| Module | Names |
| --- | --- |
| `fs.` | `read write append exists list mkdir remove home join` |
| `os.` | `args env exit platform locale config_dir` |
| `time.` | `ms now sleep` |
| `json.` | `parse write(value, indent = 0)` |
| `img.` | `new load save get set pixels from_pixels` + drawing |
| `clip.` | `get set` |
| `window.` | `open(w, h, title, options) run(w, tick) title` + drawing (not in the REPL) |
| `pen.` | `available`; pressure is `e.pen.pressure` in window events |
| drawing (`img.` and `window.`) | `size clear fill(t, [x, y, w, h], c) span(t, y, x0, x1, c) spans blit(t, source, pos)` |

Errors read `fs.read: can't read x.txt: ...` and `try` catches them. Events: `e.type` is `down up move scroll key type resize close`; `e.pos`, `e.button`, `e.key`, `e.text`, `e.pen.pressure`.

## Common errors

| Message | Fix |
| --- | --- |
| `can't assign to x: it was never declared` | `let x = ...` first |
| `identifier not found: x` | typo, or `x` is out of scope |
| `type mismatch: STRING + INTEGER` | `str(n)` |
| `type mismatch: NULL + INTEGER` | a missing index or key; check `=== null` |
| `expected "{", got "if"` | `elif`, not `else if` |
| `expected end of statement, got identifier ...` | `//` is a comment? Use `-~>` |
| `negative exponent` | use a float: `2.0 ** -1` |
| `index out of range` | `push` to add items |
| `wrong number of arguments: want 2, got 1` | check the call, or add a default |

## Running

```sh
go build                  # builds ./lizardscript
./lizardscript            # REPL; exit or Ctrl+D to leave
./lizardscript prog.liz   # run a file
./lizardscript prog.liz a b   # os.args() is ["a", "b"]
```

[Index](../README.md) · [Exercises →](exercises.md)
