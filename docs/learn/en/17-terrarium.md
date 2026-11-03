# 17. The Terrarium

[← Previous: Projects](16-projects.md) · [Index](../README.md) · [Next: Modules and tools →](18-modules-and-tools.md) · [Español](../es/17-terrarium.md)

Until now every program lived inside the interpreter: it computed things and printed them. The **Terrarium** (where lizards live) is lizard-script's standard library, and it lets a program reach the world around it: read and write files, get its arguments, tell the time, read and write JSON, make images, and even open a window and draw in it with the mouse or a pen.

This chapter shows the modules one by one with a small example each. The full reference, with every function and every error, is [TERRARIUM.md](../../TERRARIUM.md).

## Modules are hashes

Each module is a hash of builtins, used with dot access, just like `math`, `text` and `list` (chapters 8, 10 and 11). They are there in every program you run with `./lizardscript file.liz`; you don't import anything.

```
puts(type(fs), type(json), type(img))
puts(keys(time))
puts(fs.read)
```

**Output:**

```text
HASH HASH HASH
["ms", "now", "sleep"]
builtin fs.read
```

| Module | For |
| --- | --- |
| `fs` | files and folders |
| `os` | the program's arguments, the environment, the system |
| `time` | clocks and waiting |
| `json` | reading and writing JSON |
| `img` | images |
| `clip` | the clipboard: `clip.get()`, `clip.set(text)` |
| `window`, `pen` | a window to draw in, its input, pen pressure |

They are ordinary variables of your program, so a `let` with the same name replaces one:

```
let fs = "my file system"
puts(fs)
puts(fs.read("notes.txt"))
```

**Output:**

```text
my file system
[line 3, column 8] runtime error: string index must be INTEGER, got STRING
```

`fs` is a string now, so `fs.read` means `fs["read"]`, and a string can't be indexed with a string. Pick other names for your own variables.

## `fs`: files and folders

```
fs.mkdir("notes")
let path = fs.join("notes", "lizards.txt")
fs.write(path, "Liz likes sun\n")
fs.append(path, "Rex likes rocks\n")
puts(fs.read(path))
puts(fs.exists(path), fs.list("notes"))
```

**Output:**

```text
Liz likes sun
Rex likes rocks

true ["lizards.txt"]
```

- `fs.write(path, text)` replaces the whole file (or makes it); `fs.append(path, text)` adds at the end.
- `fs.read(path)` gives the whole text. The empty line in the output is the file's last `\n` plus the one `puts` adds.
- `fs.join(a, b, ...)` builds a path with `/`, which works on every system, Windows too.
- `fs.mkdir` makes a folder (and the ones above it); `fs.list` gives the names inside, in alphabetical order.
- `fs.digest(path)` is a fingerprint of the file's contents (SHA-256, as hex), the same as `text.digest` of its text: two files with the same digest hold the same bytes, so it is a good name for a thumbnail or a cache of the file. `fs.digest(path, "fnv")` is a shorter, faster one for caches.

```
fs.write("note.txt", "abc")
puts(fs.digest("note.txt"))
puts(fs.digest("note.txt", "fnv"), text.digest("abc", "fnv"))
```

**Output:**

```text
ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
e71fa2190541574b e71fa2190541574b
```

