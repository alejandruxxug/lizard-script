package task

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
)

// The wire: how a value crosses between the program and a task, which runs
// in a process of its own. A value is written as bytes and read back as a
// new value, so the two sides never share one: that is the deep copy. Only
// data can cross (numbers, strings, booleans, null, lists, hashes, points
// and colors); anything else is an error that says where it was found.
//
// Each value is a tag byte and what follows it: an integer as a varint, a
// float as its 8 bytes, a string as its length and bytes, a list or a point
// as its length and items, a hash as its length and key/value pairs, a
// color as its 4 bytes.

const (
	tagNull byte = iota
	tagFalse
	tagTrue
	tagInt
	tagFloat
	tagString
	tagList
	tagHash
	tagPoint
	tagColor
)

// dataKinds names what may cross, for the error of what may not.
const dataKinds = "numbers, strings, booleans, null, lists, hashes, points and colors"

// notData is the error of a value that can't cross: what it is and where
// it was found, from the outermost value in. The path is built on the way
// out, so a value that crosses fine costs nothing for it.
type notData struct {
	path []string // innermost first
	what string
}

func (e *notData) Error() string {
	var where strings.Builder
	for i := len(e.path) - 1; i >= 0; i-- {
		where.WriteString(e.path[i])
	}
	return where.String() + " is " + e.what
}

// inside adds the step from a value to the one inside it that failed.
func inside(err error, step string) error {
	var nd *notData
	if errors.As(err, &nd) {
		nd.path = append(nd.path, step)
	}
	return err
}

// encode writes value to buf, after what it has. root names the value in
// an error, like "args" or "the result".
func encode(buf []byte, value object.Object, root string) ([]byte, error) {
	e := encoder{buf: buf, open: map[object.Object]bool{}}
	if err := e.value(value); err != nil {
		return nil, inside(err, root)
	}
	return e.buf, nil
}

// encoder writes values. open holds the lists and hashes being written,
// to find one that holds itself.
type encoder struct {
	buf  []byte
	open map[object.Object]bool
}

func (e *encoder) value(v object.Object) error {
	switch v := v.(type) {
	case *object.Null:
		e.buf = append(e.buf, tagNull)
	case *object.Boolean:
		if v.Value {
			e.buf = append(e.buf, tagTrue)
		} else {
			e.buf = append(e.buf, tagFalse)
		}
	case *object.Integer:
		e.buf = append(e.buf, tagInt)
		e.buf = binary.AppendVarint(e.buf, v.Value)
	case *object.Float:
		e.buf = append(e.buf, tagFloat)
		e.buf = binary.LittleEndian.AppendUint64(e.buf, math.Float64bits(v.Value))
	case *object.String:
		e.buf = append(e.buf, tagString)
		e.buf = binary.AppendUvarint(e.buf, uint64(len(v.Value)))
		e.buf = append(e.buf, v.Value...)
	case *object.Color:
		e.buf = append(e.buf, tagColor, v.R, v.G, v.B, v.A)
	case *object.Point:
		e.buf = append(e.buf, tagPoint)
		return e.items(v.Coords)
	case *object.Array:
		if e.open[v] {
			return &notData{what: "a list that holds itself"}
		}
		e.open[v] = true
		defer delete(e.open, v)
		e.buf = append(e.buf, tagList)
		return e.items(v.Elements)
	case *object.Hash:
		if e.open[v] {
			return &notData{what: "a hash that holds itself"}
		}
		e.open[v] = true
		defer delete(e.open, v)
		return e.hash(v)
	default:
		return &notData{what: fmt.Sprintf("%s, and only data crosses to and from a task (%s)",
			named(v), dataKinds)}
	}
	return nil
}

// items writes the length and the items of a list or a point.
func (e *encoder) items(items []object.Object) error {
	e.buf = binary.AppendUvarint(e.buf, uint64(len(items)))
	for i, item := range items {
		if err := e.value(item); err != nil {
			return inside(err, fmt.Sprintf("[%d]", i))
		}
	}
	return nil
}

