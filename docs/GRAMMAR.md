# lizard-script grammar

EBNF notation: `{ x }` means zero or more, `[ x ]` means optional, and quoted text is a literal token.

## Statements

```ebnf
program        = { statement } ;

statement      = letStatement
               | functionDecl
               | returnStatement
               | breakStatement
               | controlStatement
               | layerStatement
               | expressionStatement
               | ";" ;

letStatement        = "let" IDENT "=" expression end ;
functionDecl        = "function" IDENT "(" [ parameters ] ")" block [ ";" ] ;
returnStatement     = "return" [ expression ] end ;
breakStatement      = "break" end ;
controlStatement    = ( ifExpression | whileExpression | forExpression ) [ ";" ] ;
layerStatement      = "layer" ( IDENT | STRING ) [ ";" ] { statement } ;
expressionStatement = expression end ;

block      = "{" { statement } "}" ;
parameters = IDENT { "," IDENT } ;
```

`end` is not a token. It is a `;`, or nothing when the next token is `}`, the end of the input, or on a later line. That is why `let x = 5 6` is an error while two statements on separate lines are fine.

## Expressions

The grammar is written by levels, one per row of the precedence table below. The parser doesn't have one function per level: the Pratt algorithm covers them all with `parseExpression` and the `precedences` table.

```ebnf
expression = assignment ;

assignment = logicalOr [ "=" assignment ] ;
logicalOr  = logicalAnd { "||" logicalAnd } ;
logicalAnd = equality { "&&" equality } ;
equality   = comparison { ( "==" | "!=" | "===" | "!==" ) comparison } ;
comparison = sum { ( "<" | ">" | "<=" | ">=" ) sum } ;
sum        = product { ( "+" | "-" ) product } ;
product    = prefix { ( "*" | "/" | "//" | "%" ) prefix } ;
prefix     = ( "-" | "!" ) prefix | power ;
power      = postfix [ "**" prefix ] ;
postfix    = primary { "(" [ arguments ] ")" | "[" expression "]" | "." IDENT } ;

primary    = IDENT | INT | FLOAT | STRING | COLOR | "true" | "false"
           | "(" expression ")"
           | pointLiteral
           | arrayLiteral
           | hashLiteral
           | functionLiteral
           | ifExpression
           | whileExpression
           | forExpression ;

arguments       = expression { "," expression } ;
pointLiteral    = "(" expression "," expression [ "," expression ] ")" ;
arrayLiteral    = "[" [ arguments ] "]" ;
hashLiteral     = "{" [ pair { "," pair } [ "," ] ] "}" ;
pair            = expression ":" expression ;
functionLiteral = "function" "(" [ parameters ] ")" block ;

ifExpression    = "if" expression block { "elif" expression block } [ "else" block ] ;
whileExpression = "while" expression block ;
forExpression   = "for" "(" [ forInit ] ";" [ expression ] ";" [ expression ] ")" block ;
forInit         = letStatement | expressionStatement ;
```

The left side of `=` must be an identifier or an index expression. The grammar above allows more, so the parser checks it and rejects things like `1 = 2`. `a.name` is `a["name"]`, the same tree, so it works wherever an index does: `a.b.c`, `a.b[0]`, `a.b(1)`, `f().x` and `a.b = 1`. After the `.` must come an identifier; a keyword, a number or anything else is the syntax error `expected a name after "."`. In a `for`, the `;` after `forInit` is the one that ends that statement.

## Precedence and associativity

From weakest to strongest:

| Level | Operators | Associativity |
| --- | --- | --- |
| `ASSIGN` | `=` | right |
| `OR` | `\|\|` | left |
| `AND` | `&&` | left |
| `EQUALS` | `==` `!=` `===` `!==` | left |
| `LESSGREATER` | `<` `>` `<=` `>=` | left |
| `SUM` | `+` `-` | left |
| `PRODUCT` | `*` `/` `//` `%` | left |
| `PREFIX` | `-x` `!x` | prefix |
| `EXPONENT` | `**` | right |
| `CALL` | `f(x)` | left |
| `INDEX` | `a[i]` `a.name` | left |

`**` is stronger than prefix operators, so `-2 ** 2` is `-(2 ** 2)`. Its right side can still start with one: `2 ** -1`.

## Lexical rules

```ebnf
IDENT   = ( letter | "_" ) { letter | digit | "_" } ;
INT     = digit { digit } [ "px" ] ;
FLOAT   = digit { digit } "." digit { digit } [ unit ]
        | digit { digit } ( "deg" | "rad" | "turn" ) ;
unit    = "deg" | "rad" | "turn" | "px" ;
STRING  = '"' { any character except '"' and '\' | escape } '"' ;
escape  = "\\" ( '"' | "\\" | "n" | "t" ) ;
COLOR   = "#" hex hex hex [ hex hex hex [ hex hex ] ] ;
hex     = digit | "a".."f" | "A".."F" ;
comment = "-~>" { any character except newline } ;
```

Whitespace and comments are skipped between tokens. Numbers are always decimal, so `010` is 10.

