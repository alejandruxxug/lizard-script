package object

import "strings"

// Joining two strings (s + t) copies both, so a string built piece by piece
// (s += piece, in a loop) costs the square of its length. Concat avoids it:
// a long string it makes lives in a buffer with room to grow, and when the
// left side is the newest string of its buffer, the right side is added to
// the buffer in place and the new string shares the bytes before it. The
// bytes a string already has never change, so sharing them is invisible;
// the buffer doubles when full, like a slice, so building a string of n
// characters copies about 2n of them in all.

// concatBuffered is the length from which Concat puts a string in a buffer;
// shorter ones are joined the plain way.
const concatBuffered = 64

// Concat is a + b as a new string.
func Concat(a, b *String) *String {
	length := len(a.Value) + len(b.Value)
	buffer := a.buffer
	switch {
	case buffer != nil && buffer.Len() == len(a.Value):
		// a is the newest string of its buffer.
		buffer.WriteString(b.Value)
	case length >= concatBuffered:
		buffer = &strings.Builder{}
		buffer.Grow(2 * length)
		buffer.WriteString(a.Value)
		buffer.WriteString(b.Value)
	default:
		return &String{Value: a.Value + b.Value}
	}
	joined := &String{Value: buffer.String(), buffer: buffer}
	// A string of ASCII characters stays one, and Len stays O(1).
	if a.scanned && a.ascii && isASCII(b.Value) {
		joined.scanned, joined.ascii = true, true
	}
	return joined
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// chars are the strings of one ASCII character, made once: what indexing
// an ASCII string gives. Their characters are filled in already, so reading
// one never writes to it, and sharing them is safe.
var chars = func() (table [128]String) {
	for i := range table {
		table[i] = String{Value: string(rune(i)), scanned: true, runes: []rune{rune(i)}}
	}
	return table
}()

// Char is the string of one character, shared when it is ASCII.
func Char(r rune) *String {
	if r >= 0 && r < 128 {
		return &chars[r]
	}
	return &String{Value: string(r)}
}

// KnownLen is Len when it is known without looking at the string again,
// and -1 before then; code that must not make calls can read it inline.
func (s *String) KnownLen() int {
	switch {
	case !s.scanned:
		return -1
	case s.ascii:
		return len(s.Value)
	}
	return len(s.runes)
}
