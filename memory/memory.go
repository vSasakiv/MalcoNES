package memory

import (
	"fmt"
	"os"
	"vsasakiv/nesemulator/cartridge"
)

// type Memory [0xFFFF]uint8

type Memory struct {
	ram [0x0800]uint8
	rom cartridge.Cartridge
}

var MainMemory Memory

// copy of memory, where we have a 1 where the memory was touched in some point, only for debug
// and dumping purposes
var modified Memory
var debug bool = true

func LoadFromCartridge(cartridge cartridge.Cartridge) {
	MainMemory.rom = cartridge
}

func MemRead(addr uint16) uint8 {
	switch {
	case addr <= 0x07FF:
		return MainMemory.ram[addr]
	case addr >= 0x8000:
		return readPrgRom(addr)
	}
	return 0
}

func MemRead16(addr uint16) uint16 {
	switch {
	case addr <= 0x07FF:
		// zero page reading, should wrap
		var low, high uint16
		if addr == 0x00FF {
			low = uint16(MainMemory.ram[addr])
			high = uint16(MainMemory.ram[0x0000]) << 8
		} else {
			low = uint16(MainMemory.ram[addr])
			high = uint16(MainMemory.ram[addr+1]) << 8
		}
		return high + low
	case addr >= 0x8000:
		low := uint16(readPrgRom(addr))
		high := uint16(readPrgRom(addr+1)) << 8
		return high + low
	}
	return 0
}

func MemWrite(addr uint16, val uint8) {
	switch {
	case addr <= 0x07FF:
		if debug {
			modified.ram[addr] = 1
		}
		MainMemory.ram[addr] = val
	case addr >= 0x8000:
		fmt.Println("Warning: cant write to ROM")
		return
	}
}

func MemWrite16(addr uint16, val uint16) {
	switch {
	case addr <= 0x07FF:
		if debug {
			modified.ram[addr] = 1
			modified.ram[addr+1] = 1
		}
		MainMemory.ram[addr] = uint8(val & 0xff)
		MainMemory.ram[addr+1] = uint8((val >> 8) & 0xff)
	case addr >= 0x8000:
		fmt.Println("Warning: cant write to ROM")
		return
	}
}

func readPrgRom(addr uint16) uint8 {
	addr -= 0x8000
	// mirrors address if needed
	if MainMemory.rom.PrgRomSize == 0x4000 && addr >= 0x4000 {
		addr = addr % 0x4000
	}
	return MainMemory.rom.PrgRom[addr]
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
