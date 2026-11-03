# LEP 0003: Classes

- **Author(s):** the lizard-script team
- **Status:** Draft <!-- Draft · Accepted · Rejected · Deferred · Final -->
- **Created:** 2026-11-03
- **Discussion:** to be opened as a pull request

## Summary

`class Node { function init(name) { self.name = name } }` defines `Node`, a new kind of value, a **class**. Calling it, `Node("root")`, makes an **instance**: a new value with its own fields, on which `n.describe()` calls the class's method with `self` set to `n`. `class Sprite extends Node { ... }` takes every method of `Node` it doesn't define again, and `super.describe()` calls the one it replaced. There is one parent at most. Plain hashes don't change: `math.sqrt(16)` still passes only `16`.

## Motivation

Game nodes and editor objects are many values of the same kind, each with its own data and the same behavior. Today that is a hash made by a function, with every method a closure that knows its own hash, and inheritance by hand:

```
-~> A node is a hash; each method is a closure that knows its own node.
function Node(name) {
  let n = {"name": name, "children": []}
  n.add = function(child) { push(n.children, child); return n }
  n.describe = function() { n.name + " (" + str(len(n.children)) + " children)" }
  return n
}

-~> A sprite is a node with more fields; the base method is kept by hand.
function Sprite(name, pos) {
  let s = Node(name)
  s.pos = pos
  let base_describe = s.describe
  s.describe = function() { base_describe() + " at " + str(s.pos) }
  return s
}

let root = Node("root")
let liz = Sprite("liz", (10, 20))
root.add(liz)
puts(root.describe())
puts(liz.describe())
puts(len(keys(liz)), keys(liz))
puts(type(liz), same(Node("a").describe, Node("a").describe))
```

Today, with `./lizardscript`:

```
root (1 children)
liz (0 children) at (10, 20)
5 ["name", "children", "add", "describe", "pos"]
HASH false
```

It works, but:

- **Every instance carries a copy of every method.** A node with ten methods is ten closures per node; a thousand nodes are ten thousand closures, and the methods show up in `keys(liz)` and in `json.write(liz)` next to the data.
- **Inheritance is by hand.** `base_describe` must be saved before it is replaced, in the right order, in every subclass.
- **A typo is silent.** `liz.nmae` is `null`, like any missing hash key; the mistake shows up somewhere else, later.
- **Field reads are slow when there are many instances.** The VM's dot-access cache remembers one hash per place in the code. Reading `nodes[i].x` from 1000 different hashes misses it every time. Measured with `./lizardscript --vm` (2,000,000 reads, same loop, one hash repeated against 1000 hashes with the same keys):

```
one hash: 151.63 ms   1000 hashes: 196.835 ms
```

The other workaround, writing the methods once in a shared hash, needs the node written twice at every call, and forgetting it is an error far from the cause:

```
-~> Methods shared by every node, written once, with an explicit self.
let NodeMethods = {
  "rename": function(self, name) { self.name = name; return self },
  "describe": function(self) { "node " + self.name },
}
function new_node(name) { hash.merge({"name": name}, NodeMethods) }
let a = new_node("a")
a.rename(a, "b")         -~> the node is written twice
puts(a.describe(a))
a.describe()             -~> forgetting it is a runtime error
```

```
node b
[line 10, column 11] runtime error: wrong number of arguments: want 1, got 0
```

### Why `self` can't be automatic on plain hashes

The tempting fix is to make `h.f(x)` pass `h` as a hidden first argument whenever `h` is a hash. That breaks the language: every module is a hash of functions, and so is every hash of closures like the one above.

```
puts(type(math), type(fs), math.sqrt(16))
let shapes = {"square": function(n) { n * n }}
puts(shapes.square(3))
```

```
HASH HASH 4.0
9
```

With an automatic `self`, `math.sqrt(16)` would call `sqrt(math, 16)`, `fs.read(path)` would get the `fs` hash first, `shapes.square(3)` would be `wrong number of arguments`, and every module written in lizard-script (`json`, the ones made with `use ... as`) would break the same way. A hash can't tell "a method of this value" from "a function kept under this key". So the method call needs a value that knows it has methods: a class instance.

## The change

