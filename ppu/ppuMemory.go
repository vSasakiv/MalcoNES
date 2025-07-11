package ppu

import (
	"fmt"
	"os"
	"vsasakiv/nesemulator/cartridge"
)

// PPU ADDRESSING SPACE

//  _______________________ 0xFFFF
// |                      |
// |       mirrors        |
//  _______________________ 0x4000
// |                      |
// |       palletes       |
//  _______________________ 0x3F00
// |                      |
// |      name tables     |
// |        (VRAM)        |
// |                      |
//  _______________________ 0x2000
// |                      |
// |    pattern tabbles   |
// |      (chr ROM)       |
//  _______________________ 0x0000

type Memory struct {
	vram       [0x0800]uint8
	rom        cartridge.Cartridge
	paletteRam [0x0100]uint8
	oam        [0x0100]uint8
}

var PpuMemory Memory

func LoadFromCartridge(cartridge cartridge.Cartridge) {
	PpuMemory.rom = cartridge
}

func PpuMemRead(addr uint16) uint8 {
	if addr <= 0x1FFF {
		return readChrRom(addr)
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		return PpuMemory.vram[mirrorVramAddress(addr)]
	} else if addr >= 0x3F00 {
		addr = (addr - 0x3F00) % 0x20
		return PpuMemory.paletteRam[addr]
	}
	return 0
}

func PpuMemWrite(addr uint16, val uint8) {
	if addr <= 0x1FFF {
		fmt.Println("PpuMemWrite: Cannot write to rom addr:", addr)
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		PpuMemory.vram[mirrorVramAddress(addr)] = val
	} else if addr >= 0x3F00 {
		addr = (addr - 0x3F00) % 0x20
		if addr%4 == 0 {
			PpuMemory.paletteRam[addr&0x0F] = val
			PpuMemory.paletteRam[addr|0x10] = val
		} else {
			PpuMemory.paletteRam[addr] = val
		}
	}
}

func PpuMemReadTile(addr uint16) [16]uint8 {
	var tile [16]uint8
	for i := range uint8(16) {
		tile[i] = PpuMemRead(addr + uint16(i))
	}
	return tile
}

func PpuOamWrite(addr uint8, val uint8) {
	PpuMemory.oam[addr] = val
}

func PpuOamRead(addr uint8) uint8 {
	return PpuMemory.oam[addr]
}

func readChrRom(addr uint16) uint8 {
	return PpuMemory.rom.ChrRom[addr]
}

func mirrorVramAddress(addr uint16) uint16 {
	// mirrors the unused memory to the vram
	if addr >= 0x3000 {
		addr -= 0x1000
	}
	// performs mirroing according to cartridge info
	switch PpuMemory.rom.MirroringType {

	case cartridge.HorizontalMirroring:
		// HORIZONTAL Mirroring
		//     [ A ] [ A ]
		//     [ B ] [ B ]
		if addr >= 0x2000 && addr <= 0x23FF {
			addr -= 0x2000
		} else if addr >= 0x2400 && addr <= 0x27FF {
			addr -= 0x2400
		} else if addr >= 0x2800 && addr <= 0x2BFF {
			addr -= 0x2400
		} else {
			addr -= 0x2800
		}

	case cartridge.VerticalMirroring:
		// Vertical Mirroring
		//     [ A ] [ B ]
		//     [ A ] [ B ]
		if addr >= 0x2000 && addr <= 0x23FF || addr >= 0x2400 && addr <= 0x27FF {
			addr -= 0x2000
		} else {
			addr -= 0x2800
		}

	case cartridge.FourScreenMirroring:
		// Four Screen Mirroring
		//     [ A ] [ B ]
		//     [ C ] [ D ]
		addr -= 0x2000
	}
	return addr
}

func GetNextNameTableAddress(baseNameTable uint16) uint16 {
	switch PpuMemory.rom.MirroringType {
	case cartridge.HorizontalMirroring:
		if baseNameTable == 0x2000 {
			return 0x2800
		} else if baseNameTable == 0x2800 {
			return 0x2400
		} else if baseNameTable == 0x2400 {
			return 0x2C00
		} else {
			return 0x2000
		}
	case cartridge.VerticalMirroring:
		if baseNameTable == 0x2000 {
			return 0x2400
		} else if baseNameTable == 0x2400 {
			return 0x2800
		} else if baseNameTable == 0x2800 {
			return 0x2C00
		} else {
			return 0x2000
		}
	}
	fmt.Println("Could not get next name table address. base name table address:", baseNameTable)
	return 0x2000
}

func HexDumpVram(filename string) {

	content := ""

	for i := range 0x0800 {
		content += fmt.Sprintf("%04X : %02X\n", i, PpuMemory.vram[uint16(i)])
	}
	content += "PALETTE RAM:\n"
	for i := range 0x0100 {
		content += fmt.Sprintf("%04X : %02X\n", i, PpuMemory.paletteRam[uint16(i)])
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
