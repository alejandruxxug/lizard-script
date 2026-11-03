# 10. Strings and `text`

[← Previous: Hashes and dot access](09-hashes.md) · [Index](../README.md) · [Next: Numbers, `math` and units →](11-numbers-and-math.md) · [Español](../es/10-strings-and-text.md)

A string is a piece of text between double quotes. This chapter covers what you can write inside one, how to take strings apart and how to build them, mostly with the `text` library.

## Special characters: escapes

Some characters can't be typed directly between quotes. A backslash `\` starts an **escape**:

| Escape | Means |
| --- | --- |
| `\"` | a double quote |
| `\\` | a backslash |
| `\n` | a line break |
| `\t` | a tab |
| `\r` | a carriage return |
| `\u00e9` or `\u{E9}` | the character with that Unicode code (here `é`) |

```
puts("she said \"hi\"")
puts("C:\\lizards")
puts("one\ntwo")
puts("name\tage")
puts("caf\u00e9", "\u{1F98E}")
```

**Output:**

```text
she said "hi"
C:\lizards
one
two
name	age
café 🦎
```

Any other backslash is an error:

```
puts("a\qb")
```

**Output:**

```text
[line 1, column 6] lexical error: unknown escape \q in string
```

A string can also span several lines; the line breaks are part of it.

## Characters, not bytes

`len` counts characters, and an index gives one character as a string. Accented letters and emoji count as one:

```
let word = "héllo"
puts(len(word), word[1])
puts(len("\u{1F98E}"))
```

**Output:**

```text
5 é
1
```

Just like arrays, an index outside the string gives `null`, and a negative index doesn't count from the end:

```
puts("abc"[5], "abc"[-1])
```

**Output:**

```text
null null
```

## Strings can't be changed

You can read `s[0]`, but not assign to it. Make a new string instead:

```
let s = "cat"
s[0] = "b"
```

**Output:**

```text
[line 2, column 2] runtime error: strings can't be changed
```

```
let s = "cat"
s = "b" + text.slice(s, 1, len(s))
puts(s)
```

**Output:**

```text
bat
```

## Joining and repeating

`+` joins two strings, `*` repeats one, and `+=` adds to the end of a string variable:

```
let line = "-" * 10
puts(line)
let greeting = "Hello"
greeting += ", Liz"
puts(greeting + "!")
```

**Output:**

```text
----------
Hello, Liz!
```

Remember that `+` doesn't convert numbers (chapter 2): `"n = " + str(n)`.

## Comparing

`==` and `===` compare text exactly (`"Liz" == "liz"` is false). `<` and `>` compare in dictionary order, with uppercase before lowercase. To compare without caring about case, lower both first:

```
puts("Liz" == "liz")
puts(text.lower("Liz") == text.lower("liz"))
```

**Output:**

```text
false
true
```

## The `text` library

`text` is a set of functions for strings, used with a dot. Indexes count characters, and where it says so, a negative index counts from the end.

### Cutting and searching

```
let s = "lizard script"
puts(text.slice(s, 0, 6))
puts(text.slice(s, -6, len(s)))
puts(text.find(s, "script"), text.find(s, "i", 2), text.find(s, "x"))
puts(text.starts(s, "liz"), text.ends(s, "script"))
```

**Output:**

```text
lizard
script
7 10 -1
true true
```

- `text.slice(s, start, end)` gives the characters from `start` up to (not including) `end`.
- `text.find(s, sub)` gives the index of `sub`, or `-1`. An optional third argument says where to start looking.

### Splitting and joining

```
puts(text.split("red,green,blue", ","))
puts(text.split("abc", ""))
puts(text.join(["a", "b", "c"], " + "))
puts(text.lines("first\nsecond\nthird"))
```

**Output:**

```text
["red", "green", "blue"]
["a", "b", "c"]
a + b + c
["first", "second", "third"]
```

`text.join` only joins strings:

```
puts(text.join([1, 2, 3], ","))
```

**Output:**

```text
[line 1, column 15] runtime error: text.join: element 0 of the list must be STRING, got INTEGER
```

Convert first, for example with a loop that pushes `str(x)`.

### Cleaning and changing

```
puts("[" + text.trim("   lots of space   ") + "]")
puts(text.upper("Liz"), text.lower("LIZ"))
puts(text.replace("the cat and the hat", "the", "a"))
```

**Output:**

```text
[lots of space]
LIZ liz
a cat and a hat
```

### Characters and codes

Each character has a number, its code. `text.code(s, i)` gives the code of the character at `i`, and `text.char(n)` the character of a code:

```
puts(text.code("A", 0), text.code("a", 0))
puts(text.char(65), text.char(955))
```

**Output:**

```text
65 97
A λ
```

### Reading numbers from text

`text.number(s)` reads the number written in a string, or gives `null` if it isn't one. Unlike `int`, it never stops your program:

```
puts(text.number("42"), text.number(" -2.5 "), text.number("1e3"))
puts(text.number("12px"), text.number("twelve"))
```

**Output:**

```text
42 -2.5 1000.0
null null
```

### Fingerprints of text

`text.digest(s)` is a fingerprint of a string: 64 hex digits (SHA-256) that change completely when the text changes at all, and are the same on every computer. `text.digest(s, "fnv")` is a shorter, faster one, good for caches but easy to fake on purpose.

```
puts(text.digest("lizard"))
puts(text.digest("lizard", "fnv"))
puts(text.digest("lizard") == text.digest("Lizard"))
```

**Output:**

```text
b95cbcfe0cd25ed09eef361239fee7d50a3f31545465ca2421f4e791be2624b6
06c79e6c935a3779
false
```

## Example: reversing and checking palindromes

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

**Output:**

```text
drazil
true false
```

> **Try it.** Write `capitalize(s)` that makes the first letter uppercase: `capitalize("liz")` is `"Liz"`. Then write `countVowels(s)`.

## Summary

- Escapes: `\"`, `\\`, `\n`, `\t`, `\r`, `\u{...}`.
- `len` and indexes count characters. Out-of-range indexes give `null`.
- Strings can't be changed; build new ones with `+`, `*` and `text`.
- `text`: `slice`, `find`, `split`, `join`, `replace`, `trim`, `lines`, `starts`, `ends`, `upper`, `lower`, `code`, `char`, `number`.

[← Previous: Hashes and dot access](09-hashes.md) · [Index](../README.md) · [Next: Numbers, `math` and units →](11-numbers-and-math.md)
