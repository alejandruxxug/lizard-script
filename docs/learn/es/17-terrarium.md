# 17. El Terrario

[← Anterior: Proyectos](16-projects.md) · [Índice](../README.md) · [Hoja de referencia →](cheatsheet.md) · [English](../en/17-terrarium.md)

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

## `time`: relojes y esperas

```
let start = time.ms()
time.sleep(100)
let took = time.ms() - start
puts(type(took), took >= 100)

let now = time.now()
puts(keys(now))
puts(now.month >= 1 && now.month <= 12)
```

**Salida:**

```text
FLOAT true
["year", "month", "day", "hour", "minute", "second"]
true
```

- `time.ms()` son los milisegundos desde que empezó el programa, como flotante. Resta dos de ellos para medir cuánto tardó algo.
- `time.sleep(ms)` espera.
- `time.now()` es la fecha y la hora como un hash de enteros, así que `now.hour` y `now.minute` están listos para imprimir o para dibujar un reloj.

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

La fila del medio es rojo semitransparente: sobre negro da un rojo oscuro (`#800000`), sobre blanco un rosado (`#ff7f7f`). Las otras funciones de dibujo son `img.clear(image, color)` (todos los píxeles, sin mezclar) e `img.blit(image, other, pos)` (pinta una imagen sobre otra).

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
| `"key"` | `key`, `mods` | se presionó una tecla |
| `"type"` | `text` | se escribió un carácter |
| `"resize"` | `size` | una ventana redimensionable cambió de tamaño |
| `"close"` | | la ventana se está cerrando: el último evento |

`mods` dice cuáles de `shift`, `alt`, `cmd` y `ctrl` estaban presionadas. Las opciones van en un hash después del título: `window.open(160, 100, "game", {"scale": 3, "fps": 30})` muestra cada píxel como un cuadrado de 3x3, ideal para pixel art. Un programa muestra una sola ventana. El REPL tiene todos los módulos menos `window` y `pen`; ahí, `window` da `identifier not found`.

**Ajustar la presión.** No todos los lápices y trackpads se sienten igual: uno llega a la presión máxima con un toque suave, otro necesita apretar fuerte. `pen.curve(...)` le da otra forma a la presión cruda antes de que tu programa la use, para que el pincel se sienta bien en tu dispositivo; cómo llamarla, y el archivo de configuración que la recuerda, están en [TERRARIUM.md](../../TERRARIUM.md#pen) (en inglés).

> **Error común: la división entera en el código de dibujo.** `brush` da un flotante, así que `r / 2` en `stroke` conserva su fracción. Con enteros, `/` la descarta (capítulo 4): `3 / 2` da `1`. Cuando la fracción importa, como en el paso 5, escribe `float(i) / steps`.

> **Pruébalo.** Haz que el botón derecho borre, dibujando con `paper` cuando `e.button == "right"` (guarda el color en una variable junto a `last`). Luego haz que la rueda cambie la tinta: en un evento `"scroll"`, toma el siguiente color de una lista.

## Los ejemplos

Los cuatro programas del Terrario en la carpeta `examples/terrarium/` del repositorio se ejecutan como cualquier otro:

| Programa | Ejecútalo con | Qué hace |
| --- | --- | --- |
| `files.liz` | `./lizardscript examples/terrarium/files.liz` | escribe, lee y lista archivos en una carpeta dentro de tu carpeta personal, y hace un viaje de ida y vuelta por JSON; al final limpia lo que creó |
| `image.liz` | `./lizardscript examples/terrarium/image.liz out.png` | dibuja un degradado píxel por píxel y lo guarda como PNG (`gradient.png` si no le das argumento) |
| `paint.liz` | `./lizardscript examples/terrarium/paint.liz` | el programa de pintura de este capítulo |
| `clock.liz` | `./lizardscript examples/terrarium/clock.liz` | una ventana con un reloj de manecillas dibujado con tramos; Esc sale |

Agrega `--vm` antes del archivo para ejecutarlos en el motor más rápido: `./lizardscript --vm examples/terrarium/clock.liz`.

## Resumen

- Los módulos del Terrario son hashes de funciones integradas que tiene todo programa: `fs`, `os`, `time`, `json`, `img`, `clip`, `window`, `pen`.
- Sus errores son errores de ejecución con el nombre de la función al principio, y `try` los atrapa.
- `fs.read`, `fs.write`, `fs.append`, `fs.list`, `fs.mkdir`, `fs.remove` (solo carpetas vacías), `fs.join`.
- `os.args()` da lo que sigue al nombre del archivo; `os.exit(code)` detiene el programa.
- `time.ms()` mide, `time.sleep(ms)` espera, `time.now()` es la fecha y la hora.
- `json.write(value, indent)` y `json.parse(text)`; los colores vuelven como cadenas.
- `img.new`, `img.get`, `img.set`, `img.save`, `img.load`; `fill`, `span`, `clear` y `blit` dibujan igual en imágenes y ventanas.
- `window.run(w, tick)` llama a `tick(events)` en cada cuadro hasta que devuelva `false`; un programa de pintura es un círculo, un arrastre, un trazo que rellena los huecos y `e.pen.pressure`.

[← Anterior: Proyectos](16-projects.md) · [Índice](../README.md) · [Hoja de referencia →](cheatsheet.md)
