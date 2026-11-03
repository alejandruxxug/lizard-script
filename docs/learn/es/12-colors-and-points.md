# 12. Colores y puntos

[← Anterior: Números, `math` y unidades](11-numbers-and-math.md) · [Índice](../README.md) · [Siguiente: Capas →](13-layers.md) · [English](../en/12-colors-and-points.md)

lizard-script tiene dos valores pensados para programas de dibujo: los **colores** y los **puntos**. Son valores normales, así que puedes guardarlos en variables, arreglos y hashes, y pasarlos a funciones.

## Colores

Un color se escribe como en CSS o en una herramienta de diseño: `#` y luego 3, 6 u 8 dígitos hexadecimales.

| Forma | Ejemplo | Significa |
| --- | --- | --- |
| `#rgb` | `#f80` | forma corta, cada dígito duplicado: `#ff8800` |
| `#rrggbb` | `#3fa34d` | rojo, verde y azul de `00` a `ff` |
| `#rrggbbaa` | `#00000080` | más alfa (opacidad): `00` es invisible, `ff` es sólido |

```
let leaf = #3fa34d
puts(leaf, type(leaf))
puts(#f80, #3FA34D, #00000080)
```

**Salida:**

```text
#3fa34d COLOR
#ff8800 #3fa34d #00000080
```

Los colores siempre se imprimen en minúsculas, y el alfa solo se muestra cuando no es `ff`.

### Leer los canales

`c.r`, `c.g`, `c.b` y `c.a` dan los cuatro canales, como enteros de 0 a 255:

```
let leaf = #3fa34d
puts(leaf.r, leaf.g, leaf.b, leaf.a)
```

**Salida:**

```text
63 163 77 255
```

### Crear colores a partir de números

| Función | Crea |
| --- | --- |
| `rgb(r, g, b)` | un color a partir de rojo, verde y azul, de 0 a 255 |
| `rgba(r, g, b, a)` | lo mismo, con alfa |
| `hsl(h, s, l)` | un color a partir de un tono en grados (0 a 360) y una saturación y una luminosidad en porcentaje (0 a 100) |

```
puts(rgb(255, 128, 0))
puts(rgba(0, 0, 0, 128))
puts(hsl(0, 100, 50), hsl(120, 100, 25), hsl(210, 50, 40))
```

**Salida:**

```text
#ff8000
#00000080
#ff0000 #008000 #336699
```

`hsl` es práctico para crear una gama de colores relacionados: deja fijos `s` y `l` y cambia el tono.

```
for (let i = 0; i < 6; i += 1) {
  puts(hsl(i * 60, 80, 50))
}
```

**Salida:**

```text
#e61919
#e6e619
#19e619
#19e6e6
#1919e6
#e619e6
```

Los valores fuera de rango son un error:

```
puts(rgb(300, 0, 0))
```

**Salida:**

```text
[line 1, column 9] runtime error: rgb: red must be from 0 to 255, got 300
```

### Los colores no se pueden cambiar

Como las cadenas, los colores son fijos. Para obtener una variación, crea un color nuevo:

```
let leaf = #3fa34d
leaf.a = 100
```

**Salida:**

```text
[line 2, column 5] runtime error: colors can't be changed
```

```
function withAlpha(c, alpha) {
  rgba(c.r, c.g, c.b, alpha)
}
puts(withAlpha(#3fa34d, 100))
```

**Salida:**

```text
#3fa34d64
```

No hay aritmética con colores (`#fff + #000` es un error). `==` compara los canales, y un color nunca es igual a una cadena:

```
puts(#fff == #ffffff, #fff == "#ffffff")
```

**Salida:**

```text
true false
```

## Puntos

Un punto son dos o tres números entre paréntesis: `(x, y)` o `(x, y, z)`. La coma es lo que lo convierte en punto; `(5)` es solo el número `5` entre paréntesis.

