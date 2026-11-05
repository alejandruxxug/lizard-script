package build

import (
	"bytes"
	"encoding/binary"
)

// icnsKinds are the PNG entries of an .icns file, with the pixel side of each:
// icp4 is 16, ic11 to ic14 are the 16, 32, 128 and 256 point icons at double
// density, and ic07 to ic10 are 128, 256, 512 and 1024 (512 at double).
var icnsKinds = []struct {
	kind string
	size int
}{
	{"icp4", 16}, {"ic11", 32}, {"ic12", 64}, {"ic07", 128}, {"ic13", 256},
	{"ic08", 256}, {"ic14", 512}, {"ic09", 512}, {"ic10", 1024},
}

// writeICNS makes the .icns file of the icons. An .icns is "icns", its
// length, then entries of a four letter kind, their length and the data, all
// numbers big endian; macOS takes a PNG as the data. It is written here so a
// build on any system needs no iconutil.
func writeICNS(s *iconSet) []byte {
	var body bytes.Buffer
	for _, e := range icnsKinds {
		if !s.has(e.size) {
			continue
		}
		data := s.png(e.size)
		body.WriteString(e.kind)
		binary.Write(&body, binary.BigEndian, uint32(8+len(data)))
		body.Write(data)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	binary.Write(&out, binary.BigEndian, uint32(8+body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}
