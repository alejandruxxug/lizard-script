# 10. Cadenas y `text`

[← Anterior: Hashes y acceso con punto](09-hashes.md) · [Índice](../README.md) · [Siguiente: Números, `math` y unidades →](11-numbers-and-math.md) · [English](../en/10-strings-and-text.md)

Una cadena es un trozo de texto entre comillas dobles. Este capítulo cubre qué puedes escribir dentro de una, cómo desarmar cadenas y cómo construirlas, sobre todo con la biblioteca `text`.

## Caracteres especiales: secuencias de escape

Algunos caracteres no se pueden escribir directamente entre comillas. Una barra invertida `\` inicia una **secuencia de escape**:

| Escape | Significa |
| --- | --- |
| `\"` | una comilla doble |
| `\\` | una barra invertida |
| `\n` | un salto de línea |
| `\t` | un tabulador |
| `\r` | un retorno de carro |
| `\u00e9` o `\u{E9}` | el carácter con ese código Unicode (aquí `é`) |

```
puts("she said \"hi\"")
puts("C:\\lizards")
puts("one\ntwo")
puts("name\tage")
puts("caf\u00e9", "\u{1F98E}")
```

**Salida:**

```text
she said "hi"
C:\lizards
one
two
name	age
café 🦎
```

Cualquier otra barra invertida es un error:

```
puts("a\qb")
```

**Salida:**

```text
[line 1, column 6] lexical error: unknown escape \q in string
```

Una cadena también puede ocupar varias líneas; los saltos de línea forman parte de ella.

## Caracteres, no bytes

`len` cuenta caracteres, y un índice da un carácter como cadena. Las letras con tilde y los emoji cuentan como uno:

```
let word = "héllo"
puts(len(word), word[1])
puts(len("\u{1F98E}"))
```

**Salida:**

```text
5 é
1
```

Igual que en los arreglos, un índice fuera de la cadena da `null`, y un índice negativo no cuenta desde el final:

```
puts("abc"[5], "abc"[-1])
```

**Salida:**

```text
null null
```

## Las cadenas no se pueden cambiar

Puedes leer `s[0]`, pero no asignarle. Crea una cadena nueva:

```
let s = "cat"
s[0] = "b"
```

**Salida:**

```text
[line 2, column 2] runtime error: strings can't be changed
```

```
let s = "cat"
s = "b" + text.slice(s, 1, len(s))
puts(s)
```

**Salida:**

```text
bat
```

## Unir y repetir

`+` une dos cadenas, `*` repite una, y `+=` agrega al final de una variable de tipo cadena:

```
let line = "-" * 10
puts(line)
let greeting = "Hello"
greeting += ", Liz"
puts(greeting + "!")
```

**Salida:**

```text
----------
Hello, Liz!
```

Recuerda que `+` no convierte números (capítulo 2): `"n = " + str(n)`.

## Comparar

`==` y `===` comparan el texto exactamente (`"Liz" == "liz"` es falso). `<` y `>` comparan en orden de diccionario, con las mayúsculas antes que las minúsculas. Para comparar sin importar mayúsculas, pasa ambas a minúsculas primero:

```
puts("Liz" == "liz")
puts(text.lower("Liz") == text.lower("liz"))
```

**Salida:**

```text
false
true
```

## La biblioteca `text`

`text` es un conjunto de funciones para cadenas, que se usan con un punto. Los índices cuentan caracteres y, donde se indica, un índice negativo cuenta desde el final.

### Cortar y buscar

```
let s = "lizard script"
puts(text.slice(s, 0, 6))
puts(text.slice(s, -6, len(s)))
puts(text.find(s, "script"), text.find(s, "i", 2), text.find(s, "x"))
puts(text.starts(s, "liz"), text.ends(s, "script"))
```

**Salida:**

```text
lizard
script
7 10 -1
true true
```

