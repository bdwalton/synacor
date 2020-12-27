package synacor

import (
	"errors"
	"fmt"
)

const (
	NREGS     = 8 // Number of registers
	MAX_15BIT = 32767
	MAX_REG   = MAX_15BIT + 8 // indirect register references
)

// CPU states
const (
	RUNNING = iota
	HALTED
	ERROR
)

// Instruction names
const (
	HALT = iota // 0: stop execution and terminate the program
	SET         // 1: set register <a> to the value of <b>
	PUSH        // 2: push <a> onto the stack
	POP         // 3: remove the top element from the stack and write it into <a>; empty stack = error
	EQ          // 4: set <a> to 1 if <b> is equal to <c>; set it to 0 otherwise
	GT          // 5: set <a> to 1 if <b> is greater than <c>; set it to 0 otherwise
	JMP         // 6: jump to <a>
	JT          // 7: if <a> is nonzero, jump to <b>
	JF          // 8: if <a> is zero, jump to <b>
	ADD         // 9: assign into <a> the sum of <b> and <c> (modulo 32768)
	MULT        // 10: store into <a> the product of <b> and <c> (modulo 32768)
	MOD         // 11: store into <a> the remainder of <b> divided by <c>
	AND         // 12: stores into <a> the bitwise and of <b> and <c>
	OR          // 13: stores into <a> the bitwise or of <b> and <c>
	NOT         // 14: stores 15-bit bitwise inverse of <b> in <a>
	RMEM        // 15: read memory at address <b> and write it to <a>
	WMEM        // 16: write the value from <b> into memory at address <a>
	CALL        // 17: write the address of the next instruction to the stack and jump to <a>
	RET         // 18: remove the top element from the stack and jump to it; empty stack = halt
	OUT         // 19: write the character represented by ascii code <a> to the terminal
	IN          // 20: read a character from the terminal and write its ascii code to <a>
	NOOP        // 21: no operation
)

var opsToString map[int]string = map[int]string{
	HALT: "HALT",
	SET:  "SET",
	PUSH: "PUSH",
	POP:  "POP",
	EQ:   "EQ",
	GT:   "GT",
	JMP:  "JMP",
	JT:   "JT",
	JF:   "JF",
	ADD:  "ADD",
	MULT: "MULT",
	MOD:  "MOD",
	AND:  "AND",
	OR:   "OR",
	NOT:  "NOT",
	RMEM: "RMEM",
	WMEM: "WMEM",
	CALL: "CALL",
	RET:  "RET",
	OUT:  "OUT",
	IN:   "IN",
	NOOP: "NOOP",
}

type Stack struct {
	data []uint16
	len  int
}

func NewStack() *Stack {
	return &Stack{data: make([]uint16, 0)}
}

func (s *Stack) Push(v uint16) {
	s.data = append(s.data, v)
	s.len += 1
}

func (s *Stack) Pop() uint16 {
	v := s.data[s.len-1]
	s.data = s.data[:s.len-2]
	s.len -= 1
	return v
}

type Machine struct {
	memory []uint16
	regs   []uint16
	pc     uint16 // program counter
	stack  *Stack
	state  int
}

func NewMachine(prog []uint16) *Machine {
	m := &Machine{
		memory: make([]uint16, 32768), // 15-bits
		regs:   make([]uint16, NREGS, NREGS),
		stack:  NewStack(),
	}

	copy(m.memory, prog)

	return m
}

func (m *Machine) Halted() bool {
	return m.state == HALTED
}

func (m *Machine) Run() {
	for !m.Halted() {
		m.Step()
	}
}

func (m *Machine) maybeIndirectArgs(arg uint16) (uint16, error) {
	if arg < MAX_15BIT {
		return arg, nil
	}

	if arg <= MAX_REG {
		return m.regs[MAX_REG-arg], nil
	}

	return 0, errors.New("Invalid argument")
}

func (m *Machine) getArgs(op uint16) []uint16 {
	args := make([]uint16, 0)
	var n int
	switch op {
	case IN, OUT, JMP:
		n = 1
	case JT, JF, SET:
		n = 2
	}

	for i := 1; i <= n; i++ {
		args = append(args, m.memory[m.pc+uint16(i)])
	}

	return args
}

func (m *Machine) updateProgramCounter(op uint16) {
	var n uint16 // Number of arguments for operand

	switch op {
	case HALT:
		n = 0
	case NOOP:
		n = 0
	case PUSH, POP, OUT, IN:
		n = 1
	case SET, JT, JF:
		n = 2
	}

	m.pc += 1 + n
}

func (m *Machine) Error(msg string) {
	fmt.Println(msg)
	m.Halt()
}

func (m *Machine) Halt() {
	m.state = HALTED
}

func (m *Machine) Step() {
	op := m.memory[m.pc]
	args := m.getArgs(op)

	switch op {
	case HALT:
		m.Halt()
	case OUT:
		fmt.Printf("%c", args[0])
		m.updateProgramCounter(op)
	case NOOP:
		m.updateProgramCounter(op)
	default:
		fmt.Printf("UNIMPLEMENTED INSTRUCTION: %q\n", opsToString[int(op)])
		m.state = HALTED
	}
}
