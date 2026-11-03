; Highlights for lizard-script. Names follow the Tree-sitter capture
; conventions that editors know (@keyword, @function, @string, ...).

; Names. These come first so the rules after them win.
(identifier) @variable

; Comments
(comment) @comment

; Keywords
[
  "let"
  "function"
  "return"
  "break"
  "continue"
  "try"
  "catch"
  "layer"
  "use"
  "as"
] @keyword

[
  "if"
  "elif"
  "else"
  "while"
  "for"
] @keyword.control

; Values
(boolean) @boolean
((identifier) @constant.builtin
  (#eq? @constant.builtin "null"))

(number) @number
(unit) @type.builtin ; 90deg, 2px: the unit
(color) @constant.numeric
(string) @string
(string_content) @string
(escape_sequence) @string.escape

; Operators
[
  "=" "+=" "-=" "*=" "/="
  "+" "-" "*" "/" "//" "%" "**"
  "==" "!=" "===" "!=="
  "<" ">" "<=" ">="
  "&&" "||" "!"
] @operator

[ "," ";" ":" "." ] @punctuation.delimiter
[ "(" ")" "[" "]" "{" "}" ] @punctuation.bracket

; Functions
(function_declaration name: (identifier) @function)
(parameter name: (identifier) @variable.parameter)
(call_expression function: (identifier) @function.call)
(call_expression function: (member_expression property: (identifier) @function.method))

; Builtin functions (they exist without being declared)
((call_expression function: (identifier) @function.builtin)
  (#any-of? @function.builtin
    "len" "puts" "push" "pop" "first" "last" "rest" "keys" "values" "delete"
    "type" "str" "error" "int" "float" "range" "rgb" "rgba" "hsl" "point" "same"))

; Terrarium modules: math.sin, window.open, pen.line, ...
((member_expression object: (identifier) @module)
  (#any-of? @module
    "math" "text" "list" "hash" "json" "fs" "os" "time" "img" "clip" "io"
    "lizard" "window" "pen"))

; Properties: a.name, and the keys of a hash literal
(member_expression property: (identifier) @property)
(pair key: (identifier) @property)

; Layers and modules
(layer_statement name: (identifier) @label)
(layer_statement name: (string) @label)
(use_statement path: (string) @string.special.path)
(use_statement alias: (identifier) @module)

; The name in catch (e)
(try_statement error: (identifier) @variable.parameter)
