# lily2d tests

The tests of the standard module `flower/lily2d`, run with the rest by `go test ./...` (on both engines, see `lily2d_test.go` in `tests/`) or on their own:

```sh
lizardscript test tests/lily2d
lizardscript test --vm tests/lily2d
lizardscript test --bench tests/lily2d     # the raster benchmark too
```

The drawing tests paint into an image and compare it, pixel by pixel, with the PNG of the same name in `golden/` (see `golden.liz`). After a change meant to alter a picture, write the golden files again and look at them before keeping them:

```sh
LILY2D_GOLDEN=update lizardscript test tests/lily2d
```
