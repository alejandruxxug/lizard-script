# 16. Proyectos

[← Anterior: Referencia de funciones integradas](15-builtins-reference.md) · [Índice](../README.md) · [Hoja de referencia →](cheatsheet.md) · [English](../en/16-projects.md)

Cuatro programas pequeños que juntan todos los capítulos. Cada uno se construye por pasos: lee un paso, intenta escribir el siguiente por tu cuenta y luego compara. Guarda cada programa en un archivo y ejecútalo con `./lizardscript archivo.liz`.

## Proyecto 1: FizzBuzz

**Meta:** imprimir los números del 1 al 15, pero escribir `Fizz` para los múltiplos de 3, `Buzz` para los múltiplos de 5 y `FizzBuzz` para los múltiplos de ambos.

**Usa:** `for`, `%`, `if`/`elif`/`else` (capítulos 4 a 6).

El truco está en el orden de las revisiones: un múltiplo de 15 también es múltiplo de 3, así que hay que revisarlo **primero**.

```
for (let i = 1; i <= 15; i += 1) {
  if (i % 15 == 0) {
    puts("FizzBuzz")
  } elif (i % 3 == 0) {
    puts("Fizz")
  } elif (i % 5 == 0) {
    puts("Buzz")
  } else {
    puts(i)
  }
}
```

**Salida:**

```text
1
2
Fizz
4
Buzz
Fizz
7
8
Fizz
Buzz
11
Fizz
13
14
FizzBuzz
```

**Paso 2:** conviértelo en una función que **devuelva** la palabra en lugar de imprimirla, para poder reutilizarla y probarla. Luego arma toda la línea con `text.join`:

```
function fizzbuzz(n) {
  if (n % 15 == 0) { return "FizzBuzz" }
  if (n % 3 == 0) { return "Fizz" }
  if (n % 5 == 0) { return "Buzz" }
  str(n)
}

let words = []
for (let i = 1; i <= 15; i += 1) {
  push(words, fizzbuzz(i))
}
puts(text.join(words, " "))
```

**Salida:**

```text
1 2 Fizz 4 Buzz Fizz 7 8 Fizz Buzz 11 Fizz 13 14 FizzBuzz
```

**Para ir más allá:** deja que quien llama elija las reglas con un hash, como `{3: "Fizz", 5: "Buzz", 7: "Bazz"}`.

## Proyecto 2: Un contador de palabras

**Meta:** dado un texto, contar cuántas veces aparece cada palabra y mostrar las más comunes.

**Usa:** cadenas y `text` (capítulo 10), hashes (capítulo 9), `list.sort` con una función (capítulo 8).

**Paso 1: separar el texto en palabras limpias.** Pasa todo a minúsculas, convierte los signos de puntuación en espacios, separa por espacios y descarta los pedazos vacíos:

```
function words(s) {
  let clean = text.lower(s)
  let marks = [",", ".", "!", "?", ";", ":", "\n"]
  for (let i = 0; i < len(marks); i += 1) {
    clean = text.replace(clean, marks[i], " ")
  }
  let out = []
  let pieces = text.split(clean, " ")
  for (let i = 0; i < len(pieces); i += 1) {
    if (pieces[i] != "") { push(out, pieces[i]) }
  }
  out
}
puts(words("The lizard sees the sun. The sun, the sun!"))
```

**Salida:**

```text
["the", "lizard", "sees", "the", "sun", "the", "sun", "the", "sun"]
```

**Paso 2: contarlas en un hash.** Una clave que falta se lee como `null`, así que la primera vez que aparece una palabra le ponemos 1:

```
function count(ws) {
  let counts = {}
  for (let i = 0; i < len(ws); i += 1) {
    let w = ws[i]
    if (counts[w] === null) {
      counts[w] = 1
    } else {
      counts[w] += 1
    }
  }
  counts
}
puts(count(["sun", "lizard", "sun"]))
```

**Salida:**

```text
{"sun": 2, "lizard": 1}
```

**Paso 3: ordenar por cantidad e imprimir las primeras.** Un hash no tiene más orden que el de inserción, así que lo convertimos en un arreglo de registros `{"word", "n"}` y ordenamos ese arreglo:

