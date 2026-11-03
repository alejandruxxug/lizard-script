# 9. Hashes y acceso con punto

[← Anterior: Arreglos](08-arrays.md) · [Índice](../README.md) · [Siguiente: Cadenas y `text` →](10-strings-and-text.md) · [English](../en/09-hashes.md)

Un hash (también llamado diccionario o mapa) guarda valores bajo **claves**. Donde un arreglo responde "¿cuál es el elemento número 2?", un hash responde "¿cuál es el `name`?".

## Crear y leer

```
let lizard = {"name": "Liz", "age": 3, "colors": ["green", "brown"]}
puts(lizard)
puts(lizard["name"], lizard["age"])
puts(lizard["colors"][1])
puts(len(lizard), type(lizard))
```

**Salida:**

```text
{"name": "Liz", "age": 3, "colors": ["green", "brown"]}
Liz 3
brown
3 HASH
```

Una clave que no está da `null`:

```
let lizard = {"name": "Liz"}
puts(lizard["wings"])
```

**Salida:**

```text
null
```

## Acceso con punto

Para claves de tipo cadena que parecen nombres, `h.clave` es exactamente lo mismo que `h["clave"]`, y se lee mucho mejor:

```
let lizard = {"name": "Liz", "pet": {"kind": "beetle"}}
puts(lizard.name)
puts(lizard.pet.kind)
puts(lizard.missing)
```

**Salida:**

```text
Liz
beetle
null
```

Usa corchetes cuando la clave está en una variable, tiene espacios o no es una cadena:

```
let lizard = {"name": "Liz", "favorite food": "crickets"}
let field = "name"
puts(lizard[field])
puts(lizard["favorite food"])
```

**Salida:**

```text
Liz
crickets
```

Después del punto tiene que venir un nombre. Un número o una palabra reservada es un error de sintaxis:

```
let h = {1: "one"}
puts(h.1)
```

**Salida:**

```text
[line 2, column 8] syntax error: expected a name after "."
```

## Agregar, cambiar y quitar

`h[clave] = valor` y `h.clave = valor` agregan la clave o reemplazan su valor. `delete(h, clave)` quita una clave y dice si estaba:

```
let lizard = {"name": "Liz", "age": 3}
lizard.age += 1
lizard.tail = true
lizard["favorite food"] = "crickets"
puts(lizard)
puts(delete(lizard, "tail"), delete(lizard, "wings"))
puts(lizard)
```

**Salida:**

```text
{"name": "Liz", "age": 4, "tail": true, "favorite food": "crickets"}
true false
{"name": "Liz", "age": 4, "favorite food": "crickets"}
```

> **Error común: `+=` sobre una clave que todavía no existe.** Una clave que falta se lee como `null`, y `null + 1` no es `1`:
>
> ```
> let counts = {}
> counts.apples += 1
> ```
>
> **Salida:**
>
> ```text
> [line 2, column 15] runtime error: type mismatch: NULL + INTEGER
> ```
>
> Créala primero, o revisa:
>
> ```
> let counts = {}
> if (counts.apples === null) { counts.apples = 0 }
> counts.apples += 1
> puts(counts)
> ```
>
> **Salida:**
>
> ```text
> {"apples": 1}
> ```

## Claves

Las claves pueden ser cadenas, enteros o booleanos. Los flotantes, arreglos, hashes, colores y puntos no pueden ser claves:

```
let h = {"a": 1, 2: "two", true: "yes"}
puts(h["a"], h[2], h[true])
let bad = {[1, 2]: "no"}
```

**Salida:**

```text
1 two yes
[line 3, column 11] runtime error: unusable as hash key: ARRAY
```

Las claves se comparan de forma exacta, así que `1` y `"1"` son claves distintas:

```
let h = {1: "the number"}
puts(h[1], h["1"])
```

**Salida:**

```text
the number null
```

## Recorrer un hash

`keys(h)` y `values(h)` dan arreglos, **en el orden en que se agregaron las claves**. Ese orden siempre es el mismo, así que la salida es predecible:

```
let stock = {"apples": 4, "pears": 0, "figs": 12}
puts(keys(stock))
puts(values(stock))

let names = keys(stock)
for (let i = 0; i < len(names); i += 1) {
  let k = names[i]
  puts(k, "->", stock[k])
}
```

**Salida:**

```text
["apples", "pears", "figs"]
[4, 0, 12]
apples -> 4
pears -> 0
figs -> 12
```

## Los hashes también se comparten

Igual que los arreglos, los hashes se comparten, no se copian. Una función que recibe un hash puede cambiarlo:

```
function birthday(l) {
  l.age += 1
}
let liz = {"name": "Liz", "age": 3}
birthday(liz)
puts(liz)
```

**Salida:**

```text
{"name": "Liz", "age": 4}
```

## Registros: arreglos de hashes

Un arreglo de hashes es la forma usual de guardar una tabla de cosas:

```
let lizards = [
  {"name": "Liz", "age": 3},
  {"name": "Iggy", "age": 7},
  {"name": "Rex", "age": 5}
]
let oldest = lizards[0]
for (let i = 1; i < len(lizards); i += 1) {
  if (lizards[i].age > oldest.age) { oldest = lizards[i] }
}
puts("the oldest is", oldest.name)
```

**Salida:**

```text
the oldest is Iggy
```

## Funciones dentro de hashes

Un hash puede guardar funciones, lo que te da algo parecido a un objeto con métodos:

```
let calc = {
  "double": function(x) { x * 2 },
  "square": function(x) { x * x }
}
puts(calc.double(5), calc.square(5))
```

**Salida:**

```text
10 25
```

Combinado con las clausuras (capítulo 7), esto permite mantener datos privados. Mira la cuenta bancaria del [capítulo 16](16-projects.md).

> **Pruébalo.** Escribe una función `countWords(words)` que reciba un arreglo de cadenas y devuelva un hash con cuántas veces aparece cada una. `countWords(["a", "b", "a"])` debería dar `{"a": 2, "b": 1}`.

## Resumen

- `{"clave": valor, ...}`; una clave que falta da `null`.
- `h.clave` es `h["clave"]`. Usa corchetes para claves en variables o con espacios.
- `h.clave = v` agrega o reemplaza; `delete(h, clave)` quita.
- Claves: cadenas, enteros, booleanos. `1` y `"1"` son distintas.
- `keys` y `values` conservan el orden en que se agregaron las claves.
- Los hashes se comparten, igual que los arreglos.

[← Anterior: Arreglos](08-arrays.md) · [Índice](../README.md) · [Siguiente: Cadenas y `text` →](10-strings-and-text.md)