```
-~> with this LEP
class Node {
  function init(name) {
    self.name = name
    self.children = []
  }
  function add(child) {
    push(self.children, child)
    return self
  }
  function describe() { self.name + " (" + str(len(self.children)) + " children)" }
}

class Sprite extends Node {
  function init(name, pos) {
    super.init(name)
    self.pos = pos
  }
  function describe() { super.describe() + " at " + str(self.pos) }
}

let root = Node("root")
let liz = Sprite("liz", (10, 20))
root.add(liz)
puts(root.describe())          -~> root (1 children)
puts(liz.describe())           -~> liz (0 children) at (10, 20)
puts(keys(liz))                -~> ["name", "children", "pos"]
puts(type(liz), is(liz, Node)) -~> INSTANCE true
puts(liz)                      -~> Sprite {"name": "liz", "children": [], "pos": (10, 20)}
let d = liz.describe
puts(d())                      -~> liz (0 children) at (10, 20)
liz.nmae                       -~> [line 32, column 4] runtime error: Sprite has no field or method "nmae"
```

Grammar (for [GRAMMAR.md](../docs/GRAMMAR.md)):

```ebnf
classStatement = "class" IDENT [ "extends" postfix ] "{" { method } "}" [ ";" ] ;
method         = "function" IDENT "(" [ parameters ] ")" block [ ";" ] ;
superCall      = "super" "." IDENT "(" [ args ] ")" ;      (* only inside a method of a class that extends one *)
```

No new keyword, the same way as `layer` and `use`: a statement that starts with the identifier `class` followed on the same line by an identifier is a class statement (`class Node` was the syntax error `expected end of statement, got identifier "Node"`). `extends` is a word only right after the class name. The body holds only methods; anything else is the syntax error `a class body only has methods (function name() { ... })`. A class statement can be anywhere a statement can, like `function name() {}`.

Meaning (for [SEMANTICS.md](../docs/SEMANTICS.md)):

