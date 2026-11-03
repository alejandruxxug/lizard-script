# 3. Variables y alcance

[← Anterior: Valores y tipos](02-values-and-types.md) · [Índice](../README.md) · [Siguiente: Operadores →](04-operators.md) · [English](../en/03-variables-and-scope.md)

Una variable es un nombre para un valor. En lizard-script hay dos acciones distintas, y no confundirlas es la clave de este capítulo:

- `let nombre = valor` **crea** una variable.
- `nombre = valor` **cambia** una variable que ya existe.

## Crear y cambiar

```
let name = "Liz"
let toes = 20
puts(name, "has", toes, "toes")

toes = toes - 1
puts("after an accident,", name, "has", toes, "toes")
```

**Salida:**

```text
Liz has 20 toes
after an accident, Liz has 19 toes
```

Una variable puede guardar cualquier tipo, y más adelante guardar un valor de otro tipo:

```
let thing = 5
thing = "five"
puts(thing)
```

**Salida:**

```text
five
```

Los nombres empiezan con una letra o `_`, seguida de letras, dígitos o `_`. Mayúsculas y minúsculas son distintas: `Name` y `name` son dos variables. Solo se permiten letras ASCII simples, así que `café` no es un nombre válido (dentro de una cadena sí puede ir).

## Error común: cambiar una variable que nunca se creó

```
total = 10
```

**Salida:**

```text
[line 1, column 1] runtime error: can't assign to total: it was never declared, use let total = ...
```

El mensaje te dice cómo arreglarlo. Exigir `let` primero atrapa errores de tipeo: si escribes mal un nombre al asignarle, obtienes un error en vez de una variable nueva silenciosa.

```
let count = 0
cuont = count + 1
```

**Salida:**

```text
[line 2, column 1] runtime error: can't assign to cuont: it was never declared, use let cuont = ...
```

Leer un nombre que no existe también es un error:

```
puts(score)
```

**Salida:**

```text
[line 1, column 6] runtime error: identifier not found: score
```

## `let` dos veces

Escribir `let` otra vez con el mismo nombre, en el mismo lugar, simplemente lo reemplaza:

```
let toes = 20
let toes = "many"
puts(toes)
```

**Salida:**

```text
many
```

## La asignación es una expresión

`=` devuelve el valor que guardó, así que se puede encadenar:

```
let a = 0
let b = 0
a = b = 7
puts(a, b)
```

**Salida:**

```text
7 7
```

## Alcance: dónde vive una variable

Cada bloque `{ ... }` (el cuerpo de un `if`, de un ciclo, de una función) es un nuevo **alcance** (*scope*). Una variable creada con `let` dentro de un bloque solo existe dentro de él:

```
if (true) {
  let secret = 42
  puts("inside:", secret)
}
puts("outside:", secret)
```

**Salida:**

```text
inside: 42
[line 5, column 18] runtime error: identifier not found: secret
```

Un bloque puede **leer y cambiar** las variables que lo rodean, siempre que use `=` y no `let`:

```
let counter = 0
function increment() {
  counter = counter + 1
}
increment()
increment()
puts("counter:", counter)
```

**Salida:**

```text
counter: 2
```

## Ocultamiento: el mismo nombre en dos alcances

Si un bloque usa `let` con un nombre que ya existe afuera, crea una variable **nueva** que oculta a la de afuera hasta que el bloque termina. La de afuera no cambia:

```
let x = "global"

function show() {
  let x = "local"
  puts("inside show, x is", x)
}
show()
puts("after show, x is still", x)

if (true) {
  let x = "in the if"
  puts("inside the if, x is", x)
}
puts("after the if, x is", x)
```

**Salida:**

```text
inside show, x is local
after show, x is still global
inside the if, x is in the if
after the if, x is global
```

Los parámetros de una función también son locales:

```
let value = 100
function double(value) {
  value * 2
}
puts(double(5), value)
```

**Salida:**

```text
10 100
```

El `let` de la cabecera de un `for` vive solo dentro del ciclo (capítulo 6):

```
let i = "outside"
for (let i = 0; i < 2; i = i + 1) {
  puts("in the loop, i is", i)
}
puts("after the loop, i is", i)
```

**Salida:**

```text
in the loop, i is 0
in the loop, i is 1
after the loop, i is outside
```

**Pruébalo.** Predice qué imprime esto y luego ejecútalo:

```
let n = 1
if (true) {
  n = 2
  let n = 3
  n = 4
}
puts(n)
```

<details><summary>Respuesta</summary>

```
let n = 1
if (true) {
  n = 2
  let n = 3
  n = 4
}
puts(n)
```

**Salida:**

```text
2
```

`n = 2` cambia la `n` de afuera. Luego `let n = 3` crea una `n` nueva dentro del bloque, así que `n = 4` cambia esa, que desaparece en el `}`.
</details>

## Hasta puedes ocultar una función integrada

Los nombres se buscan desde el alcance más interno hacia afuera, y las funciones integradas van al final. Así que una variable puede ocultar una función integrada:

```
let len = 5
puts(len)
```

**Salida:**

```text
5
```

Después de esa línea, `len("abc")` fallaría, porque ahora `len` es un número. Evita usar nombres de funciones integradas para tus variables (`len`, `type`, `str`, `list`, `text`, `math`...).

## Error común: una línea que empieza con `-`

Un salto de línea termina una instrucción solo cuando la instrucción no puede continuar. Una línea que empieza con un operador, `(` o `[` **continúa** la línea anterior:

```
let x = 10
-1
puts(x)
```

**Salida:**

```text
9
```

Eso fue `let x = 10 - 1`. Si de verdad quieres una instrucción aparte, termina la primera línea con `;`. En cambio, dos valores uno al lado del otro en la misma línea son un error:

```
let x = 5 6
```

**Salida:**

```text
[line 1, column 11] syntax error: expected end of statement, got "6"
```

## Resumen

- `let` crea, `=` cambia. Asignar a un nombre que nunca se creó es un error.
- Cada bloque `{ }` es un alcance; el `let` de adentro se queda adentro.
- Sin `let`, un bloque cambia la variable de afuera.
- Una línea que empieza con un operador, `(` o `[` continúa la anterior.

[← Anterior: Valores y tipos](02-values-and-types.md) · [Índice](../README.md) · [Siguiente: Operadores →](04-operators.md)
