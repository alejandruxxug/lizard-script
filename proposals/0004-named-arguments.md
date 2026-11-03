# LEP 0004: Named arguments

- **Author(s):** the lizard-script team
- **Status:** Draft <!-- Draft · Accepted · Rejected · Deferred · Final -->
- **Created:** 2026-11-03
- **Discussion:** to be opened as a pull request

## Summary

A call can name its arguments: `circle(center: (1, 2), r: 5)`. A named argument goes to the parameter with that name, whatever its place, so a call can give the fifth parameter and leave the ones before it to their defaults. Positional arguments come first, named ones after. Builtins take names too when they declare them, and a builtin whose last parameter is an options hash collects the extra names into it: `window.open(320, 180, "game", scale: 3)`. `name: value` in a call was a syntax error, so nothing that works changes.

## Motivation

Drawing code is full of calls whose numbers mean nothing without the definition next to them, and default parameters only help with the last ones:

```
-~> Positional: which number is which?
function rect(pos, size, radius = 0, stroke = 1, fill = #000) {
  puts(pos, size, radius, stroke, fill)
}
rect((0, 0), (40, 20), 0, 3)        -~> to give stroke, radius must be written too

-~> An options hash: readable, but a typo is silent and defaults are by hand.
function rect2(pos, size, o = {}) {
  let stroke = if (hash.has(o, "stroke")) { o.stroke } else { 1 }
  puts(pos, size, o.radius, stroke)
}
rect2((0, 0), (40, 20), {"stroke": 3})
rect2((0, 0), (40, 20), {"strok": 3})  -~> typo: no error, stroke stays 1
```

Today, with `./lizardscript`:

```
(0, 0) (40, 20) 0 3 #000000
(0, 0) (40, 20) null 3
(0, 0) (40, 20) null 1
```

The options hash is what the Terrarium does (`window.open`, `lizard.run`, `fs.list`), and each of those builtins checks the keys by hand so a typo isn't silent:

```
window.open(10, 10, "x", {"scal": 2})
```

```
[line 1, column 12] runtime error: window.open: unknown option "scal" (the options are automation, floating, fps, frameless, headless, hover, min_size, on_demand, resizable, scale, transparent)
```

A lizard function gets no such check unless its author writes it, and its defaults move from the parameter list into `if`s in the body.

The syntax is free today:

```
function circle(center, r = 1) { puts(center, r) }
circle(center: (1, 2), r: 5)
```

```
[line 2, column 14] syntax error: expected ")", got ":"
```

`=` is not free: inside a call it is already an assignment, which is why this LEP uses `:`:

```
function show(a) { puts("got", a) }
let r = 0
show(r = 5)          -~> an assignment, not a named argument
puts(r)
```

```
got 5
5
```

## The change

```
-~> with this LEP (each failing call as if it were the only one)
function rect(pos, size, radius = 0, stroke = 1, fill = #000) {
  puts(pos, size, radius, stroke, fill)
}
rect((0, 0), (40, 20), stroke: 3)              -~> (0, 0) (40, 20) 0 3 #000000
rect(size: (40, 20), pos: (0, 0))              -~> (0, 0) (40, 20) 0 1 #000000
rect((0, 0), (40, 20), strok: 3)               -~> [line 7, column 24] runtime error: no parameter named strok (the parameters are pos, size, radius, stroke, fill)
rect((0, 0), pos: (1, 1), size: (2, 2))        -~> [line 8, column 14] runtime error: pos is given twice, by position and by name
rect(stroke: 3)                                -~> [line 9, column 5] runtime error: missing argument pos
let w = window.open(320, 180, "game", scale: 3, fps: 30)
```

Grammar (for [GRAMMAR.md](../docs/GRAMMAR.md)): only the arguments of a call change; arrays, points and hashes keep `args` and `pair` as they are.

```ebnf
postfix   = primary { "(" [ callArgs ] ")" | "[" expression "]" } ;
callArgs  = callArg { "," callArg } [ "," ] ;
callArg   = [ IDENT ":" ] expression ;
```

