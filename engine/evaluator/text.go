package evaluator

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// textHash is the value of the name text in the current run. Like mathHash,
// it is made on first use and dropped by ResetMath.
var textHash *object.Hash

// textLibrary returns the value of the name text, a hash of builtins for
// cutting and joining strings. Indexes count characters, not bytes. A
// variable called text hides it.
func textLibrary() *object.Hash {
	if textHash != nil {
		return textHash
	}
	hash := object.NewHash()
	setLibrary(hash, "text.", map[string]object.BuiltinFunction{
		"slice":   textSlice,
		"find":    textFind,
		"split":   textSplit,
		"join":    textJoin,
		"replace": textReplace,
		"trim":    textTrim,
		"lines":   textLines,
		"starts":  textStarts,
		"ends":    textEnds,
		"upper":   textUpper,
		"lower":   textLower,
		"code":    textCode,
		"char":    textChar,
		"number":  textNumber,
		"digest":  textDigest,
	})
	textHash = hash
	return hash
}

// checkCount fails when a builtin got fewer than min or more than max
// arguments.
func checkCount(args []object.Object, min, max int) *object.Error {
	if len(args) >= min && len(args) <= max {
		return nil
	}
	want := strconv.Itoa(min)
	if max != min {
		want += " to " + strconv.Itoa(max)
	}
	return wrongCount(want, len(args))
}

// stringArg reads argument i as a string.
func stringArg(args []object.Object, i int) (string, *object.Error) {
	s, ok := args[i].(*object.String)
	if !ok {
		return "", argError("argument %d must be STRING, got %s", i+1, args[i].Type())
	}
	return s.Value, nil
}

// integerArg reads argument i as an integer.
func integerArg(args []object.Object, i int) (int64, *object.Error) {
	n, ok := args[i].(*object.Integer)
	if !ok {
		return 0, argError("argument %d must be INTEGER, got %s", i+1, args[i].Type())
	}
	return n.Value, nil
}

// arrayArg reads argument i as an array.
func arrayArg(args []object.Object, i int) (*object.Array, *object.Error) {
	a, ok := args[i].(*object.Array)
	if !ok {
		return nil, argError("argument %d must be ARRAY, got %s", i+1, args[i].Type())
	}
	return a, nil
}

// clampIndex turns an index that may be negative (counting from the end)
// into one between 0 and length, both included.
func clampIndex(i int64, length int) int {
	if i < 0 {
		i += int64(length)
	}
	if i < 0 {
		return 0
	}
	if i > int64(length) {
		return length
	}
	return int(i)
}

// stringAndIntegers reads a string followed by count integers.
func stringAndIntegers(args []object.Object, count int) (string, []int64, *object.Error) {
	s, numbers, err := stringObjectAndIntegers(args, count)
	if err != nil {
		return "", nil, err
	}
	return s.Value, numbers, nil
}

// stringObjectAndIntegers is stringAndIntegers that keeps the string object,
// so helpers that work by character index can use its cached characters.
func stringObjectAndIntegers(args []object.Object, count int) (*object.String, []int64, *object.Error) {
	if err := checkCount(args, count+1, count+1); err != nil {
		return nil, nil, err
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return nil, nil, argError("argument 1 must be STRING, got %s", args[0].Type())
	}
	numbers := make([]int64, count)
	for i := range numbers {
		var err *object.Error
		if numbers[i], err = integerArg(args, i+1); err != nil {
			return nil, nil, err
		}
	}
	return s, numbers, nil
}

// text.slice(s, start, end) is the characters from start up to end. Both
// are clamped to the string, and a negative one counts from the end.
func textSlice(args ...object.Object) object.Object {
	s, n, err := stringObjectAndIntegers(args, 2)
	if err != nil {
		return err
	}
	length := s.Len()
	start, end := clampIndex(n[0], length), clampIndex(n[1], length)
	if end < start {
		end = start
	}
	if length == len(s.Value) { // all ASCII: characters are bytes
		return &object.String{Value: s.Value[start:end]}
	}
	return &object.String{Value: string(s.Runes()[start:end])}
}

// text.find(s, sub, [from]) is the character index of the first sub at or
// after from, or -1.
func textFind(args ...object.Object) object.Object {
	if err := checkCount(args, 2, 3); err != nil {
		return err
	}
	str, ok := args[0].(*object.String)
	if !ok {
		return argError("argument 1 must be STRING, got %s", args[0].Type())
	}
	sub, err := stringArg(args, 1)
	if err != nil {
		return err
	}
	s := str.Value
	length := str.Len()
	from := 0
	if len(args) == 3 {
		n, err := integerArg(args, 2)
		if err != nil {
			return err
		}
		from = clampIndex(n, length)
	}
	if length == len(s) { // all ASCII: characters are bytes
		at := strings.Index(s[from:], sub)
		if at < 0 {
			return &object.Integer{Value: -1}
		}
		return &object.Integer{Value: int64(from + at)}
	}
	rest := string(str.Runes()[from:])
	at := strings.Index(rest, sub)
	if at < 0 {
		return &object.Integer{Value: -1}
	}
	return &object.Integer{Value: int64(from + utf8.RuneCountInString(rest[:at]))}
}

