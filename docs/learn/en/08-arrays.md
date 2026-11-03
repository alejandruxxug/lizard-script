# 8. Arrays

[← Previous: Functions](07-functions.md) · [Index](../README.md) · [Next: Hashes and dot access →](09-hashes.md) · [Español](../es/08-arrays.md)

An array is an ordered list of values. The values can be of any type, even mixed, and even other arrays.

## Creating and reading

```
let toes = [5, 5, 4, 5]
let mixed = [1, "two", true, [3, 4]]
puts(toes, len(toes))
puts(mixed)
puts(type(toes))
```

**Output:**

```text
[5, 5, 4, 5] 4
[1, "two", true, [3, 4]]
ARRAY
```

Strings inside an array are shown the way you write them in code, in quotes and with escapes, so you can paste the output back into a program. A string on its own is shown plain:

```
let quotes = ["say \"hi\"", "two\nlines", "back\\slash"]
puts(quotes)
puts("say \"hi\"")
```

**Output:**

```text
["say \"hi\"", "two\nlines", "back\\slash"]
say "hi"
```

Items are numbered from **0**. `a[i]` reads item `i`:

```
let pets = ["Liz", "Iggy", "Rex"]
puts(pets[0], pets[2])
puts(pets[len(pets) - 1])
```

**Output:**

```text
Liz Rex
Rex
```

An index that isn't in the array gives `null`, not an error. Negative indexes don't count from the end; they are just outside the array:

```
let pets = ["Liz", "Iggy", "Rex"]
puts(pets[3], pets[-1])
```

**Output:**

```text
null null
```

To get the last item, use `last(pets)` or `pets[len(pets) - 1]`. The index must be an integer:

```
let pets = ["Liz", "Iggy", "Rex"]
puts(pets[1.0])
```

**Output:**

```text
[line 2, column 10] runtime error: array index must be INTEGER, got FLOAT
```

## Changing items

`a[i] = x` replaces an item that exists. Compound assignment works too:

```
let toes = [5, 5, 4, 5]
toes[2] = 5
toes[0] += 1
puts(toes)
```

**Output:**

```text
[6, 5, 5, 5]
```

You can't create a new item by assigning past the end:

```
let toes = [5, 5]
toes[2] = 5
```

**Output:**

```text
[line 2, column 5] runtime error: index out of range: 2, the array has 2 elements
```

To add items, use `push`.

## Adding and removing

| Builtin | Does |
| --- | --- |
| `push(a, x)` | adds `x` at the end, returns `a` |
| `pop(a)` | removes the last item and returns it |
| `first(a)`, `last(a)` | the first or last item, or `null` if empty |
| `rest(a)` | a **new** array without the first item |
| `len(a)` | the number of items |

```
let stack = [1, 2, 3]
push(stack, 4)
puts(stack)
puts(pop(stack), stack)
puts(first(stack), last(stack), rest(stack))
puts(first([]), rest([]))
```

**Output:**

```text
[1, 2, 3, 4]
4 [1, 2, 3]
1 3 [2, 3]
null null
```

`pop` on an empty array is an error:

```
let empty = []
pop(empty)
```

**Output:**

```text
[line 2, column 4] runtime error: pop: the array is empty
```

`+` joins two arrays into a **new** one:

```
let a = [1, 2]
let b = a + [3]
puts(a, b)
```

**Output:**

```text
[1, 2] [1, 2, 3]
```

## Arrays are shared, not copied

This one surprises everybody once. After `let b = a`, `a` and `b` are **the same array**. A change made through one is visible through the other:

```
let a = [1, 2]
let b = a
push(b, 3)
puts(a)
```

**Output:**

```text
[1, 2, 3]
```

The same goes for function arguments: a function that receives an array can change it.

```
function addZero(xs) {
  push(xs, 0)
}
let numbers = [7, 8]
addZero(numbers)
puts(numbers)
```

**Output:**

```text
[7, 8, 0]
```

