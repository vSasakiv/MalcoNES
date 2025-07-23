package ppu

import (
	"fmt"
	"os"
	"vsasakiv/nesemulator/cartridge"
	"vsasakiv/nesemulator/mappers"
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
	mapper     mappers.Mapper
	paletteRam [0x0100]uint8
	oam        [0x0100]uint8
}

var PpuMemory Memory

func (ppuMemory Memory) Reset() {
	PpuMemory.vram = [0x0800]uint8{}
	PpuMemory.paletteRam = [0x0100]uint8{}
	PpuMemory.oam = [0x0100]uint8{}
}

func LoadCartridge(mapper mappers.Mapper) {
	PpuMemory.mapper = mapper
}

func PpuMemRead(addr uint16) uint8 {
	addr = addr % 0x4000
	if addr <= 0x1FFF {
		return PpuMemory.mapper.Read(addr)
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		return PpuMemory.vram[mirrorVramAddress(addr)]
	} else if addr >= 0x3F00 {
		addr = (addr - 0x3F00) % 0x20
		return PpuMemory.paletteRam[addr]
	}
	return 0
}

func PpuMemWrite(addr uint16, val uint8) {
	addr = addr % 0x4000
	if addr <= 0x1FFF {
		PpuMemory.mapper.Write(addr, val)
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

func PpuMemReadTileLine(addr uint16, fineY uint16) [2]uint8 {
	var tile [2]uint8
	tile[0] = PpuMemRead(addr + fineY)
	tile[1] = PpuMemRead(addr + fineY + 8)
	return tile
}

func PpuMemReadTile(addr uint16) []uint8 {
	var tile [16]uint8
	for i := range uint8(16) {
		tile[i] = PpuMemRead(addr + uint16(i))
	}
	return tile[:]
}

func PpuMemReadBigTile(addr uint16) []uint8 {
	var tile [32]uint8
	for i := range uint8(32) {
		tile[i] = PpuMemRead(addr + uint16(i))
	}
	return tile[:]
}

func PpuOamWrite(addr uint8, val uint8) {
	PpuMemory.oam[addr] = val
}

func PpuOamRead(addr uint8) uint8 {
	return PpuMemory.oam[addr]
}

func mirrorVramAddress(addr uint16) uint16 {
	// mirrors the unused memory to the vram
	if addr >= 0x3000 {
		addr -= 0x1000
	}
	// performs mirroing according to cartridge info
	switch PpuMemory.mapper.Mirroring() {

	case cartridge.MirroringSingle0:
		// Single 0 Mirroring
		//     [ A ] [ A ]
		//     [ A ] [ A ]
		addr = 0x2000 + addr%0x0400

	case cartridge.MirroringSingle1:
		// Single 1 Mirroring
		//     [ B ] [ B ]
		//     [ B ] [ B ]
		addr = 0x2400 + addr%0x0400

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
