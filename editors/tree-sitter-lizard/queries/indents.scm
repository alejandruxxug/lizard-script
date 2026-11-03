; Indentation for lizard-script. A block, a bracket pair and the body of a
; layer open a level.

[
  (block)
  (arguments)
  (parameters)
  (array)
  (hash)
  (point)
  (parenthesized_expression)
  (layer_statement)
] @indent.begin

[ "}" ")" "]" ] @indent.end

[ "}" ")" "]" ] @indent.branch

; else, elif and catch line up with the if or try that they continue
[ "else" "elif" "catch" ] @indent.branch
