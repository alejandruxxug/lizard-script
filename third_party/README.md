# third_party

## ebiten

A copy of [Ebitengine](https://github.com/hajimehoshi/ebiten) v2.10.5, the graphics library of the Terrarium's `window` module, under its own license (`ebiten/LICENSE`, `ebiten/NOTICE.md`). `go.mod` uses it in place of the published module:

```
replace github.com/hajimehoshi/ebiten/v2 => ./third_party/ebiten
```

It has only the packages lizard-script builds, on any system (the ones `go mod tidy` counts), and one change, in `ebiten/internal/graphicsdriver/metal/graphics_darwin.go`:

- **Before:** Ebitengine creates its Metal device in an `init` function, on the main thread, so every program that links it pays for the device when it starts, with a window or without. `lizardscript` links the window module for every program, so an empty program took about 26 ms to start on an M4, 17 ms of it the device.
- **After:** on macOS the device is created the first time `NewGraphics` runs, which is when the first window opens, on the main thread (Ebitengine's own comment there says `NewGraphics` runs on the main thread on desktops). On iOS, where it doesn't, `init` still creates it. An empty program starts in about 9 ms; a window works as before.

The change is `tools/ebitencopy/patch.go`, small enough to propose upstream. To move to a new version of Ebitengine, change its version in `go.mod`'s `require`, then:

```sh
go run ./tools/ebitencopy   # copies the packages from the module cache and applies the change
go mod tidy
go test ./...
```

`ebitencopy` stops if the change doesn't fit the new version's code, so it is looked at again by hand. Don't edit the copy itself: the next run replaces it.
