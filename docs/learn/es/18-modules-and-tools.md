# 18. Módulos y herramientas

[← Anterior: El Terrario](17-terrarium.md) · [Índice](../README.md) · [Siguiente: lily2d →](19-lily2d.md) · [English](../en/18-modules-and-tools.md)

Un programa pronto deja de caber en un archivo. Este capítulo lo divide en **módulos** con `use` y luego muestra las herramientas para un programa de varios archivos: `lizardscript run`, `lizardscript test`, `lizardscript build`, que hace de él un solo ejecutable, y `lizardscript fmt` y `lizardscript check`, que ordenan el código y encuentran errores antes de que corra. Termina con dos módulos del Terrario para programas que trabajan con código y con otros programas: `lizard` (el lenguaje mismo, como datos) e `io`.

Los ejemplos viven en una carpeta llamada `garden`:

```text
garden/
  main.liz
  plants/
    flowers.liz
    watering.liz
```

## `use`: nombres de otro archivo

`plants/flowers.liz` es un archivo de lizard-script como cualquier otro:

```
-~> plants/flowers.liz
let petals = 5

function flower(name) {
  return name + " with " + str(petals) + " petals"
}
```

`main.liz` lo usa:

```
use "plants/flowers"

puts(flower("daisy"))
puts(petals)
```

Ejecuta la carpeta con `lizardscript run garden` (ejecuta `garden/main.liz`), o el archivo mismo con `lizardscript garden/main.liz`.

**Salida:**

```text
daisy with 5 petals
5
```

- `use "plants/flowers"` ejecuta el archivo `plants/flowers.liz` y deja que este archivo vea cada nombre que define en su nivel superior: `flower` y `petals`. La ruta usa `/` en todos los sistemas, y el `.liz` se agrega solo.
- La ruta es **relativa al archivo que dice el `use`**, no a la carpeta desde donde ejecutas el programa. Un módulo puede usar otro que está a su lado: dentro de `plants/`, `use "watering"` es `plants/watering.liz`, y `use "../main_helpers"` sube una carpeta.
- Cuando el archivo no está a su lado, lizardscript busca en las carpetas de la variable de entorno `LIZARD_PATH` (separadas por `:`, o `;` en Windows), así una biblioteca puede vivir en un solo lugar para todos tus programas.
- Un `use` va en el **nivel superior** de un archivo: no dentro de una función, un `if`, un ciclo o una capa.

### Los nombres están vivos

`plants/watering.liz` guarda una variable:

```
-~> plants/watering.liz
let water = 0
let _log = []

function pour(litres) {
  water += litres
  _log = _log + [litres]
  return water
}

function history() { return _log }
```

```
use "plants/watering"

pour(2)
pour(3)
puts(water)
water = 100
puts(pour(1))
puts(history())
let water = 7
puts(water, pour(1))
```

**Salida:**

```text
5
101
[2, 3, 1]
7 102
```

- `water` no es una copia: leerla lee la variable del propio módulo, así que después de dos `pour` vale `5`.
- `water = 100` también cambia la variable del módulo (por eso el siguiente `pour(1)` da `101`). Todos los archivos que usan el módulo ven el mismo `water`.
- Un `let` en tu archivo crea un nombre tuyo, que **oculta** el del módulo: después de `let water = 7`, `water` es tuyo, mientras `pour` sigue trabajando con el del módulo (`102`).
- Un módulo corre en un **alcance propio**: no ve las variables de los archivos que lo usan, así que una biblioteca no se puede romper porque tu programa tenga una variable con el mismo nombre que una de sus funciones auxiliares.

### Nombres privados

Un nombre que empieza con `_` se queda dentro de su módulo:

```
use "plants/watering"
puts(_log)
```

**Salida:**

```text
[line 2, column 6] runtime error: identifier not found: _log
```

Las funciones del propio módulo usan `_log` sin problema (`history()` lo devuelve). Empieza con `_` las funciones auxiliares de una biblioteca y solo se verán los nombres que quieres compartir.

### `as`: un módulo en un hash

Dos módulos pueden tener nombres que choquen, o puedes querer ver de dónde viene un nombre. `use "..." as nombre` pone los nombres del módulo en un hash:

```
use "plants/flowers" as flowers
use "plants/watering" as w

puts(keys(flowers))
puts(flowers.flower("rose"))
w.pour(4)
puts(w.water, w.pour(1))
```

**Salida:**