When you want an independent array, copy it with `list.copy`:

```
let a = [1, 2]
let b = list.copy(a)
push(b, 3)
puts(a, b)
```

**Output:**

```text
[1, 2] [1, 2, 3]
```

## Going through an array

```
let prices = [12, 5, 30, 8]
let total = 0
for (let i = 0; i < len(prices); i += 1) {
  total += prices[i]
}
puts("total:", total)
```

**Output:**

```text
total: 55
```

> **Common mistake: `<=` instead of `<`.** With `i <= len(xs)`, the last round reads `xs[len(xs)]`, which is `null`. Adding `null` to a number is then a type mismatch:
>
> ```
> let prices = [12, 5]
> let total = 0
> for (let i = 0; i <= len(prices); i += 1) {
>   total += prices[i]
> }
> ```
>
> **Output:**
>
> ```text
> [line 4, column 9] runtime error: type mismatch: INTEGER + NULL
> ```

## Arrays of arrays

An array can hold arrays, which gives you a grid:

```
let grid = [
  [1, 2, 3],
  [4, 5, 6]
]
puts(grid[1][0])
grid[0][2] = 30
puts(grid)
```

**Output:**

```text
4
[[1, 2, 30], [4, 5, 6]]
```

## The `list` library

`list` has more tools for arrays. You use them with a dot: `list.sort(a)`. A negative index counts from the end here.

```
let scores = [30, 10, 20]
puts(list.sort(scores))
puts(list.reverse(scores))
puts(scores)
```

**Output:**

```text
[10, 20, 30]
[20, 10, 30]
[30, 10, 20]
```

`sort` and `reverse` make **new** arrays; `scores` didn't change. `insert` and `remove` do change the array:

```
let letters = ["a", "c"]
list.insert(letters, 1, "b")
puts(letters)
list.remove(letters, 0)
puts(letters)
list.remove(letters, -1)
puts(letters)
```

**Output:**

```text
["a", "b", "c"]
["b", "c"]
["b"]
```

`index` finds an item, `slice` cuts a piece:

```
let colors = ["red", "green", "blue", "gray"]
puts(list.index(colors, "blue"), list.index(colors, "pink"))
puts(list.slice(colors, 1, 3))
puts(list.slice(colors, -2, 4))
```

**Output:**

```text
2 -1
["green", "blue"]
["blue", "gray"]
```

### Sorting your own way

Without a second argument, `list.sort` sorts numbers by value or strings alphabetically (all items must be of one kind). To choose the order yourself, pass a function `less(x, y)` that says whether `x` goes before `y`:

```
let nums = [5, 3, 8, 1]
puts(list.sort(nums, function(x, y) { x > y }))

let lizards = [{"name": "Iggy", "age": 7}, {"name": "Liz", "age": 3}]
let byAge = list.sort(lizards, function(p, q) { p.age < q.age })
puts(byAge[0].name, byAge[1].name)
```

**Output:**

```text
[8, 5, 3, 1]
Liz Iggy
```

(`p.age` is dot access on a hash, from chapter 9.) Mixing types without a function is an error:

```
puts(list.sort([3, "a"]))
```

**Output:**

```text
[line 1, column 15] runtime error: list.sort: without a comparison function the elements must be all numbers or all strings
```

> **Try it.** Write `maxOf(xs)` that returns the largest number of an array without using `list.sort`. Then write `average(xs)` that returns a float.

## Summary

- `[a, b, c]`; items start at index 0; a missing index gives `null`.
- `push` adds, `pop` removes, `len` counts.
- `a[i] = x` changes an existing item only.
- Arrays are shared: use `list.copy` for an independent one.
- `list.sort`, `list.reverse`, `list.slice` and `list.copy` make new arrays; `list.insert` and `list.remove` change the array.

[← Previous: Functions](07-functions.md) · [Index](../README.md) · [Next: Hashes and dot access →](09-hashes.md)
