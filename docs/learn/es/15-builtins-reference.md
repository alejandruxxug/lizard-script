# 15. Referencia de funciones integradas

[← Anterior: Errores, `try`/`catch` y `null`](14-errors.md) · [Índice](../README.md) · [Siguiente: Proyectos →](16-projects.md) · [English](../en/15-builtins-reference.md)

Todas las funciones integradas (*builtins*) y todos los miembros de las bibliotecas `math`, `text`, `list` y `hash`, con un ejemplo cada uno. La columna **Resultado** es el valor del ejemplo tal como lo muestra el REPL, así que las cadenas aparecen con comillas. Cuando el ejemplo es un error, el resultado muestra el mensaje (sin la parte `[line, column]`).

Las funciones integradas se buscan **después** de tus variables, así que `let len = 5` oculta `len`. Lo mismo pasa con `math`, `text`, `list`, `hash` y `null`.

## Generales

| Función | Hace | Ejemplo | Resultado |
| --- | --- | --- | --- |
| `puts(a, b, ...)` | imprime los valores separados por espacios y un salto de línea; da `null` | `puts("hi", 1)` | imprime `hi 1` |
| `len(x)` | caracteres de una cadena, elementos de un arreglo o claves de un hash | `len("lizard")` | `6` |
| `type(x)` | el nombre del tipo | `type(1.5)` | `"FLOAT"` |
| `str(x)` | `x` como texto, tal como lo muestra `puts` | `str(3.0)` | `"3.0"` |
| `int(x)` | una cadena con un número entero, un booleano, un entero o un flotante (cortado hacia cero) como entero | `int("42")` | `42` |
| | | `int(-1.9)` | `-1` |
| | | `int("1.5")` | `runtime error: int: not a whole number: "1.5"` |
| `float(x)` | un entero, un flotante o una cadena con un número como flotante | `float("2.5")` | `2.5` |
| `range(fin)` | `[0, 1, ..., fin - 1]` | `range(4)` | `[0, 1, 2, 3]` |
| `range(inicio, fin)` | `[inicio, ..., fin - 1]`; vacío si `fin <= inicio` | `range(2, 5)` | `[2, 3, 4]` |
| `error(mensaje)` | se detiene con un error de ejecución con ese mensaje | `error("bad input")` | `runtime error: bad input` |
| `null` | el valor "ningún valor" | `type(null)` | `"NULL"` |

## Arreglos y hashes

| Función | Hace | Ejemplo | Resultado |
| --- | --- | --- | --- |
| `push(a, x)` | agrega `x` al final de `a`, cambia `a` y lo devuelve | `push([1, 2], 3)` | `[1, 2, 3]` |
| `pop(a)` | quita el último elemento y lo devuelve; error si está vacío | `pop([1, 2, 3])` | `3` |
| | | `pop([])` | `runtime error: pop: the array is empty` |
| `first(a)` | el primer elemento, o `null` | `first([7, 8])` | `7` |
| `last(a)` | el último elemento, o `null` | `last([7, 8])` | `8` |
| `rest(a)` | un arreglo nuevo sin el primer elemento, o `null` si está vacío | `rest([7, 8, 9])` | `[8, 9]` |
| `keys(h)` | las claves, en el orden en que se agregaron | `keys({"b": 1, "a": 2})` | `["b", "a"]` |
| `values(h)` | los valores, en el mismo orden | `values({"b": 1, "a": 2})` | `[1, 2]` |
| `delete(h, clave)` | quita `clave` y dice si estaba | `delete({"a": 1}, "a")` | `true` |
| `same(a, b)` | si `a` y `b` son el mismo arreglo, hash o función (para otros valores, `===`) | `same([1], [1])` | `false` |

## Colores y puntos

| Función | Hace | Ejemplo | Resultado |
| --- | --- | --- | --- |
| `rgb(r, g, b)` | un color, canales de 0 a 255 (los flotantes se redondean) | `rgb(255, 128, 0)` | `#ff8000` |
| `rgba(r, g, b, a)` | un color con alfa | `rgba(0, 0, 0, 128)` | `#00000080` |
| `hsl(h, s, l)` | un color a partir de un tono de 0 a 360, y saturación y luminosidad de 0 a 100 | `hsl(210, 50, 40)` | `#336699` |
| `point(x, y)`, `point(x, y, z)` | lo mismo que `(x, y)` y `(x, y, z)` | `point(3, 4)` | `(3, 4)` |

