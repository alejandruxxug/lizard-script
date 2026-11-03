package terrarium

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"lizardscript/evaluator"
	"lizardscript/object"
	"lizardscript/terrarium/internal/arg"
	"os"
	"sync"
)

// io: the program's standard streams, for programs that talk to another
// program through them (a server that reads requests line by line) or
// that print without the line break of puts.

// Stdin is where io.read_line reads, and Stderr where io.error writes. A
// host can replace them; io.write writes where puts does
// (evaluator.Output), so the two keep their order.
var (
	Stdin  io.Reader = os.Stdin
	Stderr io.Writer = os.Stderr
)

// MaxLine is the longest line io.read_line reads: 32 MB.
const MaxLine = 32 << 20

// lineReader is the buffer over Stdin, made again when Stdin changes.
var (
	lineMu     sync.Mutex
	lineReader *bufio.Reader
	lineSource io.Reader
)

func ioModule() *object.Hash {
	return arg.Module("io", map[string]object.BuiltinFunction{
		"read_line": ioReadLine,
		"write":     ioWrite,
		"error":     ioError,
	})
}

// io.read_line() is the next line of the input without its line break (and
// without a "\r" before it), or null when the input has ended. A last line
// with no line break still counts.
func ioReadLine(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	line, err := readLine()
	if err != nil {
		return arg.Errorf("can't read the input: %v", err)
	}
	if line == nil {
		return arg.Null
	}
	return arg.Str(string(line))
}

// io.write(text) writes text where puts writes, with no line break.
func ioWrite(args ...object.Object) object.Object {
	return writeText(args, evaluator.Output)
}

// io.error(text) writes text to the error output, with no line break.
func ioError(args ...object.Object) object.Object {
	return writeText(args, Stderr)
}

// writeText writes the one string argument to w.
func writeText(args []object.Object, w io.Writer) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	text, err := arg.String(args, 0, "text")
	if err != nil {
		return err
	}
	if _, werr := io.WriteString(w, text); werr != nil {
		return arg.Errorf("can't write: %v", werr)
	}
	return arg.Null
}

// readLine reads up to the next "\n" of Stdin. nil means the input ended.
func readLine() ([]byte, error) {
	lineMu.Lock()
	defer lineMu.Unlock()
	if lineReader == nil || lineSource != Stdin {
		lineReader, lineSource = bufio.NewReaderSize(Stdin, 1<<16), Stdin
	}
	var line []byte
	for {
		chunk, err := lineReader.ReadSlice('\n')
		line = append(line, chunk...)
		if len(line) > MaxLine {
			return nil, fmt.Errorf("a line is longer than %d bytes", MaxLine)
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil, nil
			}
			break
		}
		if err != nil {
			return nil, err
		}
		break
	}
	line = bytes.TrimSuffix(line, []byte("\n"))
	return bytes.TrimSuffix(line, []byte("\r")), nil
}
