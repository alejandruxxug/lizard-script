# Learn lizard-script 🦎 / Aprende lizard-script 🦎

A teaching guide to lizard-script, a small interpreted language built as the Compilers course project at Universidad EIA. Use it in a classroom or on your own.

Una guía para aprender y enseñar lizard-script, un pequeño lenguaje interpretado hecho como proyecto del curso de Compiladores de la Universidad EIA. Sirve para clase o para estudiar por tu cuenta.

| Chapter | English | Español |
| --- | --- | --- |
| 1 | [First steps](en/01-first-steps.md) | [Primeros pasos](es/01-first-steps.md) |
| 2 | [Values and types](en/02-values-and-types.md) | [Valores y tipos](es/02-values-and-types.md) |
| 3 | [Variables and scope](en/03-variables-and-scope.md) | [Variables y alcance](es/03-variables-and-scope.md) |
| 4 | [Operators](en/04-operators.md) | [Operadores](es/04-operators.md) |
| 5 | [Conditionals](en/05-conditionals.md) | [Condicionales](es/05-conditionals.md) |
| 6 | [Loops](en/06-loops.md) | [Ciclos](es/06-loops.md) |
| 7 | [Functions](en/07-functions.md) | [Funciones](es/07-functions.md) |
| 8 | [Arrays](en/08-arrays.md) | [Arreglos](es/08-arrays.md) |
| 9 | [Hashes and dot access](en/09-hashes.md) | [Hashes y acceso con punto](es/09-hashes.md) |
| 10 | [Strings and `text`](en/10-strings-and-text.md) | [Cadenas y `text`](es/10-strings-and-text.md) |
| 11 | [Numbers: floats, `math` and units](en/11-numbers-and-math.md) | [Números: flotantes, `math` y unidades](es/11-numbers-and-math.md) |
| 12 | [Colors and points](en/12-colors-and-points.md) | [Colores y puntos](es/12-colors-and-points.md) |
| 13 | [Layers](en/13-layers.md) | [Capas](es/13-layers.md) |
| 14 | [Errors, `try`/`catch` and `null`](en/14-errors.md) | [Errores, `try`/`catch` y `null`](es/14-errors.md) |
| 15 | [Builtins reference](en/15-builtins-reference.md) | [Referencia de funciones integradas](es/15-builtins-reference.md) |
| 16 | [Projects](en/16-projects.md) | [Proyectos](es/16-projects.md) |
| 17 | [The Terrarium: files, JSON, images, windows](en/17-terrarium.md) | [El Terrario: archivos, JSON, imágenes, ventanas](es/17-terrarium.md) |
| | [Cheat sheet](en/cheatsheet.md) | [Hoja de referencia](es/cheatsheet.md) |
| | [Exercises](en/exercises.md) | [Ejercicios](es/exercises.md) |

---

## English

### Who it's for

- **Beginners** who have never programmed: start at chapter 1 and go in order. Chapters 1 to 9 cover what every first programming course covers.
- **Programmers who know another language**: skim chapters 1 to 4 for the surprises (comments are `-~>`, `7 / 2` is `3`, `let` before `=`, `elif`), then jump to whatever you need. The [cheat sheet](en/cheatsheet.md) fits on one page.
- **Teachers**: each chapter is about one class. Every chapter has "Try it" prompts for the room, common mistakes with the real error message, and a summary. The [exercises](en/exercises.md) have solutions in folded blocks, and the [projects](en/16-projects.md) work as lab sessions.

Every example that shows output was run with the interpreter, and the output is pasted as it came out.

### How to install and run

You need [Go](https://go.dev/dl/) 1.27 or newer (check with `go version`). The first `go build` downloads the few Go modules the window needs. On Linux, the window also needs the X11 and OpenGL development packages (see [Ebitengine's install guide](https://ebitengine.org/en/documents/install.html)); macOS and Windows need nothing else.

```sh
git clone https://github.com/alejandruxxug/lizard-script.git
cd lizard-script
go build
```

`go build` creates the `lizardscript` command (`lizardscript.exe` on Windows) in the project folder. You can also skip it and write `go run .` instead of `./lizardscript`.

**The REPL** (type code, see results right away): run it with no arguments.

```sh
./lizardscript
```

Type `exit` or press Ctrl+D (Ctrl+Z then Enter on Windows) to leave.

**A file**: write your program in a file ending in `.liz` and pass its path.

```sh
./lizardscript hello.liz
```

The `examples/` folder of the repository has one program per feature you can run the same way.

---

## Español

### Para quién es

- **Principiantes** que nunca han programado: empieza en el capítulo 1 y sigue en orden. Los capítulos 1 a 9 cubren lo que cubre cualquier primer curso de programación.
- **Programadores que ya conocen otro lenguaje**: lee por encima los capítulos 1 a 4 para encontrar las sorpresas (los comentarios son `-~>`, `7 / 2` da `3`, `let` antes de `=`, `elif`) y luego salta a lo que necesites. La [hoja de referencia](es/cheatsheet.md) cabe en una página.
- **Docentes**: cada capítulo da para una clase. Todos tienen propuestas de "Pruébalo" para el salón, errores comunes con el mensaje real del intérprete y un resumen. Los [ejercicios](es/exercises.md) traen las soluciones en bloques plegables, y los [proyectos](es/16-projects.md) sirven como sesiones de laboratorio.

Cada ejemplo que muestra una salida se ejecutó con el intérprete, y la salida está pegada tal como salió.

### Cómo instalarlo y ejecutarlo

Necesitas [Go](https://go.dev/dl/) 1.27 o más reciente (compruébalo con `go version`). El primer `go build` descarga los pocos módulos de Go que necesita la ventana. En Linux, la ventana también necesita los paquetes de desarrollo de X11 y OpenGL (mira la [guía de instalación de Ebitengine](https://ebitengine.org/en/documents/install.html)); macOS y Windows no necesitan nada más.

```sh
git clone https://github.com/alejandruxxug/lizard-script.git
cd lizard-script
go build
```

`go build` crea el comando `lizardscript` (`lizardscript.exe` en Windows) en la carpeta del proyecto. También puedes saltarte este paso y escribir `go run .` en lugar de `./lizardscript`.

**El REPL** (escribes código y ves el resultado de inmediato): ejecútalo sin argumentos.

```sh
./lizardscript
```

Escribe `exit` o presiona Ctrl+D (Ctrl+Z y luego Enter en Windows) para salir.

**Un archivo**: escribe tu programa en un archivo que termine en `.liz` y pasa su ruta.

```sh
./lizardscript hola.liz
```

La carpeta `examples/` del repositorio tiene un programa por cada característica, que puedes ejecutar de la misma forma.

---

lizard-script was made by Alejandro Urrego Giraldo, Maria José Sosa Fernández and Samuel Buelvas Cabrales. / lizard-script fue hecho por Alejandro Urrego Giraldo, Maria José Sosa Fernández y Samuel Buelvas Cabrales.
