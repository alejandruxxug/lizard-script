# 17. El Terrario

[← Anterior: Proyectos](16-projects.md) · [Índice](../README.md) · [Siguiente: Módulos y herramientas →](18-modules-and-tools.md) · [English](../en/17-terrarium.md)

Hasta ahora cada programa vivía dentro del intérprete: calculaba cosas y las imprimía. El **Terrario** (*Terrarium*, donde viven los lagartos) es la biblioteca estándar de lizard-script, y le permite a un programa llegar al mundo que lo rodea: leer y escribir archivos, recibir sus argumentos, saber la hora, leer y escribir JSON, crear imágenes, e incluso abrir una ventana y dibujar en ella con el mouse o con un lápiz.

Este capítulo muestra los módulos uno por uno, con un ejemplo pequeño de cada uno. La referencia completa, con cada función y cada error, está en [TERRARIUM.md](../../TERRARIUM.md) (en inglés).

## Los módulos son hashes

Cada módulo es un hash de funciones integradas que se usa con acceso con punto, igual que `math`, `text` y `list` (capítulos 8, 10 y 11). Están en todo programa que ejecutes con `./lizardscript archivo.liz`; no hay que importar nada.

```
puts(type(fs), type(json), type(img))
puts(keys(time))
puts(fs.read)
```

**Salida:**

```text
HASH HASH HASH
["ms", "now", "sleep"]
builtin fs.read
```

| Módulo | Para |
| --- | --- |
| `fs` | archivos y carpetas |
| `os` | los argumentos del programa, el entorno, el sistema |
| `time` | relojes y esperas |
| `json` | leer y escribir JSON |
| `img` | imágenes |
| `clip` | el portapapeles: `clip.get()`, `clip.set(text)` |
| `window`, `pen` | una ventana para dibujar, su entrada, la presión del lápiz |

Son variables comunes de tu programa, así que un `let` con el mismo nombre reemplaza al módulo:

```
let fs = "my file system"
puts(fs)
puts(fs.read("notes.txt"))
```

**Salida:**

```text
my file system
[line 3, column 8] runtime error: string index must be INTEGER, got STRING
```

Ahora `fs` es una cadena, así que `fs.read` significa `fs["read"]`, y una cadena no se puede indexar con otra cadena. Usa otros nombres para tus variables.

## `fs`: archivos y carpetas

```
fs.mkdir("notes")
let path = fs.join("notes", "lizards.txt")
fs.write(path, "Liz likes sun\n")
fs.append(path, "Rex likes rocks\n")
puts(fs.read(path))
puts(fs.exists(path), fs.list("notes"))
```

**Salida:**

```text
Liz likes sun
Rex likes rocks

true ["lizards.txt"]
```

- `fs.write(path, text)` reemplaza el archivo completo (o lo crea); `fs.append(path, text)` agrega al final.
- `fs.read(path)` da todo el texto. La línea vacía de la salida es el último `\n` del archivo más el que agrega `puts`.
- `fs.join(a, b, ...)` arma una ruta con `/`, que funciona en todos los sistemas, también en Windows.
- `fs.mkdir` crea una carpeta (y las que falten encima de ella); `fs.list` da los nombres que hay adentro, en orden alfabético.
- `fs.digest(path)` es una huella del contenido del archivo (SHA-256, en hexadecimal), igual que `text.digest` de su texto: dos archivos con la misma huella tienen los mismos bytes, así que es un buen nombre para una miniatura o un caché del archivo. `fs.digest(path, "fnv")` es una más corta y rápida para cachés.

```
fs.write("note.txt", "abc")
puts(fs.digest("note.txt"))
puts(fs.digest("note.txt", "fnv"), text.digest("abc", "fnv"))
```

**Salida:**

```text
ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
e71fa2190541574b e71fa2190541574b
```

Cuando algo sale mal (un archivo que no existe, una carpeta que no está), la función lanza un error de ejecución, y `try` lo atrapa como a cualquier otro (capítulo 14):

```
try {
  fs.read("notes/missing.txt")
} catch (e) {
  puts(e.message)
}
```

**Salida:**

```text
fs.read: can't read notes/missing.txt: no such file or directory
```

`fs.remove(path)` borra un archivo o una carpeta **vacía**. Se niega a borrar una carpeta que tenga algo, para que un error no pueda borrar un árbol completo:

```
fs.mkdir("box")
fs.write("box/a.txt", "")
fs.remove("box")
```

**Salida:**

```text
[line 3, column 10] runtime error: fs.remove: can't remove box: the folder is not empty
```

Primero se borran los archivos y luego la carpeta:

