package cpu

import (
	"fmt"
	"log"
	"vsasakiv/nesemulator/memory"
)

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
	return memory.MemRead(address), size
}

// A small ammount of instructions actually use the ZeroPage with the Y register
// instead of the default X register, such as LDX, STX
func (cpu *Cpu) getAluOperandZeroPageY() (uint8, uint8) {
	address, size := cpu.getAluAddressZeroPageY()
	return memory.MemRead(address), size
}

// A small ammount of instructions actually use the ZeroPage with the Y register
// instead of the default X register, such as LDX, STX
func (cpu *Cpu) getAluAddressZeroPageY() (uint16, uint8) {
	return uint16(memory.MemRead(cpu.Pc+1) + cpu.Yidx), 2
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
		zeroPageAddress := memory.MemRead(cpu.Pc + 1)
		return memory.MemRead16(uint16(zeroPageAddress + cpu.Xidx)), 2
	case zeroPage:
		return uint16(memory.MemRead(cpu.Pc + 1)), 2
	case immediate:
		return uint16(cpu.Pc + 1), 2
	case absolute:
		return memory.MemRead16(cpu.Pc + 1), 3
	case indirectidxY:
		return memory.MemRead16(cpu.Pc+1) + uint16(cpu.Yidx), 2
	case zeroPageX:
		return uint16(memory.MemRead(cpu.Pc+1) + cpu.Xidx), 2
	case absoluteY:
		return memory.MemRead16(cpu.Pc+1) + uint16(cpu.Yidx), 3
	case absoluteX:
		return memory.MemRead16(cpu.Pc+1) + uint16(cpu.Xidx), 3
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
	memory.MemWrite(uint16(cpu.Sptr)+0x0100, val)
	cpu.Sptr -= 1
}

func (cpu *Cpu) pullFromStack() uint8 {
	cpu.Sptr += 1
	return memory.MemRead(uint16(cpu.Sptr) + 0x0100)
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
		cpu.Pc = uint16(int16(cpu.Pc) + int16(2) + int16(int8(memory.MemRead(cpu.Pc+1))))
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

func (cpu *Cpu) TraceStatus() string {
	opcode := memory.MemRead(cpu.Pc)

	pc := fmt.Sprintf("%04X  ", cpu.Pc)
	size := getInstructionSize(opcode)
	instructionHex := ""
	for i := range size {
		instructionHex += fmt.Sprintf("%02X ", memory.MemRead(cpu.Pc+uint16(i)))
	}
	for range 3 - size {
		instructionHex += "   "
	}
	instructionHex += " "
	instructionMnemonic := cpu.opcodeTable[opcode] + " "
	instructionOp := getOperand(opcode)
	return pc + instructionHex + instructionMnemonic
}

func getOperand(opcode uint8) string {

}

func getInstructionSize(opcode uint8) uint8 {
	mnemonic := cpu.opcodeTable[opcode]
	addresingMode := (opcode >> 2) & 0b00000111

	switch mnemonic {
	case ASL, ROL, LSR, ROR:
		if opcode == 0x0A || opcode == 0x2A || opcode == 0x4A || opcode == 0x6A {
			return 1
		} else {
			_, size := cpu.getAluAddress(addresingMode)
			return size
		}
	case BRK, PHP, PLP, PHA, PLA, RTI, RTS, CLC, SEC,
		CLI, SEI, CLV, CLD, SED, DEY, DEX, TYA, TXA,
		TAY, TAX, INY, INX, TSX, TXS:
		return 1
	case BPL, BMI, BVC, BVS, BCC, BCS, BNE, BEQ:
		return 2
	case JMP, JSR:
		return 3
	}
	_, size := cpu.getAluAddress(addresingMode)
	return size
}
