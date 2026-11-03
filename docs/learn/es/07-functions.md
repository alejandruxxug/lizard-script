# 7. Funciones

[← Anterior: Ciclos](06-loops.md) · [Índice](../README.md) · [Siguiente: Arreglos →](08-arrays.md) · [English](../en/07-functions.md)

Una función es un pedazo de código con nombre que puedes ejecutar cada vez que lo necesites, con entradas distintas cada vez. En lizard-script las funciones son valores, igual que los números y las cadenas: puedes guardarlas en variables, pasarlas a otras funciones y devolverlas.

## Definir y llamar

```
function add(a, b) {
  return a + b
}
puts(add(2, 3))
puts(add(10, -4))
```

**Salida:**

```text
5
6
```

`a` y `b` son los **parámetros**; `2` y `3` son los **argumentos** de esa llamada. `return` devuelve el resultado y termina la función.

## Se devuelve el último valor

Sin `return`, una función da el valor de su última instrucción. Los dos estilos son comunes:

```
function square(x) {
  x * x
}
puts(square(9))
```

**Salida:**

```text
81
```

Un `return` solo (o una función cuya última instrucción no tiene valor) da `null`:

```
function greet(name) {
  puts("hi,", name)
}
let result = greet("Liz")
puts(result)
```

**Salida:**

```text
hi, Liz
null
```

`return` también sirve para terminar una función antes de tiempo:

```
function sign(n) {
  if (n < 0) { return "negative" }
  if (n == 0) { return "zero" }
  "positive"
}
puts(sign(-5), sign(0), sign(8))
```

**Salida:**

```text
negative zero positive
```

## Las funciones son valores

`function nombre(...) { }` es lo mismo que `let nombre = function(...) { }`. La segunda forma, una **función literal**, sirve cuando quieres una función sin ponerle nombre:

```
let greet = function(name) { "Hello, " + name + "!" }
puts(greet("Liz"))
puts(type(greet))
```

**Salida:**

```text
Hello, Liz!
FUNCTION
```

## Número equivocado de argumentos

Una llamada debe dar exactamente un argumento por parámetro (salvo que el parámetro tenga un valor por defecto, más abajo):

```
function add(a, b) { a + b }
add(1)
```

**Salida:**

```text
[line 2, column 4] runtime error: wrong number of arguments: want 2, got 1
```

Llamar algo que no es una función también es un error:

```
let n = 5
n(1)
```

**Salida:**

```text
[line 2, column 2] runtime error: not a function: INTEGER
```

## Valores por defecto

Un parámetro puede tener un valor por defecto, que se usa cuando la llamada no lo da. Los parámetros con valor por defecto van al final:

```
function greet(name, greeting = "Hello") {
  greeting + ", " + name + "!"
}
puts(greet("Liz"))
puts(greet("Liz", "Good morning"))
```

**Salida:**

```text
Hello, Liz!
Good morning, Liz!
```

El valor por defecto se calcula en cada llamada, después de los parámetros anteriores, así que puede usarlos:

```
function label(name, size = 12, caption = name + " (" + str(size) + ")") {
  caption
}
puts(label("tree"))
puts(label("tree", 20))
puts(label("tree", 20, "big tree"))
```

**Salida:**

```text
tree (12)
tree (20)
big tree
```

Un `[]` por defecto es un arreglo vacío nuevo en cada llamada, no uno compartido entre llamadas:

```
function collect(x, into = []) {
  push(into, x)
}
puts(collect(1), collect(2))
```

**Salida:**

```text
[1] [2]
```

Un parámetro sin valor por defecto no puede ir después de uno que sí lo tiene:

```
function f(a = 1, b) { a + b }
```

**Salida:**

```text
[line 1, column 19] syntax error: parameter b needs a default value, it comes after one that has it
```

## Recursión

Una función puede llamarse a sí misma. Toda función recursiva necesita un **caso base** que la detenga:

```
function factorial(n) {
  if (n <= 1) {
    return 1
  }
  n * factorial(n - 1)
}
puts(factorial(5), factorial(10))

function fib(n) {
  if (n < 2) { return n }
  fib(n - 1) + fib(n - 2)
}
puts(fib(20))
```

**Salida:**

```text
120 3628800
6765
```

Sin caso base, las llamadas se acumulan hasta que el intérprete las detiene:

```
function forever(n) {
  forever(n + 1)
}
forever(0)
```

**Salida:**

```text
[line 2, column 10] runtime error: stack overflow: more than 10000 nested calls
```

## Clausuras

Una función recuerda el alcance donde fue **creada**, y puede seguir usando y cambiando sus variables incluso cuando ese alcance ya terminó. A una función así se le llama **clausura** (*closure*).