## `math`

| Miembro | Hace | Ejemplo | Resultado |
| --- | --- | --- | --- |
| `math.pi` | π | `math.pi` | `3.141592653589793` |
| `math.tau` | 2π | `math.tau` | `6.283185307179586` |
| `math.sqrt(x)` | raíz cuadrada, como flotante; error si es negativo | `math.sqrt(2)` | `1.4142135623730951` |
| `math.abs(x)` | sin el signo, del mismo tipo | `math.abs(-3.5)` | `3.5` |
| `math.floor(x)` | redondeado hacia abajo, como entero | `math.floor(-1.5)` | `-2` |
| `math.ceil(x)` | redondeado hacia arriba, como entero | `math.ceil(1.2)` | `2` |
| `math.round(x)` | al entero más cercano, las mitades alejándose de cero | `math.round(2.5)` | `3` |
| `math.min(a, b, ...)` | el menor de dos o más números | `math.min(4, 1.5, 9)` | `1.5` |
| `math.max(a, b, ...)` | el mayor de dos o más números | `math.max(4, 1.5, 9)` | `9` |
| `math.sin(x)` | seno, `x` en radianes | `math.sin(90deg)` | `1.0` |
| `math.cos(x)` | coseno | `math.cos(0)` | `1.0` |
| `math.tan(x)` | tangente | `math.tan(45deg)` | `0.9999999999999998` |
| `math.asin(x)` | el ángulo cuyo seno es `x` (-1 a 1) | `math.asin(1)` | `1.5707963267948966` |
| `math.acos(x)` | el ángulo cuyo coseno es `x` (-1 a 1) | `math.acos(1)` | `0.0` |
| `math.atan2(y, x)` | el ángulo del punto `(x, y)` | `math.atan2(1, 1) / 1deg` | `45.0` |
| `math.dist(p, q)` | distancia entre dos puntos | `math.dist((0, 0), (3, 4))` | `5.0` |
| `math.angle(p, q)` | ángulo de la línea de `p` a `q` (puntos 2D) | `math.angle((0, 0), (0, 5)) / 1deg` | `90.0` |
| `math.random()` | un flotante desde 0 hasta 1 (sin incluirlo) | `math.random()` | `0.6046602879796196` |
| `math.random(a, b)` | un entero de `a` a `b`, ambos incluidos | `math.random(1, 6)` | `3` |
| `math.seed(n)` | reinicia los números aleatorios desde `n`; da `null` | `math.seed(42)` | `null` |

Los ejemplos aleatorios muestran los primeros números de una ejecución nueva: cada ejecución empieza con la misma semilla.

## `text`

Los índices cuentan caracteres. Donde se indica, un índice negativo cuenta desde el final.

| Miembro | Hace | Ejemplo | Resultado |
| --- | --- | --- | --- |
| `text.slice(s, inicio, fin)` | los caracteres desde `inicio` hasta `fin` (negativo cuenta desde el final) | `text.slice("lizard", 1, 4)` | `"iza"` |
| `text.find(s, sub)` | la posición del primer `sub`, o `-1` | `text.find("banana", "an")` | `1` |
| `text.find(s, sub, desde)` | lo mismo, empezando en `desde` | `text.find("banana", "an", 2)` | `3` |
| `text.split(s, sep)` | los pedazos entre cada `sep`; `""` separa en caracteres | `text.split("a,b,c", ",")` | `["a", "b", "c"]` |
| `text.join(lista, sep)` | une una lista de cadenas con `sep` | `text.join(["a", "b"], "-")` | `"a-b"` |
| `text.replace(s, viejo, nuevo)` | cada `viejo` reemplazado por `nuevo` | `text.replace("aXbX", "X", "-")` | `"a-b-"` |
| `text.trim(s)` | sin espacios, tabuladores ni saltos de línea en los extremos | `text.trim("  hi  ")` | `"hi"` |
| `text.lines(s)` | las líneas de `s` | `text.lines("a\nb")` | `["a", "b"]` |
| `text.starts(s, prefijo)` | si `s` empieza con `prefijo` | `text.starts("lizard", "liz")` | `true` |
| `text.ends(s, sufijo)` | si `s` termina con `sufijo` | `text.ends("lizard", "ard")` | `true` |
| `text.upper(s)` | en mayúsculas | `text.upper("Liz")` | `"LIZ"` |
| `text.lower(s)` | en minúsculas | `text.lower("Liz")` | `"liz"` |
| `text.code(s, i)` | el código del carácter en `i`, o `null` | `text.code("A", 0)` | `65` |
| `text.char(código)` | el carácter con ese código | `text.char(955)` | `"λ"` |
| `text.number(s)` | el número escrito en `s`, o `null` | `text.number("2.5")` | `2.5` |
| | | `text.number("12px")` | `null` |

