# 8. Arreglos

[← Anterior: Funciones](07-functions.md) · [Índice](../README.md) · [Siguiente: Hashes y acceso con punto →](09-hashes.md) · [English](../en/08-arrays.md)

Un arreglo es una lista ordenada de valores. Los valores pueden ser de cualquier tipo, incluso mezclados, e incluso otros arreglos.

## Crear y leer

```
let toes = [5, 5, 4, 5]
let mixed = [1, "two", true, [3, 4]]
puts(toes, len(toes))
puts(mixed)
puts(type(toes))
```

**Salida:**

```text
[5, 5, 4, 5] 4
[1, "two", true, [3, 4]]
ARRAY
```

Las cadenas dentro de un arreglo se muestran como se escriben en el código, entre comillas y con escapes, así que puedes pegar la salida de vuelta en un programa. Una cadena sola se muestra tal cual:

```
let quotes = ["say \"hi\"", "two\nlines", "back\\slash"]
puts(quotes)
puts("say \"hi\"")
```

**Salida:**

```text
["say \"hi\"", "two\nlines", "back\\slash"]
say "hi"
```

Los elementos se numeran desde **0**. `a[i]` lee el elemento `i`:

```
let pets = ["Liz", "Iggy", "Rex"]
puts(pets[0], pets[2])
puts(pets[len(pets) - 1])
```

**Salida:**

```text
Liz Rex
Rex
```

Una posición que no está en el arreglo da `null`, no un error. Los índices negativos no cuentan desde el final; simplemente quedan fuera del arreglo:

```
let pets = ["Liz", "Iggy", "Rex"]
puts(pets[3], pets[-1])
```

**Salida:**

```text
null null
```

Para obtener el último elemento usa `last(pets)` o `pets[len(pets) - 1]`. El índice tiene que ser entero:

```
let pets = ["Liz", "Iggy", "Rex"]
puts(pets[1.0])
```

**Salida:**

```text
[line 2, column 10] runtime error: array index must be INTEGER, got FLOAT
```

## Cambiar elementos

`a[i] = x` reemplaza un elemento que existe. La asignación compuesta también funciona:

```
let toes = [5, 5, 4, 5]
toes[2] = 5
toes[0] += 1
puts(toes)
```

**Salida:**

```text
[6, 5, 5, 5]
```

No puedes crear un elemento nuevo asignando más allá del final:

```
let toes = [5, 5]
toes[2] = 5
```

**Salida:**

```text
[line 2, column 5] runtime error: index out of range: 2, the array has 2 elements
```

Para agregar elementos, usa `push`.

## Agregar y quitar

| Función | Hace |
| --- | --- |
| `push(a, x)` | agrega `x` al final y devuelve `a` |
| `pop(a)` | quita el último elemento y lo devuelve |
| `first(a)`, `last(a)` | el primer o el último elemento, o `null` si está vacío |
| `rest(a)` | un arreglo **nuevo** sin el primer elemento |
| `len(a)` | la cantidad de elementos |

```
let stack = [1, 2, 3]
push(stack, 4)
puts(stack)
puts(pop(stack), stack)
puts(first(stack), last(stack), rest(stack))
puts(first([]), rest([]))
```

**Salida:**

```text
[1, 2, 3, 4]
4 [1, 2, 3]
1 3 [2, 3]
null null
```

`pop` sobre un arreglo vacío es un error:

```
let empty = []
pop(empty)
```

**Salida:**

```text
[line 2, column 4] runtime error: pop: the array is empty
```

`+` une dos arreglos en uno **nuevo**:

```
let a = [1, 2]
let b = a + [3]
puts(a, b)
```

**Salida:**

```text
[1, 2] [1, 2, 3]
```

## Los arreglos se comparten, no se copian

Esto sorprende a todo el mundo alguna vez. Después de `let b = a`, `a` y `b` son **el mismo arreglo**. Un cambio hecho a través de uno se ve a través del otro:

```
let a = [1, 2]
let b = a
push(b, 3)
puts(a)
```

**Salida:**

```text
[1, 2, 3]
```

Lo mismo pasa con los argumentos de una función: una función que recibe un arreglo puede cambiarlo.

```
function addZero(xs) {
  push(xs, 0)
}
let numbers = [7, 8]
addZero(numbers)
puts(numbers)
```

**Salida:**

```text
[7, 8, 0]
```

