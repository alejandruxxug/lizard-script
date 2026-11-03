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
| [`window`](#window) | a window to draw in, and its input |
| [`pen`](#pen) | pen and trackpad pressure |

[Drawing targets](#drawing-targets) explains the drawing functions that `img` and `window` share.

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

Most functions take and give the usual values: strings, integers, floats, points, colors, arrays, hashes. Images and windows are a value of their own, `NATIVE`: a program keeps them in variables and passes them to the functions that know them, but can't look inside. `puts` shows them as `<image 640x480>` or `<window 600x400>`. Two of them are equal only when they are the same one.

Paths use `/` between their parts on every system; Windows takes it too.

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

### fs.list(dir)
The names of what is in the folder, files and folders, in alphabetical order. Errors when it isn't a folder.
```
puts(fs.list("."))   -~> ["README.md", "docs", "examples", ...]
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

## time

### time.ms()
The milliseconds since the program started, as a float. It only goes forward, even when the system clock changes, so it is the one to measure with.
```
let start = time.ms()
work()
puts("took", time.ms() - start, "ms")
```

### time.now()
The local date and time as a hash of integers: `year`, `month` (1 to 12), `day`, `hour` (0 to 23), `minute`, `second`.
```
let now = time.now()
puts(now.hour, ":", now.minute)
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

Images. An image keeps every pixel's color as it was set, alpha included. An image is a [drawing target](#drawing-targets), so `img.size`, `img.clear`, `img.fill`, `img.span`, `img.spans` and `img.blit` work on it too.

### img.load(path, max_side = 0)
Reads a PNG, JPEG or GIF file. With a `max_side` above 0, an image longer than that on its longer side is shrunk to fit, averaging the pixels (it is never enlarged). Errors when the file can't be read, isn't an image, or is more than 32768 pixels on a side.
```
let photo = img.load("photo.jpg", 256)
```

### img.save(image, path)
Writes the image as a PNG file, the whole file at once like `fs.write`.
```
img.save(picture, "out.png")
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

## Drawing targets

An image and a window are both drawing targets: they are drawn on with the same six functions, so a drawing library can take either one. The functions are in both modules (`img.fill(w, ...)` and `window.fill(w, ...)` are the same function), and each takes any target.

Painting mixes the color over what is there, so a color with alpha shows what is below. `clear` replaces. The ends of a span and the sides of a rectangle can be fractional: the pixel they cut is covered in part, so the edge is soft. What falls outside the target is cut.

### size(target)
`(width, height)` in pixels.

### clear(target, color)
Every pixel becomes `color`.
```
window.clear(w, #f4efe6)
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

### spans(target, list, color)
Paints every `[y, x0, x1]` of `list` in one color: the fast way to fill a shape worked out row by row.
```
window.spans(w, [[10, 5, 20], [11, 4, 21], [12, 5, 20]], #2b3a55)
```

### blit(target, source, pos)
Paints the pixels of `source`, an image or a window, over `target` with its top left at `pos`. It is how a picture gets into a window, and how a window's pixels get into an image (to save a screenshot).
```
window.blit(w, img.load("lizard.png"), (10, 10))
let shot = img.new(600, 400)
img.blit(shot, w, (0, 0))
img.save(shot, "shot.png")
```

## window

A window that shows a buffer of pixels the program draws on, with the [drawing target](#drawing-targets) functions: `window.size`, `window.clear`, `window.fill`, `window.span`, `window.spans`, `window.blit`. A program shows one window in its life.

### window.open(w, h, title, options = {})
Makes a window of `w` by `h` pixels (1 to 8192 each), black at first. It shows on the screen when `window.run` starts, so the program can draw its first picture before. The options are:

| Option | Default | What it does |
| --- | --- | --- |
| `scale` | 1 | each pixel shows as `scale` by `scale` on the screen, 1 to 8 |
| `resizable` | false | the user can resize the window; the buffer follows, and a `resize` event says the new size |
| `frameless` | false | no title bar or border |
| `fps` | 60 | frames a second, 1 to 240 |

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

### Events

Each event is a hash with a `type` and the keys that mean something for it. Positions are buffer pixels from the top left.

| type | keys | when |
| --- | --- | --- |
| `down` | `pos`, `button`, `mods`, `pen` | a mouse button or the pen goes down; `button` is `"left"`, `"right"` or `"middle"` |
| `up` | `pos`, `button`, `mods`, `pen` | it goes up |
| `move` | `pos`, `mods`, `pen` | the pointer moved (with or without a button down) |
| `scroll` | `pos`, `dy`, `mods` | the wheel or a two-finger scroll; `dy` above 0 is up |
| `key` | `key`, `mods` | a key went down, and again while it is held; `key` is its name: `"A"` to `"Z"`, `"Digit0"` to `"Digit9"`, `"Escape"`, `"Enter"`, `"Space"`, `"Backspace"`, `"Tab"`, `"ArrowLeft"`, ... |
| `type` | `text`, `mods` | a character was typed, as the keyboard composed it (`"é"`, `"C"`) |
| `resize` | `size` | a resizable window changed size |
| `close` | | the user closed the window: the last event |

`mods` is `{"shift", "alt", "cmd", "ctrl"}`, each true or false. `pen` is:

| key | what it is |
| --- | --- |
| `pressure` | 0 to 1, after the device's [response curve](#the-response-curve); always 1 for a mouse |
| `raw` | 0 to 1, as the device reported it |
| `source` | `"mouse"`, `"trackpad"` (a Force Touch trackpad) or `"tablet"` |
| `tilt` | a point, -1 to 1 on each axis (tablets) |
| `eraser` | whether the eraser end of the pen is in use |

Use `key` for keys that do something (Escape, arrows, shortcuts with `mods.cmd`) and `type` for text.

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

`terrarium.Install(env)` defines the modules in an environment; `terrarium.InstallCore(env)` defines all but `window` and `pen`, for a program with no screen (the REPL, a server). `window` and `pen` live in their own package, `lizardscript/terrarium/window`, which adds them to `Install` when a host imports it, so a host that doesn't want a window doesn't link one:

```go
import (
	"lizardscript/terrarium"
	_ "lizardscript/terrarium/window"
)

env := object.NewEnvironment()
terrarium.Args = os.Args[2:]   // what os.args() gives
terrarium.Install(env)
```

`window.run` must be called on the program's main goroutine, which is where a program runs when a host evaluates it in `main`.

| Package | What is in it |
| --- | --- |
| `terrarium` | `Install`, the core modules (one file each: `fs.go`, `os.go`, `time.go`, `img.go`, `clip.go`), the system parts (`sys_darwin.go`, `sys_windows.go`, `sys_other.go`) and the modules in lizard-script (`lib/*.liz`, run by `lizard.go`) |
| `terrarium/window` | the window module and the Ebitengine loop (`ebiten.go`) |
| `terrarium/pen` | pressure: macOS (`pen_darwin.go`), Windows Ink (`pen_windows.go`), none (`pen_other.go`); the response curve (`curve.go`) and the pen file (`config.go`) |
| `terrarium/internal/arg` | checking a builtin's arguments, making values, building a module hash |
| `terrarium/internal/pixels` | the pixel buffer under images and windows |
| `terrarium/internal/canvas` | the drawing functions every target shares |

## Adding a module

1. **Lizard-script first.** If the module only needs what lizard-script already has, write it in lizard-script, in `terrarium/lib/name.liz`, defining a variable `name` that holds a hash of its functions, and add `name` to `lizardModules` in `terrarium/lizard.go`. The file runs in an environment of its own, so a program's variables (`let text = ...`) never reach its code, and each function becomes a builtin `name.function` whose errors are reported at the program's call, like any builtin's. Keeping the helpers inside one function that returns the hash, the way `json.liz` does, keeps them out of the module.
2. **Go when it reaches the system.** Write `terrarium/name.go` with a function that returns `arg.Module("name", ...)`, and define it in `InstallCore`. Put the code that differs between systems in files with build tags, like `sys_darwin.go`. A module that needs a big dependency or a screen goes in its own package that calls `terrarium.Register` from its `init`, like `window`.
3. **Every builtin** checks its arguments with `arg` (`arg.Count`, `arg.String`, ...), returns an `*object.Error` for anything that goes wrong (no panics), and has a comment that says what it does. A value Go keeps for the program is an `object.Native` with a kind of its own; a new drawing target implements `pixels.Target`.
4. **Test it** the way the others are: a table of programs and what they give, errors included, with files in `t.TempDir()`. Nothing in a test opens a window or touches the real clipboard.
5. **Document it** here: one section, every function with its signature, what it does, its errors and an example.