## `list`

Un índice negativo cuenta desde el final.

| Miembro | Hace | Ejemplo | Resultado |
| --- | --- | --- | --- |
| `list.slice(a, inicio, fin)` | un arreglo nuevo con los elementos desde `inicio` hasta `fin` | `list.slice([1, 2, 3, 4], 1, 3)` | `[2, 3]` |
| `list.insert(a, i, x)` | pone `x` antes de la posición `i`, cambia `a` y lo devuelve | `list.insert(["a", "c"], 1, "b")` | `["a", "b", "c"]` |
| `list.remove(a, i)` | saca el elemento en `i`, cambia `a` y lo devuelve | `list.remove([1, 2, 3], -1)` | `[1, 2]` |
| `list.index(a, x)` | la posición del primer elemento `=== x`, o `-1` | `list.index([5, 6, 7], 7)` | `2` |
| `list.reverse(a)` | un arreglo nuevo en orden inverso | `list.reverse([1, 2, 3])` | `[3, 2, 1]` |
| `list.sort(a)` | un arreglo nuevo, números por valor o cadenas alfabéticamente | `list.sort([3, 1, 2])` | `[1, 2, 3]` |
| `list.sort(a, less)` | ordenado según tu función `less(x, y)`: ¿va `x` primero? | `list.sort([3, 1, 2], function(x, y) { x > y })` | `[3, 2, 1]` |
| `list.copy(a)` | un arreglo nuevo con los mismos elementos | `list.copy([1, 2])` | `[1, 2]` |

## `hash`

Ninguna cambia el hash, salvo `hash.remove`.

| Miembro | Hace | Ejemplo | Resultado |
| --- | --- | --- | --- |
| `hash.copy(h)` | un hash nuevo con las mismas claves y valores, en el mismo orden | `hash.copy({"a": 1})` | `{"a": 1}` |
| `hash.keys(h)` | las claves, como `keys` | `hash.keys({"b": 1, "a": 2})` | `["b", "a"]` |
| `hash.values(h)` | los valores, como `values` | `hash.values({"b": 1, "a": 2})` | `[1, 2]` |
| `hash.merge(a, b)` | un hash nuevo con las claves de `a` y luego las nuevas de `b`; ganan los valores de `b` | `hash.merge({"a": 1, "b": 2}, {"b": 3})` | `{"a": 1, "b": 3}` |
| `hash.has(h, clave)` | si `clave` está, aunque guarde `null` | `hash.has({"a": null}, "a")` | `true` |
| | | `hash.has({}, [1])` | `runtime error: hash.has: unusable as hash key: ARRAY` |
| `hash.remove(h, clave)` | quita `clave` y dice si estaba, como `delete` | `hash.remove({"a": 1}, "b")` | `false` |

## Valores con acceso con punto

No son funciones, pero también se leen con un punto:

| Valor | Miembros | Ejemplo | Resultado |
| --- | --- | --- | --- |
| hash | cualquier clave de tipo cadena | `{"name": "Liz"}.name` | `"Liz"` |
| color | `r`, `g`, `b`, `a` (0 a 255) | `#3fa34d.g` | `163` |
| punto | `x`, `y`, y `z` en puntos 3D | `(1, 2, 3).z` | `3` |

[← Anterior: Errores, `try`/`catch` y `null`](14-errors.md) · [Índice](../README.md) · [Siguiente: Proyectos →](16-projects.md)