// stringArgs checks that there are exactly count arguments, all strings,
// and returns them.
func stringArgs(args []object.Object, count int) ([]string, *object.Error) {
	if err := checkCount(args, count, count); err != nil {
		return nil, err
	}
	out := make([]string, count)
	for i := range out {
		var err *object.Error
		if out[i], err = stringArg(args, i); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// stringList wraps strings as an array of lizard strings.
func stringList(parts []string) object.Object {
	elements := make([]object.Object, len(parts))
	for i, part := range parts {
		elements[i] = &object.String{Value: part}
	}
	return &object.Array{Elements: elements}
}

// text.split(s, sep) cuts s at every sep. An empty sep gives the characters.
func textSplit(args ...object.Object) object.Object {
	a, err := stringArgs(args, 2)
	if err != nil {
		return err
	}
	return stringList(strings.Split(a[0], a[1]))
}

// text.join(list, sep) joins a list of strings with sep between them.
func textJoin(args ...object.Object) object.Object {
	if err := checkCount(args, 2, 2); err != nil {
		return err
	}
	list, err := arrayArg(args, 0)
	if err != nil {
		return err
	}
	sep, err := stringArg(args, 1)
	if err != nil {
		return err
	}
	parts := make([]string, len(list.Elements))
	for i, element := range list.Elements {
		s, ok := element.(*object.String)
		if !ok {
			return argError("element %d of the list must be STRING, got %s", i, element.Type())
		}
		parts[i] = s.Value
	}
	return &object.String{Value: strings.Join(parts, sep)}
}

// text.replace(s, old, new) replaces every old in s. An empty old is an
// error, since it would match between every two characters.
func textReplace(args ...object.Object) object.Object {
	a, err := stringArgs(args, 3)
	if err != nil {
		return err
	}
	if a[1] == "" {
		return argError("the text to replace can't be empty")
	}
	return &object.String{Value: strings.ReplaceAll(a[0], a[1], a[2])}
}

// text.trim(s) removes the white space around s.
func textTrim(args ...object.Object) object.Object {
	a, err := stringArgs(args, 1)
	if err != nil {
		return err
	}
	return &object.String{Value: strings.TrimSpace(a[0])}
}

// text.lines(s) cuts s at every line break, "\n" or "\r\n". Empty lines
// stay, so a string that ends in a line break ends in an empty line, and
// joining the lines with "\n" gives the text back.
func textLines(args ...object.Object) object.Object {
	a, err := stringArgs(args, 1)
	if err != nil {
		return err
	}
	return stringList(strings.Split(strings.ReplaceAll(a[0], "\r\n", "\n"), "\n"))
}

// text.starts(s, prefix) and text.ends(s, suffix).
func textStarts(args ...object.Object) object.Object {
	a, err := stringArgs(args, 2)
	if err != nil {
		return err
	}
	return nativeBoolToBooleanObject(strings.HasPrefix(a[0], a[1]))
}

func textEnds(args ...object.Object) object.Object {
	a, err := stringArgs(args, 2)
	if err != nil {
		return err
	}
	return nativeBoolToBooleanObject(strings.HasSuffix(a[0], a[1]))
}

// text.upper(s) and text.lower(s).
func textUpper(args ...object.Object) object.Object {
	a, err := stringArgs(args, 1)
	if err != nil {
		return err
	}
	return &object.String{Value: strings.ToUpper(a[0])}
}

func textLower(args ...object.Object) object.Object {
	a, err := stringArgs(args, 1)
	if err != nil {
		return err
	}
	return &object.String{Value: strings.ToLower(a[0])}
}

// text.code(s, i) is the character code of the character at index i (a
// negative index counts from the end), or null when there is no such
// character.
func textCode(args ...object.Object) object.Object {
	s, n, err := stringObjectAndIntegers(args, 1)
	if err != nil {
		return err
	}
	i := n[0]
	if i < 0 {
		i += int64(s.Len())
	}
	if i < 0 || i >= int64(s.Len()) {
		return NULL
	}
	return &object.Integer{Value: int64(s.RuneAt(int(i)))}
}

// text.char(code) is the one-character string with that character code.
func textChar(args ...object.Object) object.Object {
	if err := checkCount(args, 1, 1); err != nil {
		return err
	}
	code, err := integerArg(args, 0)
	if err != nil {
		return err
	}
	if code < 0 || code > utf8.MaxRune || (code >= 0xD800 && code <= 0xDFFF) {
		return argError("%d is not a character code", code)
	}
	return &object.String{Value: string(rune(code))}
}

// numberText is what text.number accepts: an optional sign, digits with an
// optional fraction or exponent. It leaves out the words and the hexadecimal
// forms that strconv would also take.
var numberText = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

// text.number(s) is the number written in s, an INTEGER when it has no
// fraction or exponent and fits, or a FLOAT. It is null when s is not a
// number, and never an error (except for a non-string argument).
func textNumber(args ...object.Object) object.Object {
	a, err := stringArgs(args, 1)
	if err != nil {
		return err
	}
	s := strings.TrimSpace(a[0])
	if !numberText.MatchString(s) {
		return NULL
	}
	if n, parseErr := strconv.ParseInt(s, 10, 64); parseErr == nil {
		return &object.Integer{Value: n}
	}
	f, parseErr := strconv.ParseFloat(s, 64)
	if parseErr != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return NULL
	}
	return &object.Float{Value: f}
}
