package synacor

import (
	"bufio"
	"fmt"
	"os"
)

const (
	NREGS          = 8     // Number of registers
	MAX_15BIT      = 32767 // Values are 0..MAX_15BIT
	OVERFLOW_15BIT = 32768
	MAX_REG        = MAX_15BIT + 8 // indirect register references
)

// CPU states
const (
	RUNNING = iota // Default. Next instruction pointed to by program counter
	HALTED         // Normal halt.
	ERROR          // Halted in error state.
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

// The number of arguments expected for each OP.
var argsForOp map[int]uint16 = map[int]uint16{
	HALT: 0,
	SET:  2,
	PUSH: 1,
	POP:  1,
	EQ:   3,
	GT:   3,
	JMP:  1,
	JT:   2,
	JF:   2,
	ADD:  3,
	MULT: 3,
	MOD:  3,
	AND:  3,
	OR:   3,
	NOT:  2,
	RMEM: 2,
	WMEM: 2,
	CALL: 1,
	RET:  0,
	OUT:  1,
	IN:   1,
	NOOP: 0,
}

func isReg(arg uint16) bool {
	return MAX_15BIT < arg && arg <= MAX_REG
}

func isValue(arg uint16) bool {
	return arg <= MAX_15BIT
}

func decipherReg(arg uint16) uint16 {
	return arg - MAX_15BIT - 1
}

type Stack struct {
	data []uint16
}

func NewStack() *Stack {
	return &Stack{data: make([]uint16, 0)}
}

func (s *Stack) Push(v uint16) {
	s.data = append(s.data, v)
}

func (s *Stack) IsEmpty() bool {
	return len(s.data) == 0
}

func (s *Stack) Pop() (uint16, bool) {
	if s.IsEmpty() {
		return 0, false
	}

	idx := len(s.data) - 1
	v := s.data[idx]
	s.data = s.data[:idx]

	return v, true
}

type Machine struct {
	memory       []uint16
	regs         []uint16
	pc           uint16 // program counter
	stack        *Stack
	state        int
	input        *bufio.Reader
	unused_input []uint16 // Available input
}

func NewMachine(prog []uint16) *Machine {
	m := &Machine{
		memory:       make([]uint16, 32768), // 15-bits
		regs:         make([]uint16, NREGS, NREGS),
		stack:        NewStack(),
		input:        bufio.NewReader(os.Stdin),
		unused_input: make([]uint16, 0),
	}

	copy(m.memory, prog)

	return m
}

func (m *Machine) Halted() bool {
	return m.state != RUNNING
}

func (m *Machine) Run() {
	for !m.Halted() {
		m.Step()
	}
}

func (m *Machine) readArg(arg uint16) uint16 {
	if isValue(arg) {
		return arg
	}

	if isReg(arg) {
		return m.regs[decipherReg(arg)]
	}

	m.Error(fmt.Sprintf("Invalid Argument '%d'.", arg))
	return 0
}

func (m *Machine) getArgs(op uint16) []uint16 {
	if n := argsForOp[int(op)]; n > 0 {
		return m.memory[m.pc+1 : m.pc+1+n]
	}

	return []uint16{}
}

// Move over the OP and the number of args for the OP
func (m *Machine) nextProgramCounter(op uint16) uint16 {
	return m.pc + 1 + argsForOp[int(op)]
}

// Log error and halt machine.
func (m *Machine) Error(msg string) {
	fmt.Println(msg)
	m.state = ERROR
}

// Halt machine.
func (m *Machine) Halt() {
	m.state = HALTED
}

func (m *Machine) Step() {
	op := m.memory[m.pc]
	args := m.getArgs(op)

	switch op {
	case HALT:
		m.Halt()
		return
	case SET:
		m.regs[decipherReg(args[0])] = m.readArg(args[1])
	case PUSH:
		m.stack.Push(m.readArg(args[0]))
	case POP:
		v, ok := m.stack.Pop()
		if !ok {
			m.Error("Popped an empty stack.")
			return
		}

		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = v
		} else {
			m.memory[args[0]] = v
		}
	case EQ:
		b, c := m.readArg(args[1]), m.readArg(args[2])
		var eq uint16
		if b == c {
			eq = 1
		}
		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = eq
		} else {
			m.memory[args[0]] = eq
		}
	case GT:
		b, c := m.readArg(args[1]), m.readArg(args[2])
		var gt uint16
		if b > c {
			gt = 1
		}
		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = gt
		} else {
			m.memory[args[0]] = gt
		}
	case JMP:
		m.pc = m.readArg(args[0])
		return
	case JT:
		if m.readArg(args[0]) != 0 {
			m.pc = m.readArg(args[1])
			return
		}
	case JF:
		if m.readArg(args[0]) == 0 {
			m.pc = m.readArg(args[1])
			return
		}
	case ADD:
		b, c := m.readArg(args[1]), m.readArg(args[2])
		a := (b + c) % OVERFLOW_15BIT

		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = a
		} else {
			m.memory[args[0]] = a
		}
	case MULT:
		b, c := m.readArg(args[1]), m.readArg(args[2])
		a := (b * c) % OVERFLOW_15BIT

		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = a
		} else {
			m.memory[args[0]] = a
		}
	case MOD:
		b, c := m.readArg(args[1]), m.readArg(args[2])
		a := b % c

		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = a
		} else {
			m.memory[args[0]] = a
		}
	case AND:
		b, c := m.readArg(args[1]), m.readArg(args[2])
		a := b & c

		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = a
		} else {
			m.memory[args[0]] = a
		}
	case OR:
		b, c := m.readArg(args[1]), m.readArg(args[2])
		a := b | c

		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = a
		} else {
			m.memory[args[0]] = a
		}
	case NOT:
		b := m.readArg(args[1])
		a := (b ^ MAX_15BIT)
		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = a
		} else {
			m.memory[args[0]] = a
		}
	case RMEM:
		m.regs[decipherReg(args[0])] = m.memory[m.readArg(args[1])]
	case WMEM:
		m.memory[m.readArg(args[0])] = m.readArg(args[1])
	case CALL:
		m.stack.Push(m.nextProgramCounter(op))
		m.pc = m.readArg(args[0])
		return
	case RET:
		if npc, ok := m.stack.Pop(); ok {
			m.pc = npc
		} else {
			m.Error("Popped an empty stack.")
		}
		return
	case OUT:
		fmt.Printf("%c", m.readArg(args[0]))
	case IN:
		if len(m.unused_input) == 0 {
			fmt.Printf("Input: ")
			input, _ := m.input.ReadString('\n')
			for _, c := range input {
				m.unused_input = append(m.unused_input, uint16(c))
			}
		}

		if isReg(args[0]) {
			m.regs[decipherReg(args[0])] = m.unused_input[0]
		} else {
			m.memory[m.readArg(args[0])] = m.unused_input[0]
		}

		m.unused_input = m.unused_input[1:]
	case NOOP:
	default:
		m.Error(fmt.Sprintf("UNIMPLEMENTED INSTRUCTION %q.\n\n", opsToString[int(op)]))
	}

	m.pc = m.nextProgramCounter(op)
}
