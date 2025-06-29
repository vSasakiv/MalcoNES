package main

import (
	"fmt"
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
	cpu.Sptr = 0xFD
	cpu.opcodeTable = Generate()
	return &cpu
}

func (cpu *Cpu) executeNext() {
	instruction := MainMemory.memRead(cpu.Pc)
	opcode := cpu.opcodeTable[instruction]
	addresingMode := (instruction >> 2) & 0b00000111

	switch opcode {
	// Accumulator instructions
	case ADC:
		op, size := cpu.getAluOperand(addresingMode)
		var result uint16 = uint16(cpu.Acc) + uint16(op) + uint16(cpu.getFlag(Carry))
		cpu.calcAndSetFlags([]string{Carry, Zero, Overflow, Negative}, result, cpu.Acc, op)
		cpu.Acc = uint8(result)
		cpu.Pc += uint16(size)
	case ORA:
		op, size := cpu.getAluOperand(addresingMode)
		var result uint8 = cpu.Acc | op
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
		cpu.Acc = uint8(result)
		cpu.Pc += uint16(size)
	case AND:
		op, size := cpu.getAluOperand(addresingMode)
		var result uint8 = cpu.Acc & op
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
		cpu.Acc = uint8(result)
		cpu.Pc += uint16(size)
	case EOR:
		op, size := cpu.getAluOperand(addresingMode)
		var result uint8 = cpu.Acc ^ op
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
		cpu.Acc = uint8(result)
		cpu.Pc += uint16(size)
	case STA:
		address, size := cpu.getAluAddress(addresingMode)
		MainMemory.memWrite(address, cpu.Acc)
		cpu.Pc += uint16(size)
	case LDA:
		address, size := cpu.getAluAddress(addresingMode)
		var result uint8 = MainMemory.memRead(address)
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
		cpu.Acc = result
		cpu.Pc += uint16(size)
	case CMP:
		op, size := cpu.getAluOperand(addresingMode)
		cpu.setCompareFlags(op, cpu.Acc)
		cpu.Pc += uint16(size)
	case SBC:
		op, size := cpu.getAluOperand(addresingMode)
		var result uint16 = uint16(cpu.Acc) + uint16(^op) + uint16(cpu.getFlag(Carry))
		cpu.calcAndSetFlags([]string{Carry, Zero, Overflow, Negative}, uint16(result), cpu.Acc, ^op)
		cpu.Pc += uint16(size)
	case ASL:
		// has accumulator addressing
		if instruction == 0x0A {
			cpu.setFlag(Carry, cpu.Acc>>7)
			result := cpu.Acc << 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			cpu.Acc = result
			cpu.Pc += 1
		} else {
			address, size := cpu.getAluAddress(addresingMode)
			op := MainMemory.memRead(address)
			result := op << 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			cpu.setFlag(Carry, op>>7)

			MainMemory.memWrite(address, result)
			cpu.Pc += uint16(size)
		}
	case ROL:
		// has accumulator addressing
		if instruction == 0x2A {
			cpu.setFlag(Carry, cpu.Acc>>7)
			result := (cpu.Acc << 1) + (cpu.Acc >> 7)
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			cpu.Acc = result
			cpu.Pc += 1
		} else {
			address, size := cpu.getAluAddress(addresingMode)
			op := MainMemory.memRead(address)
			cpu.setFlag(Carry, op>>7)
			result := (op << 1) + (op >> 7)
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			MainMemory.memWrite(address, result)
			cpu.Pc += uint16(size)
		}
	case LSR:
		// has accumulator addressing
		if instruction == 0x4A {
			cpu.setFlag(Carry, cpu.Acc&0b1)
			result := cpu.Acc >> 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			cpu.Acc = result
			cpu.Pc += 1
		} else {
			address, size := cpu.getAluAddress(addresingMode)
			op := MainMemory.memRead(address)
			result := op >> 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			cpu.setFlag(Carry, op&0b1)

			MainMemory.memWrite(address, result)
			cpu.Pc += uint16(size)
		}
	case ROR:
		// has accumulator addressing
		if instruction == 0x6A {
			cpu.setFlag(Carry, cpu.Acc&0b1)
			result := (cpu.Acc >> 1) + (cpu.Acc << 7)
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			cpu.Acc = result
			cpu.Pc += 1
		} else {
			address, size := cpu.getAluAddress(addresingMode)
			op := MainMemory.memRead(address)
			cpu.setFlag(Carry, op&0b1)
			result := (op >> 1) + (op << 7)
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			MainMemory.memWrite(address, result)
			cpu.Pc += uint16(size)
		}
	case DEC:
		address, size := cpu.getAluAddress(addresingMode)
		result := MainMemory.memRead(address) - 1
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
		MainMemory.memWrite(address, result)
		cpu.Pc += uint16(size)
	case INC:
		address, size := cpu.getAluAddress(addresingMode)
		result := MainMemory.memRead(address) + 1
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
		MainMemory.memWrite(address, result)
		cpu.Pc += uint16(size)
	case SLO:
		address, size := cpu.getAluAddress(addresingMode)
		op := MainMemory.memRead(address)
		cpu.setFlag(Carry, op>>7)
		result := (op<<1 | cpu.Acc)
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

		MainMemory.memWrite(address, result)
		cpu.Pc += uint16(size)
	case RLA:
		address, size := cpu.getAluAddress(addresingMode)
		op := MainMemory.memRead(address)
		cpu.setFlag(Carry, op>>7)
		result := ((op << 1) + (op >> 7)) & cpu.Acc
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

		MainMemory.memWrite(address, result)
		cpu.Pc += uint16(size)
	case SRE:
		address, size := cpu.getAluAddress(addresingMode)
		op := MainMemory.memRead(address)
		cpu.setFlag(Carry, op&0b1)
		result := (op >> 1) ^ cpu.Acc
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

		MainMemory.memWrite(address, result)
		cpu.Pc += uint16(size)
	case RRA:
		address, size := cpu.getAluAddress(addresingMode)
		op := MainMemory.memRead(address)
		cpu.setFlag(Carry, op&0b1)
		rotateResult := (op >> 1) + (op << 7)
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(rotateResult), 0, 0)

		var result uint16 = uint16(cpu.Acc) + uint16(rotateResult) + uint16(cpu.getFlag(Carry))
		cpu.calcAndSetFlags([]string{Carry, Zero, Overflow, Negative}, result, cpu.Acc, op)
		cpu.Acc = uint8(result)
		cpu.Pc += uint16(size)
	case SAX:
		const zeroPageY uint8 = 0b101
		var aluAddress uint16
		var size uint8
		if addresingMode == zeroPageY {
			aluAddress, size = cpu.getAluAddressZeroPageY()
		} else {
			aluAddress, size = cpu.getAluAddress(addresingMode)
		}
		result := cpu.Xidx + cpu.Acc
		MainMemory.memWrite(aluAddress, result)
		cpu.Pc += uint16(size)
	case LAX:
		const zeroPageY uint8 = 0b101
		var op uint8
		var size uint8
		if addresingMode == zeroPageY {
			op, size = cpu.getAluOperandZeroPageY()
		} else {
			op, size = cpu.getAluOperand(addresingMode)
		}
		cpu.Acc = op
		cpu.Xidx = op
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(op), 0, 0)
		cpu.Pc += uint16(size)
	case DCP:
		address, size := cpu.getAluAddress(addresingMode)
		result := MainMemory.memRead(address) - 1
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
		MainMemory.memWrite(address, result)
		cpu.setCompareFlags(result, cpu.Acc)
		cpu.Pc += uint16(size)
	case ISC:
		address, size := cpu.getAluAddress(addresingMode)
		result := MainMemory.memRead(address) + 1
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
		MainMemory.memWrite(address, result)

		var subResult uint16 = uint16(cpu.Acc) + uint16(^result) + uint16(cpu.getFlag(Carry))
		cpu.calcAndSetFlags([]string{Carry, Zero, Overflow, Negative}, uint16(subResult), cpu.Acc, ^result)
		cpu.Acc = uint8(subResult)
		cpu.Pc += uint16(size)

	// stack manipulating instructions
	case BRK:
		cpu.pushToStack16(cpu.Pc + 2)
		cpu.pushToStack(cpu.Psts | 0b00110000)
		cpu.setFlag(InterruptDisable, 1)
		cpu.Pc = 0xFFFE
	case PHP:
		cpu.pushToStack(cpu.Psts | 0b00110000)
		cpu.Pc += 1
	case PLP:
		cpu.Psts = cpu.pullFromStack()
		cpu.Pc += 1
	case PHA:
		cpu.pushToStack(cpu.Acc)
		cpu.Pc += 1
	case PLA:
		cpu.Acc = cpu.pullFromStack()
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Acc), 0, 0)
	case RTI:
		cpu.Psts = cpu.pullFromStack()
		cpu.Pc = cpu.pullFromStack16()
	case RTS:
		cpu.Pc = cpu.pullFromStack16() + 1

	// branching instructions
	case BPL:
		cpu.branchIfFlag(Negative, 0)
	case BMI:
		cpu.branchIfFlag(Negative, 1)
	case BVC:
		cpu.branchIfFlag(Overflow, 0)
	case BVS:
		cpu.branchIfFlag(Overflow, 1)
	case BCC:
		cpu.branchIfFlag(Carry, 0)
	case BCS:
		cpu.branchIfFlag(Carry, 1)
	case BNE:
		cpu.branchIfFlag(Zero, 0)
	case BEQ:
		cpu.branchIfFlag(Zero, 1)

	// flag manipulating instructions
	case BIT:
		op, size := cpu.getAluOperand(addresingMode)
		result := cpu.Acc & op
		cpu.calcAndSetFlags([]string{Zero}, uint16(result), 0, 0)
		cpu.setFlag(Overflow, (op>>6)&0b1)
		cpu.setFlag(Negative, (op>>7)&0b1)
		cpu.Pc += uint16(size)
	case CLC:
		cpu.setFlag(Carry, 0)
		cpu.Pc += 1
	case SEC:
		cpu.setFlag(Carry, 1)
		cpu.Pc += 1
	case CLI:
		cpu.setFlag(InterruptDisable, 0)
		cpu.Pc += 1
	case SEI:
		cpu.setFlag(InterruptDisable, 1)
		cpu.Pc += 1
	case CLV:
		cpu.setFlag(Overflow, 0)
		cpu.Pc += 1
	case CLD:
		cpu.setFlag(Decimal, 0)
		cpu.Pc += 1
	case SED:
		cpu.setFlag(Decimal, 1)
		cpu.Pc += 1

	// jumping instructions
	case JSR:
		cpu.pushToStack16(cpu.Pc + 2)
		cpu.Pc = MainMemory.memRead16(cpu.Pc + 1)
	case JMP:
		aluAddress, _ := cpu.getAluAddress(addresingMode)
		// only occourence of indirect absolute
		if instruction == 0x68 {
			cpu.Pc = MainMemory.memRead16(aluAddress)
		} else {
			cpu.Pc = aluAddress
		}

		// index register manipulating instructions
	case STY:
		aluAddress, size := cpu.getAluAddress(addresingMode)
		MainMemory.memWrite(aluAddress, cpu.Yidx)
		cpu.Pc += uint16(size)
	case STX:
		const zeroPageY uint8 = 0b101
		var aluAddress uint16
		var size uint8
		if addresingMode == zeroPageY {
			aluAddress, size = cpu.getAluAddressZeroPageY()
		} else {
			aluAddress, size = cpu.getAluAddress(addresingMode)
		}
		MainMemory.memWrite(aluAddress, cpu.Xidx)
		cpu.Pc += uint16(size)
	case DEY:
		cpu.Yidx -= 1
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Yidx), 0, 0)
		cpu.Pc += 1
	case DEX:
		cpu.Xidx -= 1
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Xidx), 0, 0)
		cpu.Pc += 1
	case TYA:
		cpu.Acc = cpu.Yidx
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Acc), 0, 0)
		cpu.Pc += 1
	case TXA:
		cpu.Acc = cpu.Xidx
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Acc), 0, 0)
		cpu.Pc += 1
	case SHY:
		address, size := cpu.getAluAddress(addresingMode)
		MainMemory.memWrite(address, cpu.Yidx&(MainMemory.memRead(cpu.Pc+2)+1))
		cpu.Pc += uint16(size)
	case SHX:
		address, size := cpu.getAluAddress(addresingMode)
		MainMemory.memWrite(address, cpu.Xidx&(MainMemory.memRead(cpu.Pc+2)+1))
		cpu.Pc += uint16(size)
	case LDY:
		op, size := cpu.getAluOperand(addresingMode)
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(op), 0, 0)
		cpu.Yidx = op
		cpu.Pc += uint16(size)
	case LDX:
		const zeroPageY uint8 = 0b101
		var op uint8
		var size uint8
		if addresingMode == zeroPageY {
			op, size = cpu.getAluOperandZeroPageY()
		} else {
			op, size = cpu.getAluOperand(addresingMode)
		}
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(op), 0, 0)
		cpu.Xidx = op
		cpu.Pc += uint16(size)
	case TAY:
		cpu.Yidx = cpu.Acc
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Acc), 0, 0)
		cpu.Pc += 1
	case TAX:
		cpu.Xidx = cpu.Acc
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Acc), 0, 0)
		cpu.Pc += 1
	case CPY:
		op, size := cpu.getAluOperand(addresingMode)
		cpu.setCompareFlags(op, cpu.Yidx)
		cpu.Pc += uint16(size)
	case CPX:
		op, size := cpu.getAluOperand(addresingMode)
		cpu.setCompareFlags(op, cpu.Xidx)
		cpu.Pc += uint16(size)
	case INY:
		cpu.Yidx += 1
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Yidx), 0, 0)
		cpu.Pc += 1
	case INX:
		cpu.Xidx += 1
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Xidx), 0, 0)
		cpu.Pc += 1
	case TSX:
		cpu.Xidx = cpu.Sptr
		cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Sptr), 0, 0)
		cpu.Pc += 1
	case TXS:
		cpu.Sptr = cpu.Xidx
		cpu.Pc += 1
	}
}

