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
               | expressionStatement
               | ";" ;

letStatement        = "let" IDENT "=" expression end ;
functionDecl        = "function" IDENT "(" [ parameters ] ")" block [ ";" ] ;
returnStatement     = "return" [ expression ] end ;
breakStatement      = "break" end ;
controlStatement    = ( ifExpression | whileExpression | forExpression ) [ ";" ] ;
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
postfix    = primary { "(" [ arguments ] ")" | "[" expression "]" } ;

primary    = IDENT | INT | STRING | "true" | "false"
           | "(" expression ")"
           | arrayLiteral
           | hashLiteral
           | functionLiteral
           | ifExpression
           | whileExpression
           | forExpression ;

arguments       = expression { "," expression } ;
arrayLiteral    = "[" [ arguments ] "]" ;
hashLiteral     = "{" [ pair { "," pair } [ "," ] ] "}" ;
pair            = expression ":" expression ;
functionLiteral = "function" "(" [ parameters ] ")" block ;

ifExpression    = "if" expression block { "elif" expression block } [ "else" block ] ;
whileExpression = "while" expression block ;
forExpression   = "for" "(" [ forInit ] ";" [ expression ] ";" [ expression ] ")" block ;
forInit         = letStatement | expressionStatement ;
```

The left side of `=` must be an identifier or an index expression. The grammar above allows more, so the parser checks it and rejects things like `1 = 2`. In a `for`, the `;` after `forInit` is the one that ends that statement.

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
| `INDEX` | `a[i]` | left |

`**` is stronger than prefix operators, so `-2 ** 2` is `-(2 ** 2)`. Its right side can still start with one: `2 ** -1`.

## Lexical rules

```ebnf
IDENT   = ( letter | "_" ) { letter | digit | "_" } ;
INT     = digit { digit } ;
STRING  = '"' { any character except '"' } '"' ;
comment = "-~>" { any character except newline } ;
```

Whitespace and comments are skipped between tokens. Integers are always decimal, so `010` is 10.

## Rules that are not in the grammar

- `break` must be inside a `while` or `for` of the same function. Otherwise it is a syntax error.
- `return` is allowed outside of a function. What it does there is up to the evaluator.
- `elif` has no node of its own: it is an `if` inside the `else` block of the previous one.
- `function add(x) { ... }` builds the same tree as `let add = function(x) { ... };`
- An `if`, `while` or `for` that starts a statement ends at its closing `}`, so the next line is a new statement. As part of a larger expression they do produce a value: `let x = if (a) { 1 } else { 2 };`
- A `{` where an expression is expected is always a hash. There are no standalone blocks.

## Known limitations

- Strings have no escape sequences, so they can't contain a `"`. They can span several lines.
- Identifiers and comments are ASCII. A non-ASCII character in code is a lexical error, although strings and comments can contain it.
- `;` is optional at the end of a line, but a line break doesn't end an expression that can continue. A line starting with an operator, `(` or `[` continues the previous expression, so `x` followed by a line with `-1` is a subtraction.
- After a syntax error the rest of that statement is skipped, so a second mistake in the same statement shows up only after fixing the first.
