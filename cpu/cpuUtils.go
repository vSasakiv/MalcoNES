package cpu

import (
	"fmt"
	"log"
	"os"
	"strings"
	"vsasakiv/nesemulator/memory"
)

const idxindirectX uint8 = 0b000
const zeroPage uint8 = 0b001
const immediate uint8 = 0b010
const absolute uint8 = 0b011
const indirectidxY uint8 = 0b100
const zeroPageX uint8 = 0b101
const absoluteY uint8 = 0b110
const absoluteX uint8 = 0b111

// Receives an adressingMode, and returns the operand and total size of the instruction
func (cpu *Cpu) getAluOperand(addressingMode uint8) (uint8, uint8) {
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
		nextByte := memory.MemRead(cpu.Pc + 1)
		return memory.MemRead16(uint16(nextByte)) + uint16(cpu.Yidx), 2
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
		if val == 1 {
			cpu.Psts |= 1
		} else {
			cpu.Psts &^= 1
		}
	case Zero:
		if val == 1 {
			cpu.Psts |= 1 << 1
		} else {
			cpu.Psts &^= 1 << 1
		}
	case InterruptDisable:
		if val == 1 {
			cpu.Psts |= 1 << 2
		} else {
			cpu.Psts &^= 1 << 2
		}
	case Decimal:
		if val == 1 {
			cpu.Psts |= 1 << 3
		} else {
			cpu.Psts &^= 1 << 3
		}
	case Overflow:
		if val == 1 {
			cpu.Psts |= 1 << 6
		} else {
			cpu.Psts &^= 1 << 6
		}
	case Negative:
		if val == 1 {
			cpu.Psts |= 1 << 7
		} else {
			cpu.Psts &^= 1 << 7
		}
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
			if uint8(result) == 0 {
				cpu.setFlag(Zero, 1)
			} else {
				cpu.setFlag(Zero, 0)
			}
		case Overflow:
			if ((uint8(result) ^ reg) & (uint8(result) ^ operand) & 0x80) != 0 {
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
	if isIllegal(opcode) {
		instructionHex += "*"
	} else {
		instructionHex += " "
	}

	instructionMnemonic := cpu.opcodeTable[opcode] + " "
	instructionOp := getOperand(opcode, cpu.Pc)
	instructionOp = instructionOp + strings.Repeat(" ", 28-len(instructionOp))
	registers := cpu.getRegisters()
	return pc + instructionHex + instructionMnemonic + instructionOp + registers
}

func (cpu *Cpu) RunAndTraceToFile(path string) {
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("runAndTraceToFile Could not open file! ", err)
	}
	defer file.Close()
	var trace string

	for {
		if memory.MemRead(cpu.Pc) == 0x00 {
			fmt.Println("BRK instruction found, stopping execution")
			return
		}
		trace = cpu.TraceStatus() + "\n"
		file.WriteString(trace)
		ExecuteNext()
	}
}

func (cpu *Cpu) getRegisters() string {
	return fmt.Sprintf("A:%02X X:%02X Y:%02X P:%02X SP:%02X", cpu.Acc, cpu.Xidx, cpu.Yidx, cpu.Psts, cpu.Sptr)
}