// Receives an adressingMode, and returns the operand and total size of the instruction
func (cpu *Cpu) getAluOperand(addressingMode uint8) (uint8, uint8) {
	const idxindirectX uint8 = 0b000
	const zeroPage uint8 = 0b001
	const immediate uint8 = 0b010
	const absolute uint8 = 0b011
	const indirectidxY uint8 = 0b100
	const zeroPageX uint8 = 0b101
	const absoluteY uint8 = 0b110
	const absoluteX uint8 = 0b111

	address, size := cpu.getAluAddress(addressingMode)
	return MainMemory.memRead(address), size
}

// A small ammount of instructions actually use the ZeroPage with the Y register
// instead of the default X register, such as LDX, STX
func (cpu *Cpu) getAluOperandZeroPageY() (uint8, uint8) {
	address, size := cpu.getAluAddressZeroPageY()
	return MainMemory.memRead(address), size
}

// A small ammount of instructions actually use the ZeroPage with the Y register
// instead of the default X register, such as LDX, STX
func (cpu *Cpu) getAluAddressZeroPageY() (uint16, uint8) {
	return uint16(MainMemory.memRead(cpu.Pc+1) + cpu.Yidx), 2
}

// Receives an adressingMode, and returns the address and total size of the instruction
func (cpu *Cpu) getAluAddress(addressingMode uint8) (uint16, uint8) {
	const idxindirectX uint8 = 0b000
	const zeroPage uint8 = 0b001
	const immediate uint8 = 0b010
	const absolute uint8 = 0b011
	const indirectidxY uint8 = 0b100
	const zeroPageX uint8 = 0b101
	const absoluteY uint8 = 0b110
	const absoluteX uint8 = 0b111

	switch addressingMode {
	case idxindirectX:
		zeroPageAddress := MainMemory.memRead(cpu.Pc + 1)
		return MainMemory.memRead16(uint16(zeroPageAddress + cpu.Xidx)), 2
	case zeroPage:
		return uint16(MainMemory.memRead(cpu.Pc + 1)), 2
	case immediate:
		return uint16(cpu.Pc + 1), 2
	case absolute:
		return MainMemory.memRead16(cpu.Pc + 1), 3
	case indirectidxY:
		return MainMemory.memRead16(cpu.Pc+1) + uint16(cpu.Yidx), 2
	case zeroPageX:
		return uint16(MainMemory.memRead(cpu.Pc+1) + cpu.Xidx), 2
	case absoluteY:
		return MainMemory.memRead16(cpu.Pc+1) + uint16(cpu.Yidx), 3
	case absoluteX:
		return MainMemory.memRead16(cpu.Pc+1) + uint16(cpu.Xidx), 3
	default:
		log.Printf("Warning: cpu.getAluAddress invalid addressingMode: %b\n", addressingMode)
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
		log.Printf("Warning: cpu.setFlag invalid flag: %s\n", flag)
	}
}

