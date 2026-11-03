# 14. Errores, `try`/`catch` y `null`

[← Anterior: Capas](13-layers.md) · [Índice](../README.md) · [Siguiente: Referencia de funciones integradas →](15-builtins-reference.md) · [English](../en/14-errors.md)

Los programas fallan. Este capítulo trata de entender los mensajes de error, lanzar tus propios errores con `error()`, recuperarte de ellos con `try`/`catch` y usar `null` para decir "ningún valor".

## Los tres tipos de error

Todos los mensajes tienen la misma forma: `[line L, column C] tipo error: mensaje`.

Los **errores léxicos** (`lexical error`) son caracteres que el lenguaje no conoce. Se detectan antes de que se ejecute nada:

```
let price = 10 @ 2
```

**Salida:**

```text
[line 1, column 16] lexical error: unexpected character "@"
```

Los **errores de sintaxis** (`syntax error`) son piezas en el orden equivocado. También se detectan antes de ejecutar nada, y el analizador los reporta todos de una vez: un error por cada equivocación, y las líneas que siguen a una equivocación se siguen revisando:

```
puts("this line never runs")
let = 5
let total = (1 + 2
if (true) { break }
```

**Salida:**

```text
[line 2, column 5] syntax error: let needs a name before the "=", like let x = 1
[line 3, column 19] syntax error: the "(" at line 3, column 13 is never closed
[line 4, column 13] syntax error: break outside of a loop
```

Un `)`, `]` o `}` que falta se reporta justo donde falta, y el mensaje dice dónde se abrió el paréntesis. Una línea tiene como mucho un error: una equivocación suele romper el resto de su línea, así que una segunda equivocación en la misma línea aparece cuando se corrige la primera.

### Mensajes que ayudan

Muchos errores de sintaxis dicen lo que probablemente quisiste escribir:

```
fucntion double(x) {
  x * 2
}
let n = 3
if (n = 3) {
  puts("three")
} elseif (n > 3) {
  puts("more")
}
let pet = {"name" = "Liz"}
puts("unclosed"
let age = 2
```

**Salida:**

```text
[line 1, column 1] syntax error: "fucntion" is not a keyword; did you mean "function"?
[line 5, column 7] syntax error: "=" in a condition gives n a value; did you mean "=="?
[line 7, column 3] syntax error: "elseif" is not a keyword; did you mean "elif"?
[line 10, column 19] syntax error: a hash puts ":" between a key and its value, not "="
[line 11, column 16] syntax error: the "(" at line 11, column 5 is never closed
```

| El mensaje empieza con | Normalmente significa |
| --- | --- |
| `"x" is not a keyword; did you mean ...` | una palabra clave mal escrita, o de otro lenguaje (`elseif`, `var`, `def`) |
| `"=" in a condition gives x a value` | `=` donde se quería `==`: `if (x == 1)` |
| `the "(" at line L, column C is never closed` | falta el `)` (o `]`, `}`) del paréntesis que está en ese lugar |
| `block is never closed` | falta un `}`; la posición es la del `{` que no lo tiene |
| `this "}" has nothing to close` | sobra un `}` (o `)`, `]`) |
| `let needs a name before the "="` | `let = 5` en vez de `let x = 5` |
| `"if" is a reserved word, it can't be a name` | una palabra clave usada como nombre de variable o parámetro |
| `a hash puts ":" between a key and its value` | `{"a" = 1}` en vez de `{"a": 1}` |
| `a let gives its value with "=", not ":"` | `let x: 1` en vez de `let x = 1` |
| `unterminated string: this " is never closed` | un string sin su `"` de cierre; la posición es donde empieza |

Los **errores de ejecución** (`runtime error`) ocurren mientras el programa corre. Todo lo anterior ya pasó:

```
let toes = 20
puts("toes per foot:", toes / 4)
puts("toes per wing:", toes / 0)
puts("this line never runs")
```

**Salida:**

```text
toes per foot: 5
[line 3, column 29] runtime error: division by zero
```