```
fs.remove("notes/lizards.txt")
fs.remove("notes")
puts(fs.exists("notes"))
```

**Salida:**

```text
false
```

Mover y copiar nunca reemplazan un archivo que ya existe, así un error de tipeo no te hace perder tu trabajo:

```
fs.write("draft.txt", "Liz likes sun\n")
fs.copy("draft.txt", "backup.txt")
fs.rename("draft.txt", "final.txt")
puts(fs.list("."))
let info = fs.info("final.txt")
puts(info.size, info.dir)
puts(fs.list(".", {"info": true})[0].name)
try { fs.rename("backup.txt", "final.txt") } catch (e) { puts(e.message) }
```

**Salida** (en una carpeta que solo tiene este programa, `fsx.liz`):

```text
["backup.txt", "final.txt", "fsx.liz"]
14 false
backup.txt
fs.rename: can't move backup.txt to final.txt: there is already something there
```

`fs.info(path)` da el tamaño (`size`) en bytes, el `mtime` (cuándo cambió por última vez, en segundos desde 1970) y si es una carpeta (`dir`); `fs.list(dir, {"info": true})` da eso para cada entrada, con su `name`. `fs.trash(path)` mueve un archivo o carpeta a la Papelera, de donde el usuario lo puede recuperar, lo cual es más amable que `fs.remove` para algo que el usuario borró.

> **Pruébalo.** Escribe un programa que guarde un contador en `count.txt`: que lea el número (o empiece en 0 si el archivo no existe), le sume 1, lo vuelva a escribir y lo imprima. Ejecútalo tres veces. (`int(text)` convierte `"3"` en `3`; `fs.exists` te dice si ya hay un archivo.)

## `os`: argumentos y sistema

Lo que escribas después del nombre del archivo le llega al programa como `os.args()`, una lista de cadenas:

```
let args = os.args()
if (len(args) == 0) {
  puts("usage: ./lizardscript greet.liz name...")
  os.exit(2)
}
for (let i = 0; i < len(args); i += 1) {
  puts("hello,", args[i])
}
```

```sh
./lizardscript greet.liz Liz Rex
```

**Salida:**

```text
hello, Liz
hello, Rex
```

Sin nombres imprime la línea de uso y se detiene con `os.exit(2)`: el programa termina ahí mismo, y el número es el código de salida que ve la terminal (0 significa "todo salió bien").

Lo demás le dice al programa dónde se está ejecutando:

```
puts(os.platform(), os.locale())
puts(os.env("LIZARD_COLOR"))
```

**Salida** (en un Mac configurado en inglés; el tuyo puede decir `windows` o `linux`, y tu idioma, por ejemplo `es`):

```text
macos en
null
```

`os.env(name)` lee una variable de entorno, y da `null` cuando no está definida. Ejecútalo como `LIZARD_COLOR=green ./lizardscript archivo.liz` (macOS y Linux) y la segunda línea será `green`. `os.config_dir()` es la carpeta donde los programas guardan su configuración.

Otras más le preguntan al escritorio: `os.program_dir()` es la carpeta del archivo principal de tu programa (el capítulo 18 la usa para encontrar los archivos que vienen con un programa), `os.reveal(path)` muestra un archivo en Finder o el Explorador, `os.open_url("https://...")` abre una página web en el navegador, y `os.memory()` es la memoria de la máquina, `{"total": ..., "free": ...}` en bytes.

## `time`: relojes y esperas

```
let start = time.ms()
time.sleep(100)
let took = time.ms() - start
puts(type(took), took >= 100)

let now = time.now()
puts(keys(now))
puts(now.month >= 1 && now.month <= 12, now.weekday >= 1 && now.weekday <= 7)
puts(type(now.epoch), time.utc().utc_offset)
```

**Salida:**

```text
FLOAT true
["year", "month", "day", "hour", "minute", "second", "weekday", "epoch", "utc_offset"]
true true
FLOAT 0
```

- `time.ms()` son los milisegundos desde que empezó el programa, como flotante. Resta dos de ellos para medir cuánto tardó algo.
- `time.sleep(ms)` espera.
- `time.now()` es la fecha y la hora como un hash, así que `now.hour` y `now.minute` están listos para imprimir o para dibujar un reloj. `weekday` es 1 para el lunes hasta 7 para el domingo.
- `now.epoch` son los segundos desde 1970 (UTC), un flotante: el mismo instante en todos los computadores, así que es el que se guarda, y dos de ellos restados son los segundos entre ellos ("hace 2 h" es `int((time.now().epoch - saved) / 3600)`). `now.utc_offset` es cuántos segundos va adelantado el reloj local respecto a UTC (`-18000` en Bogotá).
- `time.utc()` es el mismo hash en UTC, con un `utc_offset` de 0.

