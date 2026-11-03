; Highlights for Zed, from editors/tree-sitter-lizard/queries/highlights.scm
; with the capture names Zed's themes know (@type for modules, @constant
; for builtin values, @function for builtin functions). The language server
; adds what a grammar can't tell, like which names are parameters.

; Names. These come first so the rules after them win.
(identifier) @variable

(comment) @comment

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
  "if"
  "elif"
  "else"
  "while"
  "for"
] @keyword

; Values
(boolean) @boolean
((identifier) @constant
  (#eq? @constant "null"))

(number) @number
(unit) @type ; 90deg, 2px: the unit
(color) @constant
(string) @string
(string_content) @string
(escape_sequence) @string.escape

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
(call_expression function: (identifier) @function)
(call_expression function: (member_expression property: (identifier) @function))

((call_expression function: (identifier) @function.builtin)
  (#any-of? @function.builtin
    "len" "puts" "push" "pop" "first" "last" "rest" "keys" "values" "delete"
    "type" "str" "error" "int" "float" "range" "rgb" "rgba" "hsl" "point" "same"))

; Terrarium modules: math.sin, window.open, ...
((member_expression object: (identifier) @type)
  (#any-of? @type
    "math" "text" "list" "hash" "json" "fs" "os" "time" "img" "clip" "io"
    "lizard" "window" "pen"))

; Properties: a.name, and the keys of a hash literal
(member_expression property: (identifier) @property)
(pair key: (identifier) @property)

; Layers and modules
(layer_statement name: (identifier) @label)
(layer_statement name: (string) @label)
(use_statement path: (string) @string.special)
(use_statement alias: (identifier) @type)

(try_statement error: (identifier) @variable.parameter)