func getOperand(opcode uint8, pc uint16) string {
	mnemonic := cpu.opcodeTable[opcode]
	addresingMode := (opcode >> 2) & 0b00000111

	// exceptions to the rule
	switch mnemonic {
	case BPL, BMI, BVC, BVS, BCC, BCS, BNE, BEQ:
		address := uint16(int16(cpu.Pc) + int16(2) + int16(int8(memory.MemRead(cpu.Pc+1))))
		return fmt.Sprintf("$%04X", address)
	case BRK, RTI, RTS, PHP, PLP, PHA, PLA, DEY, TAY, INY, INX,
		CLC, SEC, CLI, SEI, TYA, CLV, CLD, SED, TXA, TAX, DEX,
		TXS, TSX:
		return ""
	}
	switch opcode {
	//   JMP
	case 0x6C:
		aluAddress, _ := cpu.getAluAddress(addresingMode)
		var result uint16
		if aluAddress&0x00FF == 0x00FF {
			low := uint16(memory.MemRead(aluAddress))
			high := uint16(memory.MemRead(aluAddress&0xFF00)) << 8
			result = high + low
		} else {
			result = memory.MemRead16(aluAddress)
		}
		return fmt.Sprintf("($%04X) = %04X", memory.MemRead16(pc+1), result)
	//   LDY   CPY   CPX   LDX
	case 0xA0, 0xC0, 0xE0, 0xA2:
		return fmt.Sprintf("#$%02X", memory.MemRead(pc+1))
	//   JSR
	case 0x20, 0x4C:
		return fmt.Sprintf("$%04X", memory.MemRead16(pc+1))
	//   ASL   ROL   LSR   ROR
	case 0x0A, 0x2A, 0x4A, 0x6A:
		return "A"
	//   STX   LDX   SAX   LAX
	case 0x96, 0xB6, 0x97, 0xB7:
		address, _ := cpu.getAluAddressZeroPageY()
		op, _ := cpu.getAluOperandZeroPageY()
		return fmt.Sprintf("$%02X,Y @ %02X = %02X", memory.MemRead(pc+1), address, op)
	//   LDX   SHX   SHA   LAX
	case 0xBE, 0x9E, 0x9F, 0xBF:
		address, _ := cpu.getAluAddress(absoluteY)
		op, _ := cpu.getAluOperand(absoluteY)
		return fmt.Sprintf("$%04X,Y @ %04X = %02X", memory.MemRead16(pc+1), address, op)
	//   size 1 nops
	case 0xEA, 0x1A, 0x3A, 0x5A, 0x7A, 0xDA, 0xFA:
		return ""
	//   size 2 immediate nops
	case 0x80, 0x82, 0xC2, 0xE2:
		op, _ := cpu.getAluOperand(immediate)
		return fmt.Sprintf("#$%02X", op)
	}

	switch addresingMode {
	case idxindirectX:
		nextByte := memory.MemRead(pc + 1)
		address, _ := cpu.getAluAddress(idxindirectX)
		op, _ := cpu.getAluOperand(idxindirectX)
		return fmt.Sprintf("($%02X,X) @ %02X = %04X = %02X", nextByte, nextByte+cpu.Xidx, address, op)
	case zeroPage:
		op, _ := cpu.getAluOperand(zeroPage)
		return fmt.Sprintf("$%02X = %02X", memory.MemRead(pc+1), op)
	case immediate:
		op, _ := cpu.getAluOperand(immediate)
		return fmt.Sprintf("#$%02X", op)
	case absolute:
		address, _ := cpu.getAluAddress(absolute)
		op, _ := cpu.getAluOperand(absolute)
		return fmt.Sprintf("$%04X = %02X", address, op)
	case indirectidxY:
		nextByte := memory.MemRead(pc + 1)
		prevAddress := memory.MemRead16(uint16(nextByte))
		address, _ := cpu.getAluAddress(indirectidxY)
		op, _ := cpu.getAluOperand(indirectidxY)
		return fmt.Sprintf("($%02X),Y = %04X @ %04X = %02X", nextByte, prevAddress, address, op)
	case zeroPageX:
		address, _ := cpu.getAluAddress(zeroPageX)
		op, _ := cpu.getAluOperand(zeroPageX)
		return fmt.Sprintf("$%02X,X @ %02X = %02X", memory.MemRead(pc+1), address, op)
	case absoluteY:
		address, _ := cpu.getAluAddress(absoluteY)
		op, _ := cpu.getAluOperand(absoluteY)
		return fmt.Sprintf("$%04X,Y @ %04X = %02X", memory.MemRead16(pc+1), address, op)
	case absoluteX:
		address, _ := cpu.getAluAddress(absoluteX)
		op, _ := cpu.getAluOperand(absoluteX)
		return fmt.Sprintf("$%04X,X @ %04X = %02X", memory.MemRead16(pc+1), address, op)
	}
	return ""
}

func isIllegal(opcode uint8) bool {
	mnemonic := cpu.opcodeTable[opcode]
	switch mnemonic {
	case SLO, ANC, RLA, SRE, ALR, RRA, SAX, SHA, SHX, SHY, TAS, LAX, LAS, DCP, AXS, ISB:
		return true
	case NOP:
		// special case of NOP
		if opcode != 0xEA {
			return true
		}
	}
	// special case of SBC
	if opcode == 0xEB {
		return true
	}

	return false
}

func getInstructionSize(opcode uint8) uint8 {
	mnemonic := cpu.opcodeTable[opcode]
	addresingMode := (opcode >> 2) & 0b00000111

	switch mnemonic {
	case NOP:
		switch opcode {
		// implict nops size 1
		case 0xEA, 0x1A, 0x3A, 0x5A, 0x7A, 0xDA, 0xFA:
			return 1
		// immediate nops size 2
		case 0x80, 0x82, 0xC2, 0xE2:
			return 2
		// remaining nops, size dependent on addressing
		default:
			_, size := cpu.getAluOperand(addresingMode)
			return size
		}
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
