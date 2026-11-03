# 6. Ciclos

[← Anterior: Condicionales](05-conditionals.md) · [Índice](../README.md) · [Siguiente: Funciones →](07-functions.md) · [English](../en/06-loops.md)

Un ciclo repite un bloque. lizard-script tiene dos: `while` y `for`, además de `break` y `continue` para controlarlos.

## `while`

`while (condición) { cuerpo }` ejecuta el cuerpo una y otra vez mientras la condición sea verdadera. La condición se revisa **antes** de cada vuelta:

```
let countdown = 3
while (countdown > 0) {
  puts("countdown:", countdown)
  countdown -= 1
}
puts("lift off!")
```

**Salida:**

```text
countdown: 3
countdown: 2
countdown: 1
lift off!
```

Si la condición es falsa desde el principio, el cuerpo nunca se ejecuta.

## `for`

`for` pone las tres partes de un ciclo que cuenta en una sola línea: **inicio**; **condición**; **actualización**.

```
for (let i = 1; i <= 5; i += 1) {
  puts("round", i)
}
```

**Salida:**

```text
round 1
round 2
round 3
round 4
round 5
```

Funciona así: hace el inicio una vez; revisa la condición; si es verdadera, ejecuta el cuerpo, luego la actualización, y vuelve a revisar.

```
let total = 0
for (let i = 1; i <= 100; i += 1) {
  total += i
}
puts("1 + 2 + ... + 100 =", total)
```

**Salida:**

```text
1 + 2 + ... + 100 = 5050
```

En un `for` los paréntesis son obligatorios, y cualquiera de las tres partes puede quedar vacía: `for (;;) { ... }` se repite para siempre (hasta un `break`).

> **Error común: olvidar el `let` en la cabecera del `for`.**
>
> ```
> for (i = 0; i < 3; i += 1) {
>   puts(i)
> }
> ```
>
> **Salida:**
>
> ```text
> [line 1, column 6] runtime error: can't assign to i: it was never declared, use let i = ...
> ```
>
> `i = 0` cambia una variable, y todavía no hay ninguna `i`. Escribe `let i = 0`.

La variable creada en la cabecera solo existe dentro del ciclo. Después del ciclo, `i` ya no está:

```
for (let i = 0; i < 3; i += 1) { }
puts(i)
```

**Salida:**

```text
[line 2, column 6] runtime error: identifier not found: i
```

## Recorrer un arreglo

Lo usual es un contador de `0` a `len(lista) - 1`:

```
let pets = ["Liz", "Iggy", "Rex"]
for (let i = 0; i < len(pets); i += 1) {
  puts(i, pets[i])
}
```

**Salida:**

```text
0 Liz
1 Iggy
2 Rex
```

`range(n)` construye el arreglo `[0, 1, ..., n - 1]`, útil para cantidades pequeñas:

```
puts(range(5))
puts(range(2, 6))
```

**Salida:**

```text
[0, 1, 2, 3, 4]
[2, 3, 4, 5]
```

## `break`: salir del ciclo

`break` detiene de inmediato el ciclo más interno:

```
let n = 1
while (true) {
  if (n * n > 50) {
    break
  }
  n += 1
}
puts("the first n with n * n > 50 is", n)
```

**Salida:**

```text
the first n with n * n > 50 is 8
```

Un `break` fuera de un ciclo es un error de sintaxis, que se detecta antes de ejecutar el programa:

```
puts("hello")
break
```

**Salida:**

```text
[line 2, column 1] syntax error: break outside of a loop
```

## `continue`: pasar a la siguiente vuelta

`continue` se salta el resto de la vuelta actual. En un `for` la actualización igual se ejecuta, así que el contador sigue avanzando:

```
for (let i = 1; i <= 10; i += 1) {
  if (i % 3 == 0) {
    continue
  }
  puts(i)
}
```

**Salida:**

```text
1
2
4
5
7
8
10
```

> **Cuidado con `continue` dentro de un `while`.** `continue` salta directo a la condición. Si la actualización está después del `continue`, nunca ocurre y el ciclo nunca termina. Pon la actualización primero:
>
> ```
> let i = 0
> while (i < 6) {
>   i += 1
>   if (i % 2 == 0) { continue }
>   puts(i)
> }
> ```
>
> **Salida:**
>
> ```text
> 1
> 3
> 5
> ```

`continue` solo es especial dentro de un ciclo. Fuera de uno es un nombre cualquiera, por eso el error sorprende un poco:

```
if (true) { continue }
```

**Salida:**

```text
[line 1, column 13] runtime error: identifier not found: continue
```

## Ciclos anidados

Un ciclo dentro de otro: el de adentro se ejecuta completo en cada vuelta del de afuera. `break` y `continue` solo afectan al ciclo más interno.

```
for (let row = 1; row <= 3; row += 1) {
  let line = ""
  for (let col = 1; col <= 4; col += 1) {
    line += str(row * col) + "\t"
  }
  puts(line)
}
```

**Salida:**

```text
1	2	3	4	
2	4	6	8	
3	6	9	12	
```

## Los ciclos no tienen valor

Un `if` da un valor, pero `while` y `for` siempre dan `null`. Para construir un resultado, ve juntándolo en una variable:

```
let squares = []
for (let i = 1; i <= 5; i += 1) {
  push(squares, i * i)
}
puts(squares)
```

**Salida:**

```text
[1, 4, 9, 16, 25]
```

## Ciclos infinitos

Un ciclo cuya condición nunca se vuelve falsa se ejecuta para siempre. En el REPL o en una terminal, detenlo con **Ctrl+C**. Si un programa parece congelado, revisa que la variable del ciclo de verdad cambie en cada vuelta.

> **Pruébalo.** Imprime la tabla del 7 (de `7 x 1 = 7` a `7 x 10 = 70`). Luego imprime solo las líneas cuyo resultado es par, usando `continue`.

## Resumen

- `while (cond) { }` revisa antes de cada vuelta.
- `for (let i = 0; i < n; i += 1) { }`: inicio, condición, actualización. No olvides el `let`.
- `break` sale del ciclo más interno; `continue` pasa a su siguiente vuelta.
- En un `while`, actualiza antes de cualquier `continue`.
- Los ciclos dan `null`; junta los resultados con `push`.

[← Anterior: Condicionales](05-conditionals.md) · [Índice](../README.md) · [Siguiente: Funciones →](07-functions.md)