Cuando quieras un arreglo independiente, cópialo con `list.copy`:

```
let a = [1, 2]
let b = list.copy(a)
push(b, 3)
puts(a, b)
```

**Salida:**

```text
[1, 2] [1, 2, 3]
```

## Recorrer un arreglo

```
let prices = [12, 5, 30, 8]
let total = 0
for (let i = 0; i < len(prices); i += 1) {
  total += prices[i]
}
puts("total:", total)
```

**Salida:**

```text
total: 55
```

> **Error común: `<=` en lugar de `<`.** Con `i <= len(xs)`, la última vuelta lee `xs[len(xs)]`, que es `null`. Sumar `null` a un número da entonces un error de tipos:
>
> ```
> let prices = [12, 5]
> let total = 0
> for (let i = 0; i <= len(prices); i += 1) {
>   total += prices[i]
> }
> ```
>
> **Salida:**
>
> ```text
> [line 4, column 9] runtime error: type mismatch: INTEGER + NULL
> ```

## Arreglos de arreglos

Un arreglo puede contener arreglos, y así obtienes una cuadrícula:

```
let grid = [
  [1, 2, 3],
  [4, 5, 6]
]
puts(grid[1][0])
grid[0][2] = 30
puts(grid)
```

**Salida:**

```text
4
[[1, 2, 30], [4, 5, 6]]
```

## La biblioteca `list`

`list` trae más herramientas para arreglos. Se usan con un punto: `list.sort(a)`. Aquí un índice negativo sí cuenta desde el final.

```
let scores = [30, 10, 20]
puts(list.sort(scores))
puts(list.reverse(scores))
puts(scores)
```

**Salida:**

```text
[10, 20, 30]
[20, 10, 30]
[30, 10, 20]
```

`sort` y `reverse` crean arreglos **nuevos**; `scores` no cambió. `insert` y `remove` sí cambian el arreglo:

```
let letters = ["a", "c"]
list.insert(letters, 1, "b")
puts(letters)
list.remove(letters, 0)
puts(letters)
list.remove(letters, -1)
puts(letters)
```

**Salida:**

```text
["a", "b", "c"]
["b", "c"]
["b"]
```

`index` busca un elemento y `slice` corta un pedazo:

```
let colors = ["red", "green", "blue", "gray"]
puts(list.index(colors, "blue"), list.index(colors, "pink"))
puts(list.slice(colors, 1, 3))
puts(list.slice(colors, -2, 4))
```

**Salida:**

```text
2 -1
["green", "blue"]
["blue", "gray"]
```

### Ordenar a tu manera

Sin segundo argumento, `list.sort` ordena números por valor o cadenas alfabéticamente (todos los elementos deben ser de una misma clase). Para elegir tú el orden, pasa una función `less(x, y)` que diga si `x` va antes que `y`:

```
let nums = [5, 3, 8, 1]
puts(list.sort(nums, function(x, y) { x > y }))

let lizards = [{"name": "Iggy", "age": 7}, {"name": "Liz", "age": 3}]
let byAge = list.sort(lizards, function(p, q) { p.age < q.age })
puts(byAge[0].name, byAge[1].name)
```

**Salida:**

```text
[8, 5, 3, 1]
Liz Iggy
```

(`p.age` es el acceso con punto a un hash, del capítulo 9.) Mezclar tipos sin una función es un error:

```
puts(list.sort([3, "a"]))
```

**Salida:**

```text
[line 1, column 15] runtime error: list.sort: without a comparison function the elements must be all numbers or all strings
```

> **Pruébalo.** Escribe `maxOf(xs)`, que devuelva el número más grande de un arreglo sin usar `list.sort`. Luego escribe `average(xs)`, que devuelva un flotante.

## Resumen

- `[a, b, c]`; los elementos empiezan en el índice 0; una posición que no existe da `null`.
- `push` agrega, `pop` quita, `len` cuenta.
- `a[i] = x` solo cambia un elemento que ya existe.
- Los arreglos se comparten: usa `list.copy` para tener uno independiente.
- `list.sort`, `list.reverse`, `list.slice` y `list.copy` crean arreglos nuevos; `list.insert` y `list.remove` cambian el arreglo.

[← Anterior: Funciones](07-functions.md) · [Índice](../README.md) · [Siguiente: Hashes y acceso con punto →](09-hashes.md)
