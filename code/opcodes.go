// Package code defines the bytecode of lizard-script: the instructions the
// compiler (package compiler) writes and the VM (package vm) runs (this
// file), and the compiled unit with the tables they point into (unit.go).
//
// An instruction is one byte, its Opcode, followed by its operands. Every
// operand is 4 bytes, big endian, except the count of OpPoint, which is 1.
package code

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Instructions is the code of one function.
type Instructions []byte

// Opcode is the first byte of an instruction.
type Opcode byte

// The opcodes. "tok" is an index into Unit.Tokens (where an error is
// reported), "jump" an offset into the same Instructions, "slot" a local
// variable of the frame and "free" a captured variable of the closure.
const (
	// Values
	OpConstant      Opcode = iota // const: push Unit.Constants[const]
	OpNull                        // push null
	OpTrue                        // push true
	OpFalse                       // push false
	OpPop                         // drop the top value
	OpBreakValue                  // push the BREAK value
	OpContinueValue               // push the CONTINUE value
	OpMakeReturn                  // wrap the top value in a RETURN_VALUE
	OpUnwrapReturn                // unwrap the top value if it is a RETURN_VALUE

	// Variables
	OpGetLocal     // slot: push the local
	OpSetLocal     // slot: pop into the local (let)
	OpAssignLocal  // slot: store the top value in the local, keep it (=)
	OpGetCell      // slot: like OpGetLocal, for a local that closures share
	OpSetCell      // slot: like OpSetLocal, for a shared local
	OpAssignCell   // slot: like OpAssignLocal, for a shared local
	OpGetFree      // free: push a captured variable that is surely set
	OpAssignFree   // free: assign a captured variable that is surely set
	OpGetName      // name: read through Unit.Names[name] (captured, then global)
	OpAssignName   // name: assign through Unit.Names[name]
	OpDefineGlobal // name: pop into the global environment (let at the top)
	OpEnterScope   // scope: give the shared locals of Unit.Scopes[scope] their cells
	OpSkipScope    // scope: nothing, an OpEnterScope with no shared locals

	// Parameters
	OpMoveIfGiven // param, slot: copy the argument to slot if the call gave it
	OpJumpIfGiven // param, jump: jump if the call gave the argument

	// Operators, all with tok
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpFloorDiv
	OpMod
	OpPow
	OpEqual
	OpNotEqual
	OpExact
	OpNotExact
	OpLess
	OpGreater
	OpLessEqual
	OpGreaterEqual
	OpMinus  // node: -x, Unit.Nodes[node] is the PrefixExpression
	OpPrefix // node: any other prefix operator
	OpNot    // !x
	OpAnd    // jump: if the top is falsy, replace it with false and jump; else drop it
	OpOr     // jump: if the top is truthy, replace it with true and jump; else drop it
	OpToBool // replace the top value with true or false

	// Control flow
	OpJump          // jump
	OpJumpNotTruthy // jump: pop, and jump if it was falsy
	OpStep          // tok: count a step of the step limit
	OpStatement     // stmt: call the OnStatement hook with Unit.Statements[stmt]
	OpCheckSignal   // jump, jump, jump: if the top value is a RETURN_VALUE, BREAK or CONTINUE, jump to the first, second or third
	OpReturnValue   // leave the function with the top value
	OpPushHandler   // jump: start a try whose catch block is at jump
	OpPopHandler    // end the innermost try
	OpLayerEnter    // node: a layer starts, Unit.Nodes[node] is the LayerStatement
	OpLayerExit     // the innermost layer ends
	OpFail          // fail: stop with the error Unit.Failures[fail]

	// Functions
	OpCall    // count, tok: call the value under count arguments
	OpClosure // function, captures: make a closure of Unit.Functions[function]

	// Data structures
	OpArray      // count: an array of the top count values
	OpHash       // count: a hash of the top count key, value pairs
	OpCheckKey   // tok: fail if the top value can't be a hash key
	OpPoint      // count (1 byte): a point of the top count numbers
	OpCheckCoord // tok: fail if the top value is not a number
	OpIndex      // tok: pop the index and the container, push container[index]
	OpIndexKeep  // tok: push container[index], keeping both
	OpSetIndex   // tok: pop value, index, container; container[index] = value; push value
)

