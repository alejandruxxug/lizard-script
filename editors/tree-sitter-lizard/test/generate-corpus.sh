#!/bin/sh
# Writes test/corpus/programs.txt and test/corpus/error_programs.txt from the
# programs of this repository, then asks tree-sitter to fill in the trees:
#
#   sh test/generate-corpus.sh        (tree-sitter must be on the PATH,
#                                      or set TREE_SITTER=/path/to/it)
#
# programs.txt has one test per valid program (tests/valid and examples).
# error_programs.txt has the syntax errors that a grammar can see. Some
# errors of tests/errors are checks of the parser that are not in the
# grammar (break outside a loop, a default before a parameter without one,
# a number that is too large, nesting that is too deep), so they are not
# listed.
#
# The trees in the corpus are written by `tree-sitter test --update`, so read
# the diff before keeping it: a changed tree is a changed grammar.

set -e
cd "$(dirname "$0")/.."
root=../..
TS=${TREE_SITTER:-tree-sitter}
bar='========================================'
dash='----------------------------------------'

# one_test file name [:error]
one_test() {
  printf '%s\n%s\n' "$bar" "$2"
  [ -n "$3" ] && printf '%s\n' "$3"
  printf '%s\n\n' "$bar"
  cat "$1"
  printf '\n%s\n\n' "$dash"
}

: > test/corpus/programs.txt
for f in "$root"/tests/valid/*.liz "$root"/tests/valid/modules/*.liz \
         "$root"/tests/valid/modules/shapes/*.liz "$root"/examples/*.liz \
         "$root"/examples/terrarium/*.liz "$root"/terrarium/lib/*.liz; do
  [ -f "$f" ] || continue
  one_test "$f" "${f#$root/}" >> test/corpus/programs.txt
done

: > test/corpus/error_programs.txt
for name in lexical_bad_color lexical_bad_unicode_escape lexical_character \
            lexical_non_ascii lexical_single_ampersand lexical_unknown_escape \
            lexical_unterminated_string syntax_assign_to_literal \
            syntax_dot_without_name syntax_else_if_missing_block \
            syntax_four_coordinates syntax_layer_extra_tokens \
            syntax_let_without_name syntax_line_starts_with_operator \
            syntax_missing_brace syntax_missing_paren syntax_several_errors \
            syntax_try_without_catch syntax_two_statements_one_line \
            syntax_unknown_unit syntax_unknown_unit_in_array \
            syntax_unknown_unit_in_call; do
  one_test "$root/tests/errors/$name.liz" "tests/errors/$name.liz" :error \
    >> test/corpus/error_programs.txt
done

"$TS" test --update
