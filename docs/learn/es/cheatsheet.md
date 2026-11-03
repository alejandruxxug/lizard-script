# Hoja de referencia de lizard-script

[Índice](../README.md) · [Ejercicios →](exercises.md) · [English](../en/cheatsheet.md)

## Sintaxis

| Qué | Cómo |
| --- | --- |
| comentario | `-~> hasta el final de la línea` |
| crear una variable | `let x = 5` |
| cambiar una variable | `x = 6`, `x += 1`, `-=`, `*=`, `/=` (no hay `**=`, `//=`, `%=`) |
| fin de instrucción | un salto de línea o `;` (una línea que empieza con un operador, `(` o `[` continúa la anterior) |
| condición | `if (a) { } elif (b) { } else { }` (no `else if`) |
| `if` como valor | `let s = if (n > 0) { "+" } else { "-" }` |
| ciclo while | `while (cond) { }` |
| ciclo for | `for (let i = 0; i < n; i += 1) { }` (no olvides el `let`) |
| salir / saltar | `break`, `continue` (dentro de ciclos) |
| función | `function f(a, b = 2) { a + b }` o `let f = function(a) { a }` |
| devolver | el último valor, o `return x`; nada significa `null` |
| errores | `error("mensaje")`, `try { } catch (e) { e.message }` (es una instrucción) |
| capa | `layer nombre` solo en una línea, luego un cuerpo con sangría |

## Valores

| Tipo | Literales | Notas |
| --- | --- | --- |
| `INTEGER` | `42`, `-7`, `10px` | 64 bits |
| `FLOAT` | `3.5`, `2.0`, `90deg`, `0.5turn`, `1rad` | no `.5`, no `1e3` |
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
| arreglos, hashes | `push pop first last rest keys values delete` |
| colores, puntos | `rgb rgba hsl point` |
| `math.` | `pi tau sqrt abs floor ceil round min max sin cos tan asin acos atan2 dist angle random seed` |
| `text.` | `slice find split join replace trim lines starts ends upper lower code char number` |
| `list.` | `slice insert remove index reverse sort copy` |

## Errores comunes

| Mensaje | Solución |
| --- | --- |
| `can't assign to x: it was never declared` | primero `let x = ...` |
| `identifier not found: x` | error de tipeo, o `x` está fuera de su alcance |
| `type mismatch: STRING + INTEGER` | `str(n)` |
| `type mismatch: NULL + INTEGER` | una posición o clave que falta; revisa con `=== null` |
| `expected "{", got "if"` | `elif`, no `else if` |
| `expected end of statement, got identifier ...` | ¿usaste `//` como comentario? Usa `-~>` |
| `negative exponent` | usa un flotante: `2.0 ** -1` |
| `index out of range` | usa `push` para agregar elementos |
| `wrong number of arguments: want 2, got 1` | revisa la llamada, o agrega un valor por defecto |

## Ejecutar

```sh
go build                  # crea ./lizardscript
./lizardscript            # REPL; exit o Ctrl+D para salir
./lizardscript prog.liz   # ejecuta un archivo
```

[Índice](../README.md) · [Ejercicios →](exercises.md)