```text
["flower", "petals"]
rose with 5 petals
0 5
```

El hash se hace **una vez, justo cuando el módulo terminó de correr**. Sus funciones trabajan con las variables vivas del módulo (`w.pour(1)` da `5`), pero una variable leída a través del hash tiene el valor que tenía entonces: `w.water` sigue en `0`. Lee el estado que cambia a través de una función, o usa el módulo sin `as`.

### Una vez por programa

No importa cuántos archivos usen un módulo: su archivo corre una sola vez.

```
-~> plants/loud.liz
puts("loading loud.liz")
let loud = true
```

```
-~> plants/garden_bed.liz
use "loud"
let bed = "ready"
```

```
use "plants/loud"
use "plants/garden_bed"
use "plants/loud" as again
puts(bed, loud, again.loud)
```

**Salida:**

```text
loading loud.liz
ready true true
```

`garden_bed.liz` también usa `loud.liz`, y `again` es un tercer uso, pero la línea se imprime una vez. Los nombres no se pasan de mano en mano: `main.liz` ve `loud` porque él mismo usa `plants/loud`, no porque `garden_bed` lo use.

### Cuando algo sale mal

Un módulo que no existe:

```
use "plants/cactus"
```

**Salida:**

```text
[line 1, column 1] runtime error: can't find the module "plants/cactus": there is no plants/cactus.liz
```

Un error dentro de un módulo muestra el archivo del módulo adelante, para que sepas dónde buscar:

```
-~> plants/thirsty.liz
function drink(cups) {
  return cups * "water"
}
```

```
use "plants/thirsty"
puts(drink(2))
puts(drink("two"))
```

**Salida:**

```text
waterwater
plants/thirsty.liz: [line 2, column 15] runtime error: unknown operator: STRING * STRING
```

Un `use` dentro de una función es un error de sintaxis:

```
function setup() {
  use "plants/flowers"
}
```

**Salida:**

```text
[line 2, column 3] syntax error: use must be at the top level of a file, not inside a block or a layer
```

### Módulos que se usan entre sí

Dos módulos pueden usarse entre sí: un jardín que riega sus plantas, y una regadera que cuenta las plantas del jardín.

```
-~> plants/garden.liz
let plants = ["fern"]
use "water"
function plant(name) {
  plants = plants + [name]
  return water(name)
}
```

```
-~> plants/water.liz
use "garden"
let first = plants[0]
function water(name) {
  return "watered " + name + ", " + str(len(plants)) + " plants now"
}
```

```
use "plants/garden"
use "plants/water"
puts(first)
puts(plant("cactus"))
```

**Salida:**

```text
fern
watered cactus, 2 plants now
```

Cuando `garden.liz` llega a `use "water"`, todavía está cargando. Entonces corre `water.liz`, y su `use "garden"` no vuelve a empezar el jardín: lo ve **como va hasta ahora**, vivo. `plants` ya está (se define arriba del `use`), así que `first` es `"fern"`; `plant` todavía no, y aparece cuando el jardín sigue. Las funciones se llaman entre sí después, cuando todo ya cargó, así que encuentran todos los nombres.

Lo único que hay que cuidar: una línea que corre **mientras los archivos cargan** no puede usar un nombre que el otro archivo define más abajo.

```
-~> plants/early.liz
use "late"
let sun = 3
```

```
-~> plants/late.liz
use "early"
let light = sun * 2
```

```
use "plants/early"
```

**Salida:**

```text
plants/late.liz: [line 2, column 13] runtime error: identifier not found: sun: plants/early.liz defines it further down, but it is still loading (use cycle: plants/early.liz uses plants/late.liz, which uses plants/early.liz)
```

Sube `let sun = 3` arriba del `use "late"`, o lee `sun` dentro de una función, que corre después.

### Módulos estándar

Algunos módulos vienen con lizard-script: están dentro de `lizardscript`, así que cualquier programa puede usarlos, pero solo se cargan en los programas que los piden. Se llaman **módulos estándar**. Los primeros son la familia **flower** de bibliotecas gráficas, y la de 2D es `flower/lily2d` (el capítulo 19 trata de ella):

```
use "flower/lily2d" as lily2d

let picture = img.new(40, 20, #ffffff)
lily2d.circle(picture, (10, 10), 6, #2d6a8f)
puts(img.get(picture, 10, 10))
puts(lily2d.measure("hi"))
```