- `text.slice(s, inicio, fin)` da los caracteres desde `inicio` hasta `fin` (sin incluirlo).
- `text.find(s, sub)` da la posición de `sub`, o `-1`. Un tercer argumento opcional dice desde dónde empezar a buscar.

### Dividir y unir

```
puts(text.split("red,green,blue", ","))
puts(text.split("abc", ""))
puts(text.join(["a", "b", "c"], " + "))
puts(text.lines("first\nsecond\nthird"))
```

**Salida:**

```text
["red", "green", "blue"]
["a", "b", "c"]
a + b + c
["first", "second", "third"]
```

`text.join` solo une cadenas:

```
puts(text.join([1, 2, 3], ","))
```

**Salida:**

```text
[line 1, column 15] runtime error: text.join: element 0 of the list must be STRING, got INTEGER
```

Conviértelas primero, por ejemplo con un ciclo que haga `push` de `str(x)`.

### Limpiar y cambiar

```
puts("[" + text.trim("   lots of space   ") + "]")
puts(text.upper("Liz"), text.lower("LIZ"))
puts(text.replace("the cat and the hat", "the", "a"))
```

**Salida:**

```text
[lots of space]
LIZ liz
a cat and a hat
```

### Caracteres y códigos

Cada carácter tiene un número, su código. `text.code(s, i)` da el código del carácter en `i`, y `text.char(n)` el carácter de un código:

```
puts(text.code("A", 0), text.code("a", 0))
puts(text.char(65), text.char(955))
```

**Salida:**

```text
65 97
A λ
```

### Leer números de un texto

`text.number(s)` lee el número escrito en una cadena, o da `null` si no lo es. A diferencia de `int`, nunca detiene tu programa:

```
puts(text.number("42"), text.number(" -2.5 "), text.number("1e3"))
puts(text.number("12px"), text.number("twelve"))
```

**Salida:**

```text
42 -2.5 1000.0
null null
```

### Huellas de un texto

`text.digest(s)` es una huella de una cadena: 64 dígitos hexadecimales (SHA-256) que cambian por completo cuando el texto cambia en lo más mínimo, y son los mismos en todos los computadores. `text.digest(s, "fnv")` es una más corta y rápida, buena para cachés pero fácil de falsificar a propósito.

```
puts(text.digest("lizard"))
puts(text.digest("lizard", "fnv"))
puts(text.digest("lizard") == text.digest("Lizard"))
```

**Salida:**

```text
b95cbcfe0cd25ed09eef361239fee7d50a3f31545465ca2421f4e791be2624b6
06c79e6c935a3779
false
```

## Ejemplo: invertir y reconocer palíndromos

```
function reverse(s) {
  let out = ""
  for (let i = len(s) - 1; i >= 0; i -= 1) {
    out += s[i]
  }
  out
}
function isPalindrome(s) {
  let clean = text.lower(text.replace(s, " ", ""))
  clean == reverse(clean)
}
puts(reverse("lizard"))
puts(isPalindrome("Never odd or even"), isPalindrome("lizard"))
```

**Salida:**

```text
drazil
true false
```

> **Pruébalo.** Escribe `capitalize(s)`, que ponga la primera letra en mayúscula: `capitalize("liz")` es `"Liz"`. Luego escribe `countVowels(s)`.

## Resumen

- Escapes: `\"`, `\\`, `\n`, `\t`, `\r`, `\u{...}`.
- `len` y los índices cuentan caracteres. Los índices fuera de rango dan `null`.
- Las cadenas no se pueden cambiar; construye nuevas con `+`, `*` y `text`.
- `text`: `slice`, `find`, `split`, `join`, `replace`, `trim`, `lines`, `starts`, `ends`, `upper`, `lower`, `code`, `char`, `number`.

[← Anterior: Hashes y acceso con punto](09-hashes.md) · [Índice](../README.md) · [Siguiente: Números, `math` y unidades →](11-numbers-and-math.md)