A float needs digits on both sides of the `.`: `1.5` and `10.0` are floats, `.5` is a `.` and an integer, and `1.` is an integer and a `.`, so `1.x` is dot access on an integer (an error when it runs). `1.5.x` is the float `1.5` and then dot access.

A unit must come right after the number, with no space: `45deg`, `1.5rad`, `0.25turn`, `10px`. `deg`, `rad` and `turn` make a FLOAT in radians (`45deg` is 0.785..., `1turn` is 2π), and `px` leaves the number as it is. The unit is only taken when the whole word is one of the four, so `5degx` is not a unit: it is the integer `5` and then the name `degx`, as before. When a number is directly followed by a name like that and a statement should end there, the parser says `unknown unit "kg"` instead of the plain "expected end of statement" (a syntax error, not a lexical one). A number too big for a float is the syntax error `number ... is too large`.

A color is `#` and then exactly 3, 6 or 8 hex digits: `#rgb`, `#rrggbb`, `#rrggbbaa`, in any case. The lexer takes the whole word after the `#`, so `#12`, `#1234`, `#12345` and `#12ab_z` are the lexical error `bad color #12` (with the text as written) and not a color followed by a name. A `#` with no word after it is still `unexpected character "#"`.

A point is `(x, y)` or `(x, y, z)`. A comma after the first expression inside parentheses is what makes it a point; `(x)` is still a grouped expression. `()` and four or more coordinates are the syntax error `a point has two or three coordinates`, reported at the `(`. A point is one argument: `f((1, 2))` passes a point, while `f(1, 2)` passes two numbers. A trailing comma, `(1, 2,)`, is an error.

## Layers

`layer` is not a reserved word. A statement is a layer when it starts with the identifier `layer` followed, on the same line, by a name (an identifier, or a string for a name with spaces) and then nothing else on that line except a `;` or a `-~>` comment. Two names side by side were never valid, so every program that worked before reads the same: `let layer = 1`, `layer + 1`, `layer = 2` and `f(layer)` use `layer` as the ordinary identifier it is.

The `{ statement }` in the rule is the body, and it is measured by columns, not by tokens:

- The body is every following statement that starts on a **later line** with its first token in a **column greater** than the column of `layer`. It ends at the first statement whose first token is in that column or a smaller one, at a `}` that closes the block around the layer, or at the end of the input. Further lines of a statement that is already open (`let x = 1 +` and the line after it) belong to that statement, whatever their column.
- The body may be empty.
- Layers nest: a `layer` inside a body is a child layer, and its own body is measured against its own column.
- Columns are the ones the lexer reports, 1-based, where a tab counts as one column like any other character. A file that mixes tabs and spaces is measured as it is read, and getting it to line up is up to the author. The editor writes 2 spaces.
- Inside a `{ }` block (a function body or a loop), a layer works the same way, measured from the column of its own `layer` word and not from the brace.
- After the name, anything else on the line is the syntax error `a layer is "layer name" alone on its line, got ... after the name`.

## Rules that are not in the grammar

- `break` must be inside a `while` or `for` of the same function. Otherwise it is a syntax error.
- `return` is allowed outside of a function, where it ends the program. What programs do when they run is in [SEMANTICS.md](SEMANTICS.md).
- `elif` has no node of its own: it is an `if` inside the `else` block of the previous one.
- `function add(x) { ... }` builds the same tree as `let add = function(x) { ... };`
- An `if`, `while` or `for` that starts a statement ends at its closing `}`, so the next line is a new statement. As part of a larger expression they do produce a value: `let x = if (a) { 1 } else { 2 };`
- A `{` where an expression is expected is always a hash. There are no standalone blocks.
- Nesting is limited to 10000 levels, where every expression, block and `if` inside another one counts as a level. That allows around 10000 parentheses or 3000 `if`s inside each other. Deeper code is the syntax error `the program is nested too deeply`, because the parser and the evaluator call themselves once per level and would otherwise crash.

## Known limitations

- Strings can span several lines. Inside a string, `\"` is a quote, `\\` a backslash, `\n` a line break and `\t` a tab. Any other backslash sequence, like `\q`, is the lexical error `unknown escape \q in string`, and a string that ends in a lone `\` is unterminated.
- Identifiers and comments are ASCII. A non-ASCII character in code is a lexical error, although strings and comments can contain it.
- `;` is optional at the end of a line, but a line break doesn't end an expression that can continue. A line starting with an operator, `(` or `[` continues the previous expression, so `x` followed by a line with `-1` is a subtraction.
- For the same reason, `return` takes the expression on the next line as its value, unless a `;` or `}` comes first.
- A layer's body is closed by indentation, the only place the language looks at columns. The REPL runs line by line, so a `layer` typed there is an empty layer and the indented lines after it run on their own; layers are meant for files and for hosts.
- The smallest 64-bit integer can't be written as a literal: `-9223372036854775808` is a `-` applied to a number that is too large.
- After a syntax error the rest of that statement is skipped, so a second mistake in the same statement shows up only after fixing the first. Recovery is a best effort: with a badly broken `for` header, like one with no `)`, the first error is right but a second one can follow it.
