package main

import (
	"log"
)

const Carry = "C"
const Zero = "Z"
const InterruptDisable = "I"
const Decimal = "D"
const Overflow = "O"
const Negative = "N"

type Cpu struct {
	Pc                          uint16
	Acc, Xidx, Yidx, Sptr, Psts uint8
	opcodeTable                 map[uint8]string
}

// Initialize cpu with corret parameters, also initialize instructionTable
func NewCpu() *Cpu {
	var cpu Cpu
	cpu.Psts = 0b00100000
	cpu.opcodeTable = Generate()
	return &cpu
}

func (cpu *Cpu) executeNext() {
	instruction := Memory[cpu.Pc]
	opcode := cpu.opcodeTable[instruction]
	addresingMode := (instruction >> 2) & 0b00000111

	switch opcode {
	case ADC:
	}
}

// Receives an adressingMode, and returns the operand and total size of the instruction
func (cpu *Cpu) getAluOperand(adressingMode uint8) (uint8, uint8) {
	const idxindirectX uint8 = 0b000
	const zeroPage uint8 = 0b001
	const immediate uint8 = 0b010
	const absolute uint8 = 0b011
	const indirectidxY uint8 = 0b100
	const zeroPageX uint8 = 0b101
	const absoluteY uint8 = 0b110
	const absoluteX uint8 = 0b111

	arg := uint16(Memory[cpu.Pc+1])
	switch adressingMode {
	case idxindirectX:
		return Memory[uint16(Memory[(arg+uint16(cpu.Xidx))%256])+uint16(Memory[(arg+uint16(cpu.Xidx)+uint16(1))%256])*256], 2
	case zeroPage:
		return Memory[uint16(arg)%256], 2
	case immediate:
		return Memory[cpu.Pc+1], 2
	case absolute:
		return Memory[arg], 3
	case indirectidxY:
		return Memory[uint16(Memory[arg])+uint16(Memory[(arg+1)%256])*256+1], 2
	case zeroPageX:
		return Memory[uint16(arg+uint16(cpu.Xidx))%256], 2
	case absoluteY:
		return Memory[arg+uint16(cpu.Yidx)], 3
	case absoluteX:
		return Memory[arg+uint16(cpu.Xidx)], 3
	default:
		log.Println("Warning: cpu.getAluOperand invalid addressingMode: %b", adressingMode)
		return 0, 1
	}
}

// Given a flag name and a 1 or 0, sests that flag in the cpu
// Expects only 1 or 0 as the val
func (cpu *Cpu) setFlag(flag string, val uint8) {
	switch flag {
	case Carry:
		cpu.Psts = cpu.Psts | val
	case Zero:
		cpu.Psts = cpu.Psts | (val << 1)
	case InterruptDisable:
		cpu.Psts = cpu.Psts | (val << 2)
	case Decimal:
		cpu.Psts = cpu.Psts | (val << 3)
	case Overflow:
		cpu.Psts = cpu.Psts | (val << 6)
	case Negative:
		cpu.Psts = cpu.Psts | (val << 7)
	default:
		log.Println("Warning: cpu.setFlag invalid flag: %s", flag)
	}
}

func (cpu *Cpu) getFlag(flag string) uint8 {
	var mask uint8 = 0b00000001
	switch flag {
	case Carry:
		return cpu.Psts & mask
	case Zero:
		return cpu.Psts & (mask << 1)
	case InterruptDisable:
		return cpu.Psts & (mask << 2)
	case Decimal:
		return cpu.Psts & (mask << 3)
	case Overflow:
		return cpu.Psts & (mask << 6)
	case Negative:
		return cpu.Psts & (mask << 7)
	default:
		log.Println("Warning: cpu.getFlag invalid flag: %s, returning 0 instead", flag)
		return 0
	}
}
