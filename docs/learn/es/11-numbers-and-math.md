# 11. Números: flotantes, `math` y unidades

[← Anterior: Cadenas y `text`](10-strings-and-text.md) · [Índice](../README.md) · [Siguiente: Colores y puntos →](12-colors-and-points.md) · [English](../en/11-numbers-and-math.md)

Este capítulo profundiza en los números: las dos clases, cómo se comportan los flotantes, la biblioteca `math` y las unidades `deg`, `rad`, `turn` y `px`.

## Enteros y flotantes

Un **entero** (`INTEGER`) es un número sin decimales. Un **flotante** (`FLOAT`) tiene punto decimal. Un flotante necesita dígitos a ambos lados del punto: `0.5`, no `.5`; `2.0`, no `2.`.

```
puts(type(2), type(2.0), type(0.5))
puts(2 == 2.0, 2 === 2.0)
```

**Salida:**

```text
INTEGER FLOAT FLOAT
true false
```

Las reglas para mezclarlos:

- entero con entero da entero: `7 / 2` es `3`;
- si alguno de los lados es flotante, el resultado es flotante: `7 / 2.0` es `3.5`.

```
puts(7 / 2, 7 / 2.0, 7 // 2.0, 7.5 % 2)
puts(3 * 2, 3 * 2.0)
```

**Salida:**

```text
3 3.5 3.0 1.5
6 6.0
```

Un flotante siempre se imprime con un `.` (o con un exponente), así lo distingues de un entero a simple vista: `6.0`, no `6`.

## Los flotantes son aproximados

Los flotantes se guardan en binario, así que muchos números decimales no se pueden representar exactamente. El ejemplo clásico:

```
puts(0.1 + 0.2)
puts(0.1 + 0.2 == 0.3)
```

**Salida:**

```text
0.30000000000000004
false
```

No es un error de lizard-script; todos los lenguajes con flotantes lo hacen. Para comparar flotantes, revisa que estén lo bastante cerca:

```
let a = 0.1 + 0.2
puts(math.abs(a - 0.3) < 0.000001)
```

**Salida:**

```text
true
```

Los flotantes muy grandes y muy pequeños se imprimen con exponente:

```
puts(1000000.0 * 1000000000000000.0, 0.0000001)
```

**Salida:**

```text
1e+21 1e-07
```

También puedes *escribir* un exponente: `1e3`, `2.5e-4` y `1E+6` son números válidos. Un número con exponente siempre es un flotante, aunque su valor sea entero:

```
puts(1e3, 2.5e-4, 1E+6)
puts(type(1e3), 1e3 == 1000)
puts(1e-7, 1.5e21)
```

**Salida:**

```text
1000.0 0.00025 1000000.0
FLOAT true
1e-07 1.5e+21
```

Después de la `e` tienen que venir dígitos (con un `+` o un `-` opcional antes). Un número demasiado grande para un flotante es un error:

```
puts(1e400)
```

**Salida:**

```text
[line 1, column 6] syntax error: number 1e400 is too large
```

## Convertir

```
puts(int(3.99), int(-3.99))
puts(float(3), float("0.25"))
puts(math.round(3.5), math.floor(3.99), math.ceil(3.01))
```

**Salida:**

```text
3 -3
3.0 0.25
4 3 4
```

`int` corta los decimales hacia cero; `math.round`, `math.floor` y `math.ceil` redondean de distintas maneras y dan enteros.

## Límites

Los enteros son de 64 bits. Al pasar el más grande, dan la vuelta al más negativo, como en Go:

```
puts(9223372036854775807 + 1)
```

**Salida:**

```text
-9223372036854775808
```

Los flotantes nunca se vuelven infinitos ni "no es un número" (NaN). Una operación que daría uno de ellos es un error:

```
puts(10.0 ** 400)
```

**Salida:**

```text
[line 1, column 11] runtime error: the result is not a number
```

## La biblioteca `math`

`math` contiene constantes y funciones, que se usan con un punto. Los ángulos están en **radianes**.

| Nombre | Da |
| --- | --- |
| `math.pi`, `math.tau` | π y 2π |
| `math.sqrt(x)` | raíz cuadrada |
| `math.abs(x)` | sin el signo |
| `math.floor(x)`, `math.ceil(x)`, `math.round(x)` | enteros, redondeados hacia abajo, hacia arriba o al más cercano |
| `math.min(a, b, ...)`, `math.max(a, b, ...)` | el menor o el mayor de dos o más |
| `math.sin(x)`, `math.cos(x)`, `math.tan(x)` | trigonometría |
| `math.asin(x)`, `math.acos(x)`, `math.atan2(y, x)` | trigonometría inversa |
| `math.random()`, `math.random(a, b)` | números aleatorios |
| `math.seed(n)` | reinicia los números aleatorios |
| `math.dist(p, q)`, `math.angle(p, q)` | distancia y ángulo entre puntos (capítulo 12) |