```
function words(s) {
  let clean = text.lower(s)
  let marks = [",", ".", "!", "?", ";", ":", "\n"]
  for (let i = 0; i < len(marks); i += 1) {
    clean = text.replace(clean, marks[i], " ")
  }
  let out = []
  let pieces = text.split(clean, " ")
  for (let i = 0; i < len(pieces); i += 1) {
    if (pieces[i] != "") { push(out, pieces[i]) }
  }
  out
}

function count(ws) {
  let counts = {}
  for (let i = 0; i < len(ws); i += 1) {
    let w = ws[i]
    if (counts[w] === null) { counts[w] = 1 } else { counts[w] += 1 }
  }
  counts
}

function top(counts, n) {
  let rows = []
  let ks = keys(counts)
  for (let i = 0; i < len(ks); i += 1) {
    push(rows, {"word": ks[i], "n": counts[ks[i]]})
  }
  -~> primero las más frecuentes; si empatan, en orden alfabético
  let sorted = list.sort(rows, function(a, b) {
    if (a.n != b.n) { a.n > b.n } else { a.word < b.word }
  })
  list.slice(sorted, 0, n)
}

let poem = "The lizard sees the sun.
The sun sees the lizard.
And the lizard, warm, sleeps in the sun!"

let ws = words(poem)
puts("words:", len(ws))
let best = top(count(ws), 3)
for (let i = 0; i < len(best); i += 1) {
  let row = best[i]
  puts(row.word + ":", "*" * row.n, row.n)
}
```

**Salida:**

```text
words: 18
the: ****** 6
lizard: *** 3
sun: *** 3
```

**Para ir más allá:** ignora palabras cortas como "the", "and" o "in" con una lista de palabras vacías; o muestra el porcentaje de cada palabra (¡cuidado con la división entera!).

## Proyecto 3: Una pequeña cuenta bancaria con clausuras

**Meta:** una cuenta cuyo saldo solo puede cambiar a través de `deposit` y `withdraw`, que rechazan montos inválidos.

**Usa:** clausuras (capítulo 7), hashes de funciones (capítulo 9), `error` y `try`/`catch` (capítulo 14).

El saldo es una variable local de `makeAccount`. Las funciones del hash que se devuelve son clausuras que lo comparten, y nada más en el programa puede alcanzarlo.

```
function makeAccount(owner, balance = 0) {
  let history = []

  function deposit(amount) {
    if (amount <= 0) { error("deposit must be positive, got " + str(amount)) }
    balance += amount
    push(history, "+" + str(amount))
    balance
  }

  function withdraw(amount) {
    if (amount > balance) {
      error(owner + " can't withdraw " + str(amount) + ", balance is " + str(balance))
    }
    balance -= amount
    push(history, "-" + str(amount))
    balance
  }

  {
    "owner": owner,
    "deposit": deposit,
    "withdraw": withdraw,
    "balance": function() { balance },
    "history": function() { list.copy(history) }
  }
}

let liz = makeAccount("Liz", 100)
let iggy = makeAccount("Iggy")

liz.deposit(50)
liz.withdraw(30)
iggy.deposit(10)
puts(liz.owner, liz.balance(), liz.history())
puts(iggy.owner, iggy.balance(), iggy.history())

try {
  iggy.withdraw(500)
} catch (e) {
  puts("refused:", e.message)
}
try {
  liz.deposit(-5)
} catch (e) {
  puts("refused:", e.message)
}
puts("Liz still has", liz.balance())
```

**Salida:**

```text
Liz 120 ["+50", "-30"]
Iggy 10 ["+10"]
refused: Iggy can't withdraw 500, balance is 10
refused: deposit must be positive, got -5
Liz still has 120
```

Fíjate en que:

- `liz` e `iggy` tienen cada uno su propio `balance` y su propio `history`, porque cada llamada a `makeAccount` crea unos nuevos.
- `history` devuelve una **copia**, así que quien llama no puede cambiar el historial real haciendo `push` sobre ella.
- Un `withdraw` fallido llama a `error` **antes** de cambiar nada, así que el saldo queda correcto.

**Paso 2: una transferencia.** No debe perder dinero si la segunda mitad falla. Como `withdraw` revisa primero, lo hacemos primero:

```
function makeAccount(owner, balance = 0) {
  {
    "owner": owner,
    "balance": function() { balance },
    "deposit": function(amount) { balance += amount },
    "withdraw": function(amount) {
      if (amount > balance) { error(owner + " has only " + str(balance)) }
      balance -= amount
    }
  }
}

function transfer(from, to, amount) {
  from.withdraw(amount)
  to.deposit(amount)
}

let liz = makeAccount("Liz", 100)
let iggy = makeAccount("Iggy", 5)
transfer(liz, iggy, 40)
puts(liz.balance(), iggy.balance())
try {
  transfer(iggy, liz, 1000)
} catch (e) {
  puts("transfer failed:", e.message)
}
puts(liz.balance(), iggy.balance())
```

**Salida:**

```text
60 45
transfer failed: Iggy has only 45
60 45
```

