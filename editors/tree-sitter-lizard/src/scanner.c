// External scanner of the lizard-script grammar.
//
// It does the three things that plain Tree-sitter rules can't:
//
//   TERMINATOR   the end of a statement. A line break ends a statement, and
//                nothing is needed before a `}` or at the end of the input.
//                The parser only asks for it where a statement can end, so a
//                line that stops after an operator, a comma or inside a
//                bracket never gets one.
//   HEADER_END   the end of the line `layer name` (and a `;` after it).
//   LAYER_END    nothing; the body of a layer is over. The body is every
//                statement on a later line with its first token in a column
//                greater than the column of the word `layer`.
//   *_KEYWORD    layer, use, try and continue are ordinary names in the
//                language. They are keywords only where the real parser
//                reads them as keywords, which is checked here.

#include "tree_sitter/parser.h"

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

// The same order as `externals` in grammar.js.
enum TokenType {
  TERMINATOR,
  HEADER_END,
  LAYER_KEYWORD,
  USE_KEYWORD,
  TRY_KEYWORD,
  CONTINUE_KEYWORD,
  LAYER_END,
  ERROR_SENTINEL,
};

// An open layer: the column of its word `layer` and whether its body can
// start (the header ended with a line break, so the next token is on a
// later line).
typedef struct {
  uint16_t column;
  bool body_ok;
} Layer;

#define MAX_LAYERS 200

typedef struct {
  uint16_t count;
  Layer layers[MAX_LAYERS];
} Scanner;

void *tree_sitter_lizard_external_scanner_create(void) {
  Scanner *scanner = calloc(1, sizeof(Scanner));
  return scanner;
}

void tree_sitter_lizard_external_scanner_destroy(void *payload) {
  free(payload);
}

unsigned tree_sitter_lizard_external_scanner_serialize(void *payload, char *buffer) {
  Scanner *scanner = payload;
  unsigned size = 0;
  for (uint16_t i = 0; i < scanner->count; i++) {
    buffer[size++] = (char)(scanner->layers[i].column & 0xff);
    buffer[size++] = (char)(scanner->layers[i].column >> 8);
    buffer[size++] = (char)scanner->layers[i].body_ok;
  }
  return size;
}

void tree_sitter_lizard_external_scanner_deserialize(void *payload, const char *buffer, unsigned length) {
  Scanner *scanner = payload;
  scanner->count = 0;
  for (unsigned i = 0; i + 2 < length && scanner->count < MAX_LAYERS; i += 3) {
    Layer *layer = &scanner->layers[scanner->count++];
    layer->column = (uint16_t)((unsigned char)buffer[i] | ((unsigned char)buffer[i + 1] << 8));
    layer->body_ok = buffer[i + 2] != 0;
  }
}

static void advance(TSLexer *lexer) { lexer->advance(lexer, false); }

// Moves one character. Before the token starts (`skipping`), the character
// is left out of the token. After mark_end it only looks ahead, and it must
// not skip: skipping moves the start of the token.
static void step(TSLexer *lexer, bool skipping) { lexer->advance(lexer, skipping); }

static bool is_name_start(int32_t c) {
  return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_';
}

static bool is_name_char(int32_t c) {
  return is_name_start(c) || (c >= '0' && c <= '9');
}

// At `-~>`? Looks at the comment and moves to the end of its line (not past
// the line break). If the `-` turns out to be something else it is already
// consumed, so this is only used to look ahead, or where a `-` can't be a
// token that matters.
static bool skip_comment(TSLexer *lexer, bool skipping) {
  if (lexer->lookahead != '-') return false;
  step(lexer, skipping);
  if (lexer->lookahead != '~') return false;
  step(lexer, skipping);
  if (lexer->lookahead != '>') return false;
  while (lexer->lookahead != '\n' && !lexer->eof(lexer)) step(lexer, skipping);
  return true;
}

// Skips spaces, tabs and carriage returns, but not line breaks.
static void skip_blanks(TSLexer *lexer, bool skipping) {
  while (lexer->lookahead == ' ' || lexer->lookahead == '\t' || lexer->lookahead == '\r') {
    step(lexer, skipping);
  }
}

// Skips blanks and line breaks, and stops at the first thing that is not
// one. A comment is not skipped: the scanner that returns a token has to
// leave comments to the parser, so they show up in the tree. Returns true if
// it crossed a line break.
static bool skip_lines(TSLexer *lexer, bool skipping) {
  bool crossed = false;
  for (;;) {
    int32_t c = lexer->lookahead;
    if (c == '\n') {
      crossed = true;
      step(lexer, skipping);
    } else if (c == ' ' || c == '\t' || c == '\r') {
      step(lexer, skipping);
    } else {
      return crossed;
    }
  }
}

// Like skip_lines, but it also skips comments, for looking ahead only. It
// stops at the first token. `column` gets the column of that token, and a
// lone `-` (the start of a token that is not a comment) is left unread
// only if it is the last thing, so its column is noted before it is read.
static void look_ahead(TSLexer *lexer, uint32_t *column) {
  for (;;) {
    skip_lines(lexer, false);
    *column = lexer->get_column(lexer);
    if (lexer->lookahead != '-') return;
    if (!skip_comment(lexer, false)) return;
  }
}

