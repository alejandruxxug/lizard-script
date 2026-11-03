# Ejercicios

[Índice](../README.md) · [Hoja de referencia](cheatsheet.md) · [English](../en/exercises.md)

Veinte ejercicios, más o menos en el orden de los capítulos. Intenta cada uno antes de abrir la solución. Todas las soluciones se ejecutaron, y la salida que se muestra es la real. Hay muchas respuestas correctas; la tuya no tiene que coincidir.

---

### 1. Hola, tú (capítulos 1–2)

Guarda tu nombre y tu edad en variables e imprime `My name is Liz and I am 3 years old.`, con tus datos, como una sola cadena armada con `+`.

<details><summary>Solución</summary>

```
let name = "Liz"
let age = 3
puts("My name is " + name + " and I am " + str(age) + " years old.")
```

**Salida:**

```text
My name is Liz and I am 3 years old.
```

`age` es un número, así que necesita `str` antes de unirlo con `+`.
</details>

### 2. Segundos (capítulo 4)

¿Cuántos segundos tiene una semana? ¿Y cuántas semanas completas caben en 10.000.000 de segundos, y cuántos segundos sobran?

<details><summary>Solución</summary>

```
let week = 7 * 24 * 60 * 60
puts(week)
puts(10000000 // week, "weeks and", 10000000 % week, "seconds")
```

**Salida:**

```text
604800
16 weeks and 323200 seconds
```
</details>

### 3. Predice la salida (capítulo 4)

Sin ejecutarlo, anota qué imprime cada línea. Luego compruébalo.

```
puts(9 / 2, 9 / 2.0, -9 / 2, -9 // 2)
puts(2 ** 2 ** 3, -3 ** 2)
puts(1 == "1", 1 === "1", 0 == false, "" == 0)
```

<details><summary>Solución</summary>

```
puts(9 / 2, 9 / 2.0, -9 / 2, -9 // 2)
puts(2 ** 2 ** 3, -3 ** 2)
puts(1 == "1", 1 === "1", 0 == false, "" == 0)
```

**Salida:**

```text
4 4.5 -4 -5
256 -9
true false true false
```

`2 ** 2 ** 3` es `2 ** 8`. `-3 ** 2` es `-(3 ** 2)`. `""` no es una cadena numérica, así que nunca es `==` a un número.
</details>

### 4. Corrige los errores (capítulos 3 y 6)

Este programa debería imprimir la suma de 1 a 5, pero tiene dos errores. Encuéntralos a partir de los mensajes.

```
let total = 0
for (i = 1; i <= 5; i += 1) {
  totl += i
}
puts(total)
```

**Salida:**

```text
[line 2, column 6] runtime error: can't assign to i: it was never declared, use let i = ...
```

<details><summary>Solución</summary>

La cabecera del `for` necesita `let i`. Después de arreglar eso, el siguiente error sería `identifier not found: totl` (un error de tipeo).

```
let total = 0
for (let i = 1; i <= 5; i += 1) {
  total += i
}
puts(total)
```

**Salida:**

```text
15
```
</details>

### 5. Calificación (capítulo 5)

Escribe `grade(score)`, que devuelva `"A"` para 90 o más, `"B"` de 80 a 89, `"C"` de 70 a 79 y `"F"` en otro caso.

<details><summary>Solución</summary>

```
function grade(score) {
  if (score >= 90) { "A" } elif (score >= 80) { "B" } elif (score >= 70) { "C" } else { "F" }
}
puts(grade(95), grade(85), grade(72), grade(10))
```

**Salida:**

```text
A B C F
```

El `if` es la última instrucción, así que su valor es lo que devuelve la función.
</details>

### 6. Años bisiestos (capítulo 5)

Un año es bisiesto si es divisible entre 4, excepto los divisibles entre 100, a menos que también sean divisibles entre 400. Escribe `isLeap(year)`.

<details><summary>Solución</summary>

```
function isLeap(y) {
  (y % 4 == 0 && y % 100 != 0) || y % 400 == 0
}
puts(isLeap(2024), isLeap(1900), isLeap(2000), isLeap(2026))
```

**Salida:**

```text
true false true false
```
</details>

### 7. Cuenta regresiva con `while` (capítulo 6)

