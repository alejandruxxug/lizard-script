; Scopes and names, for editors that highlight a name where it is used.
; A function (and a block) is a scope. `let`, parameters, function names,
; the catch name and the for-loop variable define names.

[
  (source_file)
  (function_declaration)
  (function_expression)
  (block)
  (layer_statement)
] @local.scope

(let_statement name: (identifier) @local.definition.var)
(let_declaration name: (identifier) @local.definition.var)
(function_declaration name: (identifier) @local.definition.function)
(parameter name: (identifier) @local.definition.parameter)
(try_statement error: (identifier) @local.definition.var)
(use_statement alias: (identifier) @local.definition.import)

(identifier) @local.reference
