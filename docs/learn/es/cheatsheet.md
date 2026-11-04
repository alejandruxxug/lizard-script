# Hoja de referencia de lizard-script

[Índice](../README.md) · [Ejercicios →](exercises.md) · [English](../en/cheatsheet.md)

## Sintaxis

| Qué | Cómo |
| --- | --- |
| comentario | `-~> hasta el final de la línea` |
| crear una variable | `let x = 5` |
| cambiar una variable | `x = 6`, `x += 1`, `-=`, `*=`, `/=` (no hay `**=`, `//=`, `%=`) |
| fin de instrucción | el final de la línea o `;`; una línea que termina con un operador, `,` o un paréntesis o corchete abierto sigue en la siguiente |
| condición | `if (a) { } elif (b) { } else { }` (`else if` también funciona) |
| `if` como valor | `let s = if (n > 0) { "+" } else { "-" }` |
| ciclo while | `while (cond) { }` |
| ciclo for | `for (let i = 0; i < n; i += 1) { }` (no olvides el `let`) |
| salir / saltar | `break`, `continue` (dentro de ciclos) |
| función | `function f(a, b = 2) { a + b }` o `let f = function(a) { a }` |
| devolver | el último valor, o `return x`; nada significa `null` |
| errores | `error("mensaje")`, `error("mensaje", {"at": "caller"})` en una biblioteca, `try { } catch (e) { e.message }` (es una instrucción) |
| capa | `layer nombre` solo en una línea, luego un cuerpo con sangría |
| módulo | `use "ruta"` (sus nombres, vivos) o `use "ruta" as m` (un hash), en el nivel superior de un archivo; los `_nombres` son privados |

## Valores

| Tipo | Literales | Notas |
| --- | --- | --- |
| `INTEGER` | `42`, `-7`, `10px` | 64 bits |
| `FLOAT` | `3.5`, `2.0`, `90deg`, `0.5turn`, `1rad` | `1e3` y `2.5e-4` son flotantes; no `.5` |
| `STRING` | `"Liz"`, `"a\n\t\"\\"`, `"\u{1F98E}"` | solo comillas dobles; no se puede cambiar |
| `BOOLEAN` | `true`, `false` | |
| `NULL` | `null` | posición o clave que falta, sin resultado |
| `ARRAY` | `[1, "a", [2]]` | compartido, índice desde 0 |
| `HASH` | `{"name": "Liz", 1: true}` | claves: cadena, entero, booleano; compartido |
| `FUNCTION` | `function(x) { x }` | clausuras |
| `COLOR` | `#f80`, `#3fa34d`, `#00000080` | `c.r c.g c.b c.a`; no se puede cambiar |
| `POINT` | `(1, 2)`, `(1, 2, 3)` | `p.x p.y p.z`; `+ -` con puntos, `* /` con números |

**Valores falsos:** `false`, `null`, `0`, `0.0`, `""`, `[]`, `{}`. Todo lo demás es verdadero.

## Operadores, del más débil al más fuerte

`=` `+=` … · `||` · `&&` · `==` `!=` `===` `!==` · `<` `>` `<=` `>=` · `+` `-` · `*` `/` `//` `%` · `-x` `!x` · `**` · `f(x)` `a[i]` `a.b`

```
puts(7 / 2, 7 / 2.0, -7 / 2, -7 // 2, -7 % 3)
puts(2 ** 3 ** 2, -2 ** 2)
puts(5 == "5", 5 === "5", 1 == 1.0, 1 === 1.0)
puts("ab" + "c", "ab" * 2, [1] + [2])
puts(0 || "", 5 && "x")
```

**Salida:**

```text
3 3.5 -3 -4 2
512 -4
true false true false
abc abab [1, 2]
false true
```

## Funciones integradas

| Grupo | Nombres |
| --- | --- |
| generales | `puts len type str int float range error` |
| arreglos, hashes | `push pop first last rest keys values delete same` |
| colores, puntos | `rgb rgba hsl point` |
| `math.` | `pi tau sqrt abs floor ceil round min max sin cos tan asin acos atan2 dist angle random seed` |
| `text.` | `slice find split join replace trim lines starts ends upper lower code char number digest` |
| `list.` | `slice insert remove index reverse sort copy` |
| `hash.` | `copy keys values merge has remove` |

## El Terrario (capítulo 17)

| Módulo | Nombres |
| --- | --- |
| `fs.` | `read write append exists list mkdir remove home join digest` |
| `os.` | `args env exit platform locale config_dir` |
| `time.` | `ms now utc sleep` (`now().epoch`, `.weekday`, `.utc_offset`) |
| `json.` | `parse write(value, indent = 0)` |
| `img.` | `new load save get set pixels from_pixels` + dibujo |
| `clip.` | `get set` |
| `window.` | `open(w, h, title, options) run(w, tick) title` + dibujo (no en el REPL) |
| `pen.` | `available`; la presión es `e.pen.pressure` en los eventos de la ventana |
| dibujo (`img.` y `window.`) | `size clear fill(t, [x, y, w, h], c) span(t, y, x0, x1, c) spans blit(t, source, pos, {"scale": k, "filter": "linear"})` |

Los errores dicen `fs.read: can't read x.txt: ...` y `try` los atrapa. Eventos: `e.type` es `down up move scroll pinch key keyup type resize close`; `e.pos`, `e.button`, `e.key`, `e.text`, `e.pen.pressure`.

## Errores comunes

| Mensaje | Solución |
| --- | --- |
| `can't assign to x: it was never declared` | primero `let x = ...` |
| `identifier not found: x` | error de tipeo, o `x` está fuera de su alcance |
| `type mismatch: STRING + INTEGER` | `str(n)` |
| `type mismatch: NULL + INTEGER` | una posición o clave que falta; revisa con `=== null` |
| `continue outside of a loop` | `continue` (igual que `break`) solo funciona dentro de `while` o `for` |
| `expected end of statement, got identifier ...` | ¿usaste `//` como comentario? Usa `-~>` |
| `unknown unit "kg"` | solo `deg`, `rad`, `turn` y `px` pueden ir pegadas a un número; escribe `5 * kg` o cambia el nombre |
| `negative exponent` | usa un flotante: `2.0 ** -1` |
| `index out of range` | usa `push` para agregar elementos |
| `wrong number of arguments: want 2, got 1` | revisa la llamada, o agrega un valor por defecto |

## Ejecutar

```sh
go build                  # crea ./lizardscript
./lizardscript            # REPL; exit o Ctrl+D para salir
./lizardscript prog.liz   # ejecuta un archivo
./lizardscript prog.liz a b   # os.args() es ["a", "b"]
```

[Índice](../README.md) · [Ejercicios →](exercises.md)