La columna señala el operador, nombre o llamada que falló. Aquí, la columna 29 es el `/` de `toes / 0`.

### Errores de ejecución comunes

| El mensaje empieza con | Normalmente significa |
| --- | --- |
| `identifier not found: x` | un error de tipeo, o una variable usada fuera de su alcance |
| `can't assign to x: it was never declared` | `x = ...` sin un `let x` antes |
| `type mismatch: STRING + INTEGER` | mezclar tipos; convierte con `str`, `int`, `float` |
| `wrong number of arguments` | una llamada con demasiados o muy pocos argumentos |
| `not a function: INTEGER` | llamar algo que no es una función |
| `index out of range` | `a[i] = x` más allá del final; usa `push` |
| `division by zero` | `/`, `//` o `%` entre `0` o `0.0` |
| `stack overflow` | recursión sin caso base |

## Lanzar tus propios errores: `error()`

`error(mensaje)` detiene el programa con tu mensaje, en la posición de la llamada. Úsalo cuando una función recibe una entrada que no puede manejar:

```
function divide(a, b) {
  if (b == 0) {
    error("can't divide " + str(a) + " by zero")
  }
  a / b
}
puts(divide(10, 2))
puts(divide(1, 0))
```

**Salida:**

```text
5
[line 3, column 10] runtime error: can't divide 1 by zero
```

### Errores que señalan a quien llama

En una biblioteca, un argumento equivocado es el error del programa que la llamó, no de la línea de la biblioteca que lo notó. `error(mensaje, {"at": "caller"})` reporta el error en la llamada del programa al archivo de la biblioteca:

```
-~> plants/pots.liz
function _need_size(size) {
  if (type(size) != "INTEGER") {
    error("make: size must be a whole number, not " + type(size), {"at": "caller"})
  }
}
function make(size) {
  _need_size(size)
  return "a pot of size " + str(size)
}
```

```
use "plants/pots"
puts(make(3))
let pot = make("big")
```

**Salida:**

```text
a pot of size 3
[line 3, column 15] runtime error: make: size must be a whole number, not STRING
```

Sin `{"at": "caller"}` el mismo error señala adentro de la biblioteca, en `plants/pots.liz: [line 3, column 10]`, donde quien escribió `make("big")` no tiene nada que arreglar. El error sale de los ayudantes de la propia biblioteca (`_need_size` se llama desde el mismo archivo) y pasa a la primera llamada desde otro archivo. `flower/lily2d` y `flower/lily3d` revisan sus argumentos así.

## Recuperarse: `try` / `catch`

`try { ... } catch (e) { ... }` ejecuta el primer bloque. Si dentro ocurre un error de ejecución (también en lo profundo de una función que llamó), se salta el resto del bloque y se ejecuta el bloque `catch`:

```
function divide(a, b) {
  if (b == 0) { error("can't divide by zero") }
  a / b
}
try {
  puts(divide(10, 2))
  puts(divide(1, 0))
  puts("not reached")
} catch (e) {
  puts("caught:", e.message)
}
puts("the program goes on")
```

**Salida:**

```text
5
caught: can't divide by zero
the program goes on
```

`e` es un hash con el mensaje (`message`), y la línea (`line`) y la columna (`column`) donde ocurrió el error:

```
try {
  let x = nope
} catch (e) {
  puts(e)
  puts("line", e.line, "column", e.column)
}
```

**Salida:**

```text
{"message": "identifier not found: nope", "line": 2, "column": 11}
line 2 column 11
```

El nombre es opcional: `catch { ... }` cuando no necesitas los detalles.

### `try` como última instrucción de una función

`try` es una **instrucción**, no una expresión, así que `let x = try { ... }` no funciona:

```
let n = try { int("12") } catch { 0 }
```

**Salida:**

```text
[line 1, column 13] syntax error: expected end of statement, got "{"
```

Pero un `try` tiene el valor del bloque que se ejecutó, así que como última instrucción de una función se convierte en el resultado de la función. Ese es el patrón habitual:

