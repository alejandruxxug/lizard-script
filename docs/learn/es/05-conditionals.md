# 5. Condicionales

[← Anterior: Operadores](04-operators.md) · [Índice](../README.md) · [Siguiente: Ciclos →](06-loops.md) · [English](../en/05-conditionals.md)

Un condicional le permite a un programa elegir qué hacer. lizard-script tiene `if`, `elif` (abreviatura de "else if", "si no, si") y `else`.

## `if`, `elif`, `else`

```
let temperature = 27

if (temperature < 15) {
  puts("too cold, the lizard hides")
} elif (temperature < 30) {
  puts("nice, the lizard sunbathes")
} else {
  puts("too hot, the lizard looks for shade")
}
```

**Salida:**

```text
nice, the lizard sunbathes
```

Las condiciones se revisan de arriba hacia abajo, y solo se ejecuta el **primer** bloque cuya condición sea verdadera. `elif` y `else` son opcionales, y puedes poner tantos `elif` como necesites.

Las llaves `{ }` siempre son obligatorias, aunque haya una sola instrucción. Los paréntesis alrededor de la condición son opcionales, así que `if temperature < 15 { ... }` también funciona; esta guía siempre los escribe porque se lee mejor.

## Error común: `else if`

Muchos lenguajes escriben `else if`. En lizard-script, después de `else` tiene que venir `{`:

```
let x = 3
if (x > 5) {
  puts("big")
} else if (x > 1) {
  puts("medium")
}
```

**Salida:**

```text
[line 4, column 8] syntax error: expected "{", got "if"
```

Escribe `elif`:

```
let x = 3
if (x > 5) {
  puts("big")
} elif (x > 1) {
  puts("medium")
}
```

**Salida:**

```text
medium
```

## Anidar

Un bloque puede contener otro `if`:

```
function describe(temperature) {
  if (temperature < 30) {
    if (temperature >= 25) {
      puts(temperature, "-> perfect")
    } else {
      puts(temperature, "-> fine")
    }
  } else {
    puts(temperature, "-> too hot")
  }
}
describe(20)
describe(27)
describe(35)
```

**Salida:**

```text
20 -> fine
27 -> perfect
35 -> too hot
```

Muchas veces puedes aplanar los `if` anidados con `&&`: `if (temperature >= 25 && temperature < 30)`.

## Cualquier valor sirve de condición

Recuerda los valores verdaderos y falsos del capítulo 2: `false`, `null`, `0`, `0.0`, `""`, `[]` y `{}` cuentan como falsos.

```
let name = ""
if (name) {
  puts("hello,", name)
} else {
  puts("you didn't tell me your name")
}

let items = []
if (!items) {
  puts("the list is empty")
}
```

**Salida:**

```text
you didn't tell me your name
the list is empty
```

## `if` da un valor

Un `if` también es una expresión: su valor es el del bloque que se ejecutó. Es práctico para elegir entre dos valores:

```
let toes = 4
let label = if (toes == 5) { "complete" } else { "missing toes" }
puts("the foot is", label)
```

**Salida:**

```text
the foot is missing toes
```

Si no se ejecuta ningún bloque (un `if` sin `else` cuya condición es falsa), el valor es `null`:

```
let x = if (false) { 1 }
puts(x)
```

**Salida:**

```text
null
```

## Condicionales en el REPL

En el REPL, cuando se cierra el bloque de un `if`, el indicador se queda en `🦎 ..` una línea más, por si la siguiente empieza con `elif` o `else`. Presiona Enter en una línea vacía para ejecutar el `if` solo:

```text
🦎 >> let x = 7
🦎 >> if (x > 5) {
🦎 ..   puts("big")
🦎 .. }
🦎 .. 
big
🦎 >> if (x > 10) { puts("huge") }
🦎 .. else { puts("not huge") }
not huge
🦎 >> exit
```

> **Pruébalo.** Escribe un programa que, dada una variable `n`, imprima `"negative"`, `"zero"` o `"positive"`. Luego conviértelo en una expresión: `let sign = if (...) { ... } elif ...`.

## Resumen

- `if (cond) { } elif (cond) { } else { }`: llaves siempre, paréntesis opcionales.
- Usa `elif`, no `else if`.
- Cualquier valor sirve de condición.
- Un `if` tiene valor: el del bloque que se ejecutó, o `null`.

[← Anterior: Operadores](04-operators.md) · [Índice](../README.md) · [Siguiente: Ciclos →](06-loops.md)
