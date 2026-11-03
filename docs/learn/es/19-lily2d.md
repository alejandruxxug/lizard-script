# 19. lily2d: gráficos 2D

[← Anterior: Módulos y herramientas](18-modules-and-tools.md) · [Índice](../README.md) · [Hoja de referencia →](cheatsheet.md) · [English](../en/19-lily2d.md)

En el capítulo 17 pintaste con `img.span` e `img.fill`: una fila, un rectángulo a la vez. Eso alcanza para un círculo, pero no para una curva suave, una estrella con borde, un cielo en degradado o un trazo de lápiz que sigue la presión. **lily2d** es una biblioteca que hace todo eso. Es un módulo estándar (capítulo 18) de la familia **flower** de bibliotecas gráficas, escrita en el propio lizard-script.

Todas las funciones están en la [referencia de lily2d](../../flower/LILY2D.md) (en inglés). Este capítulo muestra las ideas, paso a paso.

## Tu primer dibujo

```
use "flower/lily2d" as lily2d

let picture = img.new(200, 120, #f4efe6)
let sun = lily2d.circle(picture, (60, 60), 40, #f2a65a)
puts(sun.kind, sun.fill, len(sun.path))
puts(img.get(picture, 60, 60), img.get(picture, 150, 60))
img.save(picture, "sun.png")
```

**Salida:**

```text
circle #f2a65a 6
#f2a65a #f4efe6
```

- `lily2d.circle(picture, (60, 60), 40, #f2a65a)` pinta un círculo en `picture`: su centro, su radio y su color. Abre `sun.png` para verlo. Su borde está **suavizado** (anti-aliasing): los píxeles que corta el borde se mezclan con el papel, así que se ve liso y no dentado.
- El primer argumento es el **destino**. Como las funciones de dibujo del Terrario, lily2d acepta una imagen, una ventana o una superficie, así que el mismo código dibuja en cualquiera.
- Cada llamada además **devuelve la figura**, un hash. `sun.kind` es `"circle"`, `sun.fill` es su pintura y `sun.path` es su contorno como una lista de comandos (un círculo son 4 curvas, un inicio y un cierre: 6).

> **Pruébalo:** dibuja tres círculos que se superpongan, con colores con alfa (`rgba(224, 96, 122, 160)`). ¿Cuál queda encima?

## Relleno, borde y grosor

El último argumento es el **estilo**. Un color solo rellena la figura. Un hash dice más:

```
use "flower/lily2d" as lily2d

let picture = img.new(200, 120, #ffffff)
lily2d.rect(picture, (10, 10), (80, 50), {"fill": #8fb996, "stroke": #1a1418, "width": 4}, 10)
lily2d.line(picture, (10, 90), (190, 90), {"stroke": #2d6a8f, "width": 6, "dash": [12, 6], "cap": "butt"})
puts(img.get(picture, 50, 35), img.get(picture, 10, 35), img.get(picture, 15, 90), img.get(picture, 25, 90))
```

**Salida:**

```text
#8fb996 #1a1418 #2d6a8f #ffffff
```

- `fill` es el interior, `stroke` el contorno, `width` el grosor del contorno. El último argumento de `rect`, `10`, redondea sus esquinas.
- `dash` corta un trazo en pedazos: 12 píxeles sí, 6 no. El píxel en x = 25 cae en un hueco, así que sigue blanco.
- `cap` es cómo termina una línea abierta (`"round"`, `"butt"`, `"square"`) y `join` cómo se ven sus esquinas (`"round"`, `"miter"`, `"bevel"`).
- En una línea, un color solo es el trazo: `lily2d.line(picture, a, b, #000)` es una línea negra de 1 píxel.

Las otras claves del estilo son `opacity` (0 a 1), `blend` (`"multiply"`, `"screen"` y los demás modos del capítulo 17) y `rule`; la referencia tiene la tabla completa.

## Caminos: curvas que pasan por puntos

`lily2d.path` recibe una lista de puntos y dibuja un contorno suave que **pasa por** ellos. Donde el contorno gira mucho conserva una esquina: un punto donde gira más de `corner` grados (55 si no dices otra cosa) queda en punta.

