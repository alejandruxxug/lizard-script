package object

// NATIVE_OBJ is the type of a value that only Go code can look inside.
const NATIVE_OBJ = "NATIVE"

// Native is a value that belongs to Go code: a window, an image. A program
// can keep it in variables and pass it to the builtins that know it, but it
// can't look inside. Two natives are equal only when they are the same one.
type Native struct {
	// Kind names what it is, like "image". Builtins check it before they
	// use Value.
	Kind  string
	Value any
}

// Describer is implemented by a native's Value to say more about it in
// Inspect, like the size of an image.
type Describer interface {
	Describe() string
}

func (n *Native) Type() ObjectType { return NATIVE_OBJ }

// Inspect writes <kind>, or <kind description> when the value describes
// itself: <image 640x480>.
func (n *Native) Inspect() string {
	if d, ok := n.Value.(Describer); ok {
		if text := d.Describe(); text != "" {
			return "<" + n.Kind + " " + text + ">"
		}
	}
	return "<" + n.Kind + ">"
}