// After a line break: does the next line go on with an if (`else`, `elif`)?
// Skips blank lines and comments to find out.
static bool next_line_continues(TSLexer *lexer) {
  uint32_t column;
  look_ahead(lexer, &column);
  if (lexer->lookahead != 'e') return false;
  char word[8];
  int n = 0;
  while (is_name_char(lexer->lookahead) && n < 7) {
    word[n++] = (char)lexer->lookahead;
    advance(lexer);
  }
  word[n] = '\0';
  if (is_name_char(lexer->lookahead)) return false;
  return strcmp(word, "else") == 0 || strcmp(word, "elif") == 0;
}

static bool scan_terminator(TSLexer *lexer) {
  skip_blanks(lexer, true);
  if (lexer->lookahead == '\n') {
    advance(lexer);
    lexer->mark_end(lexer);
    // `else` or `elif` at the start of the next line goes on with an if.
    if (next_line_continues(lexer)) return false;
    lexer->result_symbol = TERMINATOR;
    return true;
  }
  // Nothing is needed before a `}` or at the end of the input.
  if (lexer->eof(lexer) || lexer->lookahead == '}') {
    lexer->mark_end(lexer);
    lexer->result_symbol = TERMINATOR;
    return true;
  }
  // A comment, or anything else: not the end of the statement (a comment is
  // read by the parser, and the scanner is asked again after it).
  return false;
}

// `layer name` ends at the end of its line, with an optional `;` and
// comment. The token is the `;` if there is one, otherwise empty.
static bool scan_header_end(Scanner *scanner, TSLexer *lexer) {
  skip_blanks(lexer, true);
  bool semicolon = false;
  if (lexer->lookahead == ';') {
    advance(lexer);
    semicolon = true;
  }
  lexer->mark_end(lexer);
  skip_blanks(lexer, false);
  if (lexer->lookahead == '-' && !skip_comment(lexer, false)) return false;
  bool body_ok;
  if (lexer->lookahead == '\n') {
    body_ok = true;
  } else if (lexer->eof(lexer) || lexer->lookahead == '}' || semicolon) {
    // Nothing else on the line (or `layer a; more`): the body can't start here.
    body_ok = false;
  } else {
    return false;
  }
  if (scanner->count > 0) scanner->layers[scanner->count - 1].body_ok = body_ok;
  lexer->result_symbol = HEADER_END;
  return true;
}

static bool scan_layer_end(Scanner *scanner, TSLexer *lexer) {
  // The token is empty: it ends where it starts.
  lexer->mark_end(lexer);
  Layer *layer = &scanner->layers[scanner->count - 1];
  uint32_t column;
  look_ahead(lexer, &column);
  bool ends = lexer->eof(lexer) || lexer->lookahead == '}' || !layer->body_ok ||
              column <= layer->column;
  if (!ends) return false;
  scanner->count--;
  lexer->result_symbol = LAYER_END;
  return true;
}

static bool scan_keyword(Scanner *scanner, TSLexer *lexer, const bool *valid) {
  skip_lines(lexer, true);
  uint32_t column = lexer->get_column(lexer);
  if (!is_name_start(lexer->lookahead)) return false;
  char word[10];
  int n = 0;
  while (is_name_char(lexer->lookahead)) {
    if (n >= (int)sizeof word - 1) return false;  // too long for a keyword
    word[n++] = (char)lexer->lookahead;
    advance(lexer);
  }
  word[n] = '\0';
  // The keyword is the word alone; what follows is only looked at.
  lexer->mark_end(lexer);
  skip_blanks(lexer, false);
  int32_t next = lexer->lookahead;

  if (valid[LAYER_KEYWORD] && strcmp(word, "layer") == 0) {
    // layer name, or layer "name"
    if (is_name_start(next) || next == '"') {
      if (scanner->count < MAX_LAYERS) {
        Layer *layer = &scanner->layers[scanner->count++];
        layer->column = (uint16_t)column;
        layer->body_ok = false;
      }
      lexer->result_symbol = LAYER_KEYWORD;
      return true;
    }
  } else if (valid[USE_KEYWORD] && strcmp(word, "use") == 0) {
    if (next == '"') {
      lexer->result_symbol = USE_KEYWORD;
      return true;
    }
  } else if (valid[TRY_KEYWORD] && strcmp(word, "try") == 0) {
    if (next == '{') {
      lexer->result_symbol = TRY_KEYWORD;
      return true;
    }
  } else if (valid[CONTINUE_KEYWORD] && strcmp(word, "continue") == 0) {
    bool alone = next == ';' || next == '}' || next == '\n' || lexer->eof(lexer);
    if (next == '-') alone = skip_comment(lexer, false);
    if (alone) {
      lexer->result_symbol = CONTINUE_KEYWORD;
      return true;
    }
  }
  return false;
}

bool tree_sitter_lizard_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid) {
  Scanner *scanner = payload;

  // While the parser recovers from an error every token is valid; do
  // nothing then, or an empty token could loop forever.
  if (valid[ERROR_SENTINEL]) return false;

  if (valid[LAYER_END] && scanner->count > 0) {
    if (scan_layer_end(scanner, lexer)) return true;
  }
  if (valid[HEADER_END]) {
    return scan_header_end(scanner, lexer);
  }
  if (valid[TERMINATOR]) {
    if (scan_terminator(lexer)) return true;
  }
  if (valid[LAYER_KEYWORD] || valid[USE_KEYWORD] || valid[TRY_KEYWORD] || valid[CONTINUE_KEYWORD]) {
    return scan_keyword(scanner, lexer, valid);
  }
  return false;
}