Imprime `10 9 8 ... 1 lift off!` en **una** sola línea.

<details><summary>Solución</summary>

```
let line = ""
let n = 10
while (n > 0) {
  line += str(n) + " "
  n -= 1
}
puts(line + "lift off!")
```

**Salida:**

```text
10 9 8 7 6 5 4 3 2 1 lift off!
```
</details>

### 8. Suma de los impares (capítulo 6)

Usando `continue`, suma los números impares del 1 al 99.

<details><summary>Solución</summary>

```
let sum = 0
for (let i = 1; i < 100; i += 1) {
  if (i % 2 == 0) { continue }
  sum += i
}
puts(sum)
```

**Salida:**

```text
2500
```
</details>

### 9. Primer divisor (capítulo 6)

Escribe `smallestDivisor(n)`, que devuelva el menor número mayor que 1 que divide a `n`, usando un ciclo con `break`. Úsalo para escribir `isPrime(n)`.

<details><summary>Solución</summary>

```
function smallestDivisor(n) {
  let d = 2
  while (d * d <= n) {
    if (n % d == 0) { break }
    d += 1
  }
  if (d * d > n) { n } else { d }
}
function isPrime(n) { n > 1 && smallestDivisor(n) == n }
puts(smallestDivisor(91), smallestDivisor(13))
let primes = []
for (let i = 1; i <= 30; i += 1) {
  if (isPrime(i)) { push(primes, i) }
}
puts(primes)
```

**Salida:**

```text
7 13
[2, 3, 5, 7, 11, 13, 17, 19, 23, 29]
```
</details>

### 10. Potencia, recursivamente (capítulo 7)

Escribe `power(base, exp)` sin usar `**`, con recursión. `exp` es un número entero de 0 en adelante.

<details><summary>Solución</summary>

```
function power(base, exp) {
  if (exp == 0) { return 1 }
  base * power(base, exp - 1)
}
puts(power(2, 10), power(3, 0), power(1.5, 2))
```

**Salida:**

```text
1024 1 2.25
```
</details>

### 11. Un saludo con valor por defecto (capítulo 7)

Escribe `greet(name, punctuation = "!")`, que devuelva `"Hello, Liz!"`, y que `greet("Liz", "?")` dé `"Hello, Liz?"`.

<details><summary>Solución</summary>

```
function greet(name, punctuation = "!") {
  "Hello, " + name + punctuation
}
puts(greet("Liz"), greet("Liz", "?"))
```

**Salida:**

```text
Hello, Liz! Hello, Liz?
```
</details>

### 12. Componer (capítulo 7)

Escribe `compose(f, g)`, que devuelva una función nueva que calcule `f(g(x))`.

<details><summary>Solución</summary>

```
function compose(f, g) {
  function(x) { f(g(x)) }
}
let addTen = function(x) { x + 10 }
let square = function(x) { x * x }
puts(compose(addTen, square)(3), compose(square, addTen)(3))
```

**Salida:**

```text
19 169
```
</details>

### 13. Un contador con pasos (capítulo 7)

Escribe `makeCounter(step = 1)`, que devuelva una función; cada llamada suma `step` y devuelve la cuenta nueva.

<details><summary>Solución</summary>

```
function makeCounter(step = 1) {
  let count = 0
  function() {
    count += step
  }
}
let byOne = makeCounter()
let byFive = makeCounter(5)
byOne()
byFive()
puts(byOne(), byFive(), byFive())
```

**Salida:**

```text
2 10 15
```

`count += step` es una expresión cuyo valor es la cuenta nueva, así que también es el resultado de la función.
</details>

### 14. Máximo y promedio (capítulo 8)

Escribe `maxOf(xs)` (sin `list.sort`) y `average(xs)`, que debe devolver un flotante.

<details><summary>Solución</summary>

```
function maxOf(xs) {
  let best = xs[0]
  for (let i = 1; i < len(xs); i += 1) {
    if (xs[i] > best) { best = xs[i] }
  }
  best
}
function average(xs) {
  let total = 0
  for (let i = 0; i < len(xs); i += 1) { total += xs[i] }
  float(total) / len(xs)
}
let xs = [4, 8, 15, 16, 23, 42]
puts(maxOf(xs), average(xs))
```