**Salida:**

```text
#2d6a8f
(14, 13)
```

- `use` busca un módulo estándar **al final**: después de la carpeta del archivo y de las carpetas de `LIZARD_PATH`. Así que si tu programa tiene su propio `flower/lily2d.liz`, gana ese.
- Ponle nombre con `as`. `use "flower/lily2d" as lily2d` da el hash `lily2d`; sirve cualquier nombre (`as g` da `g.circle`).
- El archivo de un módulo estándar se llama `std:/` y su ruta, y sus errores lo dicen. El mensaje igual empieza con la función que llamaste:

```
use "flower/lily2d" as lily2d

let picture = img.new(40, 20, #ffffff)
lily2d.circle(picture, (10, 10), "six", #2d6a8f)
```

**Salida:**

```text
std:/flower/lily2d/check.liz: [line 7, column 8] runtime error: lily2d.circle: r must be a number, not STRING
```

La línea es la del módulo, no la tuya: busca en tu programa la llamada que nombra el mensaje (`lily2d.circle`). La [referencia del Terrario](../../TERRARIUM.md#standard-modules) lista los módulos estándar.

## `lizardscript test`: programas que se revisan solos

Una prueba es un programa con un archivo `.expected` a su lado que contiene exactamente lo que debe imprimir (errores incluidos). Ponlas en una carpeta:

```text
garden/tests/
  flowers.liz        use "../plants/flowers"
                     puts(flower("tulip"))
  flowers.expected   tulip with 5 petals
  watering.liz       use "../plants/watering"
                     puts(pour(2), pour(2))
  watering.expected  2 4
```

```text
$ lizardscript test garden/tests
ok    garden/tests/flowers.liz (1.3 ms)
ok    garden/tests/watering.liz (0.8 ms)
2 passed, 0 failed
```

- Ejecuta cada `.liz` de la carpeta (y de las carpetas dentro de ella) que tenga un archivo `.expected`; un `.liz` sin él, como un módulo que comparten las pruebas, no es una prueba.
- Cuando la salida es distinta, muestra las dos y el comando termina con código 1, para que un script o un CI lo note. Con `watering.expected` diciendo `2 5`:

```text
FAIL  garden/tests/watering.liz
--- expected:
2 5
--- got:
2 4
1 passed, 1 failed
```

- `--update` escribe lo que imprime cada programa en su `.expected`. Úsalo para una prueba nueva, y lee el archivo antes de quedarte con él.
- `--vm` corre las pruebas en la máquina virtual de bytecode, que debe dar la misma salida.
- Un archivo que termina en `_bench.liz` es una medición: con `--bench` también corre y se muestra su tiempo (`bench garden/tests/count_bench.liz 27.0 ms`). Cada corrida tarda lo suyo, así que el tuyo será otro.

## `lizardscript build`: un solo archivo para compartir

`lizardscript build garden -o garden-app` hace **un solo archivo ejecutable** del programa: una copia de `lizardscript` con `main.liz`, cada módulo que usa y los archivos de datos que listes, adentro. Quien reciba el archivo lo ejecuta, sin instalar nada.

Los archivos de datos van en `garden/bundle.txt`, una ruta o patrón por línea (una carpeta incluye todo lo que tiene):

```text
-~> los archivos que lee el programa
notes
```

Un programa encuentra sus propios archivos desde `os.program_dir()`, la carpeta de su archivo principal:

```
use "plants/flowers"

puts(flower("daisy"))
puts(fs.read(fs.join(os.program_dir(), "notes/care.txt")))
```

```text
$ lizardscript build garden -o garden-app
built garden-app with main.liz, notes/care.txt, plants/flowers.liz
$ ./garden-app
daisy with 5 petals
Water on Mondays.

```

- Dentro del archivo construido, `os.program_dir()` es `"bundle:"` y sus archivos están en `bundle:/notes/care.txt` y así: `fs.read`, `fs.list`, `fs.exists` y `fs.info` los leen, pero nada los puede cambiar. El mismo programa funciona construido o no, siempre que encuentre sus archivos a través de `os.program_dir()`.
- `--os` y `--arch` construyen para otro sistema: `lizardscript build garden -o garden.exe --os windows --arch amd64`, o `--os linux --arch arm64`. Para eso lizardscript se construye a sí mismo para ese sistema con Go, así que necesita Go y su propio código fuente (ejecútalo dentro de la carpeta de lizard-script, o define `LIZARD_SRC`); `--runtime archivo` le da en cambio un lizardscript ya construido para ese sistema.
- `--icon icon.png` le da un icono al programa: un PNG cuadrado de 256 píxeles o más (lo mejor es 1024), y `--icon-small small.png` uno dibujado a mano para 16 píxeles. En Windows el icono queda dentro del `.exe`. En Linux se escriben `icon.png`, `icon-256.png` y una plantilla `.desktop` junto al programa. En macOS la salida es una carpeta de aplicación, así que llámala `.app`: `lizardscript build garden -o Garden.app --icon icon.png` hace `Garden.app` (con su icono, y firmada ad hoc si `codesign` está instalado). Los archivos del programa van dentro de la app; ábrela con `open Garden.app`. Dentro del programa, `window.open(w, h, título, {"icon": ruta})` pone el icono de la ventana.
- `--vm` hace que el programa corra en la máquina virtual de bytecode.
- Cada módulo que usa el programa debe estar dentro de su carpeta.

## `lizardscript fmt`: una sola forma de escribir el código

`lizardscript fmt` acomoda el código de una sola forma: el mismo programa sale igual, lo haya escrito quien lo haya escrito, así que nadie tiene que discutir por espacios. Este es `pond.liz`, escrito con prisa:

```
let frogs = [ "ana","leo" ]
function count(list)
{
    let total=0
    for(let i=0;i<len(list);i+=1){ total += 1 }
    return total
}
if (count(frogs) > 1) {
puts("a crowd") -~> more than one
}
else { puts("alone") } -~> just one
let pond = {"frogs": frogs, "lilies": 3,
            "deep": true}



puts(pond.lilies)
```

```text
$ lizardscript fmt pond.liz
let frogs = ["ana", "leo"]
function count(list) {
  let total = 0
  for (let i = 0; i < len(list); i += 1) { total += 1 }
  return total
}
if (count(frogs) > 1) {
  puts("a crowd")        -~> more than one
} else { puts("alone") } -~> just one
let pond = {
  "frogs": frogs,
  "lilies": 3,
  "deep": true
}

puts(pond.lilies)
```

- Dos espacios por cada bloque, un espacio alrededor de cada operador y después de cada coma, la `{` de un bloque en la línea de su `if` o su `function`, `} else {` en una línea, un hash que ocupa varias líneas con un par por línea, los comentarios del final de las líneas uno debajo del otro, y nunca dos líneas en blanco seguidas. [STYLE.md](../../STYLE.md) tiene todas las reglas (en inglés).
- Imprime el resultado. `-w` lo escribe de vuelta en los archivos, y `--check` solo lista los archivos que cambiarían y termina con código de salida 1 si hay alguno, para un script que revisa un proyecto. Una carpeta quiere decir cada `.liz` que hay en ella y en las carpetas de adentro: `lizardscript fmt -w garden`.
- Solo mueve espacios, y saltos de línea donde el parser lee un salto de línea como un espacio, así que el programa hace exactamente lo mismo que antes. No te parte una línea larga, y un bloque corto se queda en su línea (`if (a) { 1 }` y luego `else { 2 }` en la siguiente se quedan así).
- Un archivo con un error de sintaxis se deja como está: fmt muestra el error y termina con código de salida 2.

## `lizardscript check`: errores antes de correr

`lizardscript check` lee un programa sin ejecutarlo y te dice qué va a salir mal. `bugs.liz`:

```
let frogs = ["ana", "leo"]

function area(w, h) {
  let unit = "cm"
  return w * h
}

puts(area(3))
puts(len(forgs))
puts(math.sine(1))
if (len(frogs) == 2) { total = 5 }
```

```text
$ lizardscript check bugs.liz
bugs.liz: [line 4, column 7] warning: unit is never used (unused)
    hint: start its name with _ if that is on purpose
bugs.liz: [line 8, column 6] error: wrong number of arguments to area: want 2, got 1 (arguments)
bugs.liz: [line 9, column 10] error: identifier not found: forgs (unknown-name)
    hint: did you mean `frogs`?
bugs.liz: [line 10, column 6] error: math has no sine (member)
    hint: did you mean `math.sin`?
bugs.liz: [line 11, column 24] error: can't assign to total: it was never declared, use let total = ... (assign)
```

- Un **error** (`error`) es algo con lo que el programa se toparía al correr, o un error de sintaxis que no lo deja correr: un nombre que no está definido en ninguna parte, `=` a un nombre que no tiene `let`, una llamada con el número equivocado de argumentos, una función que un módulo no tiene, un `use` cuyo archivo no existe, un `return` cuyo valor está en la línea siguiente.
- Una **advertencia** (`warning`) es probablemente un error: una variable o un parámetro que nunca se usa (empieza su nombre con `_` para decir que es a propósito), un `let` que oculta una variable de un alcance de afuera, código después de un `return`, `break` o `continue` que nunca corre, `==` entre dos literales, `x = x`, un `use` del que no se usa ningún nombre.
- La palabra entre paréntesis es el nombre de la revisión. Termina con código de salida 1 cuando encuentra un error, para que un script se detenga ahí; si solo hay advertencias termina con 0. `--json` escribe lo que encontró como JSON, para un editor.
- Sabe cuándo ya corrió un `let`. Una función puede usar un nombre cuyo `let` viene más abajo, porque la función corre después; leer un nombre en la misma función antes de su `let` es un error: `identifier not found: x: its let, at line 4, comes later`.
- Lee los módulos que usa el programa, así que conoce sus nombres: `use "plants/flowers"` y luego `flowr("rose")` es un nombre desconocido con la pista `` did you mean `flower`? ``.
- Que no haya errores no quiere decir que el programa no pueda fallar: check no lo ejecuta, así que un tipo equivocado o un archivo que falta solo aparecen al correr.

Un programa que corre dentro de otro (una app que corre los bocetos de sus usuarios, por ejemplo) usa nombres que le da la app, y check no puede conocerlos:

```text
$ lizardscript check sketches
sketches/sky.liz: [line 1, column 1] error: identifier not found: canvas (unknown-name)
sketches/sky.liz: [line 1, column 13] error: identifier not found: brush (unknown-name)
    hint: did you mean `push`?
$ lizardscript check --names canvas,brush sketches
```

`--names` los da para una sola vez. Para decirlo una vez para toda una carpeta, pon en ella un `lizard-check.liz`: un solo hash, que check lee y nunca ejecuta, y que vale para los archivos de esa carpeta y de todas las carpetas que tiene adentro (gana el más cercano):

```
-~> Los nombres que la app de pintura le da a cada programa de esta carpeta.
{"names": ["canvas", "brush"]}
```

El editor (`lizardscript lsp`) también lo lee.

## Editores: `lizardscript lsp`

`lizardscript lsp` es un servidor de lenguaje: un editor lo inicia y le pregunta por el código mientras escribes. Muestra subrayado lo que encuentra `lizardscript check`, mientras escribes; formatea el archivo con `lizardscript fmt` (al guardar, si quieres); completa nombres, con lo que es cada uno (después de `math.` las funciones de `math`, después de `lily2d.` las del módulo, después de `config.` las llaves de tu hash); muestra la firma de una función y su comentario de documentación cuando el mouse está sobre su nombre, y cuánto vale un literal (`90deg` es `1.5708 rad`); salta a donde se declara un nombre, también en otro archivo; cambia el nombre de algo en todas partes donde significa lo mismo; y pone un selector de color en cada color.

```
-~> The area of a rectangle.
-~> w: its width
-~> h: its height
-~> example: area(3, 4)
function area(w, h = 1) {
  return w * h
}
```

El comentario encima de la función es lo que el editor muestra para `area`, leído como lo lee `lizard.docs`: las primeras líneas son el resumen, una línea `nombre: ...` explica un parámetro, y `example:` es un ejemplo. Escribirlos es como tus funciones se explican solas en cada archivo que las usa.

- **VS Code**: la extensión está en `editors/vscode`. `npm install` y `npm run package` ahí hacen `lizard-script.vsix`; se instala con `code --install-extension lizard-script.vsix`. Sus comandos corren el archivo (el botón ▷), lo corren en la VM, lo formatean y revisan toda la carpeta.
- **Zed**: la extensión está en `editors/zed`. En Zed, **zed: install dev extension** y elige esa carpeta.
- Las dos necesitan el comando `lizardscript` en tu `PATH` (o su ruta completa en sus ajustes). Cualquier otro editor con un cliente LSP (Neovim, Helix, ...) también puede iniciar `lizardscript lsp`: [editors/README.md](../../../editors/README.md) tiene los ajustes.

## `lizard`: el código como datos

El módulo `lizard` lee código de lizard-script como lo lee el lenguaje, así que un editor, un revisor o una herramienta para enseñar se pueden escribir en lizard-script.

### Tokens

```
let code = "let size = 10 * 2 -~> big"
for (let i = 0; i < len(lizard.tokens(code)); i += 1) {
  let t = lizard.tokens(code)[i]
  puts(t.kind, t.text, t.column)
}
```

**Salida:**

```text
keyword let 1
name size 5
operator = 10
number 10 12
operator * 15
number 2 17
comment -~> big 19
```

Cada token también tiene su `type` (como `"IDENT"`), su `value` y dónde termina (`end_line`, `end_column`). El `kind` es lo que usa un resaltador de sintaxis para colorear.

Un programa que recorre los tokens de un archivo grande los pide en columnas: un solo hash de listas, con las mismas claves y un elemento por token. Así no se hace un hash por cada token, lo que es unas cuatro veces más rápido en un archivo de 200.000 tokens:

```
let columns = lizard.tokens("let size = 10 * 2", {"columns": true})
puts(columns.kind)
puts(columns.text[1], columns.column[1], len(columns.type))
```

**Salida:**

```text
["keyword", "name", "operator", "number", "operator", "number"]
size 5 6
```

### El árbol de sintaxis

```
let tree = lizard.parse("let size = 10 * 2")
let let_node = tree.program.statements[0]
puts(let_node.kind, let_node.name.value)
let product = let_node.value
puts(product.kind, product.operator, product.left.value, product.right.value)
puts(product.column, product.end_column)
puts(lizard.parse("let = 1").errors)
```

**Salida:**

```text
let size
infix * 10 2
12 18
[{"kind": "syntax", "message": "let needs a name before the \"=\", like let x = 1", "line": 1, "column": 5}]
```

Cada nodo es un hash con su `kind`, sus campos y sus posiciones: `10 * 2` va de la columna 12 hasta justo antes de la 18. [TERRARIUM.md](../../TERRARIUM.md#lizard) (en inglés) lista cada tipo de nodo.

### Ejecutar código en una caja de arena

`lizard.run(code, options)` ejecuta código en un alcance nuevo y vacío: solo ve las funciones integradas y lo que le das en `globals`, y haga lo que haga, tu programa sigue como antes.

```
let user_code = "let area = w * h\nputs(\"area:\", area)"
let r = lizard.run(user_code, {"globals": {"w": 3, "h": 4}})
puts(r.output)
puts(r.globals.area, r.error)

let forever = lizard.run("while (true) {}", {"steps": 1000})
puts(forever.error.message)

let broken = lizard.run("let x = 1\nx + \"a\"", {"name": "sketch"})
puts(broken.error)

let lines = []
lizard.run("let a = 1\nlet b = a + 1", {"on_statement": function(line, column, source) { lines = lines + [line] }})
puts(lines)
```

**Salida:**

```text
area: 12

12 null
the program ran too long (more than 1000 steps)
{"kind": "runtime", "message": "type mismatch: INTEGER + STRING", "line": 2, "column": 3, "source": "sketch"}
[1, 2]
```

- Lo que imprime el código queda en `output` en vez de imprimirse, y sus nombres del nivel superior quedan en `globals` después de correr.
- `steps` detiene el código que correría para siempre. Un error no detiene tu programa: vuelve en `error`.
- `on_statement` se llama antes de cada instrucción, y `on_layer` cuando una capa empieza y termina: un editor los usa para saber qué línea hizo qué.
- El Terrario no está a menos que lo pases: `{"globals": {"fs": fs}}` le da `fs` al código.

Una biblioteca que el código debe usar viva va en `use`, no en `globals`. Un nombre dado en `globals` es una copia: cambiarlo cambia solo la copia del código. Un módulo en `use` es el módulo mismo, como si el código empezara con `use "plants/settings"`:

```
-~> plants/settings.liz
let pot_size = 2
function pot() { return "a pot of size " + str(pot_size) }
```

```
let r = lizard.run("pot_size = 5\npot()", {"name": "sketch.liz", "use": ["plants/settings"]})
puts(r.value)

let lines = []
lizard.run("use \"plants/settings\"\nlet p = pot()", {"name": "sketch.liz", "hook_source": "sketch.liz",
  "on_statement": function(line, column, source) { lines = lines + [line] }})
puts(lines)
```

**Salida:**

```text
a pot of size 5
[1, 2]
```

- Los módulos se buscan desde el `name` del código, como sus propias líneas `use`.
- `hook_source` llama a `on_statement` solo para las instrucciones de esa fuente: las dos líneas del código, no la línea de adentro de `pot`. Un editor que corre un dibujo en cada cambio se ahorra una llamada por cada instrucción de las bibliotecas.
- Correr las mismas bibliotecas una y otra vez es barato: un archivo de módulo se analiza (y se compila, en la VM) una sola vez mientras no cambie.

Un editor necesita dos cosas más: todos los errores de un texto que no se puede analizar, y las instrucciones del código mismo, no las de las funciones que llama:

```
let typos = lizard.run("let = 1\nlet ok = 2\nlet y = 5kg", {"name": "sketch"})
puts(typos.error.line, len(typos.errors))
for (let i = 0; i < len(typos.errors); i += 1) {
  puts(typos.errors[i].line, typos.errors[i].message)
}

let lines = []
let code = "function twice(n) {\n  return n * 2\n}\nlet a = twice(1)\nlet b = twice(a)"
lizard.run(code, {"hook_functions": false, "on_statement": function(line, _column, _source) { lines = lines + [line] }})
puts(lines)
```

**Salida:**

```text
1 2
1 let needs a name before the "=", like let x = 1
3 unknown unit "kg"
[1, 4, 5]
```

- `errors` tiene todos los errores de sintaxis, en orden; `error` es el primero. Cuando el código corrió y falló, `errors` tiene ese único error, y cuando funcionó, ninguno.
- Con `"hook_functions": false`, `on_statement` se llama solo para las instrucciones que están fuera de toda función: la línea 2, dentro de `twice`, corre dos veces pero nunca se avisa. Las instrucciones de una capa sí se avisan.

### Revisar código

`lizard.check(code)` encuentra errores en el código igual que `lizardscript check`, y los da como hashes, para un editor que los subraya:

```
let found = lizard.check("let size = 10\nputs(siz * 2)")
puts(len(found))
puts(found[0].severity, found[0].message)
puts(found[0].hint, found[0].line, found[0].column)
puts(lizard.check("draw(canvas)", "", ["draw", "canvas"]))
```

**Salida:**

```text
1
error identifier not found: siz
did you mean `size`? 2 6
[]
```

Cada hallazgo tiene también su `rule` y dónde termina (`end_line`, `end_column`). El código conoce los nombres del Terrario; el tercer argumento lista más nombres que el código tendrá al correr, como los que un editor les da a los programas que ejecuta.

### Comentarios de documentación

`lizard.docs(code)` lee las líneas `-~>` justo encima de cada función:

```
let library = "-~> A flower with its petals.
-~> name: what it is called
-~> example: flower(\"daisy\")
function flower(name, petals = 5) { name }"
let docs = lizard.docs(library)
puts(docs.flower.summary)
puts(docs.flower.params)
puts(docs.flower.example)
```

**Salida:**

```text
A flower with its petals.
[{"name": "name", "default": null, "doc": "what it is called"}, {"name": "petals", "default": "5", "doc": null}]
flower("daisy")
```

## `io`: hablar con otro programa

`io.read_line()` lee una línea de la entrada del programa (`null` cuando se acaba), `io.write(text)` imprime sin agregar un salto de línea, e `io.error(text)` escribe en la salida de errores, que queda aparte de la salida que lee otro programa:

```
let line = io.read_line()
while (line != null) {
  io.write(text.upper(line) + "!\n")
  line = io.read_line()
}
io.error("done\n")
```

```text
$ printf 'hello\nlizards\n' | lizardscript shout.liz
HELLO!
LIZARDS!
done
```

`done` fue a la salida de errores: con `2>/dev/null` al final del comando, solo quedan las dos líneas.

## `task`: trabajo en segundo plano

Un programa con ventana dibuja un cuadro muchas veces por segundo, así que un cálculo que tarda un segundo la congela. `task.start(module, function, args)` corre en cambio una función de un módulo en segundo plano, en un intérprete nuevo y propio, y devuelve enseguida un identificador de la tarea. El módulo se busca como en un `use`, desde el archivo que llama a `task.start`:

```
-~> count.liz: cuántos números menores que n son primos, lento a propósito.
function primes_below(n) {
  let count = 0
  for (let k = 2; k < n; k += 1) {
    let prime = true
    for (let d = 2; d * d <= k; d += 1) {
      if (k % d == 0) {
        prime = false
        break
      }
    }
    if (prime) {
      count += 1
    }
  }
  puts("counted up to " + str(n))
  count
}
```

```
let small = task.start("count", "primes_below", [1000])
let big = task.start("count", "primes_below", [200000])
puts(small.state())
puts(small.result(), big.result())
puts(big.state(), big.done())

let endless = task.start("count", "primes_below", [1000000000])
endless.cancel()
puts(endless.state())
try {
  endless.result()
} catch (e) {
  puts(e["message"])
}
try {
  task.start("count", "primes_below", [function() { 1 }])
} catch (e) {
  puts(e["message"])
}
```

```text
running
[task] counted up to 1000
[task] counted up to 200000
168 17984
done true
cancelled
task.result: the task was cancelled
task.start: args[0] is a FUNCTION, and only data crosses to and from a task (numbers, strings, booleans, null, lists, hashes, points and colors)
```

- `t.result()` espera a la tarea y da lo que devolvió su función. `t.done()` dice si ya terminó sin esperar, que es lo que el `tick` de una ventana pregunta en cada cuadro; `t.state()` es `"running"`, `"done"`, `"failed"` o `"cancelled"`.
- `t.cancel()` la detiene de inmediato, aunque esté en medio de un ciclo. Su `result()` es entonces un error.
- Un error dentro de la tarea sale por `t.result()`, con el archivo y la línea donde ocurrió.
- No se comparte nada: los argumentos y el resultado se copian, así que una tarea que cambia un hash cambia su propia copia. Solo pasan datos (números, textos, booleanos, `null`, listas, hashes, puntos y colores); una función es un error que dice dónde estaba. Una tarea que necesita una imagen recibe su ruta y la carga.
- Dentro de una tarea no hay `window`, `pen` ni `clip` (son el error `not available inside a task`), y lo que imprime va a la salida de errores con `[task] ` delante, así que el orden de esas líneas junto a las del programa puede cambiar de una ejecución a otra.
- Cada tarea es un proceso propio, así que varias corren a la vez en los procesadores de la máquina; empezar una cuesta más o menos lo que arrancar `lizardscript`, así que sirve para trabajo que tarda. Cuando el programa termina, sus tareas también.
- En un programa construido, un módulo que solo nombra `task.start` también debe usarse en alguna parte (`use "count" as count`), para que `lizardscript build` lo incluya.

## Resumen

- `use "ruta"` ejecuta un módulo una vez por programa, en su propio alcance, y deja que el archivo vea sus nombres, vivos: `=` cambia la variable del módulo, un `let` la oculta. Los nombres que empiezan con `_` son privados.
- `use "ruta" as nombre` da un hash con los nombres del módulo, hecho cuando terminó de correr.
- Las rutas son relativas al archivo que dice el `use`, luego a las carpetas de `LIZARD_PATH` y al final a los módulos estándar (`flower/lily2d`). Un `use` va en el nivel superior de un archivo.
- `lizardscript test carpeta` ejecuta cada programa que tiene un archivo `.expected` y compara; `--update`, `--vm`, `--bench`.
- `lizardscript build carpeta -o archivo` hace un ejecutable con los módulos y los archivos de `bundle.txt`; encuentra tus archivos con `os.program_dir()`. `--os` y `--arch` construyen para otros sistemas.
- `lizardscript fmt` acomoda el código de una sola forma (`-w` lo escribe, `--check` lista lo que cambiaría); `lizardscript check` encuentra errores y advertencias sin ejecutar el programa (`--json` para editores; `--names` o un `lizard-check.liz` para los nombres que da un anfitrión).
- `lizardscript lsp` es el servidor de lenguaje: lo inician las extensiones de VS Code y Zed de `editors/`, para diagnósticos, formato, completado, información al pasar el mouse, ir a la definición y un selector de color.
- `lizard.tokens`, `lizard.parse`, `lizard.check`, `lizard.run` y `lizard.docs` leen, revisan y ejecutan código; `io` lee líneas y escribe en la salida y en la salida de errores.
- `task.start(module, function, args)` corre una función en segundo plano, en un proceso propio; `done`, `state`, `result` y `cancel` la siguen, y solo pasan datos.

[← Anterior: El Terrario](17-terrarium.md) · [Índice](../README.md) · [Siguiente: lily2d →](19-lily2d.md)