When something goes wrong (a missing file, a folder that isn't there), the builtin raises a runtime error, and `try` catches it like any other (chapter 14):

```
try {
  fs.read("notes/missing.txt")
} catch (e) {
  puts(e.message)
}
```

**Output:**

```text
fs.read: can't read notes/missing.txt: no such file or directory
```

`fs.remove(path)` deletes a file or an **empty** folder. It refuses a folder with something in it, so a mistake can't delete a whole tree:

```
fs.mkdir("box")
fs.write("box/a.txt", "")
fs.remove("box")
```

**Output:**

```text
[line 3, column 10] runtime error: fs.remove: can't remove box: the folder is not empty
```

Remove the files first, then the folder:

```
fs.remove("notes/lizards.txt")
fs.remove("notes")
puts(fs.exists("notes"))
```

**Output:**

```text
false
```

Moving and copying never replace a file that is already there, so a typo can't lose your work:

```
fs.write("draft.txt", "Liz likes sun\n")
fs.copy("draft.txt", "backup.txt")
fs.rename("draft.txt", "final.txt")
puts(fs.list("."))
let info = fs.info("final.txt")
puts(info.size, info.dir)
puts(fs.list(".", {"info": true})[0].name)
try { fs.rename("backup.txt", "final.txt") } catch (e) { puts(e.message) }
```

**Output** (in a folder with only this program, `fsx.liz`):

```text
["backup.txt", "final.txt", "fsx.liz"]
14 false
backup.txt
fs.rename: can't move backup.txt to final.txt: there is already something there
```

`fs.info(path)` gives the `size` in bytes, the `mtime` (when it last changed, in seconds since 1970) and whether it is a folder (`dir`); `fs.list(dir, {"info": true})` gives that for every entry, with its `name`. `fs.trash(path)` moves a file or folder to the Trash (the Recycle Bin on Windows), where the user can get it back, which is kinder than `fs.remove` for something the user deleted.

> **Try it.** Write a program that keeps a counter in `count.txt`: it reads the number (or starts at 0 when the file doesn't exist), adds 1, writes it back and prints it. Run it three times. (`int(text)` turns `"3"` into `3`; `fs.exists` tells you whether there is a file yet.)

## `os`: arguments and the system

Whatever you write after the file name reaches the program as `os.args()`, a list of strings:

```
let args = os.args()
if (len(args) == 0) {
  puts("usage: ./lizardscript greet.liz name...")
  os.exit(2)
}
for (let i = 0; i < len(args); i += 1) {
  puts("hello,", args[i])
}
```

```sh
./lizardscript greet.liz Liz Rex
```

**Output:**

```text
hello, Liz
hello, Rex
```

Without names it prints the usage line and stops with `os.exit(2)`: the program ends right there, and the number is the exit code the terminal sees (0 means "all went well").

The rest tells the program where it is running:

```
puts(os.platform(), os.locale())
puts(os.env("LIZARD_COLOR"))
```

**Output** (on a Mac set to English; yours may say `windows` or `linux`, and your language):

```text
macos en
null
```

`os.env(name)` reads an environment variable, and is `null` when it isn't set. Run it as `LIZARD_COLOR=green ./lizardscript file.liz` (macOS and Linux) and the second line is `green`. `os.config_dir()` is the folder where programs keep their settings.

A few more ask the desktop: `os.program_dir()` is the folder of your program's main file (chapter 18 uses it to find the files that come with a program), `os.reveal(path)` shows a file in Finder or Explorer, `os.open_url("https://...")` opens a web page in the browser, and `os.memory()` is the machine's memory, `{"total": ..., "free": ...}` in bytes.

## `time`: clocks and waiting

```
let start = time.ms()
time.sleep(100)
let took = time.ms() - start
puts(type(took), took >= 100)

let now = time.now()
puts(keys(now))
puts(now.month >= 1 && now.month <= 12, now.weekday >= 1 && now.weekday <= 7)
puts(type(now.epoch), time.utc().utc_offset)
```

**Output:**

```text
FLOAT true
["year", "month", "day", "hour", "minute", "second", "weekday", "epoch", "utc_offset"]
true true
FLOAT 0
```

- `time.ms()` is milliseconds since the program started, as a float. Subtract two of them to measure how long something took.
- `time.sleep(ms)` waits.
- `time.now()` is the date and time as a hash, so `now.hour` and `now.minute` are ready to print or to draw a clock with. `weekday` is 1 for Monday to 7 for Sunday.
- `now.epoch` is the seconds since 1970 (UTC), a float: the same instant on every computer, so it is the one to store, and two of them subtracted are the seconds between them ("2 h ago" is `int((time.now().epoch - saved) / 3600)`). `now.utc_offset` is how many seconds the local clock is ahead of UTC (`-18000` in Bogotá).
- `time.utc()` is the same hash in UTC, with a `utc_offset` of 0.

> **Try it.** Measure how long `range(1000000)` takes with `time.ms()`. Then print the time as `14:05` (pad the minutes with a `"0"` when they are below 10).

## `json`: data as text

JSON is the most common way programs save data and talk to each other. `json.write` turns a value into JSON text, and `json.parse` turns the text back:

```
let liz = {"name": "Liz", "toes": [5, 5, 5, 5], "likes": {"sun": true}}
let text = json.write(liz)
puts(text)
puts(json.write(liz, 2))

let back = json.parse(text)
puts(back.name, len(back.toes), back.likes.sun)
puts(back == liz)
```

**Output:**

```text
{"name":"Liz","toes":[5,5,5,5],"likes":{"sun":true}}
{
  "name": "Liz",
  "toes": [
    5,
    5,
    5,
    5
  ],
  "likes": {
    "sun": true
  }
}
Liz 4 true
true
```

The second argument of `json.write` is the indent: `2` puts each item on its own line, nested by two spaces, which is easier to read in a file. Hashes become JSON objects, arrays become arrays, and the round trip gives back an equal value.

`json` and `fs` together save and load a program's settings. A color is written as its text:

```
let settings = {"brush": 8, "color": #2b3a55}
fs.write("settings.json", json.write(settings, 2))
let loaded = json.parse(fs.read("settings.json"))
puts(loaded)
fs.remove("settings.json")
```

**Output:**

```text
{"brush": 8, "color": "#2b3a55"}
```

Bad JSON is an error that says where the mistake is, inside the text:

```
json.parse("{\"name\": \"Liz\",\n  \"toes\": 20,\n}")
```

**Output:**

```text
[line 1, column 11] runtime error: json.parse: line 3, column 1: expected a string key in an object, got '}'
```

The first position is your program's call; the second is in the JSON text. JSON allows no comma before a closing `}`.

> **Common mistake: expecting the color back.** JSON has no colors or points, so `loaded.color` above is the *string* `"#2b3a55"`, not a `COLOR`. `loaded.color.r` would be an error. Keep colors as `[r, g, b]` and rebuild them with `rgb(c[0], c[1], c[2])` when you load.

## `img`: images

An image is a grid of colored pixels. `img.new(w, h, color)` makes one, `img.set` and `img.get` change and read a pixel (`x` from the left, `y` from the top, both from 0), and `puts` shows its size:

```
let picture = img.new(4, 2, #ffffff)
img.set(picture, 0, 0, #ff0000)
puts(picture, img.size(picture))
puts(img.get(picture, 0, 0), img.get(picture, 3, 1))
puts(img.pixels(picture))
```

**Output:**

```text
<image 4x2> (4, 2)
#ff0000 #ffffff
[#ff0000, #ffffff, #ffffff, #ffffff, #ffffff, #ffffff, #ffffff, #ffffff]
```

`type(picture)` is `NATIVE`: a value the program can keep and pass around but not look inside. `img.pixels` gives every color in one flat list, row by row.

Two loops make a gradient, and `img.save` writes it as a PNG you can open with any image viewer:

```
let w = 256
let h = 64
let picture = img.new(w, h)
for (let y = 0; y < h; y += 1) {
  for (let x = 0; x < w; x += 1) {
    img.set(picture, x, y, rgb(x, 100, 255 - x))
  }
}
img.save(picture, "gradient.png")

let back = img.load("gradient.png")
puts(img.size(back), img.get(back, 0, 0), img.get(back, 255, 0))
puts(img.size(img.load("gradient.png", 32)))
```

**Output:**

```text
(256, 64) #0064ff #ff6400
(32, 8)
```

`img.load(path)` reads PNG, JPEG and GIF files. With a second argument it shrinks the image to fit that many pixels on its longer side, which is handy before looping over a big photo.

### Drawing: `fill` and `span`

Setting pixels one by one is slow for big shapes. Two functions paint whole runs of pixels at once, and they **mix** the color over what is there, so a color with alpha (chapter 12) is see-through:

- `img.fill(image, [x, y, w, h], color)` paints a rectangle.
- `img.span(image, y, x0, x1, color)` paints row `y` from `x0` up to `x1`. A circle, a triangle or any shape is one span per row.

```
let picture = img.new(5, 3, #000000)
img.fill(picture, [1, 0, 3, 3], #ffffff)
img.span(picture, 1, 0, 5, #ff000080)
puts(img.pixels(picture))
```

**Output:**

```text
[#000000, #ffffff, #ffffff, #ffffff, #000000, #800000, #ff7f7f, #ff7f7f, #ff7f7f, #800000, #000000, #ffffff, #ffffff, #ffffff, #000000]
```

The middle row is half-transparent red: over black it gives dark red (`#800000`), over white a pink (`#ff7f7f`). The other drawing functions are `img.clear(image, color)` (every pixel, no mixing) and `img.blit(image, other, pos)` (paints one image over another). `blit` takes options to paint it at another size: `{"scale": 2}` doubles it with sharp square pixels, and `{"scale": 0.25, "filter": "linear"}` shrinks it smoothly, for a thumbnail.

```
let tile = img.from_pixels((2, 1), [#000000, #ffffff])
let big = img.new(4, 1)
img.blit(big, tile, (0, 0), {"scale": 2})
puts(img.pixels(big))
img.blit(big, tile, (0, 0), {"scale": 2, "filter": "linear"})
puts(img.pixels(big))
let small = img.new(1, 1)
img.blit(small, tile, (0, 0), {"scale": 0.5, "filter": "linear"})
puts(img.pixels(small))
```

**Output:**

```text
[#000000, #000000, #ffffff, #ffffff]
[#000000, #404040, #bfbfbf, #ffffff]
[#808080]
```


> **Common mistake: a pixel outside the image.** `img.get` and `img.set` only take pixels that exist:
>
> ```
> let picture = img.new(4, 2)
> img.get(picture, 4, 0)
> ```
>
> **Output:**
>
> ```text
> [line 2, column 8] runtime error: img.get: the pixel (4, 0) is outside the 4x2 image
> ```
>
> A 4-pixel-wide image has `x` from 0 to 3. (`fill` and `span` are friendlier: whatever falls outside is simply cut off.)

> **Try it.** Draw a filled circle of radius 20 in the middle of a 64x64 image with one `img.span` per row (for a row `dy` away from the center, the half-width is `math.sqrt(r * r - dy * dy)`), and save it as `circle.png`.

## `window` and `pen`: build a paint program

A window is a picture on the screen plus the input that reaches it. It is a drawing target like an image, so `window.fill`, `window.span`, `window.clear` and the rest work on it the same way. In this section you build `examples/terrarium/paint.liz`, a small paint program, from an empty file, in seven steps. Every step is a whole program: save it as `paint.liz`, run `./lizardscript paint.liz`, try it, then go on. None of them prints anything; everything happens in the window.

### Step 1: open a window

```
let w = window.open(600, 400, "paint 🦎")

window.run(w, function(events) {
})
```

`window.open(w, h, title)` makes a window of 600 by 400 pixels. `window.run(w, tick)` shows it and calls `tick(events)` about 60 times a second, with the list of what happened since the last time, until `tick` returns `false` or you close the window. This `tick` does nothing yet, so you get a black window; close it to stop the program.

### Step 2: paper

```
let paper = #f4efe6
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

window.run(w, function(events) {
})
```

`window.clear(w, color)` paints every pixel. It runs once, before `window.run`, so the window starts with its first picture already there.

### Step 3: a circle where you click

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> A filled circle, one span per row.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, 8, ink)
    }
  }
})
```

Each event is a hash with a `type`. A `"down"` event says a button went down, with `e.button` (`"left"`, `"right"` or `"middle"`) and `e.pos`, the point where it happened, in the window's pixels from the top left. `circle` is the circle of the last "Try it" of the `img` section, one span per row; `window.span` works exactly like `img.span`. Click around: each click leaves a dot.

### Step 4: follow the drag

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> A filled circle, one span per row.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> Where the drag is, or null when no button is down.
let last = null

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, 8, ink)
      last = e.pos
    } elif (e.type == "move" && last != null) {
      circle(e.pos, 8, ink)
      last = e.pos
    } elif (e.type == "up") {
      last = null
    }
  }
})
```

