# The Terrarium: lizard-script's standard library

The Terrarium is where lizards live. It is the set of modules every lizard-script program can use with no setup: files, the system, time, JSON, images, the clipboard, and a window to draw in with its input, pen pressure included. With it a whole app (a paint program, a game, an editor) can be written in lizard-script alone:

```
lizardscript examples/terrarium/paint.liz
```

Each module is a hash of builtins used with a dot, like `math`: `fs.read(path)`. A program's own `let fs = ...` hides the module, like it hides any name. The modules are made again for every run, so a program that changes one (`fs.read = 1`) can't affect the next.

The Go code only reaches the system. What to draw, and every pixel of it, is decided by the program: the window shows a buffer the program filled.

| Module | What it is for |
| --- | --- |
| [`fs`](#fs) | files and folders |
| [`os`](#os) | the program's arguments, the environment, the system |
| [`time`](#time) | clocks and waiting |
| [`json`](#json) | reading and writing JSON |
| [`img`](#img) | images: load, save, read and change pixels |
| [`clip`](#clip) | the system clipboard |
| [`io`](#io) | the standard input, output and error |
| [`lizard`](#lizard) | the language itself: tokens, syntax trees, runs in a sandbox, doc comments |
| [`window`](#window) | a window to draw in, its surfaces and its input; headless windows and automation |
| [`pen`](#pen) | pen and trackpad pressure |

[Drawing targets](#drawing-targets) explains the drawing functions that `img`, `window` and surfaces share.

[Standard modules](#standard-modules) are bigger libraries written in lizard-script that come with every `lizardscript` but are loaded only with `use`, like the [flower](flower/README.md) graphics family: `use "flower/lily2d" as lily2d`.

## Errors

A builtin that can't do its work stops the program with a runtime error whose message starts with the builtin's name. Every one of them can be caught with `try`:

```
try {
  fs.read("missing.txt")
} catch (e) {
  puts(e.message)   -~> fs.read: can't read missing.txt: no such file or directory
}
```

A wrong argument says what was wanted: `fs.read: want a STRING path, got INTEGER`, `wrong number of arguments: want 2, got 1`. The errors below are the ones beyond those.

## Values

Most functions take and give the usual values: strings, integers, floats, points, colors, arrays, hashes. Images, windows and surfaces are a value of their own, `NATIVE`: a program keeps them in variables and passes them to the functions that know them, but can't look inside. `puts` shows them as `<image 640x480>`, `<window 600x400>` or `<surface canvas 800x600>`. Two of them are equal only when they are the same one.

Paths use `/` between their parts on every system; Windows takes it too. In a program made with `lizardscript build`, the paths that start with `bundle:` are its own files (see [Built programs](#built-programs)).

---

## fs

Files and folders.

### fs.read(path)
The text of the file. Errors when it can't be read: `can't read notes.txt: no such file or directory`.
```
let text = fs.read("notes.txt")
```

### fs.write(path, text)
Replaces the file with `text`, or makes it. Another program never sees half a file: the text goes to a new file next to it, which then takes its place. A file that was there keeps its permissions. Errors when the folder doesn't exist or `path` is a folder.
```
fs.write("notes.txt", "lizards like sun\n")
```

### fs.append(path, text)
Adds `text` at the end of the file, or makes it.
```
fs.append("log.txt", "one more line\n")
```

### fs.exists(path)
Whether there is a file or folder at `path`.
```
if (!fs.exists("out")) { fs.mkdir("out") }
```

### fs.list(dir, options = {})
The names of what is in the folder, files and folders, in alphabetical order. Errors when it isn't a folder. With the option `{"info": true}` each one is a hash instead: its `name` and what [`fs.info`](#fsinfopath) gives. An unknown option is an error.
```
puts(fs.list("."))   -~> ["README.md", "docs", "examples", ...]
let files = fs.list("drawings", {"info": true})
puts(files[0])   -~> {"name": "cat.lzdraw", "size": 812, "mtime": 1793735180, "dir": false}
```

### fs.info(path)
What is at `path`: `{"size": bytes, "mtime": seconds since 1970 when it last changed, "dir": whether it is a folder}`. Errors when there is nothing: `can't read x: no such file or directory`.
```
let newer = fs.info("a.txt").mtime > fs.info("b.txt").mtime
```

### fs.rename(from, to)
Moves a file or folder to a new path (also to another folder). It refuses to replace something that is already at `to`: `can't move a.txt to b.txt: there is already something there`, so a mistake can't lose a file.
```
fs.rename("draft.txt", "final.txt")
```

### fs.copy(from, to)
Copies a file (not a folder), with its permissions. Like `fs.rename`, it refuses to replace something at `to`.
```
fs.copy("settings.json", "settings.backup.json")
```

### fs.trash(path)
Moves a file or folder to the Trash, where the user can get it back: `~/.Trash` on macOS (Finder, when the file is on another disk), the Recycle Bin on Windows, the freedesktop Trash on Linux (`$XDG_DATA_HOME/Trash`, with the `.trashinfo` file managers use to put it back; `gio trash` for another disk). A name already in the Trash gets ` 2`, ` 3`, ... before its extension. It never deletes: when the Trash can't take it, the file stays and it is the error `can't move x to the Trash: ...`.
```
fs.trash("old drawing.lzdraw")
```

### fs.mkdir(path)
Makes the folder and the ones above it that are missing. A folder that is already there is fine.
```
fs.mkdir("out/pictures/2026")
```

### fs.remove(path)
Deletes a file or an empty folder. It refuses a folder with something in it (`can't remove out: the folder is not empty`), so a mistake can't delete a whole tree.
```
fs.remove("notes.txt")
```

### fs.home()
The user's home folder.
```
let dir = fs.join(fs.home(), "Pictures")
```

### fs.join(a, b, ...)
Puts the parts of a path together with `/` and tidies it. Takes one or more strings.
```
fs.join("a/", "./b", "../c")   -~> "a/c"
```

### fs.digest(path, algorithm = "sha256")
The hash of the file's bytes as lowercase hex, like [`text.digest`](SEMANTICS.md#the-text-library) of its text: `"sha256"` (64 digits) or `"fnv"` (16 digits, fast, for caches). Two files with the same digest have the same contents, so it names a thumbnail or a cache entry after what is in the file. It reads a built program's own files (`bundle:`) too.
```
let name = fs.digest("drawing.lzdraw") + ".png"
fs.digest("notes.txt", "fnv")   -~> "e71fa2190541574b" for a file that holds abc
```

## os

The program's arguments and environment, and the system it runs on.

### os.args()
The program's arguments as strings, the ones after its file: `lizardscript tool.liz in.png out.png` gives `["in.png", "out.png"]`.
```
let files = os.args()
```

### os.env(name)
The value of the environment variable, or `null` when it isn't set.
```
let editor = os.env("EDITOR")
```

### os.exit(code = 0)
Ends the program now with the exit code, 0 to 255.
```
if (len(os.args()) == 0) { puts("usage: tool.liz file"); os.exit(2) }
```

### os.platform()
`"macos"`, `"windows"` or `"linux"`.

### os.locale()
The user's language as a lowercase two-letter code, like `"es"`. It asks the system, then `LC_ALL` and `LANG`, and is `"en"` when none of them says.

### os.config_dir()
The folder where programs keep the user's settings: `~/Library/Application Support` on macOS, `%AppData%` on Windows, `~/.config` on Linux. It doesn't make it.
```
let settings = fs.join(os.config_dir(), "my-tool", "settings.json")
```

### os.program_dir()
The folder of the program's main file, as an absolute path, so a program finds the files that come with it wherever it is run from. In a [built program](#built-programs) it is `"bundle:"`, where `fs` reads its own files. In the REPL it is the current folder.
```
let icons = fs.list(fs.join(os.program_dir(), "icons"))
```

### os.reveal(path)
Shows the file or folder in the file manager: selected in Finder or Explorer, its folder opened on Linux (`xdg-open`). Errors when there is nothing at `path`.

### os.open_url(url)
Opens a web address in the user's browser, or a `mailto:` address in the mail program. Only `http://`, `https://` and `mailto:` are taken (`only http, https and mailto addresses can be opened`), so a program can't run a file through it.
```
os.open_url("https://github.com/")
```

### os.memory()
The machine's memory in bytes: `{"total": all of it, "free": what can be used now}` (free and inactive memory on macOS, `MemAvailable` on Linux). `free` is never more than `total`.
```
let m = os.memory()
puts(m.free * 100 // m.total, "% free")
```

## time

### time.ms()
The milliseconds since the program started, as a float. It only goes forward, even when the system clock changes, so it is the one to measure with.
```
let start = time.ms()
work()
puts("took", time.ms() - start, "ms")
```

### time.now()
The local date and time as a hash: the integers `year`, `month` (1 to 12), `day`, `hour` (0 to 23), `minute`, `second` and `weekday` (1 for Monday to 7 for Sunday), `epoch`, the seconds since 1970-01-01 UTC as a float with the fractions (the same instant on every machine, to store and subtract), and `utc_offset`, the seconds the local time is ahead of UTC (negative behind it: `-18000` in Bogotá).
```
let now = time.now()
puts(now.hour, ":", now.minute)
let saved = now.epoch
-~> later
puts(int((time.now().epoch - saved) / 3600), " h ago")
```

### time.utc()
`time.now()` in UTC: the same keys, the date and hour of Greenwich, and a `utc_offset` of 0. The `epoch` is the same as `time.now()`'s.
```
let u = time.utc()
puts(u.year, "-", u.month, "-", u.day)
```

### time.sleep(ms)
Waits that many milliseconds, an integer or a float from 0 to 3600000 (an hour).
```
time.sleep(500)
```

## json

Written in lizard-script itself (`terrarium/lib/json.liz`); its functions are builtins like the others'.

Objects are hashes, arrays are arrays, strings are strings, `true` and `false` are booleans and `null` is `null`. A number without a fraction or exponent is an `INTEGER` (a `FLOAT` when it is too big for one); any other is a `FLOAT`.

### json.parse(text)
The value of the JSON text. A mistake is an error that says where it is: `json.parse: line 2, column 3: unexpected 'x', expected a value`. Escapes, `\u` escapes and surrogate pairs are decoded. Objects and arrays nest at most 100 deep.
```
let lizard = json.parse(fs.read("liz.json"))
puts(lizard.name)
```

### json.write(value, indent = 0)
The JSON text of `value`. With an `indent` above 0 (at most 16), each item goes on its own line, nested by that many spaces. A `COLOR` is written as its `"#rrggbb"` text and a `POINT` as `[x, y]` or `[x, y, z]`; a hash key that isn't a string is written with `str()`. A function, or a value that contains itself, is an error: `json.write: can't write a FUNCTION as JSON`.
```
fs.write("liz.json", json.write({"name": "Liz", "toes": [5, 5, 5, 5]}, 2))
```

## img

Images. An image keeps every pixel's color as it was set, alpha included. An image is a [drawing target](#drawing-targets), so every drawing function (`img.fill`, `img.spans`, `img.span_gradient`, `img.push_layer`, ...) works on it too. While a [layer](#push_layertarget) is open on an image, `img.get`, `img.set` and `img.pixels` use the layer.

### img.load(path, max_side = 0)
Reads a PNG, JPEG or GIF file (a [built program](#built-programs)'s own files too). With a `max_side` above 0, an image longer than that on its longer side is shrunk to fit, averaging the pixels (it is never enlarged). Errors when the file can't be read, isn't an image, or is more than 32768 pixels on a side.
```
let photo = img.load("photo.jpg", 256)
```

### img.save(target, path, rect = all)
Writes an image, a window or a surface as a PNG file, the whole file at once like `fs.write`. With `rect`, `[x, y, w, h]`, only that part of it.
```
img.save(picture, "out.png")
img.save(picture, "corner.png", [0, 0, 64, 64])
```

### img.png_base64(target, rect = all)
The PNG of a target (or of the part `rect` of it) as base64 text: a picture another program can take inside JSON. Errors when `rect` is all outside the target.
```
let data = img.png_base64(window.screenshot(w))
```

### img.new(w, h, color = #00000000)
A new image of `w` by `h` pixels (0 to 32768 each, at most 64M pixels in all), filled with `color`.
```
let picture = img.new(256, 128, #ffffff)
```

### img.get(image, x, y)
The color of the pixel at column `x`, row `y`, from the top left. Errors outside the image: `the pixel (2, 0) is outside the 2x2 image`.

### img.set(image, x, y, color)
Replaces the pixel with `color`. It doesn't mix it with the color below; `img.fill` and the other drawing functions do.
```
for (let x = 0; x < 256; x += 1) { img.set(picture, x, 0, rgb(x, 0, 0)) }
```

### img.pixels(image)
Every color of the image in one flat list, row by row from the top left: the pixel `(x, y)` is at index `y * width + x`. A flat list is the fastest way for a loop to read a whole image.

### img.from_pixels(size, pixels)
A new image of `size`, a point `(w, h)`, made from a flat list of `w * h` colors in the order `img.pixels` gives them.
```
let flipped = img.from_pixels(img.size(picture), list.reverse(img.pixels(picture)))
```

## clip

The system clipboard, through the system's own tools: `pbcopy` and `pbpaste` on macOS, PowerShell on Windows, `wl-copy`, `xclip` or `xsel` on Linux. When there is no tool, the clipboard still works inside the program.

### clip.get()
The text on the clipboard, `""` when there is none.

### clip.set(text)
Puts `text` on the clipboard.
```
clip.set(json.write(drawing))
```

## io

The program's standard streams, for a program that talks to another one through them, like a server that reads one request per line, or that prints without the line break of `puts`.

### io.read_line()
The next line of the input, without its line break (and without a `\r` before it), or `null` when the input has ended. A last line with no line break still counts. Lines longer than 32 MB are an error.
```
let line = io.read_line()
while (line != null) {
  puts(text.upper(line))
  line = io.read_line()
}
```

### io.write(text)
Writes `text` where `puts` writes, with nothing after it, so `io.write("a"); puts("b")` prints `ab`.

### io.error(text)
Writes `text` to the error output, with nothing after it. A program that talks through its output (a server) logs here.
```
io.error("got a request\n")
```

In Go, `terrarium.Stdin` and `terrarium.Stderr` are where they read and write (`repl.RunFile` points `Stderr` at its error writer), and `io.write` writes to `evaluator.Output`, like `puts`.

## lizard

The language itself, as data: an editor colors and checks code with it, a tool runs a user's code in a sandbox, and a help panel reads the doc comments of a library. Positions are a `line` and `column` from 1, counted in characters (a tab is one), and an `end_line` and `end_column` just after the last character.

### lizard.tokens(code, options = {})
The tokens of `code`, comments included, in order. Each one is a hash:

| Key | What it is |
| --- | --- |
| `kind` | for coloring: `"name"`, `"keyword"`, `"number"`, `"string"`, `"color"`, `"operator"`, `"punctuation"`, `"comment"` or `"error"` (something the lexer rejected, like an unterminated string) |
| `type` | the exact token: `"IDENT"`, `"INT"`, `"FLOAT"`, `"LET"`, `"+="`, `"("`, `"COMMENT"`, ... |
| `text` | what is written, quotes and escapes included |
| `value` | what it means: a string without quotes and escapes, `45deg` in radians |
| `line`, `column`, `end_line`, `end_column` | where it is |

```
puts(lizard.tokens("x = 45deg")[2])
-~> {"kind": "number", "type": "FLOAT", "text": "45deg", "value": "0.7853981633974483", "line": 1, "column": 5, "end_line": 1, "end_column": 10}
```

With `{"columns": true}` it is one hash of those keys instead, each a list with one element per token: `kind[i]`, `text[i]`, `line[i]` and the rest are token `i`. No hash is made per token, so a program that walks the tokens of a big text (an editor's scanner) gets them about four times faster (220,000 tokens: about 30 ms against 125 ms on an M4); the strings that repeat, like each kind and type and the text of each operator, are one object each. An unknown option is an error.
```
let t = lizard.tokens("x = 45deg", {"columns": true})
puts(t.kind, t.column)   -~> ["name", "operator", "number"] [1, 3, 5]
```

### lizard.parse(code, name = "")
The syntax tree of `code`: `{"program": node, "errors": [...]}`. Each error is `{"kind": "syntax" or "lexical", "message", "line", "column"}`, the message without its position. With errors, the program has the statements that parsed. `name` is the source name of the code.

Each node is a hash with its `kind`, its positions (a node in parentheses includes them; a statement includes its `;`) and the fields of its kind. A missing part (an `else` that isn't there, a parameter without a default) is `null`.

| kind | fields |
| --- | --- |
| `program`, `block` | `statements` |
| `let` | `name`, `value`, `declaration` (true when written `function name(...) { }`) |
| `return` | `value` |
| `break`, `continue` | |
| `try` | `body`, `error_name` (the name in `catch (e)`), `catch` |
| `expression` | `expression` (a statement that is an expression) |
| `layer` | `name` (a string), `body` (a `block` without braces) |
| `use` | `path` (a `string` node), `as` |
| `name` | `value` |
| `integer`, `float`, `string`, `color`, `boolean` | `value` |
| `point` | `coords` |
| `prefix` | `operator`, `right` |
| `infix` | `operator`, `left`, `right` |
| `assign` | `operator` (`"="`, `"+="`, ...), `target`, `value` |
| `if` | `condition`, `consequence`, `alternative` (an `elif` is an `if` alone in the alternative) |
| `while` | `condition`, `body` |
| `for` | `init`, `condition`, `update`, `body` |
| `function` | `params` (`name` nodes), `defaults` (one node or `null` per parameter), `body` |
| `call` | `callee`, `arguments` |
| `array` | `elements` |
| `index` | `left`, `index` |
| `dot` | `left`, `name` (a `string` node): `a.b` |
| `hash` | `pairs` (each `{"key", "value"}`) |

```
let tree = lizard.parse("let area = w * h")
let value = tree.program.statements[0].value
puts(value.kind, value.operator, value.column, value.end_column)   -~> infix * 12 17
```

### lizard.run(code, options = {})
Runs `code` in a sandbox: a new top scope with only the builtins, the libraries and what `options.globals` gives (no Terrarium, unless given), from a fresh start (its own step count, the default random seed, new libraries, its own modules). The program that calls it goes on exactly as before: its steps, random numbers, output and hooks are its own again. It gives:

| Key | What it is |
| --- | --- |
| `value` | the code's value, `null` when it failed |
| `error` | `null`, or `{"kind": "syntax", "lexical" or "runtime", "message", "line", "column", "source"}`, the message without its position (for a syntax error, the first one) |
| `errors` | every error: each syntax error of a code that doesn't parse, in order (so a host doesn't parse it again to list them), the one that stopped a code that ran, or none; `error` is the first |
| `output` | what the code printed, with `puts` or `io.write` |
| `steps` | the steps it took, while a limit was set |
| `globals` | a hash of its top-level names after the run, the given ones included |

The options:

| Option | What it does |
| --- | --- |
| `globals` | a hash of names to define first, like `{"fs": fs, "size": 10}` |
| `steps` | the most steps it may take (a loop round or a call is one), 0 for no limit. Past it, the error is `the program ran too long (more than N steps)` |
| `name` | its source name, the `source` of its errors, and where its `use` statements look for modules |
| `engine` | `"evaluator"` (the default) or `"vm"`; both give the same results |
| `use` | a list of module paths, like `["draw/shapes", "draw/colors"]`: the code sees them alive before its first line, as if it started with `use "draw/shapes"` for each one, found from its `name`. Writing a module's variable (`cache_limit = 4`) changes the module itself, which a copy in `globals` can't. They load once per run, on the code's engine; a module that fails is the run's `error` |
| `on_statement` | a function called with the line, the column and the source name before each statement runs |
| `hook_source` | a source name: `on_statement` is called only for the statements of that source, usually the code's own `name`, not for the ones of its modules. Much cheaper than leaving the filter to the hook |
| `hook_functions` | `true` (the default) or `false`: with `false`, `on_statement` is not called for the statements of function bodies, only for the ones outside every function (a module's too, and the ones in layers, blocks and loops). On the VM, the functions then run without the hook's instructions |
| `on_layer` | a function called with the layer's name, `true` or `false` and its line when a layer starts and ends |

The two hooks run as the calling program: their steps count in its limit, and what they print is its output. An error in a hook stops the run and is the error of `lizard.run` (`lizard.run: a hook failed: ...`). Modules the code uses see the same `globals` around them. A module's file is parsed (and, on the VM, compiled) once while its text stays the same, so many runs with the same libraries don't pay for them again; their code runs once in each run.
```
let r = lizard.run("let area = w * h\nputs(area)", {"globals": {"w": 3, "h": 4}, "steps": 1000})
puts(r.output, r.globals.area, r.error)   -~> 12\n 12 null

let drawing = lizard.run(code, {"name": "drawing.liz", "use": ["draw/shapes"], "hook_source": "drawing.liz",
  "on_statement": function(line, column, source) { rows = rows + [line] }})
```

### lizard.docs(code)
The doc comments of the functions at the top level of `code`: the `-~>` lines right above each `function name(...)` or `let name = function(...)`. It gives a hash from each name to `{"name", "line", "summary", "params", "example"}`:

- `summary` is the first lines, up to a blank `-~>` line, a parameter line or the example, joined with spaces;
- `params` has one `{"name", "default", "doc"}` per parameter: the default as written (`"(1, 2)"`) or `null`, and the text of its `name: ...` line (lines after it go on it) or `null`;
- `example` is what follows `example:` (lines after it go on it, with line breaks), or `null`.

```
-~> A circle around center.
-~> radius: how big, in pixels
-~> example: circle((10, 10), 5)
function circle(center, radius = 1) { ... }
```
gives `{"circle": {"name": "circle", "line": 4, "summary": "A circle around center.", "params": [{"name": "center", "default": null, "doc": null}, {"name": "radius", "default": "1", "doc": "how big, in pixels"}], "example": "circle((10, 10), 5)"}}`. Code with syntax errors gives the functions that parsed.

## Standard modules

A standard module is a library written in lizard-script that is part of every `lizardscript` (and of every built program), but is not installed into every program like the modules above: it is big, so only a program that asks for it with [`use`](SEMANTICS.md#modules) pays for loading it. `use` finds it after the program's own folder and the folders of `$LIZARD_PATH`, so a file of the program's own with the same path wins.

```
use "flower/lily2d" as lily2d
lily2d.circle(picture, (50, 50), 20, #f2a65a)
```

| Module | What it is for |
| --- | --- |
| [`flower/lily2d`](flower/LILY2D.md) | anti-aliased 2D vector graphics on any drawing target (the [flower](flower/README.md) family) |
| [`flower/lily3d`](flower/LILY3D.md) | 3D graphics on any drawing target: meshes, `.obj` files, cameras, lights, fog, flat and smooth shading, drawn with [`triangles`](#trianglestarget-list-options--) |

The files are in `terrarium/std/`, built into the binary: `use "flower/lily2d"` runs `terrarium/std/flower/lily2d.liz`, and that file uses its own parts next to it (`use "lily2d/raster"` is `terrarium/std/flower/lily2d/raster.liz`). A standard module's source name starts with `std:`, so its errors show where they come from: `std:/flower/lily2d/check.liz: [line 7, column 8] runtime error: ...`. It runs like any module, with the Terrarium around it, and `lizardscript build` leaves it out of the bundle, since every `lizardscript` has it. In Go, `evaluator.StandardModules` is the file system they are read from (the Terrarium sets it in `std.go`) and `evaluator.IsStandardModule(path)` tells their source names.

### lizard.check(code, name = "", names = [])
Finds mistakes in `code` without running it, the way `lizardscript check` does (package `check`): an array with one hash per finding, in the order of the code.

| Key | What it is |
| --- | --- |
| `severity` | `"error"` (the code fails when it runs, or doesn't run at all) or `"warning"` (probably a mistake) |
| `rule` | which check found it: `"syntax"`, `"lexical"`, `"unknown-name"`, `"assign"`, `"arguments"`, `"member"`, `"module"`, `"return"` (errors); `"unused"`, `"shadow"`, `"unreachable"`, `"literals"`, `"unused-module"`, `"self-assign"` (warnings). `member` is an error when the member is called (`math.sine(1)`), a warning when it is only read |
| `message` | what is wrong, without the position |
| `hint` | how to fix it, like ``"did you mean `size`?"``, or `null` |
| `line`, `column`, `end_line`, `end_column` | what it is about |

The code has the Terrarium's names (and the builtins and libraries); `names` lists more that the host running it will define, so they aren't unknown. `name` is its source name, where its `use` statements look for modules; a module that can't be found is an error, and then any name may come from it. With syntax errors, only the syntax errors and the rules that need no names are reported.

```
let found = lizard.check("let size = 10\nputs(siz * 2)")
puts(found[0].severity, found[0].message, found[0].hint)   -~> error identifier not found: siz did you mean `size`?
puts(lizard.check("draw(canvas)", "", ["draw", "canvas"]))   -~> []
```

## Built programs

`lizardscript build dir -o app` makes one executable of the program in `dir`: its `main.liz`, every module it uses (and the ones they use, which must be inside `dir`) and the data files that `dir/bundle.txt` lists, one path or pattern per line (`assets/*.png`; a folder takes everything in it; a line that starts with `-~>` is a comment). `--os` and `--arch` build it for another system (`--os windows --arch amd64`, `--os linux --arch arm64`), `--vm` runs it on the VM, and `--runtime FILE` gives a `lizardscript` built for the target. The executable is a copy of `lizardscript` with the files appended, so it needs nothing installed. For this system it copies the running `lizardscript`; for another one it builds `lizardscript` from its source with Go (it looks for the source in and above the current folder and the program's, or in `$LIZARD_SRC`).

While a built program runs, its files are at paths that start with `bundle:`: `os.program_dir()` is `"bundle:"`, its main file is `bundle:/main.liz`, its modules are found there, and `fs.read`, `fs.exists`, `fs.list`, `fs.info`, `fs.copy` (as the source) and `img.load` read them. They can't be changed: `fs.write` and the others are the error `the files of a built program can't be changed`. A program that finds its files with `fs.join(os.program_dir(), ...)` works the same way built or not. In Go, `bundle.ReadFile`, `bundle.Stat` and `bundle.ReadDir` read such a path, or one on disk.

`lizardscript run dir args...` runs `dir/main.liz` without building it.

## Drawing targets

An image, a window and a [surface](#surfaces) are all drawing targets: they are drawn on with the same functions, so a drawing library can take any of them. The functions are in both modules (`img.fill(w, ...)` and `window.fill(w, ...)` are the same function), and each takes any target.

Painting mixes the color over what is there, so a color with alpha shows what is below. `clear` replaces. The ends of a span and the sides of a rectangle can be fractional: the pixel they cut is covered in part, so the edge is soft. What falls outside the target, or outside its [clip](#cliptarget-rect), is cut.

### size(target)
`(width, height)` in pixels.

### clear(target, color, rect = all)
Every pixel becomes `color`, or every pixel of the rectangle `rect`. A transparent color erases.
```
window.clear(w, #f4efe6)
img.clear(layer, #00000000, [10, 10, 40, 20])
```

### fill(target, rect, color)
Paints the rectangle `rect`, an array `[x, y, w, h]` of numbers.
```
img.fill(picture, [20, 54, 215.5, 20], #ffffffc0)
```

### span(target, y, x0, x1, color)
Paints row `y` (rounded down) from `x0` up to `x1`. A shape is drawn as one span per row:
```
function circle(target, c, r, color) {
  for (let dy = -r; dy <= r; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    img.span(target, c.y + dy, c.x - half, c.x + half, color)
  }
}
```

### spans(target, list, color, offset = (0, 0))
Paints every `[y, x0, x1]` of `list` in one color, moved by `offset`: the fast way to fill a shape worked out row by row, and to draw it again somewhere else without making a new list.
```
window.spans(w, [[10, 5, 20], [11, 4, 21], [12, 5, 20]], #2b3a55)
window.spans(w, star, #f6c453, (100, 40))
```

### blit(target, source, pos, options = {})
Paints the pixels of `source`, an image or a window, over `target` with its top left at `pos`. It is how a picture gets into a window, and how a window's pixels get into an image (to save a screenshot).

The options paint it at another size: `"scale"`, a number above 0 (`2` is twice as big, `0.25` a quarter), and `"filter"`, `"nearest"` (the default: each pixel becomes a sharp square, the look of pixel art) or `"linear"` (smooth: it mixes the four nearest pixels when enlarging, and averages every pixel it covers when shrinking, so a thumbnail keeps its thin lines). The size is rounded to whole pixels, at least one; colors mix with their alpha.
```
window.blit(w, img.load("lizard.png"), (10, 10))
let shot = img.new(600, 400)
img.blit(shot, w, (0, 0))
img.save(shot, "shot.png")

let thumb = img.new(150, 100)
img.blit(thumb, shot, (0, 0), {"scale": 0.25, "filter": "linear"})
window.blit(w, sprite, (0, 0), {"scale": 4})   -~> big, sharp pixels
```

### clip(target, rect)
### unclip(target)
`clip` cuts every drawing on the target to the rectangle `rect` until `unclip`; a new clip replaces the old one. `clear` keeps to the clip too. A panel that draws its contents with no care for its edges stays inside them:
```
window.clip(w, [20, 40, 200, 120])
draw_list(items)
window.unclip(w)
```

### span_gradient(target, y, x0, x1, gradient)
### spans_gradient(target, list, gradient, offset = (0, 0))
`span` and `spans` with a gradient instead of a color: each pixel takes the gradient's color at its middle. With `spans_gradient`, the gradient is in the space of `list`, before `offset`, so a shape moved with an offset takes its gradient with it. A gradient is a hash:

| key | what it is |
| --- | --- |
| `kind` | `"linear"` or `"radial"` |
| `origin` | a point: where a linear gradient's `t` is 0, the center of a radial one |
| `axis` | linear: the direction from `t` 0 to `t` 1 divided by its squared length, so `t = (p - origin) · axis` |
| `scale` | radial: 1 / the radius, so `t` is the distance from `origin` times `scale` |
| `stops` | `[[t, color], ...]` in order of `t`, from 0 to 1; two stops at the same `t` make a hard edge |
| `steps` | optional: 2 or more cuts the gradient into that many flat bands |

Before the first stop the color is the first stop's, after the last the last one's; between two stops it is mixed, alpha included.
```
-~> From (0, 0) to (200, 0): the axis is (200, 0) / 200².
let sunset = {"kind": "linear", "origin": (0, 0), "axis": (1 / 200.0, 0),
  "stops": [[0, #f6c453], [0.5, #e2557b], [1, #3b4cc0]]}
window.spans_gradient(w, rows, sunset)
let sun = {"kind": "radial", "origin": (100, 100), "scale": 1 / 40.0, "stops": [[0, #ffe9a8], [1, #e2557b]], "steps": 4}
window.span_gradient(w, 100, 60, 140, sun)
```
Malformed gradients are errors that say what is wrong: `bad gradient: a linear gradient needs a point "axis"`.

### push_layer(target)
### pop_layer(target, mode = "normal", opacity = 1)
`push_layer` opens a new transparent layer on the target, with the clip it had: everything drawn on the target goes to the layer until `pop_layer`, which closes it and mixes it onto what is below with a blend mode and an opacity (0 to 1). Layers nest, up to 64. A group of shapes drawn at half opacity this way doesn't show its own overlaps, and a group in `"multiply"` darkens what is below like ink:
```
window.push_layer(w)
window.spans(w, leaf, #30d050)
window.spans(w, shadow, #000000)
window.pop_layer(w, "multiply", 0.8)
```
`pop_layer` with no layer open is an error: `no layer to pop: call push_layer first`.

### composite(target, source, mode = "normal", opacity = 1, pos = (0, 0))
Mixes `source`, any target, onto `target` with its top left at `pos`, with a blend mode and an opacity: `blit` with a blend mode. A layer made once in an image can be composited many times.
```
img.composite(w, glow, "screen", 0.6, (40, 40))
```

### triangles(target, list, options = {})
Fills every triangle of `list`, keeping the nearest pixel: the one step of 3D drawing done pixel by pixel (the [lily3d](flower/LILY3D.md) library does everything else in lizard-script). A triangle is `[a, b, c, color]` in one color, or `[a, color_a, b, color_b, c, color_c]` with the colors mixed from its corners. A corner is a point `(x, y, z)`: `x` and `y` in pixels, `z` its depth, smaller nearer (`(x, y)` is depth 0). A pixel is filled when its middle is inside the triangle, or on a top or left edge, so two triangles that share an edge never both fill a pixel of it, or both miss one. Its depth and color are mixed from the corners' by how near the middle is to each.

The target keeps a depth buffer: a pixel is only painted when it is nearer than what was painted there before, and then it keeps its depth. Colors with alpha are painted over what is there. Either turn of the corners is drawn (leaving out the backs is the program's job), and what is outside the target or its clip is cut. The options:

| Option | Default | What it does |
| --- | --- | --- |
| `depth` | `true` | `false` paints in order, with no depth test |
| `clear_depth` | `false` | `true` makes the depth buffer empty first: call it once per frame, before the first triangles |

```
img.triangles(picture, [[(10, 10, 0.5), (90, 20, 0.5), (40, 80, 0.5), #f2a65a]], {"clear_depth": true})
img.triangles(picture, [[(0, 0, 0.2), #ff0000, (100, 0, 0.9), #00ff00, (0, 100, 0.9), #0000ff]])
```
A malformed triangle is the error `triangle 0 must be [a, b, c, color] or [a, color, b, color, c, color] with points (x, y, z), got ...`. In Go it is `pixels.Image.Triangle` and `ClearDepth` (`internal/pixels/triangles.go`).

### Blend modes

The blend modes of `pop_layer` and `composite` follow the W3C compositing rules: where both are there, the source color becomes `(1 - ab) * Cs + ab * B(Cb, Cs)` (`Cb` the color below, `ab` its alpha, `Cs` the source color) and then goes over what is below.

| mode | `B(Cb, Cs)` | what it looks like |
| --- | --- | --- |
| `normal` | `Cs` | paint |
| `multiply` | `Cb * Cs` | ink: always darker |
| `screen` | `Cb + Cs - Cb * Cs` | light: always lighter |
| `overlay` | multiply where `Cb` is dark, screen where it is light | contrast |
| `darken` | the smaller of the two | |
| `lighten` | the bigger of the two | |
| `add` | `Cb + Cs`, at most 1 | glow |

`examples/terrarium/layers.liz` shows all seven over a gradient.

## window

A window that shows a buffer of pixels the program draws on, with the [drawing target](#drawing-targets) functions (`window.size`, `window.fill`, `window.spans`, `window.span_gradient`, `window.push_layer`, ...), and over it the [surfaces](#surfaces) the program gives it. A program shows one window on the screen in its life; [headless](#headless-windows) windows, which have no screen, can run any number of times.

### window.open(w, h, title, options = {})
Makes a window of `w` by `h` pixels (1 to 8192 each), black at first. It shows on the screen when `window.run` starts, so the program can draw its first picture before. The options are:

| Option | Default | What it does |
| --- | --- | --- |
| `scale` | 1 | each pixel shows as `scale` by `scale` on the screen, 1 to 8 |
| `resizable` | false | the user can resize the window; the buffer follows, and a `resize` event says the new size |
| `frameless` | false | no title bar or border |
| `transparent` | false | the pixels the program leaves transparent show what is behind the window (a round splash screen); only `window.open` can choose it, and it stays for the window's life |
| `floating` | false | the window stays above every other window |
| `min_size` | none | a point `(w, h)`: the smallest buffer the user can resize the window to |
| `fps` | 60 | frames a second, 1 to 240 |
| `hover` | true | `move` events also come with no button down; with false, only while a button is held |
| `on_demand` | false | `tick` runs only in frames that have events, after [`window.needs_paint`](#windowneeds_paintw), and in the first one: an app that waits for input costs nothing while it waits |
| `headless` | false | no screen: see [Headless windows](#headless-windows) |
| `automation` | false | another program can drive the window: see [Automation](#automation) |

An unknown option is an error, so a typo doesn't go unnoticed.
```
let w = window.open(320, 180, "game", {"scale": 3, "fps": 30})
```

### window.run(w, tick)
Shows the window and calls `tick(events)` every frame, with the list of events since the last frame (often empty), until `tick` returns `false` or the window is closed. Closing it sends one last `close` event first. The window shows the buffer again after each frame in which it was drawn on. `tick` can be any function, with default parameters too, and it runs the same on both engines. An error in `tick` closes the window and is `run`'s error, reported like an error in `list.sort`'s comparison: the place of the `window.run` call, then the error inside `tick` (`[line 3, column 11] runtime error: window.run: [line 5, column 9] runtime error: ...`).
```
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    if (events[i].type == "key" && events[i].key == "Escape") {
      return false
    }
  }
  draw()
})
```

### window.title(w, text)
Changes the title.

### window.set(w, options)
Changes the window while it runs, or before: the options of `window.open` but `transparent`, `headless` and `automation`, and these:

| Option | What it does |
| --- | --- |
| `size` | a point `(w, h)`: the new size of the buffer, never below `min_size`; the pixels that still fit stay, and a `resize` event comes next frame |
| `center` | `true` moves the window to the middle of the screen (after the new size) |
| `title` | the new title |

The buffer takes a new size or scale at once, so what the program draws next is already at the new size; the window on the screen follows at the end of the frame. This is how a splash screen becomes the app's window:
```
window.set(w, {"frameless": false, "floating": false, "scale": 2, "size": (640, 400),
  "min_size": (320, 200), "resizable": true, "center": true, "title": "my app"})
```

### window.needs_paint(w)
Asks for one more frame: with `on_demand`, `tick` runs next frame even if nothing happens. An animation calls it in every `tick` until it ends.

### Events

Each event is a hash with a `type` and the keys that mean something for it. Positions are buffer pixels from the top left.

| type | keys | when |
| --- | --- | --- |
| `down` | `pos`, `button`, `mods`, `pen` | a mouse button or the pen goes down; `button` is `"left"`, `"right"` or `"middle"` |
| `up` | `pos`, `button`, `mods`, `pen` | it goes up |
| `move` | `pos`, `mods`, `pen` | the pointer moved (with or without a button down) |
| `scroll` | `pos`, `dy`, `mods` | the wheel or a two-finger scroll; `dy` above 0 is up |
| `drop` | `pos`, `paths`, `mods` | files were dropped on the window; `paths` is a list of their absolute paths, which `fs` reads. The system gives real paths on macOS, Windows and Linux; a file with no real path (a browser) is left out |
| `key` | `key`, `mods` | a key went down, and again while it is held; `key` is its name: `"A"` to `"Z"`, `"Digit0"` to `"Digit9"`, `"Escape"`, `"Enter"`, `"Space"`, `"Backspace"`, `"Tab"`, `"ArrowLeft"`, ... |
| `type` | `text`, `mods` | a character was typed, as the keyboard composed it (`"é"`, `"C"`) |
| `resize` | `size` | the window changed size: the user resized it, or `window.set` did |
| `focus` | `focused` | the window got (`true`) or lost (`false`) the keyboard focus |
| `leave` | | the pointer left the window (or it lost the focus) |
| `enter` | | the pointer came back over it |
| `close` | | the user closed the window: the last event |

A pointer event (`down`, `up`, `move`, `scroll`, `drop`) over a [surface](#surfaces) also has `surface`, the surface's name, and `surface_pos`, the point in the surface's own pixels (the newest surface on top wins). A `move` has `samples`: every point the pointer went through since the last frame, not only the last one, so a fast stroke with a pen keeps its shape. Each sample is a hash with `pos`, `pressure`, `raw`, `t` (seconds, the clock of the system's pen events) and, over a surface, `surface` and `surface_pos`; the last one is the event's own point. Where the system sends no samples (a mouse, an injected event), `samples` has one, the event's point.
```
if (e.type == "move" && drawing) {
  for (let i = 0; i < len(e.samples); i += 1) {
    dab(e.samples[i].surface_pos, e.samples[i].pressure)
  }
}
```

`mods` is `{"shift", "alt", "cmd", "ctrl"}`, each true or false. `pen` is:

| key | what it is |
| --- | --- |
| `pressure` | 0 to 1, after the device's [response curve](#the-response-curve); always 1 for a mouse |
| `raw` | 0 to 1, as the device reported it |
| `source` | `"mouse"`, `"trackpad"` (a Force Touch trackpad) or `"tablet"` |
| `tilt` | a point, -1 to 1 on each axis (tablets) |
| `eraser` | whether the eraser end of the pen is in use |

Use `key` for keys that do something (Escape, arrows, shortcuts with `mods.cmd`) and `type` for text.

### Surfaces

A surface is a buffer of its own shown in a rectangle of the window, at a scale of its own, over the window's buffer. A window at scale 2 (big, sharp pixels for the interface) with a surface at scale 1 shows the surface at the screen's full resolution: a drawing canvas in the middle of a pixel-art app. A surface is a [drawing target](#drawing-targets) like the window.

#### window.surface(w, name, rect, scale = 1)
The window's surface called `name`, shown in `rect`, `[x, y, w, h]` in the window's buffer pixels, each of its pixels `scale` by `scale` screen pixels. Its buffer covers the rect: `window.size(s)` of a surface over `[0, 0, 100, 50]` at scale 1 in a window at scale 2 is `(200, 100)`. The first call makes it, transparent; later calls with the same name move or resize it (keeping the pixels that still fit) and give the same value. An empty rect removes it and gives `null`. Surfaces show in the order they were made, the newest on top; a window can have 64.
```
let w = window.open(640, 400, "app", {"scale": 2})
let canvas = window.surface(w, "canvas", [20, 20, 600, 360], 1)
window.clear(canvas, #ffffff)
```

#### window.dirty(target, rect)
Says which part of the window or of one of its surfaces changed this frame. Without it, a target that was drawn on is shown again whole; with it, only the rectangles declared, each on its own, so a brush dab in one corner and a cursor in the other don't cost the screen between them. Only windows with surfaces compose the frame; a window with none shows its buffer as it is.
```
window.spans(canvas, dab, ink)
window.dirty(canvas, [x - r, y - r, 2 * r, 2 * r])
```

#### window.screenshot(w, options = {})
An image of the window as it shows: its buffer with the surfaces over it, at the finest scale they share (the buffer's own size when it has none, twice it for a window at scale 2 with a surface at scale 1). With `{"full": true}` it is every pixel of the screen, the window's size times its scale, surfaces or not: what a user sees, to compare with a picture taken at another scale.
```
img.save(window.screenshot(w), "shot.png")
img.save(window.screenshot(w, {"full": true}), "screen.png")
```

### Headless windows

`window.open(w, h, title, {"headless": true})` makes a window with no screen: `window.run` runs its frames one after the other, as fast as `tick` goes, with the events the program injects, and stops when `tick` returns `false` (or when an `on_demand` window has nothing left to do). Everything else is the same, surfaces and screenshots included, so an app's tests and benchmarks run it as it is, in `go test` or on a machine with no display. A headless window doesn't count as the program's one window.

#### window.inject(w, event)
Adds `event`, a hash like the ones `tick` gets, to the next frame, as if the user made it. Keys left out take a plain value: the left button, no `mods`, a mouse's `pen` (a `pen` hash takes `pressure`, `raw`, `source` and `eraser`). A pointer event needs a `pos`, which can be fractional to reach a finer surface's pixels; its `surface` and `surface_pos` are worked out. A `drop` event needs a `pos` and a list of `paths` (strings). A `resize` event resizes the window like `window.set`.
```
let w = window.open(200, 100, "", {"headless": true})
window.inject(w, {"type": "down", "pos": (10, 10)})
window.inject(w, {"type": "move", "pos": (50, 10), "pen": {"pressure": 0.5}})
window.inject(w, {"type": "up", "pos": (50, 10)})
let frames = 0
window.run(w, function(events) {
  handle(events)
  frames += 1
  frames < 2
})
img.save(window.screenshot(w), "after-a-stroke.png")
```

### Automation

A window with automation on can be driven by another program, the way a person would use it: to test it from outside, or to let an agent work with it. It is off unless the program turns it on, with the `automation` option, `window.automation(w)`, or the environment variable `LIZARD_AUTOMATION=1` (for every window the program opens). While the window runs it listens on `127.0.0.1` only, on a free port, and writes the port and a random token to `<os.config_dir()>/lizard/drive-<pid>.json`, a file only the user can read, removed when the window closes. A request without the token is refused.

#### window.automation(w, hooks = true)
Turns automation on (`false` turns it off). `hooks` can instead be a hash of the program's own parts:

| hook | what it is |
| --- | --- |
| `dump` | a function with no arguments: what it gives is the `app` part of `Dump` (the program's own state) |
| `stats` | a function with no arguments: what it gives is the `app` part of `Stats` |
| `commands` | a hash of `name: function(args)`: `Call name args...` calls it with the list of words, numbers as numbers |

```
window.automation(w, {
  "dump": function() { {"tool": tool, "layers": len(layers)} },
  "commands": {"Tool": function(args) { tool = args[0] }},
})
```

#### lizardscript drive [--pid N] Command [args...]
Sends one command to the newest running window with automation (or the one of the program `N`) and prints the JSON reply; a failed command prints `{"error": ...}` and exits with 1. Input commands go through the same path as real input, and each reply comes after the frame that handled it, so the next command sees what the program did. Coordinates are the window's buffer pixels, what events call `pos` (they can be fractional); with `--full` they are screen pixels.

| Command | What it does |
| --- | --- |
| `List` | the commands, and the program's own (`commands` hook) |
| `Dump` | the window (size, scale, frame), its surfaces, the stats and the `dump` hook's value |
| `Stats` | `ticks`, `frames` (ticks that drew), `events`, `tick_ms`, `compose_ms` and the `stats` hook's value |
| `Screenshot [path] [--base64]` | saves `window.screenshot` as a PNG (to a new file in the temp folder without `path`) and gives its path, or with `--base64` gives the PNG itself |
| `Click x y [mods] [--button right]` | a move to the point (a hand arrives before it presses), a press and a release; mods are `shift`, `alt`, `cmd`, `ctrl`, or joined: `shift+cmd` |
| `DoubleClick x y [mods]` | a move there, then two clicks |
| `Drag x1 y1 x2 y2 [steps] [mods] [--pressure a,b,...]` | a press, `steps` moves (10) and a release; `--pressure` ramps the pressure through the values, from a `"tablet"` |
| `Press x y [mods] [--button right] [--pressure p]` | a move to the point, then the button goes down and stays down until `Release`; with `--pressure` the pen is a `"tablet"` of that pressure |
| `Move x y [mods] [--pressure a,b,...] [--steps n] [--samples k]` | the pointer goes to the point in `steps` move events (1), from where the last command left it. After `Press` it drags, with the pressure of the press unless `--pressure` ramps it; with nothing down it hovers. `--samples k` puts `k` points in each event's `samples`, as a fast hand does between two frames |
| `Release x y` | the button goes up at the point (after a move there if the pointer is elsewhere), with the pen, button and mods of the `Press`; an error when nothing is down |
| `Path x1 y1 x2 y2 ... [mods] [--pressure a,b,...] [--steps n] [--samples k] [--close] [--button right]` | a whole gesture: a move to the first point, a press there, `steps` moves (10) along each line to the next point, and a release at the last. `--close` goes back to the first point and releases there, to draw a loop (a lasso). `--pressure` is spread along the length of the path. Not while a `Press` is down |
| `Scroll dy [x y]` | the wheel |
| `Drop x y path...` | files dropped on the window at the point; the paths are made absolute, and the files need not exist |
| `Key name [mods]` | a key, by its name: `Key S cmd` |
| `Type text` | each character of the text, as `type` events |
| `Resize w h` | `window.set` with that size, then waits for the program to settle |
| `Focus true\|false`, `Leave`, `Enter` | the window gets or loses the focus; the pointer leaves or comes back |
| `Wait` | replies at the first frame with nothing to do (at most 90 frames) |
| `Quit` | the program gets a `close` event, and the window stops after that frame |
| `Call Name args...` | calls the program's command `Name` and gives what it returns; an error in it is the reply, and the window goes on |

```
LIZARD_AUTOMATION=1 lizardscript examples/terrarium/splash.liz &
lizardscript drive Drag 40 60 250 150 30 --pressure 0.1,1
lizardscript drive Screenshot shot.png
lizardscript drive Quit
```

`Press`, `Move` and `Release` split a drag so that a driver can look at the window half way (a preview that only shows while the button is down, a selection outline, the stabilizer). `Path` makes a whole gesture, with a loop if it must. `Drag` stays the plain straight line, with no move before its press.

```
lizardscript drive Press 40 60 --pressure 0.3
lizardscript drive Move 120 90 --steps 8 --pressure 0.3,0.9
lizardscript drive Screenshot half-way.png
lizardscript drive Release 120 90
lizardscript drive Path 50 50 150 60 140 140 60 130 --close --steps 6
```

Every command sends the events of the input it names and nothing else: a command that sets the program's own state (a tool, a mode) is the program's own `Call` command, not an input, so the program decides what it arms.

## pen

Pen pressure comes in the window's pointer events, as `e.pen`. macOS reports it from Force Touch trackpads and tablets; Windows 8 and later from Windows Ink pens (Wacom, Surface, Huion, XP-Pen). Other systems, and a plain mouse anywhere, report a pressure of 1. Each source has a response curve that shapes the pressure.

### pen.available()
Whether this system can report pressure at all. Even when it is true, a mouse reports 1.
```
let size = 2 + e.pen.pressure * 14
```

### pen.curve(source)
### pen.curve(source, curve)
The response curve of a source (`"trackpad"`, `"tablet"` or `"mouse"`), as a hash `{"low", "high", "gamma"}`. With a second argument it changes the curve first: the keys left out keep their value, and the call fails with a clear message for an unknown source or key, a value that isn't a number, or one out of range. It returns the curve now in use.
```
pen.curve("trackpad")                   -~> {"low": 0.8, "high": 1.0, "gamma": 1.5}
pen.curve("trackpad", {"gamma": 1.2})   -~> {"low": 0.8, "high": 1.0, "gamma": 1.2}
```

### The response curve

A Force Touch trackpad needs a firm press to register a click (about 90% of its range), so by the time drawing starts the raw pressure is already high and strokes come out thick. A curve fixes that for one source. The pressure a program sees is:

```
clamp((raw - low) / (high - low), 0, 1) ** gamma
```

`low` is the raw pressure that counts as zero, `high` the one that counts as full, and `gamma` bends what is between: above 1 keeps light pressures small, below 1 raises them. Valid curves have `0 <= low < high <= 1` and `0.1 <= gamma <= 10`. `e.pen.raw` is always the device's own value.

| source | default curve |
| --- | --- |
| `"trackpad"` | `{"low": 0.8, "high": 1.0, "gamma": 1.5}` |
| `"tablet"` | `{"low": 0, "high": 1, "gamma": 1}` (no change) |
| `"mouse"` | `{"low": 0, "high": 1, "gamma": 1}` (no change) |

The trackpad default is a starting point, not a measured value: tune it for your trackpad (below).

### The pen file

Instead of calling `pen.curve` in every program, put the curves in `<os.config_dir()>/lizard/pen.liz` (on macOS `~/Library/Application Support/lizard/pen.liz`). It is lizard-script code; a variable named `trackpad`, `tablet` or `mouse` that holds a hash sets that curve, and keys left out keep their default:

```
-~> My trackpad registers a click late: ignore the first 85%.
let trackpad = {"low": 0.85, "gamma": 1.4}
```

The file is read once, when the program starts. A file that doesn't exist is fine. A file with an error (a syntax error, a value out of range, a variable that isn't a hash) prints a warning to stderr and every curve keeps its default; it never stops the program. Calling `pen.curve` in a program overrides the file.

### Tuning the curve

1. Run `lizardscript examples/terrarium/paint.liz` and draw with your trackpad.
2. While you draw, the title shows `pressure 0.42 (raw 0.91) · trackpad`: the pressure the program uses, the device's raw value, and the source.
3. Press as lightly as you can while still drawing and watch `raw`. Set `low` just below it. Press as hard as you comfortably do and set `high` to that raw value (or leave it at 1).
4. Raise `gamma` to make light strokes thinner, lower it to make them thicker. Edit `pen.liz`, run paint again, repeat.

---

## In Go

`terrarium.Install(env)` defines the modules in an environment; `terrarium.InstallCore(env)` defines all but `window` and `pen`, for a program with no screen (the REPL, a server). `window` and `pen` live in their own package, `github.com/alejandruxxug/lizard-script/terrarium/window`, which adds them to `Install` when a host imports it, so a host that doesn't want a window doesn't link one:

```go
import (
	"github.com/alejandruxxug/lizard-script/terrarium"
	_ "github.com/alejandruxxug/lizard-script/terrarium/window"
)

env := object.NewEnvironment()
terrarium.Args = os.Args[2:]   // what os.args() gives
terrarium.Install(env)
```

`window.run` must be called on the program's main goroutine, which is where a program runs when a host evaluates it in `main` (a headless window can run on any goroutine). `window.Drive(args, out)` is the `lizardscript drive` command, for a host of its own.

| Package | What is in it |
| --- | --- |
| `terrarium` | `Install`, the core modules (one file each: `fs.go`, `os.go`, `time.go`, `img.go`, `clip.go`, `io.go`; `lizard` in `lizard_tree.go`, `lizard_run.go`, `lizard_docs.go` and `lizard_check.go`), the system parts (`sys_*.go` for the clipboard and the locale; `shell.go` and `shell_*.go` for the Trash, the file manager, the browser and the memory) and the modules in lizard-script (`lib/*.liz`, run by `lizard.go`) |
| `terrarium/window` | the window module (`window.go`), events (`events.go`), surfaces and composing (`surface.go`), the Ebitengine loop (`ebiten.go`), the headless loop (`headless.go`), focus and the pointer leaving (`presence.go`), automation (`automation.go`) and `lizardscript drive` (`drive.go`) |
| `terrarium/pen` | pressure: macOS (`pen_darwin.go`), Windows Ink (`pen_windows.go`), none (`pen_other.go`); the response curve (`curve.go`) and the pen file (`config.go`) |
| `terrarium/internal/arg` | checking a builtin's arguments, making values, building a module hash |
| `terrarium/internal/pixels` | the pixel buffer under images, windows and surfaces: clip and layers (`pixels.go`, `blend.go`), blend modes (`blend.go`), gradients (`gradient.go`) |
| `terrarium/internal/canvas` | the drawing functions every target shares (`canvas.go`, and the per-pixel ones in `paint.go`) |
| `terrarium/std` | the [standard modules](#standard-modules), lizard-script files built in by `std.go` (`flower/lily2d.liz` and its parts in `flower/lily2d/`) |

## Adding a module

1. **Lizard-script first.** If the module only needs what lizard-script already has, write it in lizard-script, in `terrarium/lib/name.liz`, defining a variable `name` that holds a hash of its functions, and add `name` to `lizardModules` in `terrarium/lizard.go`. The file runs in an environment of its own, so a program's variables (`let text = ...`) never reach its code, and each function becomes a builtin `name.function` whose errors are reported at the program's call, like any builtin's. Keeping the helpers inside one function that returns the hash, the way `json.liz` does, keeps them out of the module.
2. **Go when it reaches the system.** Write `terrarium/name.go` with a function that returns `arg.Module("name", ...)`, and define it in `InstallCore`. Put the code that differs between systems in files with build tags, like `sys_darwin.go`. A module that needs a big dependency or a screen goes in its own package that calls `terrarium.Register` from its `init`, like `window`.
3. **Every builtin** checks its arguments with `arg` (`arg.Count`, `arg.String`, ...), returns an `*object.Error` for anything that goes wrong (no panics), and has a comment that says what it does. A value Go keeps for the program is an `object.Native` with a kind of its own; a new drawing target implements `pixels.Target` (`Base`, its own image; `Pixels`, where drawing goes, the newest open layer; `Touched`).
4. **Test it** the way the others are: a table of programs and what they give, errors included, with files in `t.TempDir()`. Nothing in a test opens a window or touches the real clipboard: a test of a window app uses a headless window.
5. **Document it** here: one section, every function with its signature, what it does, its errors and an example.