> **Pruébalo.** Mide con `time.ms()` cuánto tarda `range(1000000)`. Luego imprime la hora como `14:05` (completa los minutos con un `"0"` cuando sean menos de 10).

## `json`: datos como texto

JSON es la forma más común en que los programas guardan datos y se comunican entre sí. `json.write` convierte un valor en texto JSON, y `json.parse` convierte el texto de vuelta:

```
let liz = {"name": "Liz", "toes": [5, 5, 5, 5], "likes": {"sun": true}}
let text = json.write(liz)
puts(text)
puts(json.write(liz, 2))

let back = json.parse(text)
puts(back.name, len(back.toes), back.likes.sun)
puts(back == liz)
```

**Salida:**

```text
{"name":"Liz","toes":[5,5,5,5],"likes":{"sun":true}}
{
  "name": "Liz",
  "toes": [
    5,
    5,
    5,
    5
  ],
  "likes": {
    "sun": true
  }
}
Liz 4 true
true
```

El segundo argumento de `json.write` es la sangría: `2` pone cada elemento en su propia línea, anidado con dos espacios, lo que es más fácil de leer en un archivo. Los hashes se vuelven objetos JSON, los arreglos se vuelven arreglos, y el viaje de ida y vuelta devuelve un valor igual.

`json` y `fs` juntos guardan y cargan la configuración de un programa. Un color se escribe como su texto:

```
let settings = {"brush": 8, "color": #2b3a55}
fs.write("settings.json", json.write(settings, 2))
let loaded = json.parse(fs.read("settings.json"))
puts(loaded)
fs.remove("settings.json")
```

**Salida:**

```text
{"brush": 8, "color": "#2b3a55"}
```

Un JSON mal escrito es un error que dice dónde está la equivocación, dentro del texto:

```
json.parse("{\"name\": \"Liz\",\n  \"toes\": 20,\n}")
```

**Salida:**

```text
[line 1, column 11] runtime error: json.parse: line 3, column 1: expected a string key in an object, got '}'
```

La primera posición es la llamada de tu programa; la segunda está en el texto JSON. JSON no permite una coma antes de la `}` que cierra.

> **Error común: esperar que el color vuelva.** JSON no tiene colores ni puntos, así que `loaded.color` en el ejemplo es la *cadena* `"#2b3a55"`, no un `COLOR`. `loaded.color.r` sería un error. Guarda los colores como `[r, g, b]` y vuelve a armarlos con `rgb(c[0], c[1], c[2])` al cargar.

## `img`: imágenes

Una imagen es una cuadrícula de píxeles de colores. `img.new(w, h, color)` crea una, `img.set` e `img.get` cambian y leen un píxel (`x` desde la izquierda, `y` desde arriba, ambos desde 0), y `puts` muestra su tamaño:

```
let picture = img.new(4, 2, #ffffff)
img.set(picture, 0, 0, #ff0000)
puts(picture, img.size(picture))
puts(img.get(picture, 0, 0), img.get(picture, 3, 1))
puts(img.pixels(picture))
```

**Salida:**

```text
<image 4x2> (4, 2)
#ff0000 #ffffff
[#ff0000, #ffffff, #ffffff, #ffffff, #ffffff, #ffffff, #ffffff, #ffffff]
```

`type(picture)` es `NATIVE`: un valor que el programa puede guardar y pasar de un lado a otro, pero no mirar por dentro. `img.pixels` da todos los colores en una sola lista plana, fila por fila.

Dos ciclos hacen un degradado, y `img.save` lo escribe como un PNG que puedes abrir con cualquier visor de imágenes:

```
let w = 256
let h = 64
let picture = img.new(w, h)
for (let y = 0; y < h; y += 1) {
  for (let x = 0; x < w; x += 1) {
    img.set(picture, x, y, rgb(x, 100, 255 - x))
  }
}
img.save(picture, "gradient.png")

let back = img.load("gradient.png")
puts(img.size(back), img.get(back, 0, 0), img.get(back, 255, 0))
puts(img.size(img.load("gradient.png", 32)))
```

**Salida:**

```text
(256, 64) #0064ff #ff6400
(32, 8)
```

`img.load(path)` lee archivos PNG, JPEG y GIF. Con un segundo argumento reduce la imagen para que quepa en esa cantidad de píxeles por su lado más largo, lo que sirve antes de recorrer una foto grande.

### Dibujar: `fill` y `span`

