# 4. Operadores

[← Anterior: Variables y alcance](03-variables-and-scope.md) · [Índice](../README.md) · [Siguiente: Condicionales →](05-conditionals.md) · [English](../en/04-operators.md)

Los operadores combinan valores: `+` suma, `<` compara, `&&` une condiciones. Este capítulo los cubre todos, junto con las pocas sorpresas que esconden.

## Aritmética

| Operador | Significado |
| --- | --- |
| `+` `-` `*` | sumar, restar, multiplicar |
| `/` | dividir |
| `//` | dividir redondeando **hacia abajo** |
| `%` | residuo (módulo) |
| `**` | potencia |
| `-x` | negativo |

```
puts(7 + 2, 7 - 2, 7 * 2)
puts(7 / 2, 7 // 2, 7 % 2)
puts(2 ** 10)
```

**Salida:**

```text
9 5 14
3 3 1
1024
```

### División entera vs. división con flotantes

Esta es la sorpresa más común de todo el lenguaje. **Dos enteros siempre dan un entero**, así que `/` descarta los decimales:

```
puts(1 / 2)
puts(10 / 4)
```

**Salida:**

```text
0
2
```

En cuanto uno de los lados es flotante, el resultado es flotante y `/` es la división real:

```
puts(1 / 2.0)
puts(10.0 / 4)
puts(float(10) / 4)
```

**Salida:**

```text
0.5
2.5
2.5
```

> **Error común: un promedio que sale mal.**
>
> ```
> let total = 7
> let count = 2
> puts("average:", total / count)
> puts("average:", float(total) / count)
> ```
>
> **Salida:**
>
> ```text
> average: 3
> average: 3.5
> ```

### `/` vs. `//` con números negativos

Con números positivos, `/` y `//` dan el mismo entero. Se diferencian con los negativos: `/` redondea **hacia cero** y `//` redondea **hacia abajo**:

```
puts(-7 / 2, -7 // 2)
puts(7 / 2, 7 // 2)
```

**Salida:**

```text
-3 -4
3 3
```

`%` va de la mano con `//`, así que el residuo tiene el signo del lado derecho. Es lo que quieres para cosas como "qué día de la semana es":

```
puts(7 % 3, -7 % 3, 7 % -3)
```

**Salida:**

```text
1 2 -2
```

Dividir entre cero siempre es un error, también con `//` y `%`:

```
puts(5 % 0)
```

**Salida:**

```text
[line 1, column 8] runtime error: division by zero
```

### Potencia

`**` agrupa desde la **derecha**, y es más fuerte que el signo menos que lleva delante:

```
puts(2 ** 3 ** 2)
puts(-2 ** 2)
puts((-2) ** 2)
```

**Salida:**

```text
512
-4
4
```

`2 ** 3 ** 2` es `2 ** 9`, y `-2 ** 2` es `-(2 ** 2)`. Con enteros, un exponente negativo es un error porque el resultado no sería entero; usa un flotante:

```
puts(2.0 ** -1)
puts(2 ** -1)
```

**Salida:**

```text
0.5
[line 2, column 8] runtime error: negative exponent: 2 ** -1
```

### Operadores con cadenas y arreglos

`+` también une dos cadenas o dos arreglos, y `*` repite una cadena:

```
puts("Liz" + "ard")
puts("ab" * 3)
puts([1, 2] + [3])
```

**Salida:**

```text
Lizard
ababab
[1, 2, 3]
```

## Comparación

`<`, `>`, `<=` y `>=` comparan dos números, o dos cadenas en orden de diccionario. Comparar tipos distintos es un error:

```
puts(3 < 5, 3 >= 5, 1.5 < 2)
puts("apple" < "banana")
puts(3 < "5")
```

**Salida:**

```text
true false true
true
[line 3, column 8] runtime error: type mismatch: INTEGER < STRING
```

El orden de las cadenas es por código de carácter, así que todas las mayúsculas van antes que todas las minúsculas, y los dígitos se comparan uno por uno:

```
puts("Zebra" < "apple", "10" < "9")
```

**Salida:**

```text
true true
```

## Igualdad: `==` vs. `===`

lizard-script tiene dos igualdades:

- `===` es **exacta**: mismo tipo y mismo valor. `!==` es su contraria.
- `==` es **flexible**: si los tipos son distintos, primero convierte booleanos y cadenas numéricas en números. `!=` es su contraria.