```
use "flower/lily2d" as lily2d

let corners = [(20, 20), (180, 20), (180, 100), (20, 100)]
let box = lily2d.path(null, corners, #000)
let pillow = lily2d.path(null, corners, {"fill": #000, "corner": 120})
puts(box.path[1][0], pillow.path[1][0])
let frame = lily2d.path(null, corners, {"fill": #000, "holes": [[(60, 40), (140, 40), (140, 80), (60, 80)]]})
puts(lily2d.contains(frame, (30, 30)), lily2d.contains(frame, (100, 60)))
```

**Salida:**

```text
L C
true false
```

- Un destino `null` solo **crea** la figura, sin pintarla en ningún lado. Sirve para figuras que vas a mover, probar o dibujar después.
- Las cuatro esquinas de un rectángulo giran 90 grados, más de 55, así que `box` son líneas rectas (`"L"`). Con `"corner": 120` ninguna queda en punta, así que `pillow` son curvas (`"C"`): un cojín.
- `holes` recorta figuras: `frame` es un marco. `lily2d.contains(figura, punto)` dice si la figura pinta ese punto, así que el centro del marco da `false`.

También puedes escribir los comandos tú mismo, como en SVG: `["M", p]` mueve, `["L", p]` es una línea, `["C", c1, c2, p]` una curva de Bézier cúbica, `["Q", c, p]` una cuadrática y `["Z"]` cierra.

> **Pruébalo:** dibuja un corazón con dos comandos `"C"` desde la punta de abajo hasta el centro de arriba y de vuelta.

## Texto

```
use "flower/lily2d" as lily2d

let picture = img.new(120, 40, #ffffff)
let hello = lily2d.text(picture, (60, 20), "hello", {"fill": #1a1418, "size": 2, "align": "center", "valign": "middle"})
puts(lily2d.measure("hello", 2), lily2d.bounds(hello))
```

**Salida:**

```text
(70, 26) [25.0, 11.0, 68.0, 18.0]
```

- El texto se dibuja con una fuente de mapa de bits de 7x13, en `size` de 1 a 8 (cada píxel de la fuente se vuelve de `size` por `size`). `align` y `valign` dicen dónde está `pos`: aquí, en el centro del texto.
- `lily2d.measure` es la caja que la fuente le da al texto; `lily2d.bounds` es la caja de los píxeles que de verdad pintó (`[x, y, w, h]`), que es más pequeña, porque las letras no llenan toda su celda.

## Degradados

Un degradado va donde va un color:

```
use "flower/lily2d" as lily2d

let picture = img.new(200, 60, #ffffff)
let sky = lily2d.linear([#2b1d3f, #e0607a, #f6c453])
lily2d.rect(picture, (0, 0), (200, 60), sky)
puts(img.get(picture, 100, 0), img.get(picture, 100, 30), img.get(picture, 100, 59))
let bands = lily2d.linear([#000000, #ffffff], 0deg, 4)
lily2d.rect(picture, (0, 0), (200, 60), bands)
puts(img.get(picture, 10, 30), img.get(picture, 60, 30), img.get(picture, 110, 30), img.get(picture, 190, 30))
```

**Salida:**

```text
#2e1e40 #e06279 #f6c254
#000000 #555555 #aaaaaa #ffffff
```

- `lily2d.linear(stops)` va del primer color al último **a lo ancho de la figura que pinta**: de arriba abajo (`90deg`) si no le das un ángulo. Así el mismo `sky` sirve para un botón pequeño o para toda la ventana.
- `0deg` va de izquierda a derecha, y el tercer argumento, `4`, corta el degradado en 4 franjas planas.
- `lily2d.radial(stops)` sale del centro en círculos. `lily2d.linear_at(from, to, stops)` y `lily2d.radial_at(center, r, stops)` quedan fijos en el dibujo, en píxeles, así que varias figuras pueden compartir un degradado, como un atardecer detrás de muchas colinas.
- `lily2d.hard(color)` como parada hace un borde nítido en lugar de una mezcla suave.

## Un lápiz que siente la presión

`lily2d.brush(target, points, pressures, color, pen)` dibuja un trazo suave cuyo grosor sigue una presión, un número de 0 a 1 por cada punto. La presión viene de los eventos de la ventana (`e.pen.pressure`, capítulo 17), y el **lápiz** (pen) decide cómo se ve:

```
use "flower/lily2d" as lily2d

let ink = lily2d.brush(null, [(10, 50), (60, 20), (110, 50)], [0.1, 1.0, 0.1], #1a1418, lily2d.pens.ink)
let marker = lily2d.brush(null, [(10, 50), (60, 20), (110, 50)], [0.1, 1.0, 0.1], #1a1418, lily2d.pens.marker)
puts(ink.widths)
puts(marker.widths, marker.fill)
puts(keys(lily2d.pens))
```

**Salida:**

```text
[2.777809742251543, 6.670711017102133, 2.9631859934349043]
[10.0, 10.0, 10.0] #1a14188c
["ink", "pencil", "marker", "brush", "calligraphy"]
```

- El lápiz de tinta (`ink`) engrosa donde presionas más (y además suaviza un poco las presiones, para que la línea sea estable). El marcador (`marker`) mantiene un solo grosor y es transparente: su relleno tiene alfa `8c`.
- Un lápiz es un hash, así que puedes escribir el tuyo: `{"name": "felt", "min": 4, "max": 6, ...}`. La referencia lista sus claves, y el **bloque de presión** que dice, por ejemplo, que un trackpad necesita apretar más.

Ejecuta `lizardscript examples/lily2d/doodle.liz` para dibujar con los cinco lápices; las teclas 1 a 5 cambian el lápiz.

## Mover figuras

Una **matriz** mueve, gira o cambia el tamaño de una figura. `lily2d.transform(figura, m)` da una copia movida:

```
use "flower/lily2d" as lily2d

let leaf = lily2d.ellipse(null, (100, 60), (10, 30), #3fa34d)
let turned = lily2d.transform(leaf, lily2d.rotate(90deg, (100, 100)))
puts(lily2d.bounds(leaf))
puts(lily2d.bounds(turned))
let m = lily2d.combine(lily2d.scale(2), lily2d.translate((5, 0)))
puts(lily2d.apply(m, (10, 10)))
```

**Salida:**

```text
[90.0, 30.0, 20.0, 60.0]
[110.0, 90.0, 60.0, 20.0]
(25.0, 20.0)
```

- `lily2d.rotate(90deg, (100, 100))` gira un cuarto de vuelta en el sentido del reloj alrededor del punto `(100, 100)`: la hoja alta que estaba encima queda ancha a su derecha.
- `lily2d.combine(first, second)` hace una y luego la otra: escalar por 2 y después mover 5 a la derecha, así que `(10, 10)` va a `(25, 20)`.
- `lily2d.draw(target, figura)` pinta una figura que hiciste antes. Una flor son ocho copias de un pétalo, cada una girada 45 grados más: mira `examples/lily2d/scene.liz`.

## Escenas: pintar solo lo que cambió

Una animación podría borrar la ventana y dibujar todas las figuras en cada cuadro, pero la mayoría no se mueve. Una **escena** guarda las figuras y, cada vez que la dibujas, vuelve a pintar solo alrededor de las que cambiaron:

```
use "flower/lily2d" as lily2d

let pond = lily2d.scene({"background": #2d6a8f})
let frog = lily2d.circle(pond, (40, 40), 10, #3fa34d)
lily2d.circle(pond, (150, 80), 20, #f4b6c2)
let picture = img.new(200, 120)
puts(pond.draw(picture))
puts(pond.draw(picture))
frog = pond.replace(frog, lily2d.transform(frog, lily2d.translate((20, 0))))
puts(pond.draw(picture))
puts(same(pond.at((60, 40)), frog), pond.at((5, 5)))
```

**Salida:**

```text
[[0, 0, 200, 120]]
[]
[[29, 29, 42, 22]]
true null
```

- Con una escena como destino, una llamada de dibujo **agrega** la figura a la escena en lugar de pintarla.
- `pond.draw(picture)` devuelve los rectángulos que pintó. La primera vez es todo; sin cambios no es nada; cuando la rana se mueve es una caja pequeña alrededor de donde estaba y de donde está.
- `pond.replace(old, new)` deja la figura nueva a la misma profundidad (delante de las mismas figuras). Guarda lo que devuelve: la escena reconoce una figura por ser exactamente el mismo valor.
- `pond.at(punto)` es la figura de más adelante bajo un punto, para los clics.