A name is an identifier followed by `:`. After the first named argument every argument must be named: `f(a: 1, 2)` is the syntax error `a positional argument can't come after a named one`. The same name twice in one call is the syntax error `r is named twice`. Anything else before a `:` (`f("r": 5)`, `f(a.b: 1)`) is the syntax error `a named argument is a name followed by ":", like r: 5`. Inside the parentheses a line break is a space, as today ([LEP 0001](0001-statement-ends-at-newline.md)), so a long call can put one named argument per line.

Meaning (for [SEMANTICS.md](../docs/SEMANTICS.md#functions)):

1. **Evaluation order.** The argument expressions run left to right as written, positional and named alike, before anything is bound. `f(b: g(), a: h())` runs `g` before `h`.
2. **Binding.** The positional arguments fill the first parameters, as today. Each named argument then goes to the parameter of its name. Errors, at the position of the name (or of the call for a missing one): `no parameter named strok (the parameters are pos, size, radius, stroke, fill)`, `pos is given twice, by position and by name`, and `missing argument pos` for a parameter that got nothing and has no default.
3. **Defaults.** Every parameter still empty gets its default, from left to right, in the call's own scope, exactly as today. So a default sees every earlier parameter, whether it came by position, by name or by its own default: `function f(a, b = a * 2)` called as `f(b: 1, a: 5)` has `a` 5, `b` 1, and `f(a: 5)` has `b` 10.
4. **A call without names** is exactly what it is today: same binding, same messages (`wrong number of arguments: want 1 to 2, got 0`), same speed.
5. **Builtins.** A Go builtin can declare its parameter names (and which of them have a default). Then named arguments are bound to it the same way, and a gap left by a missing optional argument gets the builtin's declared default. A builtin that declares no names takes none: `len(x: [1])` is `len takes no named arguments`. A builtin whose last parameter is an **options hash** (`window.open(w, h, title, options = {})`) also takes, by name, every option it knows: `window.open(320, 180, "game", scale: 3)` is `window.open(320, 180, "game", {"scale": 3})`, so the builtin's own check still reports `unknown option "scal"`. Giving both the hash and option names is an error.
6. **Other function values** take names the same way: closures of the VM, the functions of a Terrarium module written in lizard-script (`json.write(value, indent: 2)`), and the functions a builtin calls (`list.sort`'s comparison is called by position, as today).
7. **Steps and hooks.** A call is one step whatever its arguments, and the hooks see the same statements.
8. **Parameter names become part of a function's interface.** Renaming a parameter now can break a caller that names it. The guide and `lizard.docs` say so; `lizard.docs` already lists every parameter with its name and default, which is what an editor needs to complete `circle(r: `.

What `lizard.docs` gives today, with `./lizardscript`:

```
let code = "-~> A circle around center.\n-~> r: how big, in pixels\nfunction circle(center, r = 1) { 0 }"
let d = lizard.docs(code)
puts(d.circle.params)
```

```
[{"name": "center", "default": null, "doc": null}, {"name": "r", "default": "1", "doc": "how big, in pixels"}]
```

`lizard.parse` gives a call with named arguments one more key, `"names"`: an array with one entry per argument, `null` for a positional one and the name for a named one. A call with no names has no `"names"` key, so the trees of programs that work today are the same.

## How other languages do it

| Language | Syntax | Mixing positional and named | Must callers name them? | Unknown names |
|---|---|---|---|---|
| Python | `f(x, r=5)` | Positional first; positional after a keyword is a `SyntaxError`. `*` makes keyword-only parameters, `/` positional-only. | No. | `TypeError`; `**kwargs` collects them instead. |
| Kotlin | `f(x, r = 5)` | Since 1.4 a named argument may be followed by positional ones when all are in their own place. Not for Java methods. | No. | Compile error. |
| Swift | `f(center: p, r: 5)` | Labels are part of the function's name and must be in declaration order. | Yes, by default; `_` makes a label optional. | Compile error. |
| C# | `f(center: p, r: 5)` | Since 7.2 a named argument in its own position may be followed by positional ones. | No. | Compile error. |
| JavaScript, Lua | none; an object or table: `f({r: 5})`, `f{r = 5}` | — | — | Silent, like the options hash above. |
| **lizard-script** | `f(center: p, r: 5)` | Positional first, then named, in any order. | No. | Runtime error with the list of parameters. |

lizard-script takes the `:` of Swift and C# (the `=` of Python and Kotlin is already an assignment inside a call), the any-order, optional names of Python and C#, and Python's simplest mixing rule (positional first). Its defaults are expressions run at each call, like Kotlin's, not constants like C#'s.

## What breaks

Nothing. `IDENT ":"` inside a call's parentheses was a syntax error (run above). Hash literals inside calls (`f({"r": 5})`) are read as before, and so is every call without a name, in meaning, messages and bytecode. `lizard.parse` adds `"names"` only to calls that have named arguments.

## Implementation

- **ast:** `CallExpression` gains `Names []*Identifier`, nil when no argument is named, otherwise one entry per argument (nil for a positional one). `String()` prints `r: 5`.
- **parser:** `parseCallArguments` reads `IDENT ":"` before an argument (one token of look-ahead past the identifier), with the three syntax errors above.
- **evaluator:** one function, `bindNamed(params, defaults, args, names)`, in `engines.go` so the VM calls the same code and the messages can't drift apart. `applyFunction` uses it when a call has names, for `*object.Function`, `object.Callable` (which gains `ParameterNames()`) and builtins that declare names.
- **object:** `Builtin` gains `Params []Param{Name, Default Object, Options bool}`, empty for builtins that declare none.
- **Terrarium:** `arg.Module` takes the parameter names of each function (they are already in TERRARIUM.md, `img.save(target, path, rect = all)`), and the options builtins mark their last one. The modules written in lizard-script pass their functions' names through. This can be done module by module: until a builtin declares its names, it takes none.
- **compiler and VM:** a call with names compiles to `OpCallNamed argc names tok`, where `names` is a constant (the names of the trailing named arguments). The VM keeps, per call site, the last function it bound there and the order it found, so a call in a loop binds by copying, not by looking names up. A call without names is the `OpCall` it is today.
- **Rough size:** small: a parser rule, one binding function, an opcode, and the names of the Terrarium builtins, with tests in each package and programs in `tests/valid` and `tests/errors`.

## Alternatives

- **Do nothing; use an options hash.** Works today (above), but every function checks its keys and fills its defaults by hand, and a typo is silent unless it does.
- **`f(r = 5)` as in Python and Kotlin.** Reads well, but `f(r = 5)` already assigns `5` to `r` and passes it (run above). Changing that would change working programs.
- **Swift's required labels.** Clear at every call, but every existing call would break, and short calls (`len(x)`, `puts(a)`) don't need them.
- **Only for builtins (the options hash with nicer syntax).** Smaller, but lizard functions are where the defaults and the typos are.
- **Hash shorthand, `f({r: 5})` with a bare word as a string key.** `{r: 5}` already uses the variable `r` as the key.

## Open questions

- Should lizard functions be able to **collect unknown names**, like Python's `**kwargs`, so a function written in lizard can take options the way `window.open` does? A marker on the last parameter (`function f(a, ...options)`) would do it.
- Should a **positional argument be allowed after a named one** in its own place, as Kotlin and C# allow? Python's rule is easier to teach; this can be relaxed later without breaking anything.
- **Positional-only parameters** (Python's `/`): should some lizard functions refuse names, so their parameters can be renamed freely? Builtins that declare no names already do.
- Which Terrarium builtins should declare names first? All of them in one change, or only the ones with optional arguments (`img.save`, `fs.list`, `clear`, `spans`, `composite`, `window.open`)?
- With classes ([LEP 0003](0003-classes.md)), calling a class with names (`Sprite(name: "liz", pos: (1, 2))`) would bind to its `init`; both LEPs should agree.