```
puts(math.pi, math.sqrt(16), math.abs(-7))
puts(math.min(3, 1.5, 8), math.max(3, 1.5, 8))
puts(math.round(2.5), math.round(-2.5), math.floor(-1.5))
```

**Salida:**

```text
3.141592653589793 4.0 7
1.5 8
3 -3 -2
```

`math.round` redondea las mitades alejándose de cero. `math.min` y `math.max` devuelven el argumento tal cual, así que `math.min(1, 2.5)` es el entero `1`.

Algunas entradas no tienen sentido y son un error:

```
puts(math.sqrt(-1))
```

**Salida:**

```text
[line 1, column 15] runtime error: math.sqrt: can't take the square root of a negative number: -1
```

## Unidades: `deg`, `rad`, `turn`, `px`

Pensar en radianes es difícil, así que lizard-script te deja escribir una unidad justo después de un número, **sin espacio**:

| Unidad | Significa | Ejemplo |
| --- | --- | --- |
| `deg` | grados, convertidos a radianes | `90deg` es `1.5707...` |
| `rad` | radianes, tal cual | `1.5rad` es `1.5` |
| `turn` | vueltas completas, convertidas a radianes | `0.25turn` es `90deg` |
| `px` | píxeles: el número queda igual | `10px` es `10` |

```
puts(180deg, 180deg == math.pi)
puts(0.25turn == 90deg)
puts(math.sin(90deg), math.cos(0deg))
puts(10px, type(10px), 10.5px)
```

**Salida:**

```text
3.141592653589793 true
true
1.0 1.0
10 INTEGER 10.5
```

Para pasar de radianes a grados, divide entre `1deg`:

```
let angle = math.atan2(1, 1)
puts(angle, angle / 1deg)
```

**Salida:**

```text
0.7853981633974483 45.0
```

Solo esas cuatro palabras son unidades. Cualquier otra cosa pegada a un número es un error:

```
let weight = 5kg
```

**Salida:**

```text
[line 1, column 14] syntax error: unknown unit "kg"
```

Se informa igual dentro de una llamada, un arreglo o un hash:

```
puts(5kg)
```

**Salida:**

```text
[line 1, column 6] syntax error: unknown unit "kg"
```

## Números aleatorios

`math.random()` da un flotante desde 0 hasta 1 (sin incluirlo); `math.random(a, b)` un entero de `a` a `b`, ambos incluidos.

Los números aleatorios de lizard-script son **repetibles**: cada ejecución empieza con la misma semilla, así que el mismo programa imprime los mismos números cada vez. Eso hace que los programas sean fáciles de probar. `math.seed(n)` elige otra secuencia; el mismo `n` siempre da la misma.

```
math.seed(7)
let first = [math.random(1, 6), math.random(1, 6), math.random(1, 6)]
math.seed(7)
let again = [math.random(1, 6), math.random(1, 6), math.random(1, 6)]
puts(first, again, first == again)
```

**Salida:**

```text
[2, 1, 1] [2, 1, 1] true
```

## Ejemplo: puntos sobre un círculo

```
let radius = 100px
for (let i = 0; i < 4; i += 1) {
  let angle = i * 90deg
  let x = math.round(radius * math.cos(angle))
  let y = math.round(radius * math.sin(angle))
  puts(i * 90, "degrees ->", x, y)
}
```

**Salida:**

```text
0 degrees -> 100 0
90 degrees -> 0 100
180 degrees -> -100 0
270 degrees -> 0 -100
```

> **Pruébalo.** Escribe `hypotenuse(a, b)` con `math.sqrt`. Luego escribe `toFahrenheit(c)`, que dé un flotante (`c * 9 / 5 + 32`). ¿Qué pasa con `toFahrenheit(37)` si olvidas el flotante?

## Resumen

- `7 / 2` es `3`; `7 / 2.0` es `3.5`. Un flotante se imprime con `.`.
- Los flotantes son aproximados: compáralos con una pequeña tolerancia.
- `1e3` y `2.5e-4` son literales flotantes; no hay Infinity ni NaN.
- `math.*` para raíces, redondeo, trigonometría y números aleatorios.
- `45deg`, `0.5turn` y `1.2rad` son radianes; `10px` es simplemente `10`.
- Los números aleatorios se repiten entre ejecuciones; `math.seed(n)` los cambia.

[← Anterior: Cadenas y `text`](10-strings-and-text.md) · [Índice](../README.md) · [Siguiente: Colores y puntos →](12-colors-and-points.md)