**Para ir más allá:** agrega una función de intereses que multiplique el saldo por `1.05` (¿qué le pasa al tipo de `balance`?), o un límite de retiros por cuenta.

## Proyecto 4: Un dibujo en la terminal

**Meta:** describir una pequeña escena con colores, puntos y capas, y luego dibujarla como texto.

**Usa:** colores y puntos (capítulo 12), `math` y unidades (capítulo 11), capas (capítulo 13), arreglos y hashes.

**Paso 1: las figuras son hashes.** Cada figura tiene una clase, una posición, un color y el nombre del grupo al que pertenece. En una ejecución normal el programa no sabe en qué capa está, así que guardamos el nombre del grupo en una variable que cada capa actualiza.

**Paso 2: un lienzo.** El lienzo es un arreglo de filas, y cada fila un arreglo de caracteres. `plot` pone un carácter en un punto e ignora los puntos que quedan afuera.

**Paso 3: dibujar cada figura.** Un círculo se dibuja recorriéndolo con `math.cos` y `math.sin`; una línea, avanzando paso a paso de un punto al otro.

```
let W = 32
let H = 12
let shapes = []
let group = ""

function add(shape) {
  shape.group = group
  push(shapes, shape)
}
function circle(center, r, color, mark) {
  add({"kind": "circle", "at": center, "r": r, "color": color, "mark": mark})
}
function line(from, to, color, mark) {
  add({"kind": "line", "at": from, "to": to, "color": color, "mark": mark})
}

-~> La escena. Las capas nombran los grupos; las líneas con sangría les pertenecen.
layer sky
  group = "sky"
  circle((25, 3), 2, #ffcc00, "o")
layer ground
  group = "ground"
  line((0, 10), (31, 10), #3fa34d, "=")
  layer tree
    group = "tree"
    line((7, 9), (7, 7), #8b5a2b, "|")
    circle((7, 4), 2, hsl(120, 60, 30), "*")

-~> El lienzo.
let canvas = []
for (let y = 0; y < H; y += 1) {
  let row = []
  for (let x = 0; x < W; x += 1) { push(row, " ") }
  push(canvas, row)
}

function plot(p, mark) {
  let x = math.round(p.x)
  let y = math.round(p.y)
  if (x >= 0 && x < W && y >= 0 && y < H) {
    canvas[y][x] = mark
  }
}

function draw(s) {
  if (s.kind == "circle") {
    for (let i = 0; i < 24; i += 1) {
      let a = i * 1turn / 24
      plot(s.at + (2 * s.r * math.cos(a), s.r * math.sin(a)), s.mark)
    }
  } elif (s.kind == "line") {
    let steps = math.max(math.abs(s.to.x - s.at.x), math.abs(s.to.y - s.at.y))
    for (let i = 0; i <= steps; i += 1) {
      plot(s.at + (s.to - s.at) * (float(i) / steps), s.mark)
    }
  }
}

for (let i = 0; i < len(shapes); i += 1) { draw(shapes[i]) }

for (let y = 0; y < H; y += 1) {
  puts(text.join(canvas[y], ""))
}

-~> Una leyenda: una línea por figura.
for (let i = 0; i < len(shapes); i += 1) {
  let s = shapes[i]
  puts(s.mark, s.kind, "in", s.group, "at", s.at, "color", s.color)
}
```

**Salida:**

```text
                                
                       ooooo    
     *****           oo     oo  
   **     **         o       o  
   *       *         oo     oo  
   **     **           ooooo    
     *****                      
       |                        
       |                        
       |                        
================================
                                
o circle in sky at (25, 3) color #ffcc00
= line in ground at (0, 10) color #3fa34d
| line in tree at (7, 9) color #8b5a2b
* circle in tree at (7, 4) color #1f7a1f
```

Algunas cosas para notar:

- Los círculos se dibujan el doble de anchos que de altos (`2 * s.r` en x) porque un carácter de la terminal es más o menos el doble de alto que de ancho.
- `s.at + (s.to - s.at) * t` es la fórmula clásica para el punto que está a una fracción `t` del camino sobre una línea. Funciona directamente con puntos.
- `float(i) / steps` importa: con dos enteros, `i / steps` daría `0` en todos los pasos menos el último.
- La capa `tree` está dentro de `ground`, así que se dibuja después de la línea del suelo y encima de ella.

**Para ir más allá:** agrega una figura `rect(corner, size, color, mark)`; dibuja los rayos del sol con ocho `line` a `i * 45deg`; u ordena `shapes` con `list.sort` para que una clave `"z"` decida qué se dibuja encima.

[← Anterior: Referencia de funciones integradas](15-builtins-reference.md) · [Índice](../README.md) · [Hoja de referencia →](cheatsheet.md)