**Salida:**

```text
42 18.0
```
</details>

### 15. ¿Compartido o copiado? (capítulo 8)

Predice la salida y luego compruébala.

```
let a = [1, 2]
let b = a
let c = list.copy(a)
push(b, 3)
push(c, 4)
puts(a, b, c)
```

<details><summary>Solución</summary>

```
let a = [1, 2]
let b = a
let c = list.copy(a)
push(b, 3)
push(c, 4)
puts(a, b, c)
```

**Salida:**

```text
[1, 2, 3] [1, 2, 3] [1, 2, 4]
```

`b` es el mismo arreglo que `a`; `c` es una copia aparte.
</details>

### 16. Contar palabras (capítulo 9)

Escribe `countWords(words)`, que reciba un arreglo de cadenas y devuelva un hash con cuántas veces aparece cada una.

<details><summary>Solución</summary>

```
function countWords(words) {
  let counts = {}
  for (let i = 0; i < len(words); i += 1) {
    let w = words[i]
    if (counts[w] === null) { counts[w] = 0 }
    counts[w] += 1
  }
  counts
}
puts(countWords(["a", "b", "a", "c", "a"]))
```

**Salida:**

```text
{"a": 3, "b": 1, "c": 1}
```
</details>

### 17. Mayúscula inicial (capítulo 10)

Escribe `capitalize(s)`, que ponga la primera letra en mayúscula y el resto en minúscula. Luego `titleCase(s)`, que lo haga con cada palabra.

<details><summary>Solución</summary>

```
function capitalize(s) {
  text.upper(text.slice(s, 0, 1)) + text.lower(text.slice(s, 1, len(s)))
}
function titleCase(s) {
  let words = text.split(s, " ")
  let out = []
  for (let i = 0; i < len(words); i += 1) { push(out, capitalize(words[i])) }
  text.join(out, " ")
}
puts(capitalize("lIZARD"))
puts(titleCase("the lizard and the sun"))
```

**Salida:**

```text
Lizard
The Lizard And The Sun
```
</details>

### 18. Tabla de temperaturas (capítulo 11)

Imprime los grados Celsius de 0 a 100 de 25 en 25, con los Fahrenheit (`c * 9 / 5 + 32`) como flotante redondeado a un decimal.

<details><summary>Solución</summary>

```
for (let c = 0; c <= 100; c += 25) {
  let f = c * 9 / 5.0 + 32
  puts(c, "C =", math.round(f * 10) / 10.0, "F")
}
```

**Salida:**

```text
0 C = 32.0 F
25 C = 77.0 F
50 C = 122.0 F
75 C = 167.0 F
100 C = 212.0 F
```

Dividir entre `5.0` lo vuelve flotante. Para redondear a un decimal, multiplica por 10, redondea y divide entre `10.0`.
</details>

### 19. Punto medio y un color más claro (capítulo 12)

Escribe `midpoint(p, q)` y `lighter(c)`, que mueva cada canal de un color hasta la mitad del camino hacia 255.

<details><summary>Solución</summary>

```
function midpoint(p, q) { (p + q) / 2.0 }
function lighter(c) {
  rgb((c.r + 255) / 2.0, (c.g + 255) / 2.0, (c.b + 255) / 2.0)
}
puts(midpoint((0, 0), (10, 5)))
puts(lighter(#3fa34d), lighter(#000000))
```

**Salida:**

```text
(5.0, 2.5)
#9fd1a6 #808080
```

`rgb` redondea los canales flotantes al entero más cercano.
</details>

### 20. División segura (capítulo 14)

Escribe `safeDivide(a, b)`, que devuelva `a / b`, o la cadena `"undefined"` cuando la división falle, usando `try`/`catch`.

<details><summary>Solución</summary>

```
function safeDivide(a, b) {
  try { a / b } catch { "undefined" }
}
puts(safeDivide(10, 2), safeDivide(1, 0), safeDivide(1.0, 0.0))
```

**Salida:**

```text
5 undefined undefined
```

El `try` es la última instrucción de la función, así que su valor es el resultado.
</details>

---

[Índice](../README.md) · [Hoja de referencia](cheatsheet.md)