En una ventana, llama `pond.draw(w)` en `tick`; la escena además le dice a la ventana qué rectángulos cambiaron, así que solo esos van a la pantalla. `lizardscript examples/lily2d/scene.liz` muestra cuánta ventana pinta cada cuadro.

## Guardar como SVG

`lily2d.svg(shapes, size, background)` escribe las figuras como un archivo SVG, que abre cualquier navegador o editor vectorial:

```
use "flower/lily2d" as lily2d

let dot = lily2d.circle(null, (10, 10), 5, #f2a65a)
puts(lily2d.svg([dot], (20, 20)))
```

**Salida:**

```text
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20">
  <path d="M 15 10 C 15 12.76 12.76 15 10 15 C 7.24 15 5 12.76 5 10 C 5 7.24 7.24 5 10 5 C 12.76 5 15 7.24 15 10 Z" fill="#f2a65a"/>
</svg>

```

Guárdalo con `fs.write("dot.svg", lily2d.svg(...))`. Las figuras de una escena son `pond.shapes()`.

## Errores comunes

**Una posición como dos números.** Los puntos son un solo valor: `(50, 50)`, no `50, 50`.

```
use "flower/lily2d" as lily2d

let picture = img.new(100, 100)
lily2d.circle(picture, 50, 50, 20, #fff)
```

**Salida:**

```text
[line 4, column 14] runtime error: wrong number of arguments: want 4, got 5
```

**Una clave de estilo que no existe.** El color del relleno es `"fill"`, y el mensaje lista todas las claves:

```
use "flower/lily2d" as lily2d

let picture = img.new(100, 100)
lily2d.circle(picture, (50, 50), 20, {"color": #fff})
```

**Salida:**

```text
std:/flower/lily2d/check.liz: [line 7, column 8] runtime error: lily2d.circle: the style has no key color; it can have fill, stroke, width, join, cap, miter, dash, dash_offset, rule, opacity, blend
```

La posición está dentro de lily2d (capítulo 18): la llamada a buscar es la que nombra el mensaje, `lily2d.circle`.

**Olvidar lo que devuelve `replace`.** Después de un replace, la figura vieja ya no está en la escena:

```
use "flower/lily2d" as lily2d

let pond = lily2d.scene()
let frog = lily2d.circle(pond, (40, 40), 10, #3fa34d)
let moved = lily2d.transform(frog, lily2d.translate((5, 0)))
pond.replace(frog, moved)
pond.remove(frog)
```

**Salida:**

```text
std:/flower/lily2d/check.liz: [line 7, column 8] runtime error: lily2d.scene.remove: this shape is not in the scene
```

Escribe `frog = pond.replace(frog, moved)`.

**Un `use` sin `as`.** `use "flower/lily2d"` sin `as` funciona, pero convierte cada función de lily2d en un nombre de tu archivo, y una de ellas es `text`, que tapa la biblioteca `text` (`text.join` deja de funcionar). Usa `as lily2d`.

## Resumen

- `use "flower/lily2d" as lily2d`, y luego `lily2d.circle(target, center, r, style)` y las demás figuras: `ellipse`, `rect`, `line`, `polyline`, `polygon`, `path`, `arc`, `star`, `text`, `brush`, `group`.
- El destino es una imagen, una ventana o una superficie (pinta ya), una escena (la guarda) o `null` (solo la crea). Cada llamada devuelve la figura.
- Un estilo es un color, un degradado o un hash: `fill`, `stroke`, `width`, `join`, `cap`, `dash`, `opacity`, `blend`, ...
- `lily2d.linear` y `lily2d.radial` se ajustan a cada figura; las versiones `_at` quedan fijas en píxeles.
- `lily2d.brush` con un lápiz convierte presiones en un trazo.
- `lily2d.transform(figura, matriz)` mueve figuras; `rotate`, `scale`, `translate`, `combine`.
- `lily2d.contains` y `scene.at` encuentran qué hay bajo un punto; `lily2d.svg` exporta.
- Una escena vuelve a pintar solo lo que cambió: `add`, `replace`, `remove`, `draw`.

[← Anterior: Módulos y herramientas](18-modules-and-tools.md) · [Índice](../README.md) · [Hoja de referencia →](cheatsheet.md)
