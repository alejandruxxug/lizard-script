# 9. Hashes and dot access

[← Previous: Arrays](08-arrays.md) · [Index](../README.md) · [Next: Strings and `text` →](10-strings-and-text.md) · [Español](../es/09-hashes.md)

A hash (also called a dictionary or map) stores values under **keys**. Where an array answers "what is item number 2?", a hash answers "what is the `name`?".

## Creating and reading

```
let lizard = {"name": "Liz", "age": 3, "colors": ["green", "brown"]}
puts(lizard)
puts(lizard["name"], lizard["age"])
puts(lizard["colors"][1])
puts(len(lizard), type(lizard))
```

**Output:**

```text
{"name": "Liz", "age": 3, "colors": ["green", "brown"]}
Liz 3
brown
3 HASH
```

A key that isn't there gives `null`:

```
let lizard = {"name": "Liz"}
puts(lizard["wings"])
```

**Output:**

```text
null
```

## Dot access

For string keys that look like names, `h.key` is exactly the same as `h["key"]`, and it is much easier to read:

```
let lizard = {"name": "Liz", "pet": {"kind": "beetle"}}
puts(lizard.name)
puts(lizard.pet.kind)
puts(lizard.missing)
```

**Output:**

```text
Liz
beetle
null
```

Use brackets when the key is in a variable, has spaces, or isn't a string:

```
let lizard = {"name": "Liz", "favorite food": "crickets"}
let field = "name"
puts(lizard[field])
puts(lizard["favorite food"])
```

**Output:**

```text
Liz
crickets
```

After the dot must come a name. A number or a keyword is a syntax error:

```
let h = {1: "one"}
puts(h.1)
```

**Output:**

```text
[line 2, column 8] syntax error: expected a name after "."
```

## Adding, changing and removing

`h[key] = value` and `h.key = value` add the key or replace its value. `delete(h, key)` removes a key and says whether it was there:

```
let lizard = {"name": "Liz", "age": 3}
lizard.age += 1
lizard.tail = true
lizard["favorite food"] = "crickets"
puts(lizard)
puts(delete(lizard, "tail"), delete(lizard, "wings"))
puts(lizard)
```

**Output:**

```text
{"name": "Liz", "age": 4, "tail": true, "favorite food": "crickets"}
true false
{"name": "Liz", "age": 4, "favorite food": "crickets"}
```

> **Common mistake: `+=` on a key that doesn't exist yet.** A missing key reads as `null`, and `null + 1` is not `1`:
>
> ```
> let counts = {}
> counts.apples += 1
> ```
>
> **Output:**
>
> ```text
> [line 2, column 15] runtime error: type mismatch: NULL + INTEGER
> ```
>
> Create it first, or check:
>
> ```
> let counts = {}
> if (counts.apples === null) { counts.apples = 0 }
> counts.apples += 1
> puts(counts)
> ```
>
> **Output:**
>
> ```text
> {"apples": 1}
> ```

## Keys

Keys can be strings, integers or booleans. Floats, arrays, hashes, colors and points can't be keys:

```
let h = {"a": 1, 2: "two", true: "yes"}
puts(h["a"], h[2], h[true])
let bad = {[1, 2]: "no"}
```

**Output:**

```text
1 two yes
[line 3, column 11] runtime error: unusable as hash key: ARRAY
```

Keys are compared exactly, so `1` and `"1"` are different keys:

```
let h = {1: "the number"}
puts(h[1], h["1"])
```

**Output:**

```text
the number null
```

## Going through a hash

`keys(h)` and `values(h)` give arrays, **in the order the keys were added**. That order is always the same, so output is predictable:

```
let stock = {"apples": 4, "pears": 0, "figs": 12}
puts(keys(stock))
puts(values(stock))

let names = keys(stock)
for (let i = 0; i < len(names); i += 1) {
  let k = names[i]
  puts(k, "->", stock[k])
}
```

**Output:**

```text
["apples", "pears", "figs"]
[4, 0, 12]
apples -> 4
pears -> 0
figs -> 12
```

## Hashes are shared too

Like arrays, hashes are shared, not copied. A function that receives a hash can change it:

```
function birthday(l) {
  l.age += 1
}
let liz = {"name": "Liz", "age": 3}
birthday(liz)
puts(liz)
```

**Output:**

```text
{"name": "Liz", "age": 4}
```

## Records: arrays of hashes

An array of hashes is the usual way to store a table of things:

```
let lizards = [
  {"name": "Liz", "age": 3},
  {"name": "Iggy", "age": 7},
  {"name": "Rex", "age": 5}
]
let oldest = lizards[0]
for (let i = 1; i < len(lizards); i += 1) {
  if (lizards[i].age > oldest.age) { oldest = lizards[i] }
}
puts("the oldest is", oldest.name)
```

**Output:**

```text
the oldest is Iggy
```

## Functions inside hashes

A hash can hold functions, which gives you something like an object with methods:

```
let calc = {
  "double": function(x) { x * 2 },
  "square": function(x) { x * x }
}
puts(calc.double(5), calc.square(5))
```

**Output:**

```text
10 25
```

Combined with closures (chapter 7), this lets you keep data private. See the bank account in [chapter 16](16-projects.md).

> **Try it.** Write a function `countWords(words)` that takes an array of strings and returns a hash with how many times each one appears. `countWords(["a", "b", "a"])` should give `{"a": 2, "b": 1}`.

## Summary

- `{"key": value, ...}`; a missing key gives `null`.
- `h.key` is `h["key"]`. Use brackets for keys in variables or with spaces.
- `h.key = v` adds or replaces; `delete(h, key)` removes.
- Keys: strings, integers, booleans. `1` and `"1"` are different.
- `keys` and `values` keep the order the keys were added in.
- Hashes are shared, like arrays.

[← Previous: Arrays](08-arrays.md) · [Index](../README.md) · [Next: Strings and `text` →](10-strings-and-text.md)
