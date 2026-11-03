# lily3d tests

The tests of the standard module `flower/lily3d`, run with the rest by `go test ./...` (on both engines, see `lily3d_test.go` in `tests/`) or on their own:

```sh
lizardscript test tests/lily3d
lizardscript test --vm tests/lily3d
lizardscript test --vm --bench tests/lily3d   # the spinning cube benchmark too
```

`math`, `clip`, `shapes`, `obj` and `errors` print what they work out. `render`, `shading` and `window` draw a scene and compare it, pixel by pixel, with the PNG of the same name in `golden/` (see `golden.liz`); both engines must give the same pixels. After a change meant to alter a picture, write the golden files again and look at them before keeping them:

```sh
LILY3D_GOLDEN=update lizardscript test tests/lily3d
```