// Definition is the name of an opcode and the width of its operands.
type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[Opcode]*Definition{
	OpConstant:      {"OpConstant", []int{4}},
	OpNull:          {"OpNull", []int{}},
	OpTrue:          {"OpTrue", []int{}},
	OpFalse:         {"OpFalse", []int{}},
	OpPop:           {"OpPop", []int{}},
	OpBreakValue:    {"OpBreakValue", []int{}},
	OpContinueValue: {"OpContinueValue", []int{}},
	OpMakeReturn:    {"OpMakeReturn", []int{}},
	OpUnwrapReturn:  {"OpUnwrapReturn", []int{}},

	OpGetLocal:     {"OpGetLocal", []int{4}},
	OpSetLocal:     {"OpSetLocal", []int{4}},
	OpAssignLocal:  {"OpAssignLocal", []int{4}},
	OpGetCell:      {"OpGetCell", []int{4}},
	OpSetCell:      {"OpSetCell", []int{4}},
	OpAssignCell:   {"OpAssignCell", []int{4}},
	OpGetFree:      {"OpGetFree", []int{4}},
	OpAssignFree:   {"OpAssignFree", []int{4}},
	OpGetName:      {"OpGetName", []int{4}},
	OpAssignName:   {"OpAssignName", []int{4}},
	OpDefineGlobal: {"OpDefineGlobal", []int{4}},
	OpEnterScope:   {"OpEnterScope", []int{4}},
	OpSkipScope:    {"OpSkipScope", []int{4}},

	OpMoveIfGiven: {"OpMoveIfGiven", []int{4, 4}},
	OpJumpIfGiven: {"OpJumpIfGiven", []int{4, 4}},

	OpAdd:          {"OpAdd", []int{4}},
	OpSub:          {"OpSub", []int{4}},
	OpMul:          {"OpMul", []int{4}},
	OpDiv:          {"OpDiv", []int{4}},
	OpFloorDiv:     {"OpFloorDiv", []int{4}},
	OpMod:          {"OpMod", []int{4}},
	OpPow:          {"OpPow", []int{4}},
	OpEqual:        {"OpEqual", []int{4}},
	OpNotEqual:     {"OpNotEqual", []int{4}},
	OpExact:        {"OpExact", []int{4}},
	OpNotExact:     {"OpNotExact", []int{4}},
	OpLess:         {"OpLess", []int{4}},
	OpGreater:      {"OpGreater", []int{4}},
	OpLessEqual:    {"OpLessEqual", []int{4}},
	OpGreaterEqual: {"OpGreaterEqual", []int{4}},
	OpMinus:        {"OpMinus", []int{4}},
	OpPrefix:       {"OpPrefix", []int{4}},
	OpNot:          {"OpNot", []int{}},
	OpAnd:          {"OpAnd", []int{4}},
	OpOr:           {"OpOr", []int{4}},
	OpToBool:       {"OpToBool", []int{}},

	OpJump:          {"OpJump", []int{4}},
	OpJumpNotTruthy: {"OpJumpNotTruthy", []int{4}},
	OpStep:          {"OpStep", []int{4}},
	OpStatement:     {"OpStatement", []int{4}},
	OpCheckSignal:   {"OpCheckSignal", []int{4, 4, 4}},
	OpReturnValue:   {"OpReturnValue", []int{}},
	OpPushHandler:   {"OpPushHandler", []int{4}},
	OpPopHandler:    {"OpPopHandler", []int{}},
	OpLayerEnter:    {"OpLayerEnter", []int{4}},
	OpLayerExit:     {"OpLayerExit", []int{}},
	OpFail:          {"OpFail", []int{4}},

	OpCall:    {"OpCall", []int{4, 4}},
	OpClosure: {"OpClosure", []int{4, 4}},

	OpArray:      {"OpArray", []int{4}},
	OpHash:       {"OpHash", []int{4}},
	OpCheckKey:   {"OpCheckKey", []int{4}},
	OpPoint:      {"OpPoint", []int{1}},
	OpCheckCoord: {"OpCheckCoord", []int{4}},
	OpIndex:      {"OpIndex", []int{4}},
	OpIndexKeep:  {"OpIndexKeep", []int{4}},
	OpSetIndex:   {"OpSetIndex", []int{4}},
}

// Lookup returns the definition of op, or an error for an unknown opcode.
func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}
	return def, nil
}

// Make encodes one instruction. An unknown opcode gives an empty slice.
func Make(op Opcode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	length := 1
	for _, w := range def.OperandWidths {
		length += w
	}

	instruction := make([]byte, length)
	instruction[0] = byte(op)
	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 4:
			binary.BigEndian.PutUint32(instruction[offset:], uint32(o))
		case 1:
			instruction[offset] = byte(o)
		}
		offset += width
	}
	return instruction
}

// ReadOperands decodes the operands of an instruction whose opcode is def,
// from ins (which starts right after the opcode). It returns them and how
// many bytes they took.
func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0
	for i, width := range def.OperandWidths {
		switch width {
		case 4:
			operands[i] = int(ReadUint32(ins[offset:]))
		case 1:
			operands[i] = int(ins[offset])
		}
		offset += width
	}
	return operands, offset
}

// ReadUint32 reads a 4-byte operand.
func ReadUint32(ins Instructions) uint32 {
	return binary.BigEndian.Uint32(ins)
}

// String writes the instructions one per line, with the offset of each:
//
//	0000 OpConstant 0
//	0005 OpPop
func (ins Instructions) String() string {
	var out bytes.Buffer
	i := 0
	for i < len(ins) {
		def, err := Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "ERROR: %s\n", err)
			i++
			continue
		}
		operands, read := ReadOperands(def, ins[i+1:])
		fmt.Fprintf(&out, "%04d %s\n", i, formatInstruction(def, operands))
		i += 1 + read
	}
	return out.String()
}

func formatInstruction(def *Definition, operands []int) string {
	if len(operands) != len(def.OperandWidths) {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n", len(operands), len(def.OperandWidths))
	}
	text := def.Name
	for _, o := range operands {
		text += fmt.Sprintf(" %d", o)
	}
	return text
}
