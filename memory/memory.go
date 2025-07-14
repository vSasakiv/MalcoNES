package memory

import (
	"fmt"
	"os"
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

const CONTROLLER1 = 0x4016

type Memory struct {
	ram             [0x0800]uint8
	mapper          mappers.Mapper
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
	MainMemory.mapper = mapper
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
	case addr >= 0x8000:
		return MainMemory.mapper.Read(addr)
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
	case addr >= 0x8000:
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
	// OAMDMA, using interrupt
	case addr == OAMDMA:
		MainMemory.OamDmaInterrupt = true
		MainMemory.OamDmaPage = val
	case addr == CONTROLLER1:
		joyPad1.ReceiveWrite(val)
	// cartridge
	case addr >= 0x8000:
		MainMemory.mapper.Write(addr, val)
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
	case addr >= 0x8000:
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
