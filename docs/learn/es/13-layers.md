# 13. Capas

[← Anterior: Colores y puntos](12-colors-and-points.md) · [Índice](../README.md) · [Siguiente: Errores, `try`/`catch` y `null` →](14-errors.md) · [English](../en/13-layers.md)

Una **capa** (`layer`) le pone nombre a un grupo de instrucciones. Es la única parte de lizard-script que usa la **sangría** en lugar de llaves.

## Escribir una capa

`layer` y un nombre, solos en una línea. El **cuerpo** son todas las líneas siguientes que tengan más sangría que la palabra `layer`:

```
let picture = []

layer sky
  push(picture, "sun")
  push(picture, "cloud")

layer ground
  push(picture, "grass")

puts(picture)
```

**Salida:**

```text
["sun", "cloud", "grass"]
```

La primera línea que vuelve a la columna de `layer` (aquí `layer ground`, y luego `puts`) termina el cuerpo. El nombre puede ser una palabra, o una cadena cuando tiene espacios:

```
layer "tall grass"
  puts("swaying")
```

**Salida:**

```text
swaying
```

## Las capas se anidan

Un `layer` dentro de un cuerpo es una capa hija. Cada cuerpo se mide desde la columna de su propia palabra `layer`:

```
let picture = []
layer sky
  push(picture, "sun")
  layer clouds
    push(picture, "cloud 1")
    push(picture, "cloud 2")
  push(picture, "birds")
puts(picture)
```

**Salida:**

```text
["sun", "cloud 1", "cloud 2", "birds"]
```

`"birds"` vuelve a la columna de `layer clouds`, así que pertenece a `sky`, no a `clouds`.

## Qué hace una capa al ejecutarse

Una capa ejecuta su cuerpo **una vez**, justo donde está, como un bloque con su propio alcance. Así que funciona igual que el código sin la línea `layer`, salvo que el `let` de adentro se queda adentro:

```
let count = 0
layer counting
  let step = 5
  count += step
puts(count)
puts(step)
```

**Salida:**

```text
5
[line 6, column 6] runtime error: identifier not found: step
```

`count += step` cambió la variable de afuera; `step` no sobrevivió a la capa.

## Para qué existen las capas

Por sí sola, una capa no cambia nada de lo que calcula un programa. Las capas son **nombres para el programa que ejecuta tu código**. A un editor o una herramienta de dibujo que incluya lizard-script se le puede avisar cuándo empieza y termina cada capa, para que etiquete todo lo que se crea adentro, por ejemplo para mostrar "sky" y "ground" como grupos separados que se pueden ocultar o seleccionar. En una ejecución normal con `./lizardscript`, son una forma ordenada de dividir programas largos en secciones con nombre.

## Capas dentro de funciones y ciclos

Las capas también funcionan dentro de llaves, medidas desde su propia palabra `layer`. Un `return` dentro de una capa sale de la función que la rodea:

```
function draw(n) {
  layer body
    puts("drawing", n)
    if (n > 1) { return "big" }
  "small"
}
puts(draw(1))
puts(draw(2))
```

**Salida:**

```text
drawing 1
small
drawing 2
big
```

## Reglas y errores

El nombre tiene que ir solo en su línea (se permite un comentario `-~>`):

```
layer sky blue
  puts("x")
```

**Salida:**

```text
[line 1, column 11] syntax error: a layer is "layer name" alone on its line, got identifier "blue" after the name
```

`layer` no es una palabra reservada: puedes seguir teniendo una variable llamada `layer`. Solo inicia una capa cuando la sigue un nombre y nada más:

```
let layer = 1
puts(layer + 1)
```

**Salida:**

```text
2
```

Usa espacios para la sangría; lo habitual son dos por nivel. Un tabulador cuenta como una columna, así que mezclar tabuladores y espacios hace que el cuerpo sea difícil de predecir.

> **Las capas no funcionan en el REPL.** El REPL ejecuta una línea a la vez, así que un `layer` escrito ahí tiene el cuerpo vacío y las líneas con sangría se ejecutan por su cuenta. Usa las capas en archivos.

> **Pruébalo.** Escribe un programa con las capas `background`, `house` (con las capas hijas `walls` y `roof`) y `garden`, donde cada capa agregue los nombres de sus partes a un arreglo. Imprime el arreglo al final.

## Resumen

- `layer nombre` solo en una línea, y luego un cuerpo con sangría.
- El cuerpo termina en la primera línea que vuelve a la columna de `layer`.
- Las capas se anidan, tienen su propio alcance y se ejecutan una vez.
- Son nombres para los programas anfitriones; en una ejecución normal organizan el código.
- Solo en archivos, no en el REPL.

[← Anterior: Colores y puntos](12-colors-and-points.md) · [Índice](../README.md) · [Siguiente: Errores, `try`/`catch` y `null` →](14-errors.md)
