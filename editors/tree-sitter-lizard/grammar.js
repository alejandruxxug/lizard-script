// Tree-sitter grammar for lizard-script.
//
// It follows docs/GRAMMAR.md, LEP 0001 (a statement ends at the end of its
// line) and LEP 0002 (use). Three things in the language need more than
// plain rules, and the external scanner (src/scanner.c) does them:
//
//   * Newlines. A newline ends a statement, but only where a statement can
//     end. The scanner is asked for `_terminator` only in the places where
//     the grammar can accept one, so a line that ends with an operator, a
//     comma or an open bracket goes on by itself: no terminator is valid
//     there, and the newline is just a space.
//   * Layers. `layer name` owns the lines indented more than the word
//     `layer`. The scanner keeps a stack of those columns and says when a
//     body is over (`_layer_end`).
//   * Words that are not reserved: layer, use, try and continue. The scanner
//     only gives the keyword where the real parser would, so `let use = 1`
//     and `layer + 1` still parse as ordinary names.

const PREC = {
  assign: 1,
  or: 2,
  and: 3,
  equal: 4,
  compare: 5,
  sum: 6,
  product: 7,
  unary: 8,
  power: 9,
  call: 10,
};

// A comma separated list with at least one item.
function commaSep1(rule) {
  return seq(rule, repeat(seq(',', rule)));
}

function commaSep(rule) {
  return optional(commaSep1(rule));
}

