# 4. Operators

[← Previous: Variables and scope](03-variables-and-scope.md) · [Index](../README.md) · [Next: Conditionals →](05-conditionals.md) · [Español](../es/04-operators.md)

Operators combine values: `+` adds, `<` compares, `&&` joins conditions. This chapter covers all of them and the few surprises they hide.

## Arithmetic

| Operator | Meaning |
| --- | --- |
| `+` `-` `*` | add, subtract, multiply |
| `/` | divide |
| `//` | divide, rounding **down** |
| `%` | remainder (modulo) |
| `**` | power |
| `-x` | negative |

```
puts(7 + 2, 7 - 2, 7 * 2)
puts(7 / 2, 7 // 2, 7 % 2)
puts(2 ** 10)
```

**Output:**

```text
9 5 14
3 3 1
1024
```

### Integer division vs float division

This is the most common surprise in the whole language. **Two integers always give an integer**, so `/` throws the decimals away:

```
puts(1 / 2)
puts(10 / 4)
```

**Output:**

```text
0
2
```

As soon as one side is a float, the result is a float and `/` is real division:

```
puts(1 / 2.0)
puts(10.0 / 4)
puts(float(10) / 4)
```

**Output:**

```text
0.5
2.5
2.5
```

> **Common mistake: an average that comes out wrong.**
>
> ```
> let total = 7
> let count = 2
> puts("average:", total / count)
> puts("average:", float(total) / count)
> ```
>
> **Output:**
>
> ```text
> average: 3
> average: 3.5
> ```

### `/` vs `//` with negative numbers

For positive numbers `/` and `//` give the same integer. They differ with negative numbers: `/` rounds **towards zero**, `//` rounds **down**:

```
puts(-7 / 2, -7 // 2)
puts(7 / 2, 7 // 2)
```

**Output:**

```text
-3 -4
3 3
```

`%` goes with `//`, so the remainder has the sign of the right side. This is what you want for things like "which day of the week":

```
puts(7 % 3, -7 % 3, 7 % -3)
```

**Output:**

```text
1 2 -2
```

Dividing by zero is always an error, also with `//` and `%`:

```
puts(5 % 0)
```

**Output:**

```text
[line 1, column 8] runtime error: division by zero
```

### Power

`**` groups from the **right**, and it is stronger than the minus sign in front:

```
puts(2 ** 3 ** 2)
puts(-2 ** 2)
puts((-2) ** 2)
```

**Output:**

```text
512
-4
4
```

`2 ** 3 ** 2` is `2 ** 9`, and `-2 ** 2` is `-(2 ** 2)`. With integers, a negative exponent is an error because the result wouldn't be an integer; use a float:

```
puts(2.0 ** -1)
puts(2 ** -1)
```

**Output:**

```text
0.5
[line 2, column 8] runtime error: negative exponent: 2 ** -1
```

### Operators on strings and arrays

`+` also joins two strings or two arrays, and `*` repeats a string:

```
puts("Liz" + "ard")
puts("ab" * 3)
puts([1, 2] + [3])
```

**Output:**

```text
Lizard
ababab
[1, 2, 3]
```

## Comparison

`<`, `>`, `<=`, `>=` compare two numbers, or two strings in dictionary order. Comparing different types is an error:

```
puts(3 < 5, 3 >= 5, 1.5 < 2)
puts("apple" < "banana")
puts(3 < "5")
```

**Output:**

```text
true false true
true
[line 3, column 8] runtime error: type mismatch: INTEGER < STRING
```

String order is by character code, so all uppercase letters come before all lowercase ones, and digits are compared one by one:

```
puts("Zebra" < "apple", "10" < "9")
```

**Output:**

```text
true true
```

## Equality: `==` vs `===`

lizard-script has two equalities:

- `===` is **exact**: same type and same value. `!==` is its opposite.
- `==` is **loose**: if the types differ, it first converts booleans and numeric strings to numbers. `!=` is its opposite.

```
puts(5 == "5", 5 === "5")
puts(1 == true, 1 === true)
puts(1 == 1.0, 1 === 1.0)
puts("five" == 5, null == 0, null == null)
```

**Output:**

```text
true false
true false
true false
false false true
```

Arrays and hashes are equal when their contents are, using the same kind of equality for the elements:

```
puts([1, 2] == [1, 2])
puts([1, "2"] == [1, 2], [1, "2"] === [1, 2])
puts({"a": 1} == {"a": 1})
```

