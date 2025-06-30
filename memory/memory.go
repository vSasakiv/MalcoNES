package memory

import (
	"fmt"
	"os"
	"vsasakiv/nesemulator/cartridge"
)

type Memory [0xFFFF]uint8

var MainMemory Memory

// copy of memory, where we have a 1 where the memory was touched in some point, only for debug
// and dumping purposes
var modified Memory
var debug bool = true

func LoadFromCartridge(cartridge cartridge.Cartridge) {
	// if size is 16KiB we have to mirror to the reamining 16KiB
	if cartridge.PrgRomSize == 0x4000 {
		copy(MainMemory[0x8000:], cartridge.PrgRom)
		copy(MainMemory[0xB000:], cartridge.PrgRom)
	} else {
		copy(MainMemory[0x8000:], cartridge.PrgRom)
	}
}

func MemRead(addr uint16) uint8 {
	return MainMemory[addr]
}

func MemRead16(addr uint16) uint16 {
	low := uint16(MainMemory[addr])
	high := uint16(MainMemory[addr+1]) << 8
	return high + low
}

func MemWrite(addr uint16, val uint8) {
	if debug {
		modified[addr] = 1
	}
	MainMemory[addr] = val
}

func MemWrite16(addr uint16, val uint16) {
	if debug {
		modified[addr] = 1
		modified[addr+1] = 1
	}
	MainMemory[addr] = uint8(val & 0xff)
	MainMemory[addr+1] = uint8((val >> 8) & 0xff)
}

func HexDump(filename string) {

	content := ""

	for i := range 0xffff {
		if modified[i] == 1 {
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