module.exports = grammar({
  name: 'lizard',

  // Keep these in the same order as TokenType in src/scanner.c.
  externals: $ => [
    $._terminator, // the end of a statement: a line break, or nothing before } and the end
    $._header_end, // the end of the line of `layer name`
    $._layer_keyword,
    $._use_keyword,
    $._try_keyword,
    $._continue_keyword,
    $._layer_end, // nothing: the body of a layer is over
    $._error_sentinel, // only valid while the parser recovers from an error
  ],

  // Line breaks are spaces except where the scanner says otherwise.
  extras: $ => [/\s/, $.comment],

  word: $ => $.identifier,

  conflicts: $ => [
    // `if`, `while` and `for` at the start of a statement end at their `}`
    // with no terminator; elsewhere they are expressions.
    [$._control_statement, $._primary],
  ],

  rules: {
    source_file: $ => repeat($._top_statement),

    // `use` is only allowed at the top level of a file.
    _top_statement: $ => choice($.use_statement, $._statement),

    _statement: $ => choice(
      $.let_statement,
      $.function_declaration,
      $.return_statement,
      $.break_statement,
      $.continue_statement,
      $.try_statement,
      $._control_statement,
      $.layer_statement,
      $.expression_statement,
      $.empty_statement,
    ),

    // A statement ends at a `;` or at the end of its line.
    _end: $ => choice(';', $._terminator),

    empty_statement: _ => ';',

    let_statement: $ => seq($._let_binding, $._end),

    _let_binding: $ => seq(
      'let',
      field('name', $.identifier),
      '=',
      field('value', $._expression),
    ),

    function_declaration: $ => seq(
      'function',
      field('name', $.identifier),
      field('parameters', $.parameters),
      field('body', $.block),
    ),

    return_statement: $ => seq('return', optional($._expression), $._end),

    break_statement: $ => seq('break', $._end),

    // `continue` alone on its line; anywhere else it is a name.
    continue_statement: $ => seq(alias($._continue_keyword, 'continue'), $._end),

    try_statement: $ => seq(
      alias($._try_keyword, 'try'),
      field('body', $.block),
      'catch',
      optional(seq('(', field('error', $.identifier), ')')),
      field('handler', $.block),
    ),

    // An if, while or for that starts a statement ends at its `}`, so the
    // next statement may follow on the same line. A `;` after it is an
    // empty statement. It is shown as an expression statement, like it is
    // when the line ends after it.
    _control_statement: $ => alias(
      prec.dynamic(1, $._control_expression),
      $.expression_statement,
    ),

    _control_expression: $ => choice($.if_expression, $.while_expression, $.for_expression),

    expression_statement: $ => seq($._expression, $._end),

    // layer name
    //   statements indented more than the word layer
    layer_statement: $ => seq(
      alias($._layer_keyword, 'layer'),
      field('name', choice($.identifier, $.string)),
      $._header_end,
      repeat($._statement),
      $._layer_end,
    ),

    // use "path" [as name]
    use_statement: $ => seq(
      alias($._use_keyword, 'use'),
      field('path', $.string),
      optional(seq('as', field('alias', $.identifier))),
      $._end,
    ),

    block: $ => seq('{', repeat($._statement), '}'),

    parameters: $ => seq('(', commaSep($.parameter), ')'),

    parameter: $ => seq(
      field('name', $.identifier),
      optional(seq('=', field('default', $._expression))),
    ),

    // Expressions, from the weakest operator to the strongest.
    _expression: $ => choice(
      $.assignment_expression,
      $.binary_expression,
      $.unary_expression,
      $.call_expression,
      $.member_expression,
      $.subscript_expression,
      $._primary,
    ),

    assignment_expression: $ => prec.right(PREC.assign, seq(
      field('left', choice($.identifier, $.member_expression, $.subscript_expression)),
      field('operator', choice('=', '+=', '-=', '*=', '/=')),
      field('right', $._expression),
    )),

    binary_expression: $ => {
      const table = [
        [PREC.or, '||'],
        [PREC.and, '&&'],
        [PREC.equal, choice('==', '!=', '===', '!==')],
        [PREC.compare, choice('<', '>', '<=', '>=')],
        [PREC.sum, choice('+', '-')],
        [PREC.product, choice('*', '/', '//', '%')],
      ];
      const power = prec.right(PREC.power, seq(
        field('left', $._expression),
        field('operator', '**'),
        field('right', $._expression),
      ));
      return choice(
        power,
        ...table.map(([precedence, operator]) =>
          prec.left(precedence, seq(
            field('left', $._expression),
            field('operator', operator),
            field('right', $._expression),
          ))),
      );
    },

    unary_expression: $ => prec(PREC.unary, seq(
      field('operator', choice('-', '!')),
      field('argument', $._expression),
    )),

    call_expression: $ => prec(PREC.call, seq(
      field('function', $._expression),
      field('arguments', $.arguments),
    )),

    arguments: $ => seq('(', commaSep($._expression), ')'),

    // a.name is the same as a["name"]
    member_expression: $ => prec(PREC.call, seq(
      field('object', $._expression),
      '.',
      field('property', $.identifier),
    )),

    subscript_expression: $ => prec(PREC.call, seq(
      field('object', $._expression),
      '[',
      field('index', $._expression),
      ']',
    )),

    _primary: $ => choice(
      $.identifier,
      $.number,
      $.string,
      $.color,
      $.boolean,
      $.parenthesized_expression,
      $.point,
      $.array,
      $.hash,
      $.function_expression,
      $._control_expression,
    ),

    parenthesized_expression: $ => seq('(', $._expression, ')'),

    // (x, y) or (x, y, z). A comma after the first value is what makes it a point.
    point: $ => seq('(', $._expression, ',', $._expression, optional(seq(',', $._expression)), ')'),

    array: $ => seq('[', commaSep($._expression), ']'),

    hash: $ => seq('{', optional(seq(commaSep1($.pair), optional(','))), '}'),

    pair: $ => seq(field('key', $._expression), ':', field('value', $._expression)),

    function_expression: $ => seq(
      'function',
      field('parameters', $.parameters),
      field('body', $.block),
    ),

    if_expression: $ => seq(
      'if',
      field('condition', $._expression),
      field('consequence', $.block),
      repeat($.elif_clause),
      optional($.else_clause),
    ),

    // `elif` and `else if` are the same
    elif_clause: $ => seq(
      choice('elif', seq('else', 'if')),
      field('condition', $._expression),
      field('consequence', $.block),
    ),

    else_clause: $ => seq('else', field('alternative', $.block)),

    while_expression: $ => seq('while', field('condition', $._expression), field('body', $.block)),

    for_expression: $ => seq(
      'for',
      '(',
      optional(field('init', choice(alias($._let_binding, $.let_declaration), $._expression))),
      ';',
      optional(field('condition', $._expression)),
      ';',
      optional(field('update', $._expression)),
      ')',
      field('body', $.block),
    ),

    // Names, numbers and the other tokens.
    identifier: _ => /[A-Za-z_][A-Za-z0-9_]*/,

    // 12, 1.5, 1e3, 2.5e-4. A unit must touch the number: 45deg, 2px.
    number: $ => seq(
      token(seq(/[0-9]+/, optional(seq('.', /[0-9]+/)), optional(seq(/[eE]/, optional(/[+-]/), /[0-9]+/)))),
      optional(alias(token.immediate(/deg|rad|turn|px/), $.unit)),
    ),

    // #rgb, #rrggbb, #rrggbbaa
    color: _ => token(/#([0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{3})/),

    boolean: _ => choice('true', 'false'),

    // Strings can span lines.
    string: $ => seq(
      '"',
      repeat(choice(
        alias(token.immediate(prec(1, /[^"\\]+/)), $.string_content),
        $.escape_sequence,
      )),
      '"',
    ),

    escape_sequence: _ => token.immediate(seq('\\', choice(
      /["\\ntr]/,
      /u[0-9a-fA-F]{4}/,
      /u\{[0-9a-fA-F]{1,6}\}/,
    ))),

    comment: _ => token(seq('-~>', /[^\n]*/)),
  },
});