Poner los píxeles de uno en uno es lento para figuras grandes. Dos funciones pintan tramos enteros de píxeles a la vez, y **mezclan** el color con lo que ya hay, así que un color con alfa (capítulo 12) es transparente:

- `img.fill(image, [x, y, w, h], color)` pinta un rectángulo.
- `img.span(image, y, x0, x1, color)` pinta la fila `y` desde `x0` hasta `x1`. Un círculo, un triángulo o cualquier figura es un tramo por fila.

```
let picture = img.new(5, 3, #000000)
img.fill(picture, [1, 0, 3, 3], #ffffff)
img.span(picture, 1, 0, 5, #ff000080)
puts(img.pixels(picture))
```

**Salida:**

```text
[#000000, #ffffff, #ffffff, #ffffff, #000000, #800000, #ff7f7f, #ff7f7f, #ff7f7f, #800000, #000000, #ffffff, #ffffff, #ffffff, #000000]
```

La fila del medio es rojo semitransparente: sobre negro da un rojo oscuro (`#800000`), sobre blanco un rosado (`#ff7f7f`). Las otras funciones de dibujo son `img.clear(image, color)` (todos los píxeles, sin mezclar) e `img.blit(image, other, pos)` (pinta una imagen sobre otra). `blit` recibe opciones para pintarla de otro tamaño: `{"scale": 2}` la duplica con píxeles cuadrados nítidos, y `{"scale": 0.25, "filter": "linear"}` la achica suavemente, para una miniatura.

```
let tile = img.from_pixels((2, 1), [#000000, #ffffff])
let big = img.new(4, 1)
img.blit(big, tile, (0, 0), {"scale": 2})
puts(img.pixels(big))
img.blit(big, tile, (0, 0), {"scale": 2, "filter": "linear"})
puts(img.pixels(big))
let small = img.new(1, 1)
img.blit(small, tile, (0, 0), {"scale": 0.5, "filter": "linear"})
puts(img.pixels(small))
```

**Salida:**

```text
[#000000, #000000, #ffffff, #ffffff]
[#000000, #404040, #bfbfbf, #ffffff]
[#808080]
```


> **Error común: un píxel fuera de la imagen.** `img.get` e `img.set` solo aceptan píxeles que existen:
>
> ```
> let picture = img.new(4, 2)
> img.get(picture, 4, 0)
> ```
>
> **Salida:**
>
> ```text
> [line 2, column 8] runtime error: img.get: the pixel (4, 0) is outside the 4x2 image
> ```
>
> Una imagen de 4 píxeles de ancho tiene `x` de 0 a 3. (`fill` y `span` son más amables: lo que queda por fuera simplemente se recorta.)

> **Pruébalo.** Dibuja un círculo relleno de radio 20 en el centro de una imagen de 64x64 con un `img.span` por fila (para una fila a `dy` del centro, la mitad del ancho es `math.sqrt(r * r - dy * dy)`), y guárdalo como `circle.png`.

## `window` y `pen`: construye un programa de pintura

Una ventana es una imagen en la pantalla más la entrada que le llega. Es un lienzo de dibujo igual que una imagen, así que `window.fill`, `window.span`, `window.clear` y las demás funcionan de la misma forma. En esta sección construyes `examples/terrarium/paint.liz`, un pequeño programa de pintura, desde un archivo vacío, en siete pasos. Cada paso es un programa completo: guárdalo como `paint.liz`, ejecuta `./lizardscript paint.liz`, pruébalo y sigue. Ninguno imprime nada; todo pasa en la ventana.

### Paso 1: abrir una ventana

```
let w = window.open(600, 400, "paint 🦎")

window.run(w, function(events) {
})
```

`window.open(w, h, title)` crea una ventana de 600 por 400 píxeles. `window.run(w, tick)` la muestra y llama a `tick(events)` unas 60 veces por segundo, con la lista de lo que pasó desde la vez anterior, hasta que `tick` devuelva `false` o cierres la ventana. Este `tick` todavía no hace nada, así que ves una ventana negra; ciérrala para terminar el programa.

### Paso 2: el papel

```
let paper = #f4efe6
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

window.run(w, function(events) {
})
```

`window.clear(w, color)` pinta todos los píxeles. Se ejecuta una vez, antes de `window.run`, así que la ventana empieza con su primera imagen ya lista.