func (cpu *Cpu) getFlag(flag string) uint8 {
	var mask uint8 = 0b00000001
	switch flag {
	case Carry:
		return cpu.Psts & mask
	case Zero:
		return (cpu.Psts >> 1) & mask
	case InterruptDisable:
		return (cpu.Psts >> 2) & mask
	case Decimal:
		return (cpu.Psts >> 3) & mask
	case Overflow:
		return (cpu.Psts >> 6) & mask
	case Negative:
		return (cpu.Psts >> 7) & mask
	default:
		log.Printf("Warning: cpu.getFlag invalid flag: %s, returning 0 instead\n", flag)
		return 0
	}
}

// Receivesa list with all the that should be set by the instruction
func (cpu *Cpu) calcAndSetFlags(flags []string, result uint16, reg uint8, operand uint8) {
	for _, flag := range flags {
		switch flag {
		case Carry:
			if result > 0xff {
				cpu.setFlag(Carry, 1)
			} else {
				cpu.setFlag(Carry, 0)
			}
		case Zero:
			if result == 0 {
				cpu.setFlag(Zero, 1)
			} else {
				cpu.setFlag(Zero, 0)
			}
		case Overflow:
			if ((uint8(result) ^ reg) & (uint8(result) ^ operand) & 0x80) == 1 {
				cpu.setFlag(Overflow, 1)
			} else {
				cpu.setFlag(Overflow, 0)
			}
		case Negative:
			if (result >> 7 & 0x01) == 1 {
				cpu.setFlag(Negative, 1)
			} else {
				cpu.setFlag(Negative, 0)
			}
		}
	}
}