```
puts(5 == "5", 5 === "5")
puts(1 == true, 1 === true)
puts(1 == 1.0, 1 === 1.0)
puts("five" == 5, null == 0, null == null)
```

**Salida:**

```text
true false
true false
true false
false false true
```

Los arreglos y los hashes son iguales cuando su contenido lo es, usando la misma clase de igualdad para los elementos:

```
puts([1, 2] == [1, 2])
puts([1, "2"] == [1, 2], [1, "2"] === [1, 2])
puts({"a": 1} == {"a": 1})
```

**Salida:**

```text
true
true false
true
```

La igualdad nunca falla, compares lo que compares. **Si tienes dudas, usa `===`**: nunca convierte nada a tus espaldas.

## Lógica

| Operador | Es verdadero cuando |
| --- | --- |
| `a && b` | ambos son verdaderos |
| `a \|\| b` | al menos uno es verdadero |
| `!a` | `a` es falso |

Siempre dan `true` o `false`, y funcionan con los valores verdaderos y falsos del capítulo 2:

```
puts(true && false, true || false, !true)
puts(5 && "hi", 0 || "")
```

**Salida:**

```text
false true false
true false
```

`&&` y `||` **se detienen antes**: el lado derecho solo se ejecuta cuando hace falta. Eso permite proteger una operación que podría fallar:

```
let d = 0
puts(d != 0 && 10 / d > 1)
```

**Salida:**

```text
false
```

Como `d != 0` fue falso, `10 / d` nunca se ejecutó, así que no hubo división entre cero.

## Asignación compuesta

`x += e` es una forma corta de `x = x + e`. Lo mismo vale para `-=`, `*=` y `/=`:

```
let score = 10
score += 5
score -= 3
score *= 2
score /= 4
puts(score)

let word = "ab"
word += "c"
word *= 2
puts(word)
```

**Salida:**

```text
6
abcabc
```

Igual que con `=`, la variable ya debe existir. No hay `**=`, `//=` ni `%=`:

```
let x = 2
x **= 3
```

**Salida:**

```text
[line 2, column 5] syntax error: expected an expression, got "="
```

Escribe `x = x ** 3` en su lugar. La asignación compuesta también funciona con elementos de arreglos y claves de hashes (capítulos 8 y 9).

## Precedencia

Cuando una expresión mezcla operadores, los más fuertes van primero. Del más débil al más fuerte:

| Nivel | Operadores | Agrupa desde |
| --- | --- | --- |
| asignación | `=` `+=` `-=` `*=` `/=` | la derecha |
| o | `\|\|` | la izquierda |
| y | `&&` | la izquierda |
| igualdad | `==` `!=` `===` `!==` | la izquierda |
| comparación | `<` `>` `<=` `>=` | la izquierda |
| suma | `+` `-` | la izquierda |
| producto | `*` `/` `//` `%` | la izquierda |
| prefijo | `-x` `!x` | |
| potencia | `**` | la derecha |
| llamada, índice | `f(x)` `a[i]` `a.name` | la izquierda |

```
puts(1 + 2 * 3, (1 + 2) * 3)
puts(10 - 4 - 3)
puts(true || false && false)
```

**Salida:**

```text
7 9
3
true
```

Si no estás seguro, agrega paréntesis. No cuestan nada y hacen el código más fácil de leer.

> **Pruébalo.** Sin ejecutarlos, calcula `17 // 5`, `17 % 5`, `-17 // 5`, `-17 % 5` y `2 * 3 ** 2`. Luego compruébalo.

## Resumen

- Dos enteros dan un entero: `1 / 2` es `0`. Haz flotante uno de los lados para la división real.
- `/` redondea hacia cero, `//` hacia abajo; `%` va con `//`.
- `**` agrupa desde la derecha y es más fuerte que `-`.
- `===` es exacta, `==` convierte. Prefiere `===`.
- `&&` y `||` se detienen antes y siempre dan un booleano.
- Existen `+=`, `-=`, `*=`, `/=`; no existen `**=`, `//=` ni `%=`.

[← Anterior: Variables y alcance](03-variables-and-scope.md) · [Índice](../README.md) · [Siguiente: Condicionales →](05-conditionals.md)
