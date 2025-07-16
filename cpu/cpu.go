package cpu

import (
	"vsasakiv/nesemulator/memory"
	"vsasakiv/nesemulator/ppu"
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
	cycles                      uint
	cpuCycle                    uint
	clockCounter                uint
	hasNmiInterrupt             bool
	hasOamDmaInterrupt          bool
	hasIrqInterrupt             bool
}

// Initialize cpu with corret parameters, also initialize instructionTable
func NewCpu() *Cpu {
	var cpu Cpu
	cpu.Psts = 0b00100100
	cpu.Sptr = 0xFD
	cpu.opcodeTable = Generate()
	cpu.cycles = 7
	return &cpu
}

func (cpu *Cpu) Reset() {
	cpu.Acc = 0
	cpu.Xidx = 0
	cpu.Yidx = 0
	cpu.Psts = 0b00100100
	cpu.Sptr = 0xFD
	cpu.cpuCycle = 0
	cpu.clockCounter = 0
	cpu.hasNmiInterrupt = false
	cpu.hasOamDmaInterrupt = false
	cpu.hasIrqInterrupt = false
	cpu.Pc = memory.MemRead16(0xFFFC)
}

func GetCpu() *Cpu {
	return &cpu
}

var cpu Cpu = *NewCpu()

func Clock() {
	cpu.clockCounter++
	if cpu.clockCounter == 3 {
		cpu.cpuCycle++
		cpu.clockCounter = 0
	}

	switch {
	case cpu.hasNmiInterrupt:
		if cpu.cpuCycle == 7 {
			cpu.treatNmiInterrupt()
			cpu.hasNmiInterrupt = false
			cpu.cpuCycle = 0
		}
		return
	case cpu.hasOamDmaInterrupt:
		if cpu.cpuCycle == 514 {
			cpu.OamDmaWrite(memory.MainMemory.OamDmaPage)
			cpu.hasOamDmaInterrupt = false
			cpu.cpuCycle = 0
		}
		return
	case cpu.hasIrqInterrupt:
		if cpu.cpuCycle == 7 {
			cpu.treatIrqInterrupt()
			cpu.hasIrqInterrupt = false
			cpu.cpuCycle = 0
		}
		return
	}

	if ppu.GetPpu().PollForNmiInterrupt() {
		cpu.hasNmiInterrupt = true
		return
	}

	if memory.PoolOamDmaInterrupt() {
		cpu.hasOamDmaInterrupt = true
		return
	}

	if memory.MainMemory.Mapper.PollInterrupt() && cpu.getFlag(InterruptDisable) == 0 {
		cpu.hasIrqInterrupt = true
		return
	}

	instruction := memory.MemRead(cpu.Pc)
	opcode := cpu.opcodeTable[instruction]
	addresingMode := (instruction >> 2) & 0b00000111
	// execute next
	if cpu.calcCycles(instruction, opcode, addresingMode) == cpu.cpuCycle {
		cpu.cpuCycle = 0
		// cpu.LastInstructionCycles = cpu.calcCycles(instruction, opcode, addresingMode)
		// cpu.cycles += cpu.LastInstructionCycles
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
			memory.MemWrite(address, cpu.Acc)
			cpu.Pc += uint16(size)
		case LDA:
			address, size := cpu.getAluAddress(addresingMode)
			var result uint8 = memory.MemRead(address)
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
			cpu.Acc = uint8(result)
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
				op := memory.MemRead(address)
				result := op << 1
				cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
				cpu.setFlag(Carry, op>>7)

				memory.MemWrite(address, result)
				cpu.Pc += uint16(size)
			}
		case ROL:
			// has accumulator addressing
			if instruction == 0x2A {
				carry := (cpu.Acc & 0b10000000) >> 7
				result := (cpu.Acc << 1) + cpu.getFlag(Carry)
				cpu.setFlag(Carry, carry)
				cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

				cpu.Acc = result
				cpu.Pc += 1
			} else {
				address, size := cpu.getAluAddress(addresingMode)
				op := memory.MemRead(address)
				carry := (op & 0b10000000) >> 7
				result := (op << 1) + cpu.getFlag(Carry)
				cpu.setFlag(Carry, carry)
				cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

				memory.MemWrite(address, result)
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
				op := memory.MemRead(address)
				result := op >> 1
				cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
				cpu.setFlag(Carry, op&0b1)

				memory.MemWrite(address, result)
				cpu.Pc += uint16(size)
			}
		case ROR:
			// has accumulator addressing
			if instruction == 0x6A {
				carry := cpu.Acc & 0b1
				result := (cpu.Acc >> 1) + (cpu.getFlag(Carry) << 7)
				cpu.setFlag(Carry, carry)
				cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

				cpu.Acc = result
				cpu.Pc += 1
			} else {
				address, size := cpu.getAluAddress(addresingMode)
				op := memory.MemRead(address)
				carry := op & 0b1
				result := (op >> 1) + (cpu.getFlag(Carry) << 7)
				cpu.setFlag(Carry, carry)
				cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

				memory.MemWrite(address, result)
				cpu.Pc += uint16(size)
			}
		case DEC:
			address, size := cpu.getAluAddress(addresingMode)
			result := memory.MemRead(address) - 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			memory.MemWrite(address, result)
			cpu.Pc += uint16(size)
		case INC:
			address, size := cpu.getAluAddress(addresingMode)
			result := memory.MemRead(address) + 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			memory.MemWrite(address, result)
			cpu.Pc += uint16(size)
		case SLO:
			address, size := cpu.getAluAddress(addresingMode)
			op := memory.MemRead(address)
			cpu.setFlag(Carry, op>>7)
			result := (op<<1 | cpu.Acc)
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			memory.MemWrite(address, op<<1)
			cpu.Acc = result
			cpu.Pc += uint16(size)
		case RLA:
			address, size := cpu.getAluAddress(addresingMode)
			op := memory.MemRead(address)
			carry := (op & 0b10000000) >> 7
			mem := (op << 1) + cpu.getFlag(Carry)
			result := mem & cpu.Acc
			cpu.setFlag(Carry, carry)
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			memory.MemWrite(address, mem)
			cpu.Acc = result
			cpu.Pc += uint16(size)
		case SRE:
			address, size := cpu.getAluAddress(addresingMode)
			op := memory.MemRead(address)
			cpu.setFlag(Carry, op&0b1)
			result := (op >> 1) ^ cpu.Acc
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)

			memory.MemWrite(address, op>>1)
			cpu.Acc = result
			cpu.Pc += uint16(size)
		case RRA:
			address, size := cpu.getAluAddress(addresingMode)
			op := memory.MemRead(address)
			carry := op & 0b1
			mem := (op >> 1) + (cpu.getFlag(Carry) << 7)
			cpu.setFlag(Carry, carry)

			var result uint16 = uint16(cpu.Acc) + uint16(mem) + uint16(cpu.getFlag(Carry))
			cpu.calcAndSetFlags([]string{Carry, Zero, Overflow, Negative}, result, cpu.Acc, mem)
			memory.MemWrite(address, mem)
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
			result := cpu.Xidx & cpu.Acc
			memory.MemWrite(aluAddress, result)
			cpu.Pc += uint16(size)
		case LAX:
			const zeroPageY uint8 = 0b101
			var op uint8
			var size uint8
			if addresingMode == zeroPageY {
				op, size = cpu.getAluOperandZeroPageY()
			} else if instruction == 0xBF {
				// absolute Y
				op, size = cpu.getAluOperand(absoluteY)
			} else {
				op, size = cpu.getAluOperand(addresingMode)
			}
			cpu.Acc = op
			cpu.Xidx = op
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(op), 0, 0)
			cpu.Pc += uint16(size)
		case DCP:
			address, size := cpu.getAluAddress(addresingMode)
			result := memory.MemRead(address) - 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			memory.MemWrite(address, result)
			cpu.setCompareFlags(result, cpu.Acc)
			cpu.Pc += uint16(size)
		case ISB:
			address, size := cpu.getAluAddress(addresingMode)
			result := memory.MemRead(address) + 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			memory.MemWrite(address, result)

			var subResult uint16 = uint16(cpu.Acc) + uint16(^result) + uint16(cpu.getFlag(Carry))
			cpu.calcAndSetFlags([]string{Carry, Zero, Overflow, Negative}, uint16(subResult), cpu.Acc, ^result)
			cpu.Acc = uint8(subResult)
			cpu.Pc += uint16(size)
		case ANC:
			op, size := cpu.getAluOperand(addresingMode)
			result := cpu.Acc & op
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			cpu.setFlag(Carry, cpu.getFlag(Negative))
			cpu.Acc = result
			cpu.Pc += uint16(size)
		case ALR:
			op, size := cpu.getAluOperand(addresingMode)
			andResult := op & cpu.Acc
			result := andResult >> 1
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			cpu.setFlag(Carry, andResult&0b1)
			cpu.Acc = result
			cpu.Pc += uint16(size)
		case ARR:
			op, size := cpu.getAluOperand(addresingMode)
			andResult := op & cpu.Acc
			result := (andResult >> 1) + (andResult << 7)
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			bit6 := (result >> 6) & 1
			bit5 := (result >> 5) & 1
			if bit6 == 1 && bit5 == 1 {
				cpu.setFlag(Carry, 1)
				cpu.setFlag(Overflow, 0)
			} else if bit6 == 0 && bit5 == 0 {
				cpu.setFlag(Carry, 0)
				cpu.setFlag(Overflow, 0)
			} else if bit6 == 0 && bit5 == 1 {
				cpu.setFlag(Carry, 0)
				cpu.setFlag(Overflow, 1)
			} else {
				cpu.setFlag(Carry, 1)
				cpu.setFlag(Overflow, 1)
			}
			cpu.Acc = result
			cpu.Pc += uint16(size)
		case LXA:
			op, size := cpu.getAluOperand(addresingMode)
			result := op & cpu.Acc
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			cpu.Acc = result
			cpu.Xidx = result
			cpu.Pc += uint16(size)
		case AXS:
			op, size := cpu.getAluOperand(addresingMode)
			andResult := cpu.Acc & cpu.Xidx
			cpu.setCompareFlags(op, andResult)
			cpu.Xidx = andResult - op
			cpu.Pc += uint16(size)
		case SHA:
			address, size := cpu.getAluAddress(addresingMode)
			result := (cpu.Xidx & cpu.Acc) & (memory.MemRead(cpu.Pc+2) + 1)
			memory.MemWrite(address, result)
			cpu.Pc += uint16(size)
		case LAS:
			op, size := cpu.getAluOperand(addresingMode)
			result := op & cpu.Sptr
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(result), 0, 0)
			cpu.Acc = result
			cpu.Xidx = result
			cpu.Sptr = result
			cpu.Pc += uint16(size)
		case TAS:
			address, size := cpu.getAluAddress(addresingMode)
			andResult := cpu.Acc & cpu.Xidx
			result := (cpu.Xidx & cpu.Acc) & (memory.MemRead(cpu.Pc+2) + 1)
			cpu.Sptr = andResult
			memory.MemWrite(address, result)
			cpu.Pc += uint16(size)

		// stack manipulating instructions
		case BRK:
			cpu.pushToStack16(cpu.Pc + 2)
			cpu.pushToStack(cpu.Psts | 0b00110000)
			cpu.setFlag(InterruptDisable, 1)
			cpu.Pc = 0xFFFE
		case PHP:
			cpu.pushToStack(cpu.Psts | 0b00010000)
			cpu.Pc += 1
		case PLP:
			// B and Unused flags
			cpu.Psts = (cpu.pullFromStack() & 0b11101111) | 0b00100000
			cpu.Pc += 1
		case PHA:
			cpu.pushToStack(cpu.Acc)
			cpu.Pc += 1
		case PLA:
			cpu.Acc = cpu.pullFromStack()
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(cpu.Acc), 0, 0)
			cpu.Pc += 1
		case RTI:
			// B and Unused flags
			cpu.Psts = (cpu.pullFromStack() & 0b11101111) | 0b00100000
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
			cpu.Pc = memory.MemRead16(cpu.Pc + 1)
		case JMP:
			aluAddress, _ := cpu.getAluAddress(addresingMode)
			// only occourence of indirect absolute
			if instruction == 0x6C {
				// instruction has bug
				if aluAddress&0x00FF == 0x00FF {
					low := uint16(memory.MemRead(aluAddress))
					high := uint16(memory.MemRead(aluAddress&0xFF00)) << 8
					cpu.Pc = high + low
				} else {
					cpu.Pc = memory.MemRead16(aluAddress)
				}
			} else {
				cpu.Pc = aluAddress
			}

		// index register manipulating instructions
		case STY:
			aluAddress, size := cpu.getAluAddress(addresingMode)
			memory.MemWrite(aluAddress, cpu.Yidx)
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
			memory.MemWrite(aluAddress, cpu.Xidx)
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
			memory.MemWrite(address, cpu.Yidx&(memory.MemRead(cpu.Pc+2)+1))
			cpu.Pc += uint16(size)
		case SHX:
			address, size := cpu.getAluAddress(addresingMode)
			memory.MemWrite(address, cpu.Xidx&(memory.MemRead(cpu.Pc+2)+1))
			cpu.Pc += uint16(size)
		case LDY:
			var op, size uint8
			// out of pattern operand
			if instruction == 0xA0 {
				op, size = cpu.getAluOperand(immediate)
			} else {
				op, size = cpu.getAluOperand(addresingMode)
			}
			cpu.calcAndSetFlags([]string{Zero, Negative}, uint16(op), 0, 0)
			cpu.Yidx = op
			cpu.Pc += uint16(size)
		case LDX:
			const zeroPageY uint8 = 0b101
			var op, size uint8
			if instruction == 0xA2 {
				op, size = cpu.getAluOperand(immediate)
			} else if instruction == 0xBE {
				// absolute Y
				op, size = cpu.getAluOperand(absoluteY)
			} else if addresingMode == zeroPageY {
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
			var op, size uint8
			// out of pattern operand
			if instruction == 0xC0 {
				op, size = cpu.getAluOperand(immediate)
			} else {
				op, size = cpu.getAluOperand(addresingMode)
			}
			cpu.setCompareFlags(op, cpu.Yidx)
			cpu.Pc += uint16(size)
		case CPX:
			var op, size uint8
			// out of pattern operand
			if instruction == 0xE0 {
				op, size = cpu.getAluOperand(immediate)
			} else {
				op, size = cpu.getAluOperand(addresingMode)
			}
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
		case NOP:
			switch instruction {
			// implict nops size 1
			case 0xEA, 0x1A, 0x3A, 0x5A, 0x7A, 0xDA, 0xFA:
				cpu.Pc += 1
			// immediate nops size 2
			case 0x80, 0x82, 0xC2, 0xE2:
				cpu.Pc += 2
			// remaining nops, size dependent on addressing
			default:
				_, size := cpu.getAluOperand(addresingMode)
				cpu.Pc += uint16(size)
			}
		}
	}
}

// treats NMI interrupt, pushes context to stack, jump to vector
func (cpu *Cpu) treatNmiInterrupt() {
	cpu.pushToStack16(cpu.Pc)
	cpu.pushToStack(cpu.Psts & 0b11101111)

	cpu.setFlag(InterruptDisable, 1)
	cpu.Pc = memory.MemRead16(0xFFFA)
}

func (cpu *Cpu) OamDmaWrite(page uint8) {
	for i := range uint16(256) {
		val := memory.MemRead((uint16(page) << 8) + i)
		memory.MemWrite(memory.OAMDATA, val)
	}
}

func (cpu *Cpu) treatIrqInterrupt() {
	cpu.pushToStack16(cpu.Pc)
	cpu.pushToStack(cpu.Psts | 0b00010000)
	cpu.setFlag(InterruptDisable, 1)
	cpu.Pc = memory.MemRead16(0xFFFE)
}
