package build

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// The icon of a Windows program is a resource of its .exe: a tree of tables
// (type, then name, then language) in a section called .rsrc, which the
// resource entry of the PE header points to. Go makes that section only
// from a .syso file at link time, and that needs Go and lizard-script's source
// for every build, even of this system, where the running lizardscript is
// copied. So the section is added to the finished file instead: it works on
// any lizardscript for Windows, and the program's files are appended after.

const (
	rtIcon      = 3
	rtGroupIcon = 14
	langUS      = 0x409
)

var le = binary.LittleEndian

// peLayout is what addIcon needs to know of a PE file.
type peLayout struct {
	sections   int // offset of the section table
	count      int
	optional   int // offset of the optional header
	dirs       int // offset of the data directories
	sectionAl  uint32
	fileAl     uint32
	headerSize uint32
}

// readPE reads the headers of a 64 or 32 bit PE file.
func readPE(exe []byte) (peLayout, error) {
	var l peLayout
	if len(exe) < 0x40 || string(exe[:2]) != "MZ" {
		return l, errors.New("the runtime is not a Windows program")
	}
	pe := int(le.Uint32(exe[0x3c:]))
	if pe+24 > len(exe) || string(exe[pe:pe+4]) != "PE\x00\x00" {
		return l, errors.New("the runtime is not a Windows program")
	}
	l.count = int(le.Uint16(exe[pe+6:]))
	l.optional = pe + 24
	l.sections = l.optional + int(le.Uint16(exe[pe+20:]))
	switch le.Uint16(exe[l.optional:]) {
	case 0x20b:
		l.dirs = l.optional + 112
	case 0x10b:
		l.dirs = l.optional + 96
	default:
		return l, errors.New("the runtime is not a 32 or 64 bit Windows program")
	}
	if l.sections+40*l.count > len(exe) || l.dirs+8*3 > len(exe) {
		return l, errors.New("the runtime's headers are damaged")
	}
	l.sectionAl = le.Uint32(exe[l.optional+32:])
	l.fileAl = le.Uint32(exe[l.optional+36:])
	l.headerSize = le.Uint32(exe[l.optional+60:])
	if l.sectionAl == 0 || l.fileAl == 0 {
		return l, errors.New("the runtime's headers are damaged")
	}
	return l, nil
}

// alignUp rounds n up to a multiple of a.
func alignUp(n, a uint32) uint32 { return (n + a - 1) / a * a }

// addIcon returns exe with the icon resources of images added as a new
// section. It fails if the file already has a resource section, or its
// headers have no room for one more section.
func addIcon(exe []byte, images []icoImage) ([]byte, error) {
	l, err := readPE(exe)
	if err != nil {
		return nil, err
	}
	if le.Uint32(exe[l.dirs+16:]) != 0 {
		return nil, errors.New("the runtime already has resources, so it has no room for an icon")
	}
	room := l.headerSize
	var nextVA, rawEnd uint32
	for i := 0; i < l.count; i++ {
		s := exe[l.sections+40*i:]
		nextVA = max(nextVA, le.Uint32(s[12:])+max(le.Uint32(s[8:]), le.Uint32(s[16:])))
		if raw := le.Uint32(s[20:]); raw != 0 {
			room = min(room, raw)
		}
	}
	if uint32(l.sections+40*(l.count+1)) > room {
		return nil, errors.New("the runtime's headers have no room for one more section")
	}
	nextVA = alignUp(nextVA, l.sectionAl)
	rawEnd = alignUp(uint32(len(exe)), l.fileAl)
	data := resourceSection(images, nextVA)
	rawSize := alignUp(uint32(len(data)), l.fileAl)

	out := make([]byte, int(rawEnd+rawSize))
	copy(out, exe)
	copy(out[rawEnd:], data)
	// The new entry of the section table.
	s := out[l.sections+40*l.count:]
	copy(s, ".rsrc")
	le.PutUint32(s[8:], uint32(len(data)))
	le.PutUint32(s[12:], nextVA)
	le.PutUint32(s[16:], rawSize)
	le.PutUint32(s[20:], rawEnd)
	le.PutUint32(s[36:], 0x40000040) // initialized data, readable
	// The headers that count the sections and the image.
	pe := l.optional - 24
	le.PutUint16(out[pe+6:], uint16(l.count+1))
	le.PutUint32(out[l.optional+8:], le.Uint32(out[l.optional+8:])+rawSize)
	le.PutUint32(out[l.optional+56:], alignUp(nextVA+uint32(len(data)), l.sectionAl))
	le.PutUint32(out[l.optional+64:], 0) // no checksum
	le.PutUint32(out[l.dirs+16:], nextVA)
	le.PutUint32(out[l.dirs+20:], uint32(len(data)))
	return out, nil
}

// resourceSection is the bytes of the .rsrc section for an image placed at
// virtual address rva. Its tree is: the root (types RT_ICON and
// RT_GROUP_ICON), under each type the names, under each name the one
// language, and there the data entry (address, size, code page).
func resourceSection(images []icoImage, rva uint32) []byte {
	n := len(images)
	// Where each table and blob goes, all relative to the section.
	rootAt := uint32(0)
	iconDirAt := rootAt + 16 + 2*8
	groupDirAt := iconDirAt + 16 + 8*uint32(n)
	iconLangAt := groupDirAt + 16 + 8 // one table of one language per icon
	groupLangAt := iconLangAt + 24*uint32(n)
	entriesAt := groupLangAt + 24
	blobsAt := entriesAt + 16*uint32(n+1)

	group := make([]byte, 6+14*n)
	le.PutUint16(group[2:], 1)
	le.PutUint16(group[4:], uint16(n))
	for i, im := range images {
		e := group[6+14*i:]
		e[0], e[1] = iconByte(im.side), iconByte(im.side)
		le.PutUint16(e[4:], 1)
		le.PutUint16(e[6:], 32)
		le.PutUint32(e[8:], uint32(len(im.png)))
		le.PutUint16(e[12:], uint16(i+1))
	}
	blobs := make([][]byte, 0, n+1)
	for _, im := range images {
		blobs = append(blobs, im.png)
	}
	blobs = append(blobs, group)
	blobAt := make([]uint32, len(blobs))
	at := blobsAt
	for i, b := range blobs {
		blobAt[i] = at
		at += alignUp(uint32(len(b)), 4)
	}

	var out bytes.Buffer
	table := func(entries ...[2]uint32) {
		out.Write(make([]byte, 12))
		binary.Write(&out, le, [2]uint16{0, uint16(len(entries))})
		for _, e := range entries {
			binary.Write(&out, le, e)
		}
	}
	const sub = 0x80000000
	table([2]uint32{rtIcon, sub | iconDirAt}, [2]uint32{rtGroupIcon, sub | groupDirAt})
	names := make([][2]uint32, n)
	for i := range names {
		names[i] = [2]uint32{uint32(i + 1), sub | (iconLangAt + 24*uint32(i))}
	}
	table(names...)
	table([2]uint32{1, sub | groupLangAt})
	for i := 0; i <= n; i++ {
		table([2]uint32{langUS, entriesAt + 16*uint32(i)})
	}
	for i, b := range blobs {
		binary.Write(&out, le, [4]uint32{rva + blobAt[i], uint32(len(b)), 0, 0})
	}
	for _, b := range blobs {
		out.Write(b)
		out.Write(make([]byte, alignUp(uint32(len(b)), 4)-uint32(len(b))))
	}
	return out.Bytes()
}