### Paso 3: un círculo donde hagas clic

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> Un círculo relleno, un tramo por fila.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, 8, ink)
    }
  }
})
```

Cada evento es un hash con un `type`. Un evento `"down"` dice que se presionó un botón, con `e.button` (`"left"`, `"right"` o `"middle"`) y `e.pos`, el punto donde pasó, en píxeles de la ventana desde la esquina de arriba a la izquierda. `circle` es el círculo del último "Pruébalo" de la sección de `img`, un tramo por fila; `window.span` funciona exactamente igual que `img.span`. Haz clic en varios lugares: cada clic deja un punto.

### Paso 4: seguir el arrastre

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> Un círculo relleno, un tramo por fila.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> Dónde va el arrastre, o null cuando no hay ningún botón presionado.
let last = null

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, 8, ink)
      last = e.pos
    } elif (e.type == "move" && last != null) {
      circle(e.pos, 8, ink)
      last = e.pos
    } elif (e.type == "up") {
      last = null
    }
  }
})
```

`last` vive fuera de `tick`, así que se recuerda de un cuadro al siguiente: es la última posición del arrastre, o `null` mientras no haya ningún botón presionado. Cada `"move"` durante el arrastre dibuja otro círculo, y `"up"` termina el arrastre. Arrastra rápido y mira de cerca: quedan huecos entre los círculos, porque una mano rápida avanza muchos píxeles entre dos cuadros.

### Paso 5: rellenar los huecos

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> Un círculo relleno, un tramo por fila.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> Un arrastre puede avanzar mucho en un cuadro: rellena el hueco con círculos.
function stroke(from, to, r) {
  let steps = math.max(1, math.ceil(math.dist(from, to) / (r / 2)))
  for (let i = 1; i <= steps; i += 1) {
    circle(from + (to - from) * (i / float(steps)), r, ink)
  }
}

-~> Dónde va el arrastre, o null cuando no hay ningún botón presionado.
let last = null

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, 8, ink)
      last = e.pos
    } elif (e.type == "move" && last != null) {
      stroke(last, e.pos, 8)
      last = e.pos
    } elif (e.type == "up") {
      last = null
    }
  }
})
```

`stroke(from, to, r)` dibuja círculos a lo largo de la línea que va de la última posición a la nueva, lo bastante juntos (cada medio radio) para que se toquen. `from + (to - from) * t` es el punto que está a una fracción `t` del camino (capítulo 12), y `i / float(steps)` importa: con dos enteros, `i / steps` daría `0` hasta el último paso. Ahora un arrastre rápido deja una línea continua.

### Paso 6: la presión

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> Un círculo relleno, un tramo por fila.
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> El tamaño del pincel para una presión de 0 a 1.
function brush(pressure) {
  2 + pressure * 14
}

-~> Un arrastre puede avanzar mucho en un cuadro: rellena el hueco con círculos.
function stroke(from, to, r) {
  let steps = math.max(1, math.ceil(math.dist(from, to) / (r / 2)))
  for (let i = 1; i <= steps; i += 1) {
    circle(from + (to - from) * (i / float(steps)), r, ink)
  }
}

-~> Dónde va el arrastre, o null cuando no hay ningún botón presionado.
let last = null

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, brush(e.pen.pressure), ink)
      last = e.pos
    } elif (e.type == "move" && last != null) {
      stroke(last, e.pos, brush(e.pen.pressure))
      last = e.pos
    } elif (e.type == "up") {
      last = null
    }
  }
})
```

Los eventos del puntero traen `e.pen`, y `e.pen.pressure` va de 0 a 1. `brush` la convierte en un radio de 2 a 16. Un mouse siempre dice 1, así que con mouse la línea simplemente es gruesa; con un trackpad Force Touch en un Mac, o una tableta con lápiz en macOS o Windows, presionar más fuerte dibuja una línea más ancha. `e.pen.source` dice cuál es (`"mouse"`, `"trackpad"` o `"tablet"`), y `pen.available()` dice si tu sistema puede reportar presión.

### Paso 7: las teclas

```
let paper = #f4efe6
let ink = #2b3a55
let w = window.open(600, 400, "paint 🦎")
window.clear(w, paper)

-~> Un círculo relleno, un tramo por fila. (/ de dos enteros es un
-~> número entero, así que las fracciones de abajo dividen entre un float.)
function circle(center, r, color) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    window.span(w, center.y + dy, center.x - half, center.x + half, color)
  }
}

-~> El tamaño del pincel para una presión de 0 a 1.
function brush(pressure) {
  2 + pressure * 14
}

-~> Un número redondeado a dos decimales, como texto.
function hundredths(x) {
  str(math.round(x * 100) / 100.0)
}

-~> Muestra la presión del lápiz en el título: "pressure 0.42 (raw 0.91) · trackpad".
function show(pen) {
  window.title(w, "pressure " + hundredths(pen.pressure) + " (raw " + hundredths(pen.raw) + ") · " + pen.source)
}

-~> Un arrastre puede avanzar mucho en un cuadro: rellena el hueco con círculos.
function stroke(from, to, r) {
  let steps = math.max(1, math.ceil(math.dist(from, to) / (r / 2)))
  for (let i = 1; i <= steps; i += 1) {
    circle(from + (to - from) * (i / float(steps)), r, ink)
  }
}

let last = null

window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "down" && e.button == "left") {
      circle(e.pos, brush(e.pen.pressure), ink)
      show(e.pen)
      last = e.pos
    } elif (e.type == "move" && last != null) {
      stroke(last, e.pos, brush(e.pen.pressure))
      show(e.pen)
      last = e.pos
    } elif (e.type == "up") {
      window.title(w, "paint 🦎")
      last = null
    } elif (e.type == "type" && e.text == "c") {
      window.clear(w, paper)
    } elif (e.type == "key" && e.key == "Escape") {
      return false
    }
  }
})
```