`last` lives outside `tick`, so it is remembered from one frame to the next: it is the drag's last position, or `null` while no button is down. Each `"move"` while dragging draws another circle, and `"up"` ends the drag. Drag quickly and look closely: the circles have gaps between them, because a fast hand moves many pixels between two frames.

### Step 5: fill the gaps

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> A filled circle, one span per row.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> A drag can move far in one frame: fill the gap with circles.
function stroke(from, to, r) {
  let steps = math.max(1, math.ceil(math.dist(from, to) / (r / 2)))
  for (let i = 1; i <= steps; i += 1) {
    circle(from + (to - from) * (i / float(steps)), r, ink)
  }
}

-~> Where the drag is, or null when no button is down.
let last = null

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, 8, ink)
      last = e.pos
    } elif (e.type == "move" && last != null) {
      stroke(last, e.pos, 8)
      last = e.pos
    } elif (e.type == "up") {
      last = null
    }
  }
})
```

`stroke(from, to, r)` draws circles along the line from the last position to the new one, close enough together (every half radius) to touch. `from + (to - from) * t` is the point a fraction `t` of the way (chapter 12), and `i / float(steps)` matters: with two integers, `i / steps` would be `0` until the last step. Now a fast drag gives a solid line.

### Step 6: pressure

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> A filled circle, one span per row.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> The size of the brush for a pressure from 0 to 1.
function brush(pressure) {
  2 + pressure * 14
}

-~> A drag can move far in one frame: fill the gap with circles.
function stroke(from, to, r) {
  let steps = math.max(1, math.ceil(math.dist(from, to) / (r / 2)))
  for (let i = 1; i <= steps; i += 1) {
    circle(from + (to - from) * (i / float(steps)), r, ink)
  }
}

-~> Where the drag is, or null when no button is down.
let last = null

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, brush(e.pen.pressure), ink)
      last = e.pos
    } elif (e.type == "move" && last != null) {
      stroke(last, e.pos, brush(e.pen.pressure))
      last = e.pos
    } elif (e.type == "up") {
      last = null
    }
  }
})
```

