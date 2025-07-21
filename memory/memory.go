package memory

import (
	"fmt"
	"os"
	"vsasakiv/nesemulator/apu"
	"vsasakiv/nesemulator/controller"
	"vsasakiv/nesemulator/mappers"
	"vsasakiv/nesemulator/ppu"
)

// PPU registers mapped to CPU
const PPUCTRL = 0x2000
const PPUMASK = 0x2001
const PPUSTATUS = 0x2002
const OAMADDR = 0x2003
const OAMDATA = 0x2004
const PPUSCROLL = 0x2005
const PPUADDR = 0x2006
const PPUDATA = 0x2007
const OAMDMA = 0x4014
const APU_PULSE1_DUTY = 0x4000
const APU_PULSE1_SWEEP = 0x4001
const APU_PULSE1_TIMER_LOW = 0x4002
const APU_PULSE1_TIMER_HIGH = 0x4003

const APU_PULSE2_DUTY = 0x4004
const APU_PULSE2_SWEEP = 0x4005
const APU_PULSE2_TIMER_LOW = 0x4006
const APU_PULSE2_TIMER_HIGH = 0x4007

const APU_TRIANGLE_CONTROL = 0x4008
const APU_TRIANGLE_TIMER_LOW = 0x400A
const APU_TRIANGLE_TIMER_HIGH = 0x400B

const APU_NOISE_CONTROL = 0x400C
const APU_NOISE_PERIOD = 0x400E
const APU_NOISE_LENGTH_COUNTER = 0x400F

const APU_STATUS = 0x4015

const CONTROLLER1 = 0x4016

type Memory struct {
	ram             [0x0800]uint8
	Mapper          mappers.Mapper
	OamDmaInterrupt bool
	OamDmaPage      uint8
}

var joyPad1 *controller.JoyPad
var MainMemory Memory

// copy of memory, where we have a 1 where the memory was touched in some point, only for debug
// and dumping purposes
var modified Memory
var debug bool = true

func LoadCartridge(mapper mappers.Mapper) {
	MainMemory.Mapper = mapper
}

func ConnectJoyPad1(joyPad *controller.JoyPad) {
	joyPad1 = joyPad
}

func MemRead(addr uint16) uint8 {
	switch {
	case addr <= 0x07FF:
		return MainMemory.ram[addr]
	// ppu registers mapped to cpu memory
	case addr >= 0x2000 && addr <= 0x3FFF:
		addr = ((addr - 0x2000) % 0x0008) + 0x2000
		switch addr {
		case PPUSTATUS:
			return ppu.GetPpu().ReadPpuStatusRegister()
		case OAMDATA:
			return ppu.GetPpu().ReadOamDataRegister()
		case PPUDATA:
			return ppu.GetPpu().ReadPpuDataRegister()
		}
	// OAMDMA returns placeholder 0x40
	case addr == OAMDMA:
		return 0x40
	case addr == CONTROLLER1:
		return joyPad1.ReceiveRead()
	case addr >= 0x6000:
		return MainMemory.Mapper.Read(addr)
	}
	return 0
}

func MemRead16(addr uint16) uint16 {
	switch {
	case addr <= 0x07FF:
		// zero page reading, should wrap
		var low, high uint16
		if addr == 0x00FF {
			low = uint16(MemRead(addr))
			high = uint16(MemRead(0)) << 8
		} else {
			low = uint16(MemRead(addr))
			high = uint16(MemRead(addr+1)) << 8
		}
		return high + low
	case addr >= 0x6000:
		low := uint16(MemRead(addr))
		high := uint16(MemRead(addr+1)) << 8
		return high + low
	}
	return 0
}