Un evento `"type"` es un carácter tal como lo compuso el teclado (`e.text`), lo indicado para las letras: `c` borra el papel. Un evento `"key"` es una tecla por su nombre (`e.key`: `"Escape"`, `"Enter"`, `"A"`, `"ArrowLeft"`...), lo indicado para las teclas que no son texto: Esc devuelve `false`, lo que termina `window.run` y el programa. Este es el código de `examples/terrarium/paint.liz`.

Mientras dibujas, el título de la ventana muestra `pressure 0.42 (raw 0.91) · trackpad`: `e.pen.raw` es lo que reportó el dispositivo y `e.pen.pressure` es ese valor tras la curva de respuesta del dispositivo, que es la que usa `brush`. Un trackpad necesita una pulsación firme para hacer clic, así que su curva viene atenuada por defecto; `pen.curve("trackpad", {"low": 0.8, "gamma": 1.5})` la cambia. Mira la sección `pen` de `docs/TERRARIUM.md`.

### Eventos

| `type` | otras claves | cuándo |
| --- | --- | --- |
| `"down"`, `"up"` | `pos`, `button`, `mods`, `pen` | un botón o el lápiz baja o sube |
| `"move"` | `pos`, `mods`, `pen` | el puntero se movió |
| `"scroll"` | `pos`, `dy`, `mods` | la rueda o un desplazamiento con dos dedos |
| `"drop"` | `pos`, `paths`, `mods` | se soltaron archivos sobre la ventana; `paths` son sus rutas absolutas |
| `"key"` | `key`, `mods` | se presionó una tecla |
| `"type"` | `text` | se escribió un carácter |
| `"resize"` | `size` | la ventana cambió de tamaño |
| `"focus"` | `focused` | la ventana ganó o perdió el foco del teclado |
| `"leave"`, `"enter"` | | el puntero salió de la ventana, o volvió |
| `"close"` | | la ventana se está cerrando: el último evento |

Un `"move"` también trae `samples`, todos los puntos por los que pasó el puntero desde el cuadro anterior (más abajo).

`mods` dice cuáles de `shift`, `alt`, `cmd` y `ctrl` estaban presionadas. Las opciones van en un hash después del título: `window.open(160, 100, "game", {"scale": 3, "fps": 30})` muestra cada píxel como un cuadrado de 3x3, ideal para pixel art. Un programa muestra una sola ventana. El REPL tiene todos los módulos menos `window` y `pen`; ahí, `window` da `identifier not found`.