```
function makeCounter() {
  let count = 0
  function() {
    count += 1
    count
  }
}
let a = makeCounter()
let b = makeCounter()
a()
a()
puts("a:", a(), "b:", b())
```

**Salida:**

```text
a: 3 b: 1
```

Cada llamada a `makeCounter` crea un `count` nuevo, así que `a` y `b` cuentan por separado. Nadie más puede tocar `count`: es privado del contador.

Una función que construye funciones:

```
function adder(x) {
  function(y) { x + y }
}
let addTen = adder(10)
let addOne = adder(1)
puts(addTen(5), addOne(5))
```

**Salida:**

```text
15 6
```

## Funciones de orden superior

Una función que recibe o devuelve otra función se llama *de orden superior*. Aquí tienes tres clásicas que puedes escribir tú mismo:

```
function map(xs, f) {
  let out = []
  for (let i = 0; i < len(xs); i += 1) { push(out, f(xs[i])) }
  out
}
function filter(xs, keep) {
  let out = []
  for (let i = 0; i < len(xs); i += 1) {
    if (keep(xs[i])) { push(out, xs[i]) }
  }
  out
}
function reduce(xs, f, start) {
  let acc = start
  for (let i = 0; i < len(xs); i += 1) { acc = f(acc, xs[i]) }
  acc
}

let evens = filter(range(1, 11), function(n) { n % 2 == 0 })
puts(evens)
puts(map(evens, function(n) { n * n }))
puts(reduce(evens, function(a, b) { a + b }, 0))
```

**Salida:**

```text
[2, 4, 6, 8, 10]
[4, 16, 36, 64, 100]
30
```

> **Pruébalo.** Escribe `compose(f, g)`, que devuelva una función que calcule `f(g(x))`. Pruébala con `compose(addTen, square)(3)`, que debería dar `19`.

## ¿En qué se va el tiempo?

Cuando un programa es lento, mide antes de cambiarlo. Córrelo con `--profile` (corre en el motor de bytecode, el más rápido) y, después de la salida del programa, lizard-script muestra cada función con cuántas veces se llamó y el tiempo que pasó en ella. **total** es el tiempo desde la llamada hasta que devuelve; **self** deja por fuera el tiempo de las funciones que llamó. Las funciones anónimas aparecen como `function@línea`. Los tiempos cambian de una corrida a otra; las llamadas no.

Guarda esto como `slow.liz` y corre `lizardscript --profile slow.liz`:

```
function fib(n) {
  if (n < 2) { return n }
  fib(n - 1) + fib(n - 2)
}
let square = function(x) { x * x }
let total = 0
for (let i = 0; i < 1000; i += 1) { total += square(i) + len(str(i)) }
puts(fib(20), total)
```

**Salida** (el perfil va a la salida de errores, después de la del programa):

```text
6765 332836390
profile: 2.77ms

functions by self time
        self        total      calls  name
     1.878ms      1.878ms      21891  fib (slow.liz:1)
     0.300ms      2.297ms          1  program (slow.liz:1)
     0.023ms      0.023ms       1000  square (slow.liz:5)

functions by total time
        self        total      calls  name
     0.300ms      2.297ms          1  program (slow.liz:1)
     1.878ms      1.878ms      21891  fib (slow.liz:1)
     0.023ms      0.023ms       1000  square (slow.liz:5)

builtins by self time
        self        total      calls  name
     0.067ms      0.067ms       1000  str
     0.023ms      0.023ms       1000  len
     0.006ms      0.006ms          1  puts
```

`fib` se llama 21891 veces para calcular `fib(20)`: casi todo el tiempo está ahí, y es la función que conviene hacer más rápida (calcula los mismos valores una y otra vez). `square` se llamó 1000 veces pero no cuesta nada.

> **Pruébalo.** Escribe un `fib` que guarde en un hash los valores que ya calculó, y vuelve a perfilarlo. ¿Cuántas llamadas quedan?

## Resumen

- `function nombre(params) { cuerpo }` o `let nombre = function(params) { cuerpo }`.
- Se devuelve el último valor; `return` devuelve antes; si no se devuelve nada, es `null`.
- Valores por defecto: `function f(a, b = 2)`. Van al final.
- Una función recursiva necesita un caso base.
- Una clausura recuerda las variables del alcance donde fue creada.
- `lizardscript --profile archivo.liz` muestra en qué se va el tiempo: llamadas, tiempo propio (self) y total de cada función.

[← Anterior: Ciclos](06-loops.md) · [Índice](../README.md) · [Siguiente: Arreglos →](08-arrays.md)
