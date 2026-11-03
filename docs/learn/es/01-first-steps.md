# 1. Primeros pasos

[Índice](../README.md) · [Siguiente: Valores y tipos →](02-values-and-types.md) · [English](../en/01-first-steps.md)

En este capítulo vas a abrir el intérprete, escribir tus primeras líneas, ejecutar un programa desde un archivo y aprender a escribir comentarios.

## Dos formas de ejecutar código

lizard-script ejecuta código de dos maneras:

- **El REPL** (Read, Eval, Print, Loop: leer, evaluar, mostrar, repetir): escribes una línea, se ejecuta y ves el resultado de inmediato. Ideal para experimentar.
- **Un archivo**: escribes un programa completo en un archivo que termina en `.liz` y lo ejecutas. Ideal para cualquier cosa de más de unas pocas líneas.

Si todavía no compilaste el intérprete, mira [Cómo instalarlo y ejecutarlo](../README.md#cómo-instalarlo-y-ejecutarlo). De aquí en adelante el comando se llama `./lizardscript` (en Windows, `lizardscript.exe`).

## El REPL

Ejecuta el intérprete sin argumentos:

```sh
./lizardscript
```

Aparece un indicador con un lagarto, `🦎 >>`. Escribe una expresión y presiona Enter. El REPL muestra su valor:

```text
🦎 >> 1 + 2
3
🦎 >> "Liz" * 2
"LizLiz"
🦎 >> let toes = 20
🦎 >> toes - 1
19
🦎 >> exit
```

Fíjate en algunas cosas:

- `"Liz" * 2` muestra `"LizLiz"` **con comillas**. El REPL muestra los valores como los escribirías en el código, así puedes distinguir la cadena `"5"` del número `5`.
- `let toes = 20` no muestra nada. Crear una variable no tiene un valor que mostrar.
- Las variables se conservan entre líneas: `toes` sigue existiendo en la línea siguiente.
- `exit` sale. Ctrl+D (Ctrl+Z y luego Enter en Windows) también funciona.

Si una línea está incompleta, por ejemplo con un `(`, `[` o `{` abierto, el indicador cambia a `🦎 ..` y el REPL sigue leyendo hasta que lo cierres:

```text
🦎 >> function double(n) {
🦎 ..   n * 2
🦎 .. }
🦎 >> double(21)
42
🦎 >> exit
```

> **Pruébalo.** Abre el REPL y calcula cuántos segundos tiene un día (`24 * 60 * 60`).

## Imprimir con `puts`

`puts` escribe sus argumentos separados por espacios y luego salta de línea. Las cadenas se escriben sin comillas:

```
puts("Hello, lizard!")
puts("I have", 4, "legs and", 1, "tail")
puts()
puts("puts() alone wrote an empty line")
```

**Salida:**

```text
Hello, lizard!
I have 4 legs and 1 tail

puts() alone wrote an empty line
```

`puts` acepta cualquier cantidad de argumentos y cualquier tipo de valor. En el REPL, `puts` imprime su texto y el REPL no muestra nada más, porque `puts` en sí da `null` (ningún valor).

## Ejecutar un archivo

Crea un archivo llamado `hello.liz` con este contenido:

```
puts("Hello from a file")
puts("2 + 2 is", 2 + 2)
```

**Salida:**

```text
Hello from a file
2 + 2 is 4
```

y ejecútalo:

```sh
./lizardscript hello.liz
```

Un archivo **no** muestra el valor de cada línea como el REPL: solo aparece lo que imprimes con `puts`.

## Comentarios

Un comentario empieza con `-~>` y llega hasta el final de la línea. Parece un lagartito corriendo hacia la derecha. El intérprete lo ignora.

```
-~> Toda esta línea es un comentario.
puts("code runs")   -~> y esta parte también es un comentario
-~> puts("esto no se ejecuta")
```

**Salida:**

```text
code runs
```

¿Por qué no `//` como en muchos lenguajes? Porque en lizard-script `//` es un operador: la división entera (capítulo 4). Usarlo como comentario es un error común:

```
puts("hi") // a comment
```

**Salida:**

```text
[line 1, column 17] syntax error: expected end of statement, got identifier "comment"
```

El intérprete leyó `// a comment` como "dividir entre `a`" y luego se confundió con `comment`. Y `#` tampoco sirve, porque `#` inicia un color (capítulo 12):

```
puts("hi")
# a comment
```

**Salida:**

```text
[line 2, column 1] lexical error: unexpected character "#"
```

## Punto y coma y líneas

Una instrucción termina al final de su línea, así que el punto y coma es opcional. Puedes usarlo para poner varias instrucciones en una misma línea:

```
let a = 1; let b = 2; puts(a + b)
let c = 3
puts(c)
```

**Salida:**

```text
3
3
```

## Cómo leer los mensajes de error

Todo error dice **dónde** ocurrió (línea y columna) y **de qué tipo** es:

- **lexical error** (error léxico): un carácter que el lenguaje no conoce, como `@`;
- **syntax error** (error de sintaxis): las piezas están en el orden equivocado, como un `)` que falta;
- **runtime error** (error de ejecución): el programa estaba bien escrito pero algo salió mal al ejecutarlo, como dividir entre cero.

```
puts("before the mistake")
puts(10 / 0)
puts("never printed")
```

**Salida:**

```text
before the mistake
[line 2, column 9] runtime error: division by zero
```

La primera línea se ejecutó y el programa se detuvo en la línea 2, columna 9 (el `/`). Con un error de sintaxis no se ejecuta nada, ni siquiera la primera línea, porque todo el programa se revisa antes de empezar:

```
puts("this line never runs")
let total = (1 + 2
```

**Salida:**

```text
[line 3, column 1] syntax error: expected ")", got end of input
```

El capítulo 14 trata por completo de los errores.

## Resumen

- `./lizardscript` abre el REPL; `./lizardscript archivo.liz` ejecuta un archivo.
- `puts(a, b, ...)` imprime valores separados por espacios.
- Los comentarios empiezan con `-~>`. Ni `//` ni `#`.
- El `;` es opcional al final de una línea.
- Los errores se ven así: `[line L, column C] kind error: message`.

[Índice](../README.md) · [Siguiente: Valores y tipos →](02-values-and-types.md)