**Ajustar la presión.** No todos los lápices y trackpads se sienten igual: uno llega a la presión máxima con un toque suave, otro necesita apretar fuerte. `pen.curve(...)` le da otra forma a la presión cruda antes de que tu programa la use, para que el pincel se sienta bien en tu dispositivo; cómo llamarla, y el archivo de configuración que la recuerda, están en [TERRARIUM.md](../../TERRARIUM.md#pen) (en inglés).

> **Error común: la división entera en el código de dibujo.** `brush` da un flotante, así que `r / 2` en `stroke` conserva su fracción. Con enteros, `/` la descarta (capítulo 4): `3 / 2` da `1`. Cuando la fracción importa, como en el paso 5, escribe `float(i) / steps`.

> **Pruébalo.** Haz que el botón derecho borre, dibujando con `paper` cuando `e.button == "right"` (guarda el color en una variable junto a `last`). Luego haz que la rueda cambie la tinta: en un evento `"scroll"`, toma el siguiente color de una lista.

## Más de un búfer: degradados, capas y superficies

El programa de pintura dibuja con colores planos en un solo búfer. Una aplicación de dibujo de verdad quiere más: un cielo degradado, formas que oscurecen lo de abajo como tinta, una interfaz de píxeles grandes alrededor de un lienzo nítido. Cada una de estas cosas es una función de dibujo más, y funcionan igual en imágenes, ventanas y superficies.

### Degradados

`span_gradient` y `spans_gradient` son `span` y `spans` con un degradado en vez de un color. Un degradado es un hash: su tipo (`kind`), dónde empieza (`origin`), hacia dónde va, y sus paradas (`stops`), colores en lugares de 0 a 1:

```
let w = window.open(400, 200, "sky")
let rows = []
for (let y = 0; y < 200; y += 1) { push(rows, [y, 0, 400]) }
window.spans_gradient(w, rows, {"kind": "linear", "origin": (0, 0), "axis": (0, 1 / 200.0),
  "stops": [[0, #3b4cc0], [0.6, #e2557b], [1, #f6c453]]})
```

El `axis` es la dirección del inicio al final dividida por su longitud al cuadrado: de `(0, 0)` a `(0, 200)` es `(0, 200) / 200²`, o sea `(0, 1 / 200.0)`. Un degradado `"radial"` lleva en cambio una `scale`, 1 dividido entre su radio. Agrega `"steps": 5` y el degradado se vuelve cinco franjas planas, ideal para pixel art.

### Capas y modos de mezcla

`push_layer(w)` abre una capa nueva y transparente: todo lo que se dibuja en `w` va a ella, hasta que `pop_layer(w, mode, opacity)` la mezcla con lo de abajo. Dos formas dibujadas en una capa a media opacidad no muestran dónde se cruzan, y en `"multiply"` oscurecen el cielo como tinta:

```
window.push_layer(w)
window.spans(w, hill(100), #2b3a55)
window.spans(w, hill(260), #2b3a55)
window.pop_layer(w, "multiply", 0.7)
```

Los modos son `normal`, `multiply`, `screen`, `overlay`, `darken`, `lighten` y `add`. `examples/terrarium/layers.liz` los muestra uno al lado del otro.

### Superficies

Una ventana con escala 2 tiene píxeles grandes y nítidos: buenos para una interfaz, demasiado gruesos para un dibujo. Una superficie es un segundo búfer que se muestra en un rectángulo de la ventana con una escala propia:

```
let w = window.open(320, 200, "app", {"scale": 2})
let canvas = window.surface(w, "canvas", [10, 20, 300, 170], 1)
window.clear(canvas, #f4efe6)
```

La superficie cubre `[10, 20, 300, 170]` en píxeles de la ventana, pero con escala 1 tiene el doble de píxeles en cada sentido: `window.size(canvas)` da `(600, 340)`. Se dibuja en ella como en la ventana. Un evento del puntero sobre ella lo dice: `e.surface` es `"canvas"` y `e.surface_pos` es el punto en los píxeles del lienzo.

Un trazo rápido puede cruzar muchos píxeles entre dos cuadros. Cada `"move"` trae `e.samples`, todos los puntos por los que pasó el puntero desde el cuadro anterior, cada uno con su `surface_pos` y su `pressure`, así el dibujo conserva la forma de la mano que lo hizo:

```
for (let i = 0; i < len(e.samples); i += 1) {
  let s = e.samples[i]
  if (s.surface == "canvas") { dab(s.surface_pos, s.pressure) }
}
```

### Una pantalla de inicio que se vuelve la ventana

`window.open` acepta `"frameless": true` (sin barra de título), `"transparent": true` (los píxeles que dejes transparentes muestran el escritorio) y `"floating": true` (encima de todas las ventanas): una pantalla de inicio con esquinas redondas. `window.set` cambia la ventana mientras corre, así la misma ventana se vuelve la aplicación:

```
window.set(w, {"frameless": false, "floating": false, "scale": 2, "size": (320, 200),
  "resizable": true, "center": true, "title": "my app"})
```

`examples/terrarium/splash.liz` hace exactamente esto, y luego pinta en una superficie lienzo con la presión del lápiz.

### Probar una ventana sin pantalla

Con `"headless": true` una ventana no tiene pantalla: `window.run` corre sus cuadros tan rápido como puede, el programa le da eventos con `window.inject`, y `window.screenshot` da la imagen. Así se prueba una aplicación, en cualquier computador:

```
let w = window.open(200, 100, "", {"headless": true})
window.inject(w, {"type": "down", "pos": (10, 10)})
window.inject(w, {"type": "up", "pos": (10, 10)})
window.run(w, function(events) {
  handle(events)
  false
})
img.save(window.screenshot(w), "after-a-click.png")
```

Devolver `false` en el primer `tick` termina la ejecución después de un cuadro. `window.screenshot(w, {"full": true})` da en cambio todos los píxeles de la pantalla, el tamaño de la ventana por su escala, como la ve el usuario.

### Dejar que otro programa la maneje

`window.automation(w)` deja que otro programa use la ventana como lo haría una persona. Mientras corre, una terminal le puede mandar comandos:

```
lizardscript drive Click 40 30
lizardscript drive Drag 10 10 200 80 --pressure 0.2,1
lizardscript drive Press 20 20
lizardscript drive Move 100 60 --steps 5
lizardscript drive Release 100 60
lizardscript drive Path 20 20 120 20 120 80 --close
lizardscript drive Screenshot shot.png
lizardscript drive Quit
```

Cada respuesta llega después del cuadro que atendió el comando, así el siguiente comando ve lo que hizo el programa. `Click` mueve el puntero al punto antes de presionar, como lo hace una mano. `Drag` va en línea recta de inicio a fin en un solo comando; para mirar la ventana a mitad del camino, pártelo con `Press`, `Move` y `Release`, y para dibujar una línea quebrada o un lazo usa `Path` con `--close`. El programa puede agregar sus propios comandos, y su propio estado para `Dump`; la lista completa está en [TERRARIUM.md](../../TERRARIUM.md#automation) (en inglés).

> **Pruébalo.** Ejecuta `examples/terrarium/layers.liz`, apunta a las columnas y lee el modo en el título. Luego cambia `0.9` en `pop_layer` por `0.4` y mira qué modos cambian más.

## Los ejemplos

Los programas del Terrario en la carpeta `examples/terrarium/` del repositorio se ejecutan como cualquier otro:

| Programa | Ejecútalo con | Qué hace |
| --- | --- | --- |
| `files.liz` | `./lizardscript examples/terrarium/files.liz` | escribe, lee y lista archivos en una carpeta dentro de tu carpeta personal, y hace un viaje de ida y vuelta por JSON; al final limpia lo que creó |
| `image.liz` | `./lizardscript examples/terrarium/image.liz out.png` | dibuja un degradado píxel por píxel y lo guarda como PNG (`gradient.png` si no le das argumento) |
| `paint.liz` | `./lizardscript examples/terrarium/paint.liz` | el programa de pintura de este capítulo |
| `clock.liz` | `./lizardscript examples/terrarium/clock.liz` | una ventana con un reloj de manecillas dibujado con tramos; Esc sale |
| `layers.liz` | `./lizardscript examples/terrarium/layers.liz` | los siete modos de mezcla sobre un degradado (agrega un nombre de archivo para guardar una imagen sin abrir ventana) |
| `splash.liz` | `./lizardscript examples/terrarium/splash.liz` | una pantalla de inicio sin marco y transparente que se vuelve una ventana normal con una superficie lienzo para pintar |

Agrega `--vm` antes del archivo para ejecutarlos en el motor más rápido: `./lizardscript --vm examples/terrarium/clock.liz`.

## Resumen

- Los módulos del Terrario son hashes de funciones integradas que tiene todo programa: `fs`, `os`, `time`, `json`, `img`, `clip`, `window`, `pen`.
- Sus errores son errores de ejecución con el nombre de la función al principio, y `try` los atrapa.
- `fs.read`, `fs.write`, `fs.append`, `fs.list`, `fs.mkdir`, `fs.remove` (solo carpetas vacías), `fs.join`, `fs.digest`.
- `os.args()` da lo que sigue al nombre del archivo; `os.exit(code)` detiene el programa.
- `time.ms()` mide, `time.sleep(ms)` espera, `time.now()` es la fecha y la hora (con `epoch` para guardar), `time.utc()` lo mismo en UTC.
- `json.write(value, indent)` y `json.parse(text)`; los colores vuelven como cadenas.
- `img.new`, `img.get`, `img.set`, `img.save`, `img.load`; `fill`, `span`, `clear` y `blit` dibujan igual en imágenes y ventanas.
- `window.run(w, tick)` llama a `tick(events)` en cada cuadro hasta que devuelva `false`; un programa de pintura es un círculo, un arrastre, un trazo que rellena los huecos y `e.pen.pressure`.
- `spans_gradient` pinta con un degradado, `push_layer`/`pop_layer` mezclan un grupo de formas con un modo de mezcla, y `window.surface` le agrega a una ventana un búfer nítido propio.
- `window.set` cambia una ventana que corre; una ventana `"headless"` corre sin pantalla, para pruebas; `window.automation` deja que `lizardscript drive` la use.

[← Anterior: Proyectos](16-projects.md) · [Índice](../README.md) · [Siguiente: Módulos y herramientas →](18-modules-and-tools.md)
