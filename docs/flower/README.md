# flower: graphics for lizard-script

**flower** is the family of graphics libraries that come with lizard-script. They are [standard modules](../TERRARIUM.md#standard-modules): written in lizard-script, built into every `lizardscript`, and loaded only by the programs that ask for them, with `use`:

```
use "flower/lily2d" as lily2d
```

| Module | What it is for | Reference |
| --- | --- | --- |
| `flower/lily2d` | anti-aliased 2D vector graphics: shapes, gradients, strokes, pressure brushes, transforms, hit tests, SVG export and a retained scene that repaints only what changed | [LILY2D.md](LILY2D.md) |
| `flower/lily3d` | 3D graphics: meshes and shapes, `.obj` files, matrices, cameras, lights, fog, flat and smooth shading, wireframes, drawn with a depth buffer | [LILY3D.md](LILY3D.md) |

Every flower library draws on the Terrarium's [drawing targets](../TERRARIUM.md#drawing-targets), so the same call draws into an image, a window or a surface. The source is in `terrarium/std/flower/`, one file per concept, and each module's tests are in `tests/` (for lily2d, `tests/lily2d/`, with golden pictures).

The guide has a chapter on each: [chapter 19, lily2d](../learn/en/19-lily2d.md) ([español](../learn/es/19-lily2d.md)).

lily3d's tests are in `tests/lily3d/`, and its chapter is [chapter 20, lily3d](../learn/en/20-lily3d.md) ([español](../learn/es/20-lily3d.md)).
