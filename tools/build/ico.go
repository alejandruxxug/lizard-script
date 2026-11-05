package build

import (
	"bytes"
	"encoding/binary"
)

// icoSizes are the sizes of a Windows icon: the taskbar, the title bar, the
// explorer's lists and its big icons.
var icoSizes = []int{16, 32, 48, 256}

// icoImage is one picture of an icon: its side and its PNG file.
type icoImage struct {
	side int
	png  []byte
}

// icoImages makes the pictures of a Windows icon.
func icoImages(s *iconSet) []icoImage {
	var images []icoImage
	for _, size := range icoSizes {
		if s.has(size) {
			images = append(images, icoImage{size, s.png(size)})
		}
	}
	return images
}

// iconByte is how an icon file writes a side: 256 is 0.
func iconByte(side int) byte {
	if side >= 256 {
		return 0
	}
	return byte(side)
}

// writeICO makes an .ico file: a header of the picture count, one 16 byte
// entry per picture (size, planes, bits, length, offset), then the pictures,
// which Windows Vista and later take as PNG.
func writeICO(images []icoImage) []byte {
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(images))})
	offset := 6 + 16*len(images)
	for _, im := range images {
		out.Write([]byte{iconByte(im.side), iconByte(im.side), 0, 0})
		binary.Write(&out, binary.LittleEndian, [2]uint16{1, 32})
		binary.Write(&out, binary.LittleEndian, [2]uint32{uint32(len(im.png)), uint32(offset)})
		offset += len(im.png)
	}
	for _, im := range images {
		out.Write(im.png)
	}
	return out.Bytes()
}
