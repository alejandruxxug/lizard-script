; The outline and the breadcrumbs: functions, top-level lets and layers.
(function_declaration
  "function" @context
  name: (identifier) @name) @item

(source_file
  (let_statement
    "let" @context
    name: (identifier) @name) @item)

(layer_statement
  "layer" @context
  name: (_) @name) @item

(use_statement
  "use" @context
  path: (string) @name) @item