func MemWrite(addr uint16, val uint8) {
	switch {
	// cpu RAM
	case addr <= 0x07FF:
		if debug {
			modified.ram[addr] = 1
		}
		MainMemory.ram[addr] = val
	// ppu registers mapped to cpu memory
	case addr >= 0x2000 && addr <= 0x3FFF:
		addr = ((addr - 0x2000) % 0x0008) + 0x2000
		switch addr {
		case PPUCTRL:
			ppu.GetPpu().WriteToPpuControl(val)
		case PPUMASK:
			ppu.GetPpu().WriteToPpuMask(val)
		case PPUSCROLL:
			ppu.GetPpu().WriteToPpuScroll(val)
		case OAMADDR:
			ppu.GetPpu().WriteToOamAddrRegister(val)
		case OAMDATA:
			ppu.GetPpu().WriteToOamDataRegister(val)
		case PPUADDR:
			ppu.GetPpu().WriteToAddrRegister(val)
		case PPUDATA:
			ppu.GetPpu().WriteToPpuDataRegister(val)
		}
		// APU registers
	case addr == APU_PULSE1_DUTY:
		apu.GetApu().Pulse1.WriteToDutyCycleAndVolume(val)
	case addr == APU_PULSE1_SWEEP:
		apu.GetApu().Pulse1.WriteToSweep(val)
	case addr == APU_PULSE1_TIMER_LOW:
		apu.GetApu().Pulse1.WriteToTimerLow(val)
	case addr == APU_PULSE1_TIMER_HIGH:
		apu.GetApu().Pulse1.WriteToTimerHigh(val)

	case addr == APU_PULSE2_DUTY:
		apu.GetApu().Pulse2.WriteToDutyCycleAndVolume(val)
	case addr == APU_PULSE2_SWEEP:
		apu.GetApu().Pulse2.WriteToSweep(val)
	case addr == APU_PULSE2_TIMER_LOW:
		apu.GetApu().Pulse2.WriteToTimerLow(val)
	case addr == APU_PULSE2_TIMER_HIGH:
		apu.GetApu().Pulse2.WriteToTimerHigh(val)

	case addr == APU_TRIANGLE_CONTROL:
		apu.GetApu().Triangle.WriteToLinearCounter(val)
	case addr == APU_TRIANGLE_TIMER_LOW:
		apu.GetApu().Triangle.WriteToTimerLow(val)
	case addr == APU_TRIANGLE_TIMER_HIGH:
		apu.GetApu().Triangle.WriteToTimerHigh(val)

	case addr == APU_NOISE_CONTROL:
		apu.GetApu().Noise.WriteToVolume(val)
	case addr == APU_NOISE_PERIOD:
		apu.GetApu().Noise.WriteToModeAndPeriod(val)
	case addr == APU_NOISE_LENGTH_COUNTER:
		apu.GetApu().Noise.WriteToLengthCounter(val)

	case addr == APU_STATUS:
		apu.GetApu().WriteToStatusRegister(val)
	// OAMDMA, using interrupt
	case addr == OAMDMA:
		MainMemory.OamDmaInterrupt = true
		MainMemory.OamDmaPage = val
	case addr == CONTROLLER1:
		joyPad1.ReceiveWrite(val)
	// cartridge
	case addr >= 0x6000:
		MainMemory.Mapper.Write(addr, val)
	}
}

func MemWrite16(addr uint16, val uint16) {
	switch {
	case addr <= 0x07FF:
		if debug {
			modified.ram[addr] = 1
			modified.ram[addr+1] = 1
		}
		MemWrite(addr, uint8(val&0xff))
		MemWrite(addr+1, uint8((val>>8)&0xff))
	case addr >= 0x6000:
		fmt.Println("Warning: cant write to ROM")
		return
	}
}

func PoolOamDmaInterrupt() bool {
	if MainMemory.OamDmaInterrupt {
		MainMemory.OamDmaInterrupt = false
		return true
	}
	return false
}

func HexDump(filename string) {

	content := ""

	for i := range 0xffff {
		if modified.ram[i] == 1 {
			content += fmt.Sprintf("%4x : %2x\n", i, MemRead(uint16(i)))
		}
	}

	file, err := os.Create(filename)

	if err != nil {
		fmt.Println("Error creating the file", err)
		return
	}

	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println("Error writing to file", err)
		return
	}

	fmt.Println("Successfully dumped memory to ", filename)

}