Pointer events carry `e.pen`, and `e.pen.pressure` goes from 0 to 1. `brush` turns it into a radius from 2 to 16. A mouse always says 1, so with a mouse the line is simply thick; with a Force Touch trackpad on a Mac, or a pen tablet on macOS or Windows, pressing harder draws a wider line. `e.pen.source` says which one it is (`"mouse"`, `"trackpad"` or `"tablet"`), and `pen.available()` says whether your system can report pressure at all.

### Step 7: keys

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> A filled circle, one span per row. (/ of two integers is a whole
-~> number, so the fractions below divide by a float.)
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> The size of the brush for a pressure from 0 to 1.
function brush(pressure) {
  2 + pressure * 14
}

-~> A number rounded to two decimals, as text.
function hundredths(x) {
  str(math.round(x * 100) / 100.0)
}

-~> Shows the pen's pressure in the title: "pressure 0.42 (raw 0.91) · trackpad".
function show(pen) {
  window.title(w, "pressure " + hundredths(pen.pressure) + " (raw " + hundredths(pen.raw) + ") · " + pen.source)
}

-~> A drag can move far in one frame: fill the gap with circles.
function stroke(from, to, r) {
  let steps = math.max(1, math.ceil(math.dist(from, to) / (r / 2)))
  for (let i = 1; i <= steps; i += 1) {
    circle(from + (to - from) * (i / float(steps)), r, ink)
  }
}