```
function toNumber(s, fallback = 0) {
  try { int(s) } catch { fallback }
}
puts(toNumber("12"), toNumber("twelve"), toNumber("twelve", -1))
```

**Salida:**

```text
12 0 -1
```

### Errores dentro del `catch`

Un error dentro del bloque `catch` no lo atrapa el mismo `try`, pero sí un `try` de afuera. Así se agrega información y se pasa el error hacia arriba:

```
function load(name) {
  try {
    error("file missing")
  } catch (e) {
    error("couldn't load " + name + ": " + e.message)
  }
}
try {
  load("song.txt")
} catch (e) {
  puts(e.message)
}
```

**Salida:**

```text
couldn't load song.txt: file missing
```

### Lo que `try` no puede atrapar

Un desbordamiento de pila (*stack overflow*), y el límite de pasos que ponen algunos programas anfitriones, nunca se atrapan, para que un programa roto no pueda ocultarlos:

```
function forever(n) { forever(n + 1) }
try {
  forever(0)
} catch (e) {
  puts("caught it")
}
```

**Salida:**

```text
[line 1, column 30] runtime error: stack overflow: more than 10000 nested calls
```

Tampoco se atrapan los errores de sintaxis: detienen el programa antes de que empiece.

### En el REPL

En el REPL, un bloque `try` se ejecuta apenas su línea está completa, así que escribe `} catch (e) {` en la **misma** línea que la `}` que cierra el `try`.

## `null`: "ningún valor"

`null` es el valor para "aquí no hay nada". Lo obtienes de una posición o clave que falta, de un `if` sin `else`, de una función sin resultado, y también puedes escribirlo tú:

```
let h = {"a": 1}
puts(h.b, [1, 2][9], type(null))
puts(null == null, null == 0, null == false)
if (null) { puts("never") } else { puts("null counts as false") }
```

**Salida:**

```text
null null NULL
true false false
null counts as false
```

### ¿`null` o `error`?

Una función tiene dos formas de decir "no pude hacerlo":

- **devolver `null`** cuando "no se encontró" es una respuesta normal que quien llama debe revisar, como `text.number("abc")` o buscar una clave que falta;
- **llamar a `error`** cuando es una equivocación de verdad que no debería pasar inadvertida, como dividir entre cero.

```
function findPet(pets, name) {
  for (let i = 0; i < len(pets); i += 1) {
    if (pets[i].name == name) { return pets[i] }
  }
  null
}
let pets = [{"name": "Liz", "age": 3}]
let found = findPet(pets, "Rex")
if (found === null) {
  puts("no pet called Rex")
}
```

**Salida:**

```text
no pet called Rex
```

> **Error común: usar un resultado `null`.** Si olvidas revisarlo, el error aparece después, lejos de la causa:
>
> ```
> let pets = {"Liz": 3}
> let age = pets["Rex"]
> puts(age + 1)
> ```
>
> **Salida:**
>
> ```text
> [line 3, column 10] runtime error: type mismatch: NULL + INTEGER
> ```
>
> `NULL + INTEGER` casi siempre significa "algo que esperaba que existiera no estaba".

> **Pruébalo.** Escribe `safeDivide(a, b)`, que devuelva `a / b`, o la cadena `"undefined"` cuando `b` es `0`, usando `try`/`catch` (no un `if`).

## Resumen

- Los errores léxicos y de sintaxis detienen el programa antes de empezar; los de ejecución lo detienen donde ocurren.
- `error(mensaje)` lanza tu propio error de ejecución.
- `try { } catch (e) { }` recupera; `e.message`, `e.line`, `e.column`.
- `try` es una instrucción; úsala como última instrucción de una función para obtener un valor.
- Los desbordamientos de pila no se pueden atrapar.
- `null` significa "ningún valor": revísalo con `=== null`.

[← Anterior: Capas](13-layers.md) · [Índice](../README.md) · [Siguiente: Referencia de funciones integradas →](15-builtins-reference.md)
