# Pen checks on real hardware

The `pen` module's conversions are unit-tested with fake events (`terrarium/pen/ink_test.go`, `xinput_test.go`), but the calls into Windows and the X server can only be checked with a real pen. This is the list for someone with a Windows tablet or a Linux tablet. Write down the system, the tablet and the driver, and what each step gave.

## The test program

Save this as `pentest.liz` and run it with `lizardscript pentest.liz`. It prints `pen.available()`, the title shows the last pointer event's pen (`down tablet pressure 0.42 raw 0.42 tilt 0.1,-0.3 eraser`, and how many `samples` a move has), and the window draws a dot at every sample of a stroke, sized by its pressure. `c` clears it, Esc ends it. `PENTEST_HEADLESS=1 lizardscript pentest.liz` checks the program itself with no pen: it injects a short stroke and prints the three events.

```
let paper = #ffffff
let ink = #202030
let headless = os.env("PENTEST_HEADLESS") != null
let w = window.open(400, 300, "pen", {"headless": headless})
window.clear(w, paper)
puts("pen.available() = " + str(pen.available()))

-~> A filled circle, one span per row.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> A number rounded to two decimals, as text.
function hundredths(x) {
  str(math.round(x * 100) / 100.0)
}

-~> The pen of an event, as one line.
function describe(e) {
  let p = e.pen
  let line = e.type + " " + p.source + " pressure " + hundredths(p.pressure) + " raw " + hundredths(p.raw) +
    " tilt " + hundredths(p.tilt.x) + "," + hundredths(p.tilt.y)
  if (p.eraser) { line = line + " eraser" }
  if (e.type == "move") { line = line + " samples " + str(len(e.samples)) }
  line
}

let frames = 0
if (headless) {
  window.inject(w, {"type": "down", "pos": (50, 50), "pen": {"pressure": 0.5, "source": "tablet"}})
  window.inject(w, {"type": "move", "pos": (60, 55), "pen": {"pressure": 0.7, "source": "tablet"}})
  window.inject(w, {"type": "up", "pos": (60, 55)})
}

window.run(w, function(events) {
  frames += 1
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" || e.type == "move" || e.type == "up") {
      let line = describe(e)
      window.title(w, line)
      if (headless) { puts(line) }
    }
    if (e.type == "move" && e.pen.pressure > 0) {
      for (let k = 0; k < len(e.samples); k += 1) {
        let s = e.samples[k]
        circle(s.pos, 1 + s.pressure * 6, ink)
      }
    } elif (e.type == "type" && e.text == "c") {
      window.clear(w, paper)
    } elif (e.type == "key" && e.key == "Escape") {
      return false
    }
  }
  !(headless && frames > 3)
})
```

## Windows (Windows 10 or 11, a Windows Ink pen)

Turn Windows Ink on in the tablet's driver (Wacom: *Use Windows Ink* in the mapping settings; Huion and XP-Pen: *Windows Ink* in their driver).

1. `pen.available()` is `true` (print it before `window.open`).
2. Hover the pen over the window without touching: nothing is drawn and the title doesn't change (the window has no `hover` option, so only a held button sends moves).
3. Touch and draw slowly, pressing lighter and harder: the title goes from about 0 to 1, the dots grow and shrink with it, `down` and `up` come once each.
4. Draw a fast circle: the dots follow the circle with no straight jumps (each `move` has many `samples`).
5. Tilt the pen left and right, then towards you and away: `tilt` goes towards -1 and 1 on each axis, and is near 0,0 upright.
6. Turn the pen over and draw with the eraser end (a pen that has one): the title says `eraser`. Turn it back: `eraser` is gone.
7. Hold a side button and draw: it still draws with pressure (Windows reports a barrel click as a right button on some drivers; note what happens).
8. On a display scaled to 150% or 200% (*Settings, Display, Scale*), draw: the dots are under the pen, not off by the scale.
9. Move the window to a second monitor with another scale, and draw again.
10. With the pen hovering over the window, touch the screen with a finger and drag (on a touch screen), then drag with the mouse: the title says `mouse` and pressure 1, never the pen's 0. (A mouse used while the pen still hovers is the one case Windows can't tell us about; note what it gives.)
11. Lift the pen out of range, then use the mouse: `mouse`, pressure 1.
12. Close the window while the pen hovers over it: the program ends with no error.

## Linux (an X11 desktop, and a Wayland desktop with XWayland)

Do the list once on X11 (for example GNOME on Xorg, KDE Plasma X11, Xfce) and once on Wayland (GNOME or KDE Plasma on Wayland). First note what the system sees: `xinput list` shows the devices, and `xinput list-props "<the stylus>"` its axes; there should be an `Abs Pressure` valuator.

1. `pen.available()` is `true`. On a system with no X server, or with `libXi.so.6` missing, it is `false` and the program still runs with a mouse.
2. Do steps 2 to 6 of the Windows list. The eraser is a device of its own on Linux (`... eraser`, `... Eraser (0)` or `xwayland-tablet eraser`), so check that its name shows up in `xinput list`.
3. Pressure goes the full way from 0 to 1, not to a half or a tenth: the range comes from the driver's valuator (0 to 2047, or 0 to 65535).
4. Tilt goes the full way to -1 and 1, and the signs match macOS and Windows (left and away are negative).
5. With `Xft.dpi` at 192 (`xrdb -query | grep dpi`; set it with `echo "Xft.dpi: 192" | xrdb -merge` before starting), draw: the dots are under the pen.
6. Draw a stroke that leaves the window and comes back while the tip is down: it keeps drawing, with pressure, when it comes back.
7. Start a stroke outside the window and drag it in: nothing is drawn by the pen's samples (the window didn't get the press).
8. Hover the pen, then use the mouse or a touchpad: `mouse`, pressure 1.
9. Unplug the tablet and plug it in again while the program runs: the pen still has pressure after it is back.
10. With an art pen or a pen that reports rotation, check whether `xinput list-props` names an `Abs Rz` axis, and note it: rotation is only read from that one.
11. Close the window while the pen moves over it, then keep moving the pen (if the program goes on after `window.run`): no X error, no crash.
12. Under Wayland, also check a second monitor with another scale: XWayland scales the window, and the dots should still be under the pen (note what happens if not).

## What to send back

For each step: works, or what happened instead. For a pressure or tilt that stops short, the raw numbers help: `xinput test-xi2 --root` on Linux shows the raw valuators of each event.
