package main

type Cpu struct {
	Pc                          uint16
	Acc, Xidx, Yidx, Sptr, Psts uint8
	instructionTable            map[uint8]map[uint8]uint8
}

// Initialize cpu with corret parameters, also initialize instructionTable
func NewCpu() *Cpu {
	var cpu Cpu
	cpu.instructionTable = make(map[uint8]map[uint8]uint8)

	return &cpu
}

type Memory [1 << 16]uint8

func (cpu *Cpu) executeNext(memory *Memory) {
	instruction := memory[cpu.Pc]

	id, addrMode, group := decode(instruction)
}

// Receives a 8 bit instruction and returns the 3 bit identifier (most significant)
// the 3 bits representing the addressing mode, and the last 2 bits representing
// the instruction group
func decode(instruction uint8) (uint8, uint8, uint8) {
	a := (instruction >> 5) & 0b111 // 3 most significant bits
	b := (instruction >> 2) & 0b111 // middle 3 bits
	c := (instruction) & 0b11       // last 2 bits
	return a, b, c
}
