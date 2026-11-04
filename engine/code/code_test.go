package code

import "testing"

func TestMake(t *testing.T) {
	tests := []struct {
		op       Opcode
		operands []int
		expected []byte
	}{
		{OpConstant, []int{65534}, []byte{byte(OpConstant), 0, 0, 255, 254}},
		{OpPop, []int{}, []byte{byte(OpPop)}},
		{OpCall, []int{2, 70000}, []byte{byte(OpCall), 0, 0, 0, 2, 0, 1, 17, 112}},
		{OpPoint, []int{3}, []byte{byte(OpPoint), 3}},
	}

	for _, tt := range tests {
		instruction := Make(tt.op, tt.operands...)
		if string(instruction) != string(tt.expected) {
			t.Errorf("Make(%d, %v) = %v, want %v", tt.op, tt.operands, instruction, tt.expected)
		}
	}
}

func TestReadOperands(t *testing.T) {
	tests := []struct {
		op        Opcode
		operands  []int
		bytesRead int
	}{
		{OpConstant, []int{65535}, 4},
		{OpCheckSignal, []int{1, 2, 3}, 12},
		{OpPoint, []int{2}, 1},
		{OpNull, []int{}, 0},
	}

	for _, tt := range tests {
		instruction := Make(tt.op, tt.operands...)
		def, err := Lookup(byte(tt.op))
		if err != nil {
			t.Fatal(err)
		}
		operands, read := ReadOperands(def, instruction[1:])
		if read != tt.bytesRead {
			t.Errorf("%s: read %d bytes, want %d", def.Name, read, tt.bytesRead)
		}
		for i, want := range tt.operands {
			if operands[i] != want {
				t.Errorf("%s: operand %d is %d, want %d", def.Name, i, operands[i], want)
			}
		}
	}
}

func TestInstructionsString(t *testing.T) {
	var ins Instructions
	for _, part := range [][]byte{Make(OpConstant, 1), Make(OpGetLocal, 2), Make(OpAdd, 7), Make(OpPoint, 2), Make(OpReturnValue)} {
		ins = append(ins, part...)
	}
	expected := "0000 OpConstant 1\n0005 OpGetLocal 2\n0010 OpAdd 7\n0015 OpPoint 2\n0017 OpReturnValue\n"
	if ins.String() != expected {
		t.Errorf("wrong disassembly:\n%s\nwant:\n%s", ins.String(), expected)
	}
}

func TestEveryOpcodeIsDefined(t *testing.T) {
	for op := OpConstant; op <= OpSetIndex; op++ {
		if _, err := Lookup(byte(op)); err != nil {
			t.Errorf("opcode %d has no definition", op)
		}
	}
	if _, err := Lookup(byte(OpSetIndex) + 1); err == nil {
		t.Error("an opcode past the last one should be undefined")
	}
}