**Output:**

```text
true
true false
true
```

Equality never fails, whatever you compare. **When in doubt, use `===`**: it never converts anything behind your back.

## Logic

| Operator | Is true when |
| --- | --- |
| `a && b` | both are true |
| `a \|\| b` | at least one is true |
| `!a` | `a` is false |

They always give `true` or `false`, and they work with truthiness (chapter 2):

```
puts(true && false, true || false, !true)
puts(5 && "hi", 0 || "")
```

**Output:**

```text
false true false
true false
```

`&&` and `||` **stop early**: the right side only runs when it's needed. That lets you protect an operation that could fail:

```
let d = 0
puts(d != 0 && 10 / d > 1)
```

**Output:**

```text
false
```

Since `d != 0` was false, `10 / d` never ran, so there was no division by zero.

## Compound assignment

`x += e` is short for `x = x + e`. The same goes for `-=`, `*=` and `/=`:

```
let score = 10
score += 5
score -= 3
score *= 2
score /= 4
puts(score)

let word = "ab"
word += "c"
word *= 2
puts(word)
```

**Output:**

```text
6
abcabc
```

Like `=`, the variable must already exist. There is no `**=`, `//=` or `%=`:

```
let x = 2
x **= 3
```

**Output:**

```text
[line 2, column 5] syntax error: expected an expression, got "="
```

Write `x = x ** 3` instead. Compound assignment also works on array items and hash keys (chapters 8 and 9).

## Precedence

When an expression mixes operators, the stronger ones go first. From weakest to strongest:

| Level | Operators | Groups from |
| --- | --- | --- |
| assignment | `=` `+=` `-=` `*=` `/=` | right |
| or | `\|\|` | left |
| and | `&&` | left |
| equality | `==` `!=` `===` `!==` | left |
| comparison | `<` `>` `<=` `>=` | left |
| sum | `+` `-` | left |
| product | `*` `/` `//` `%` | left |
| prefix | `-x` `!x` | |
| power | `**` | right |
| call, index | `f(x)` `a[i]` `a.name` | left |

```
puts(1 + 2 * 3, (1 + 2) * 3)
puts(10 - 4 - 3)
puts(true || false && false)
```

**Output:**

```text
7 9
3
true
```

When you're not sure, add parentheses. They cost nothing and make the code easier to read.

> **Try it.** Without running them, work out `17 // 5`, `17 % 5`, `-17 // 5`, `-17 % 5` and `2 * 3 ** 2`. Then check.

## Surprises that stay

These behaviours look odd at first. They are on purpose, so they won't change; learn them once.

**A negative index is `null` with `[]`, but counts from the end in `text` and `list`.** The `[]` operator never wraps around, so a bug that goes below 0 shows up as `null` instead of quietly reading the last item:

```
let a = [10, 20, 30]
puts(a[-1], "abc"[-1])
puts(text.slice("hello", -3, 5), list.slice(a, -2, 3))
```

**Output:**

```text
null null
llo [20, 30]
```

For the last item with `[]`, use `a[len(a) - 1]`.

**`"5" * 2` is `"55"`.** `*` repeats a string. A string that looks like a number is not a number for arithmetic (`"5" + 5` is an error), but the loose `==` does convert it. Use `int("5")` to get the number:

```
puts("5" * 2, "5" == 5, int("5") * 2)
puts("5" + 5)
```

**Output:**

```text
55 true 10
[line 2, column 10] runtime error: type mismatch: STRING + INTEGER
```

**Strings are compared byte by byte,** not by alphabet. Uppercase comes before lowercase, digits are compared one at a time, and accented letters come after `z`:

```
puts("B" < "a", "10" < "9", "é" < "z")
```

**Output:**

```text
true true false
```

## Summary

- Two integers give an integer: `1 / 2` is `0`. Make one side a float for real division.
- `/` rounds towards zero, `//` rounds down; `%` goes with `//`.
- `**` groups from the right and binds tighter than `-`.
- `===` is exact, `==` converts. Prefer `===`.
- `&&` and `||` stop early and always give a boolean.
- `+=`, `-=`, `*=`, `/=` exist; `**=`, `//=` and `%=` don't.
- Surprises that stay: `a[-1]` is `null`, `"5" * 2` is `"55"`, strings compare byte by byte.

[← Previous: Variables and scope](03-variables-and-scope.md) · [Index](../README.md) · [Next: Conditionals →](05-conditionals.md)