// hash writes a hash, its pairs in order.
func (e *encoder) hash(h *object.Hash) error {
	e.buf = append(e.buf, tagHash)
	e.buf = binary.AppendUvarint(e.buf, uint64(h.Len()))
	var err error
	h.Each(func(pair object.HashPair) {
		if err != nil {
			return
		}
		if err = e.value(pair.Key); err != nil {
			return
		}
		if err = e.value(pair.Value); err != nil {
			err = inside(err, "["+arg.Show(pair.Key)+"]")
		}
	})
	return err
}

// named is how a message names v: "a FUNCTION", or for a native its kind,
// "an image".
func named(v object.Object) string {
	if _, ok := v.(*object.Native); ok {
		return arg.Name(v)
	}
	return "a " + arg.Name(v)
}

// errCorrupt is the error of bytes that are not a value: a task's process
// that broke off in the middle of its answer.
var errCorrupt = errors.New("the bytes of a value are broken")

// decode reads one value from data, and returns it with the bytes after it.
func decode(data []byte) (object.Object, []byte, error) {
	d := decoder{data: data}
	v, err := d.value()
	return v, d.data, err
}

// decoder reads values from the front of data.
type decoder struct {
	data []byte
}

func (d *decoder) value() (object.Object, error) {
	if len(d.data) == 0 {
		return nil, errCorrupt
	}
	tag := d.data[0]
	d.data = d.data[1:]
	switch tag {
	case tagNull:
		return arg.Null, nil
	case tagFalse:
		return arg.False, nil
	case tagTrue:
		return arg.True, nil
	case tagInt:
		n, size := binary.Varint(d.data)
		if size <= 0 {
			return nil, errCorrupt
		}
		d.data = d.data[size:]
		return &object.Integer{Value: n}, nil
	case tagFloat:
		if len(d.data) < 8 {
			return nil, errCorrupt
		}
		f := math.Float64frombits(binary.LittleEndian.Uint64(d.data))
		d.data = d.data[8:]
		return &object.Float{Value: f}, nil
	case tagString:
		n, err := d.length(1)
		if err != nil {
			return nil, err
		}
		s := string(d.data[:n])
		d.data = d.data[n:]
		return &object.String{Value: s}, nil
	case tagColor:
		if len(d.data) < 4 {
			return nil, errCorrupt
		}
		c := &object.Color{R: d.data[0], G: d.data[1], B: d.data[2], A: d.data[3]}
		d.data = d.data[4:]
		return c, nil
	case tagPoint:
		coords, err := d.items()
		if err != nil {
			return nil, err
		}
		return &object.Point{Coords: coords}, nil
	case tagList:
		elements, err := d.items()
		if err != nil {
			return nil, err
		}
		return &object.Array{Elements: elements}, nil
	case tagHash:
		return d.hash()
	}
	return nil, errCorrupt
}

// length reads the length of something whose items take at least least
// bytes each, and checks that the data has room for it, so broken bytes
// never ask for a huge slice.
func (d *decoder) length(least int) (int, error) {
	n, size := binary.Uvarint(d.data)
	if size <= 0 {
		return 0, errCorrupt
	}
	d.data = d.data[size:]
	if n > uint64(len(d.data)/least) {
		return 0, errCorrupt
	}
	return int(n), nil
}

// items reads the length and the items of a list or a point.
func (d *decoder) items() ([]object.Object, error) {
	n, err := d.length(1)
	if err != nil {
		return nil, err
	}
	items := make([]object.Object, n)
	for i := range items {
		if items[i], err = d.value(); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// hash reads the pairs of a hash.
func (d *decoder) hash() (object.Object, error) {
	n, err := d.length(2)
	if err != nil {
		return nil, err
	}
	h := object.NewHashSized(n)
	for i := 0; i < n; i++ {
		key, err := d.value()
		if err != nil {
			return nil, err
		}
		hashable, ok := key.(object.Hashable)
		if !ok {
			return nil, errCorrupt
		}
		value, err := d.value()
		if err != nil {
			return nil, err
		}
		h.Set(hashable, value)
	}
	return h, nil
}