let last = null

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, brush(e.pen.pressure), ink)
      show(e.pen)
      last = e.pos
    } elif (e.type == "move" && last != null) {
      stroke(last, e.pos, brush(e.pen.pressure))
      show(e.pen)
      last = e.pos
    } elif (e.type == "up") {
      window.title(w, "paint 🦎")
      last = null
    } elif (e.type == "type" && e.text == "c") {
      window.clear(w, paper)
    } elif (e.type == "key" && e.key == "Escape") {
      return false
    }
  }
})
```

A `"type"` event is a character as the keyboard composed it (`e.text`), the right choice for letters: `c` clears the paper. A `"key"` event is a key by its name (`e.key`: `"Escape"`, `"Enter"`, `"A"`, `"ArrowLeft"`...), the right choice for keys that aren't text: Esc returns `false`, which ends `window.run` and the program. This is the code of `examples/terrarium/paint.liz`.

While you draw, the window title shows `pressure 0.42 (raw 0.91) · trackpad`: `e.pen.raw` is what the device reported and `e.pen.pressure` is that value after the device's response curve, which is the one `brush` uses. A trackpad needs a firm press to click, so its curve is dampened by default; `pen.curve("trackpad", {"low": 0.8, "gamma": 1.5})` changes it. See the `pen` section of `docs/TERRARIUM.md`.

### Events

| `type` | other keys | when |
| --- | --- | --- |
| `"down"`, `"up"` | `pos`, `button`, `mods`, `pen` | a button or the pen goes down or up |
| `"move"` | `pos`, `mods`, `pen` | the pointer moved |
| `"scroll"` | `pos`, `dy`, `mods` | the wheel or a two-finger scroll |
| `"key"` | `key`, `mods` | a key went down |
| `"type"` | `text` | a character was typed |
| `"resize"` | `size` | the window changed size |
| `"focus"` | `focused` | the window got or lost the keyboard focus |
| `"leave"`, `"enter"` | | the pointer left the window, or came back |
| `"close"` | | the window is being closed: the last event |

A `"move"` also has `samples`, every point the pointer went through since the last frame (see below).

`mods` says which of `shift`, `alt`, `cmd` and `ctrl` were held. Options go in a hash after the title: `window.open(160, 100, "game", {"scale": 3, "fps": 30})` shows each pixel as a 3x3 square, nice for pixel art. A program shows one window. The REPL has every module but `window` and `pen`; there, `window` is `identifier not found`.

**Tuning pressure.** Pens and trackpads don't all feel the same: one reaches full pressure with a light touch, another needs a hard press. `pen.curve(...)` reshapes the raw pressure before your program uses it, so the brush can feel right on your device; how to call it, and the settings file that remembers it, are in [TERRARIUM.md](../../TERRARIUM.md#pen).

> **Common mistake: integer division in drawing code.** `brush` gives a float, so `r / 2` in `stroke` keeps its fraction. With integers, `/` drops it (chapter 4): `3 / 2` is `1`. When a fraction matters, as in step 5, write `float(i) / steps`.

> **Try it.** Make the right button erase, by drawing with `paper` when `e.button == "right"` (remember the color in a variable next to `last`). Then make the scroll wheel change the ink: on a `"scroll"` event, pick the next color of a list.

## More than one buffer: gradients, layers and surfaces

The paint program draws with flat colors on one buffer. A real drawing app wants more: a gradient sky, shapes that darken what is below like ink, an interface with big pixels around a sharp canvas. Each of these is one more drawing function, and they work on images, windows and surfaces alike.

### Gradients

`span_gradient` and `spans_gradient` are `span` and `spans` with a gradient instead of a color. A gradient is a hash: its `kind`, where it starts (`origin`), which way it runs, and its `stops`, colors at places from 0 to 1:

```
let w = window.open(400, 200, "sky")
let rows = []
for (let y = 0; y < 200; y += 1) { push(rows, [y, 0, 400]) }
window.spans_gradient(w, rows, {"kind": "linear", "origin": (0, 0), "axis": (0, 1 / 200.0),
  "stops": [[0, #3b4cc0], [0.6, #e2557b], [1, #f6c453]]})
```

The `axis` is the direction from the start to the end divided by its squared length: from `(0, 0)` to `(0, 200)` it is `(0, 200) / 200²`, which is `(0, 1 / 200.0)`. A `"radial"` gradient has a `scale` instead, 1 divided by its radius. Add `"steps": 5` and the gradient becomes five flat bands, nice for pixel art.

### Layers and blend modes

`push_layer(w)` opens a new transparent layer: everything drawn on `w` goes to it, until `pop_layer(w, mode, opacity)` mixes it onto what is below. Two shapes drawn on one layer at half opacity don't show where they overlap, and in `"multiply"` they darken the sky like ink:

```
window.push_layer(w)
window.spans(w, hill(100), #2b3a55)
window.spans(w, hill(260), #2b3a55)
window.pop_layer(w, "multiply", 0.7)
```

The modes are `normal`, `multiply`, `screen`, `overlay`, `darken`, `lighten` and `add`. `examples/terrarium/layers.liz` shows them side by side.

### Surfaces

A window at scale 2 has big, crisp pixels: good for an interface, too coarse for a drawing. A surface is a second buffer shown in a rectangle of the window at a scale of its own:

```
let w = window.open(320, 200, "app", {"scale": 2})
let canvas = window.surface(w, "canvas", [10, 20, 300, 170], 1)
window.clear(canvas, #f4efe6)
```

The surface covers `[10, 20, 300, 170]` of the window's pixels, but at scale 1 it has twice as many pixels each way: `window.size(canvas)` is `(600, 340)`. It is drawn on like the window. A pointer event over it says so: `e.surface` is `"canvas"` and `e.surface_pos` is the point in the canvas's own pixels.

A fast stroke can cross many pixels between two frames. Every `"move"` has `e.samples`, all the points the pointer went through since the last frame, each with its `surface_pos` and `pressure`, so a drawing keeps the shape of the hand that drew it:

```
for (let i = 0; i < len(e.samples); i += 1) {
  let s = e.samples[i]
  if (s.surface == "canvas") { dab(s.surface_pos, s.pressure) }
}
```

### A splash screen that becomes the window

`window.open` takes `"frameless": true` (no title bar), `"transparent": true` (the pixels you leave transparent show the desktop) and `"floating": true` (above every window): a splash screen with round corners. `window.set` changes the window while it runs, so the same window becomes the app:

```
window.set(w, {"frameless": false, "floating": false, "scale": 2, "size": (320, 200),
  "resizable": true, "center": true, "title": "my app"})
```

`examples/terrarium/splash.liz` does exactly this, and then paints on a canvas surface with pen pressure.

### Testing a window with no screen

With `"headless": true` a window has no screen at all: `window.run` runs its frames as fast as it can, the program feeds it events with `window.inject`, and `window.screenshot` gives the picture. That is how an app is tested, on any computer:

```
let w = window.open(200, 100, "", {"headless": true})
window.inject(w, {"type": "down", "pos": (10, 10)})
window.inject(w, {"type": "up", "pos": (10, 10)})
window.run(w, function(events) {
  handle(events)
  false
})
img.save(window.screenshot(w), "after-a-click.png")
```

Returning `false` from the first `tick` ends the run after one frame. `window.screenshot(w, {"full": true})` gives every pixel of the screen instead, the window's size times its scale, as a user sees it.

### Letting another program drive it

`window.automation(w)` lets another program use the window the way a person would. While it runs, a terminal can send it commands:

```
lizardscript drive Click 40 30
lizardscript drive Drag 10 10 200 80 --pressure 0.2,1
lizardscript drive Screenshot shot.png
lizardscript drive Quit
```

Each reply comes after the frame that handled the command, so the next command sees what the program did. The program can add its own commands, and its own state for `Dump`; the whole list is in [TERRARIUM.md](../../TERRARIUM.md#automation).

> **Try it.** Run `examples/terrarium/layers.liz`, point at the columns and read the mode in the title. Then change `0.9` in `pop_layer` to `0.4` and see which modes change the most.

## The examples

The Terrarium programs in the repository's `examples/terrarium/` folder run like any other:

| Program | Run it with | What it does |
| --- | --- | --- |
| `files.liz` | `./lizardscript examples/terrarium/files.liz` | writes, reads and lists files in a folder of your home, and does a JSON round trip; it cleans up after itself |
| `image.liz` | `./lizardscript examples/terrarium/image.liz out.png` | draws a gradient pixel by pixel and saves it as a PNG (`gradient.png` without an argument) |
| `paint.liz` | `./lizardscript examples/terrarium/paint.liz` | the paint program of this chapter |
| `clock.liz` | `./lizardscript examples/terrarium/clock.liz` | a window with an analog clock drawn with spans; Esc quits |
| `layers.liz` | `./lizardscript examples/terrarium/layers.liz` | the seven blend modes over a gradient (add a file name to save a picture of it with no window) |
| `splash.liz` | `./lizardscript examples/terrarium/splash.liz` | a frameless, transparent splash screen that becomes a normal window with a canvas surface to paint on |

Add `--vm` before the file to run them on the faster engine: `./lizardscript --vm examples/terrarium/clock.liz`.

## Summary

- The Terrarium's modules are hashes of builtins that every program has: `fs`, `os`, `time`, `json`, `img`, `clip`, `window`, `pen`.
- Their errors are runtime errors with the function's name in front, and `try` catches them.
- `fs.read`, `fs.write`, `fs.append`, `fs.list`, `fs.mkdir`, `fs.remove` (empty folders only), `fs.join`, `fs.digest`.
- `os.args()` gives what follows the file name; `os.exit(code)` stops the program.
- `time.ms()` measures, `time.sleep(ms)` waits, `time.now()` is the date and time (with `epoch` to store), `time.utc()` the same in UTC.
- `json.write(value, indent)` and `json.parse(text)`; colors come back as strings.
- `img.new`, `img.get`, `img.set`, `img.save`, `img.load`; `fill`, `span`, `clear` and `blit` draw on images and windows alike.
- `window.run(w, tick)` calls `tick(events)` every frame until it returns `false`; a paint program is a circle, a drag, a stroke that fills the gaps, and `e.pen.pressure`.
- `spans_gradient` paints with a gradient, `push_layer`/`pop_layer` mix a group of shapes with a blend mode, and `window.surface` adds a sharp buffer of its own to a window.
- `window.set` changes a running window; a `"headless"` window runs with no screen, for tests; `window.automation` lets `lizardscript drive` use it.

[← Previous: Projects](16-projects.md) · [Index](../README.md) · [Next: Modules and tools →](18-modules-and-tools.md)