func (cpu *Cpu) pushToStack(val uint8) {
	MainMemory.memWrite(uint16(cpu.Sptr)+0x0100, val)
	cpu.Sptr -= 1
}

func (cpu *Cpu) pullFromStack() uint8 {
	cpu.Sptr += 1
	return MainMemory.memRead(uint16(cpu.Sptr) + 0x0100)
}

func (cpu *Cpu) pushToStack16(val uint16) {
	cpu.pushToStack(uint8(val >> 8))
	cpu.pushToStack(uint8(val))
}

func (cpu *Cpu) pullFromStack16() uint16 {
	low := cpu.pullFromStack()
	high := cpu.pullFromStack()
	return (uint16(high) << 8) + uint16(low)
}

// executes simples branch branch if flag is equal to val
func (cpu *Cpu) branchIfFlag(flag string, val uint8) {
	if cpu.getFlag(flag) == val {
		cpu.Pc = uint16(int16(cpu.Pc) + int16(2) + int16(int8(MainMemory.memRead(cpu.Pc+1))))
	} else {
		cpu.Pc += 2
	}
}

func (cpu *Cpu) setCompareFlags(operand uint8, reg uint8) {
	var result uint8 = reg - operand
	cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
	if reg >= operand {
		cpu.setFlag(Carry, 1)
	} else {
		cpu.setFlag(Carry, 0)
	}
}

func (cpu *Cpu) printStatus() {
	fmt.Println("####  Current CPU Register Status  ####")
	fmt.Printf("Program Counter: %4x   Accumulator: %2x\n", cpu.Pc, cpu.Acc)
	fmt.Printf("X Index: %2x   Y Index: %2x Stack Pointer: %2x\n", cpu.Xidx, cpu.Yidx, cpu.Sptr)
	fmt.Printf("Carry: %1b Zero: %1b InterruptDisable: %1b Decimal %1b Overflow %1b Negative %1b\n",
		cpu.getFlag(Carry), cpu.getFlag(Zero), cpu.getFlag(InterruptDisable),
		cpu.getFlag(Decimal), cpu.getFlag(Overflow), cpu.getFlag(Negative))
	fmt.Println("#### Current CPU Register Status  ####")
}