1. **A class is a value.** `class Node { ... }` defines `Node` in the current scope, like `let`. Its type is `CLASS`, it is shown as `class Node`, it is equal only to itself, and it can't be changed (`Node.x = 1` is `classes can't be changed`). It can be passed, returned and exported by a module like any value.
2. **Calling a class makes an instance.** `Node("root")` makes a new instance with no fields, runs `init` with `self` set to it and the call's arguments (defaults work as in any function), and gives the instance; what `init` returns is ignored. A class without `init` uses its parent's; with none in the chain it takes no arguments. A wrong count is the usual `wrong number of arguments`, at the call.
3. **Fields.** `self.name = v` (or `n.name = v` from outside) adds the field or changes it. `n.name` and `n["name"]` read it. Reading a name that is neither a field nor a method is the error `Node has no field or method "nmae"`, so typos are found where they are written. `keys(n)` and `values(n)` give the fields in the order they were added. A field name is a string; there is no `delete` of a field.
4. **Methods.** `n.add(c)` looks for `add` among `n`'s fields first (so a field can hold a function of its own, like a callback), then in `n`'s class, then up the parents. A method found that way is called with `self` set to `n`. Read without a call (`let f = n.add`), it is a function that keeps `n` as its `self` (a *bound method*), so `f(c)` and `list.sort(a, n.less)` work: unlike JavaScript, `self` is never lost.
5. **`self`** is a name that every method has, like a parameter that the call fills. A function written inside a method sees it like any other variable of the method (it's a closure), so `function(x) { self.size + x }` works inside a method. `self` can't be assigned or redefined in a method (syntax error `self can't be changed`). Outside class bodies, `self` is an ordinary name, as it is today.
6. **`extends` and `super`.** `class Sprite extends Node` reads `Node` when the class statement runs; it must be a class (`Sprite can't extend an INTEGER, only a class`). `extends m.Node` works with a class from `use "nodes" as m`. Inside a method, `super.name(args)` calls `name` as found starting at the parent of the class that wrote the method (not of `self`'s class), with the same `self`. `super` on its own or without a call is a syntax error (`super is only used as super.method(...)`), and so is `super` in a class without a parent. Outside methods `super` is an ordinary name.
7. **Equality and identity.** An instance is equal (`==` and `===`) only to itself, like a function, and `same(a, b)` agrees. It is always truthy, and it is not a hash key.
8. **Builtins.** `type(n)` is `"INSTANCE"` and `type(Node)` is `"CLASS"`. A new builtin `is(value, class)` says whether `value` is an instance of `class` or of one that extends it. `puts` and `str` show an instance as its class name and its fields, written like a hash, with `{...}` for an instance that contains itself.
9. **Both engines.** Classes, instances and bound methods made by one engine work in the other, and a Terrarium builtin that takes a function (`window.run`, `list.sort`) takes a bound method.

## How other languages do it

| Language | How objects are made | `self` | Inheritance and `super` | Fields |
|---|---|---|---|---|
| Lua | Tables with a metatable whose `__index` points at the "class" table. No class syntax, a convention. | `obj:m(x)` passes `obj`; `obj.m(x)` doesn't. Two call operators. | Chained `__index`; `Base.m(self, x)` by hand. | Any key, a typo reads `nil`. |
| Python | `class B(A):`, called to make an instance. | An explicit first parameter of every method, filled by `obj.m()`. Bound methods keep it. | Several parents (MRO); `super().m()`. | A dict per instance; a typo is an `AttributeError`. `__slots__` fixes the fields. |
| JavaScript | Prototypes; `class B extends A` since ES2015 is syntax over them. | `this` is implicit but decided by the call: `let f = obj.m; f()` loses it. | `extends`, `super.m()`, one parent. | Any key, a typo reads `undefined`. V8 makes field reads fast with hidden classes. |
| GDScript (Godot) | A file is a class (`class_name`, `extends Node2D`); `Thing.new()`. | Implicit; fields are read without `self.`. | One parent; `super.m()` (Godot 4). | Declared with `var`, optionally typed; a typo is an error. |
| Luau (Roblox) | Lua's metatables and `:`, and a type checker that understands the `setmetatable` class pattern. | `:` as in Lua. | By hand, as in Lua. | Sealed table types catch typos at check time. |
| **lizard-script** | `class B extends A { ... }`, called to make an instance. | Implicit in methods, always written `self.x`; bound methods keep it. | One parent; `super.m()`. | Added by assignment; a typo is an error when read. |

lizard-script is closest to GDScript and Python: one call operator (no Lua `:`), `self` that can't be lost (no JavaScript `this`), one parent. Fields are written `self.x`, not bare `x` as in GDScript, because a bare name already means a local or a global, which the VM decides at compile time; and fields need no declaration, so a class reads like the hash code it replaces.

## What breaks

Nothing that works today:

- `class` followed on the same line by a name was a syntax error (run above). `class` anywhere else (`let class = 1`, `class(x)`, `class + 1`) stays an ordinary name, and so do `extends`, `self` and `super` outside a class body.
- No `.liz` file in this repository (tests, examples, Terrarium libraries) uses the names `class`, `extends`, `self` or `super`, and no Go test string uses them as lizard code.
- `is` is a new builtin; a program's own `is` hides it, like `let len = 5`.
- Hashes, their dot access and their calls don't change at all.

## Implementation

- **ast:** `ClassStatement{Token, Name, Parent Expression, Methods []*FunctionLiteral}` (each method keeps its name), `SuperCall{Token, Method, Arguments}`. A method is a `FunctionLiteral` with `Method: true`, whose first, hidden parameter is `self`, so the evaluator and the compiler treat it as an ordinary function with one more parameter.
- **parser:** `startsClass`, `parseClassStatement`, `parseSuperCall`; a flag for "inside a method" and "this class has a parent" for the `self` and `super` errors. `lizard.parse` gets the kinds `class` and `super_call`; `lizard.docs` gives the doc comments of methods as `"Node.add"`.
- **object:** `Class{Name, Parent, Methods}`, `Instance{Class, Shape, Slots}`, `Shape` (below), and `BoundMethod{Self, Method}`, an `object.Callable`. `Inspect` of an instance uses the existing cycle-safe printing.
- **evaluator:** the class statement, calling a class, dot access and index on an instance (fields, then methods), assignment to a field, `SuperCall`, `is`, `type`, equality. A method call `n.m(x)` on an instance calls the method with `(n, x)`; on anything else, the call is what it was.
- **compiler and VM:** `OpClass` (methods and parent on the stack), `OpSuperCall method argc`, `OpCallMethod name argc` (a dot access followed by a call, so a method call makes no bound method), and `OpGetField`/`OpSetField` learn instances. `self` is slot 0 of a method's frame, so reading it costs what reading a parameter costs.
- **Rough size:** like modules (LEP 0002): a few hundred lines in each of parser, evaluator and VM, and their tests: a table of programs per package, programs in `tests/valid` and `tests/errors`, and the differential test of both engines.

### Fast fields: shapes

The dot-access cache of the VM (`code.FieldCache`) remembers one hash and its version, so it only helps when the same hash is read again (the measurement above). Instances can do better, the way V8 (hidden classes), SpiderMonkey and LuaJIT do:

- An instance keeps its fields in a slice, `Slots`, and a pointer to a **shape**: the list of its field names in order, a map from name to slot, and a map of *transitions*, the shape you get by adding one more name.
- Every instance starts at its class's empty shape. Adding `name`, then `children`, moves it along the same transitions as every other instance whose `init` adds the same fields in the same order, so all the nodes of a class share one shape. A field added later, or in another order, makes a branch; shapes only grow, because a field can't be deleted.
- A dot access on an instance caches **(shape, slot)** instead of (hash, version): the next read of any instance with that shape is one pointer comparison and a slice index, however many instances there are. The cache can hold a few shapes (four, like V8's polymorphic caches), so a loop over nodes and sprites stays fast; a place that sees more falls back to the map of the shape.
- Methods are cached the same way, by class: classes can't change, so a method found once for a class is that class's method forever.
- The evaluator uses the same shapes and slots (a map lookup in the shape), so both engines read the same data.

Formulas (`OpFormula`) can read instance fields as leaves, like hash fields today, so `self.x * scale + self.dx` stays one unboxed computation.

## Alternatives

- **Do nothing.** The closure pattern above works and stays. It costs a closure per method per instance, methods mixed with the data, inheritance by hand, silent typos and the slow field reads measured above.
- **Lua's `:` call operator on hashes.** `h:f(x)` would mean `h.f(h, x)`, with no new value kind. Small, but it brings two call operators and the rule of which one to use, methods still live in each hash (or in a shared hash merged by hand), there are no typo errors and no shapes. `:` would also be a third meaning of the colon (hash pairs, and named arguments in [LEP 0004](0004-named-arguments.md)).
- **Explicit `self` parameter (Python).** `function describe(self) { ... }` in the class body is clearer about where `self` comes from, but every method repeats it, and a learner who forgets it gets `wrong number of arguments`. GDScript, JavaScript, Kotlin and C# all leave it implicit.
- **Prototypes over hashes (JavaScript, Lua).** A hash with a `proto` key that dot access follows. No new value kind, but it changes what dot access does on every hash that has that key, and hashes keep their any-key, deletable layout, so no shapes.
- **A file is a class (GDScript).** It fits `use`, but one class per file is heavy for small node types, and a class is still needed as a value.
- **Bare field names (`name` instead of `self.name`).** Shorter, but a name would mean a local, a field or a global depending on the class, and the VM resolves names when it compiles, before it knows the instance.

## Open questions

- Should `type(n)` be the class name (`"Sprite"`) instead of `"INSTANCE"`? It reads well in `if (type(x) == "Sprite")`, but a class could be called `HASH`, and `is(x, Sprite)` already answers the question.
- Should instances be **sealed** after `init` (a new field outside `init` is an error), or should a class **declare** its fields? Either catches more typos (assignments too) and makes shapes certain, at the cost of more rules.
- Equality by identity is right for nodes; is a value class (a vector, where `==` compares fields) needed too? Should instances be usable as **hash keys** (a set of selected nodes)? Today only integers, booleans and strings are keys.
- How should `json.write`, `hash.copy` and the Terrarium builtins that take hashes (`window.open` options) treat an instance: as a hash of its fields, or as an error?
- Methods that `puts` and `str` use (`function text()`), and operators (`+` on two vectors): left for later, so this LEP stays small.
- Class-level values and functions (`Node.count`, `Node.make_root()`): not in this LEP; a module (`use`) holds them today.
- With named arguments ([LEP 0004](0004-named-arguments.md)), `Sprite(name: "liz", pos: (1, 2))` would bind to `init`'s parameters; both LEPs should agree on that.