```
let a = (10, 20)
let b = (40, 60)
puts(a, type(a))
puts(a.x, a.y)
puts((1, 2, 3).z)
puts(point(3, 4))
```

**Salida:**

```text
(10, 20) POINT
10 20
3
(3, 4)
```

Cada coordenada conserva su propio tipo, así que `(0.5, 2)` tiene una `x` flotante y una `y` entera.

### Aritmética con puntos

| Expresión | Da |
| --- | --- |
| `p + q`, `p - q` | coordenada por coordenada |
| `p * n`, `n * p`, `p / n` | cada coordenada multiplicada o dividida por el número `n` |
| `-p` | cada coordenada con el signo cambiado |

```
let a = (10, 20)
let b = (40, 60)
puts(a + b, b - a)
puts(a * 2, b / 2, -a)
puts((3, 4) / 2, (3, 4) / 2.0)
```

**Salida:**

```text
(50, 80) (30, 40)
(20, 40) (20, 30) (-10, -20)
(1, 2) (1.5, 2.0)
```

A cada coordenada se le aplican las reglas usuales de los números: `(3, 4) / 2` redondea como `3 / 2`. No se pueden mezclar puntos con distinta cantidad de coordenadas, y un punto más un número no tiene sentido:

```
puts((1, 2) + (1, 2, 3))
```

**Salida:**

```text
[line 1, column 13] runtime error: can't mix points of 2 and 3 coordinates: POINT + POINT
```

```
puts((1, 2) + 1)
```

**Salida:**

```text
[line 1, column 13] runtime error: type mismatch: POINT + INTEGER
```

Los puntos tampoco se pueden cambiar (`p.x = 3` da el error `points can't be changed`); crea uno nuevo: `(p.x + 3, p.y)`.

### Distancia y ángulo

```
let a = (0, 0)
let b = (3, 4)
puts(math.dist(a, b))
puts(math.angle(a, (10, 10)) / 1deg)
```

**Salida:**

```text
5.0
45.0
```

`math.angle(p, q)` es la dirección de `p` hacia `q`, en radianes, medida desde la dirección positiva de x.

### Un punto es un solo argumento

Dentro de una llamada, `f((1, 2))` pasa **un** punto, mientras que `f(1, 2)` pasa dos números:

```
function show(p) {
  puts("x:", p.x, "y:", p.y)
}
show((1, 2))
show(1, 2)
```

**Salida:**

```text
x: 1 y: 2
[line 5, column 5] runtime error: wrong number of arguments: want 1, got 2
```

## Ejemplo: una estrella de puntos

Puntos, unidades y `math` juntos. Cada punta de una estrella de 5 puntas es una rotación alrededor del centro:

```
let center = (100, 100)
let radius = 50
for (let i = 0; i < 5; i += 1) {
  let angle = i * 72deg - 90deg
  let offset = (math.round(radius * math.cos(angle)), math.round(radius * math.sin(angle)))
  puts(i, center + offset)
}
```

**Salida:**

```text
0 (100, 50)
1 (148, 85)
2 (129, 140)
3 (71, 140)
4 (52, 85)
```

> **Pruébalo.** Escribe `midpoint(p, q)`, que dé el punto a mitad de camino entre dos puntos. Luego escribe `lighter(c)`, que dé un color con cada canal movido hasta la mitad del camino hacia 255.

## Resumen

- Colores: `#rgb`, `#rrggbb`, `#rrggbbaa`, o `rgb`, `rgba`, `hsl`. Se leen con `c.r c.g c.b c.a`.
- Puntos: `(x, y)` o `(x, y, z)`, o `point(...)`. Se leen con `p.x p.y p.z`.
- `+` y `-` entre puntos; `*` y `/` con un número.
- Ambos son fijos: crea nuevos en lugar de cambiarlos.
- `math.dist` y `math.angle` miden entre puntos.

[← Anterior: Números, `math` y unidades](11-numbers-and-math.md) · [Índice](../README.md) · [Siguiente: Capas →](13-layers.md)
