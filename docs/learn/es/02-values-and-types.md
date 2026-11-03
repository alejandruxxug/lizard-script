# 2. Valores y tipos

[← Anterior: Primeros pasos](01-first-steps.md) · [Índice](../README.md) · [Siguiente: Variables y alcance →](03-variables-and-scope.md) · [English](../en/02-values-and-types.md)

Todo aquello con lo que trabaja un programa es un **valor**, y todo valor tiene un **tipo**. lizard-script es de *tipado dinámico*: nunca escribes los tipos, pero cada valor conoce el suyo, y la función integrada `type(x)` te dice cuál es.

## Los tipos de un vistazo

```
puts(type(42))
puts(type(3.5))
puts(type("Liz"))
puts(type(true))
puts(type(null))
puts(type([1, 2, 3]))
puts(type({"name": "Liz"}))
puts(type(function(x) { x }))
puts(type(len))
puts(type(#3fa34d))
puts(type((10, 20)))
```

**Salida:**

```text
INTEGER
FLOAT
STRING
BOOLEAN
NULL
ARRAY
HASH
FUNCTION
BUILTIN
COLOR
POINT
```

| Tipo | Ejemplos | Capítulo |
| --- | --- | --- |
| `INTEGER` (entero) | `0`, `42`, `-7` | 4, 11 |
| `FLOAT` (flotante) | `3.5`, `0.1`, `45deg` | 11 |
| `STRING` (cadena) | `"Liz"`, `""` | 10 |
| `BOOLEAN` (booleano) | `true`, `false` | 4, 5 |
| `NULL` | `null` | 14 |
| `ARRAY` (arreglo) | `[1, "a", true]` | 8 |
| `HASH` | `{"name": "Liz"}` | 9 |
| `FUNCTION` (función) | `function(x) { x * 2 }` | 7 |
| `BUILTIN` (función integrada) | `puts`, `len` | 15 |
| `COLOR` | `#3fa34d` | 12 |
| `POINT` (punto) | `(10, 20)` | 12 |

## Números

Hay dos clases de números. Los **enteros** no tienen punto decimal; los **flotantes** sí. La diferencia importa, sobre todo al dividir:

```
puts(7 / 2)
puts(7.0 / 2)
puts(type(7), type(7.0))
```

**Salida:**

```text
3
3.5
INTEGER FLOAT
```

Dos enteros siempre dan un entero. Los capítulos 4 y 11 entran en detalle.

## Cadenas

El texto va entre **comillas dobles**. Las comillas simples no existen en el lenguaje.

```
puts("Hello")
puts("")
puts(len("lizard"))
```

**Salida:**

```text
Hello

6
```

(La segunda línea imprimió una cadena vacía, por eso se ve como una línea en blanco.)

## Booleanos y `null`

`true` y `false` son los dos booleanos. Salen de las comparaciones:

```
puts(3 > 2, 3 == 4)
```

**Salida:**

```text
true false
```

`null` significa "ningún valor". Muchas cosas dan `null`: una función que no devuelve nada, leer una posición que no existe, el propio `puts`.

```
let nothing = null
puts(nothing, type(nothing))
puts([1, 2, 3][10])
```

**Salida:**

```text
null NULL
null
```

## Valores verdaderos y falsos

Las condiciones (`if`, `while`, `&&`, `||`, `!`) aceptan **cualquier** valor, no solo booleanos. Estos cuentan como falsos:

`false`, `null`, `0`, `0.0`, `""`, `[]`, `{}`

Todo lo demás cuenta como verdadero. `!x` convierte cualquier valor en el booleano contrario, así que `!!x` te dice cómo cuenta un valor:

```
puts(!!0, !!"", !![], !!{}, !!null)
puts(!!1, !!"a", !![0], !!-1)
```

**Salida:**

```text
false false false false false
true true true true
```

## Convertir entre tipos

| Función | Convierte en | Ejemplo |
| --- | --- | --- |
| `str(x)` | una cadena | `str(42)` es `"42"` |
| `int(x)` | un entero | `int("42")` es `42`, `int(2.9)` es `2` |
| `float(x)` | un flotante | `float(3)` es `3.0` |

```
puts(str(42) + "!")
puts(int("41") + 1)
puts(int(2.9), int(-1.9), int(true))
puts(float(3), float("2.5"))
```

**Salida:**

```text
42!
42
2 -1 1
3.0 2.5
```

`int` solo acepta una cadena que contenga un número entero:

```
puts(int("1.5"))
```

**Salida:**

```text
[line 1, column 9] runtime error: int: not a whole number: "1.5"
```

> **Error común: unir una cadena con un número.** `+` no convierte por ti.
>
> ```
> let age = 3
> puts("Liz is " + age + " years old")
> ```
>
> **Salida:**
>
> ```text
> [line 2, column 16] runtime error: type mismatch: STRING + INTEGER
> ```
>
> Usa `str`, o deja que `puts` ponga los espacios:
>
> ```
> let age = 3
> puts("Liz is " + str(age) + " years old")
> puts("Liz is", age, "years old")
> ```
>
> **Salida:**
>
> ```text
> Liz is 3 years old
> Liz is 3 years old
> ```

## Cómo se muestran los valores

`puts` y `str` escriben las cadenas sin comillas. Dentro de un arreglo o un hash, las cadenas conservan sus comillas para que se vea dónde termina cada una:

```
puts("Liz")
puts(["Liz", "Iggy"])
puts({"name": "Liz"})
```

**Salida:**

```text
Liz
["Liz", "Iggy"]
{"name": "Liz"}
```

> **Pruébalo.** Adivina el tipo de cada valor y luego compruébalo con `type`: `"5"`, `5`, `5.0`, `5 > 4`, `[5]`, `(5, 5)`.

## Resumen

- Todo valor tiene un tipo; `type(x)` da su nombre.
- Dos enteros dan un entero: `7 / 2` es `3`. Escribe `7.0` para obtener `3.5`.
- `false`, `null`, `0`, `0.0`, `""`, `[]` y `{}` cuentan como falsos.
- `str`, `int` y `float` convierten. `+` nunca convierte un número en cadena por ti.

[← Anterior: Primeros pasos](01-first-steps.md) · [Índice](../README.md) · [Siguiente: Variables y alcance →](03-variables-and-scope.md)
