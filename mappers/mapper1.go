package mappers

import (
	"fmt"
	"vsasakiv/nesemulator/cartridge"
)

const MIRRORING = "MIRRORING"
const PRG_ROM_MODE = "PRG_ROM_MODE"
const CHR_ROM_MODE = "CHR_ROM_MODE"

type Mapper1 struct {
	cartridge     *cartridge.Cartridge
	loadRegister  uint8
	counter       uint
	control       uint8
	chrBank0      uint8
	chrBank1      uint8
	prgBank       uint8
	totalPrgBanks uint
}

func NewMapper1(cartridge *cartridge.Cartridge) Mapper {
	// each prg rom bank has 16kB
	mapper := Mapper1{}
	mapper.loadRegister = 0x10
	mapper.cartridge = cartridge
	mapper.totalPrgBanks = cartridge.PrgRomSize / 0x4000
	mapper.control = 0b01100
	return &mapper
}

func (mapper *Mapper1) Read(address uint16) uint8 {
	switch {

	case address <= 0x0FFF:
		// 8kb bank
		if mapper.getControlRegister(CHR_ROM_MODE) == 0 {
			bank := uint(mapper.chrBank0) & 0x0E
			address := (bank * 0x1000) + uint(address)
			if mapper.cartridge.ChrRamSize > 0 {
				return mapper.cartridge.ChrRam[address]
			}
			return mapper.cartridge.ChrRom[address]
		} else
		// 4kb bank each
		if mapper.getControlRegister(CHR_ROM_MODE) == 1 {
			bank := uint(mapper.chrBank0) & 0x0F
			address := (bank * 0x1000) + uint(address)
			if mapper.cartridge.ChrRamSize > 0 {
				return mapper.cartridge.ChrRam[address]
			}
			return mapper.cartridge.ChrRom[address]
		}

	case address >= 0x1000 && address <= 0x1FFF:
		// 8kb bank
		if mapper.getControlRegister(CHR_ROM_MODE) == 0 {
			bank := uint(mapper.chrBank0) & 0x0E
			address := (bank * 0x1000) + uint(address)
			if mapper.cartridge.ChrRamSize > 0 {
				return mapper.cartridge.ChrRam[address]
			}
			return mapper.cartridge.ChrRom[address]
		} else
		// 4kb bank each
		if mapper.getControlRegister(CHR_ROM_MODE) == 1 {
			bank := uint(mapper.chrBank1) & 0x0F
			address := (bank * 0x1000) + uint(address-0x1000)
			if mapper.cartridge.ChrRamSize > 0 {
				return mapper.cartridge.ChrRam[address]
			}
			return mapper.cartridge.ChrRom[address]
		}

	case address >= 0x6000 && address <= 0x7FFF:
		return mapper.cartridge.SRam[address-0x6000]

	case address >= 0x8000 && address <= 0xBFFF:
		// fixed first 16kb bank at 0x8000
		if mapper.getControlRegister(PRG_ROM_MODE) == 2 {
			return mapper.cartridge.PrgRom[address-0x8000]
		} else
		// switchable 16kb bank
		if mapper.getControlRegister(PRG_ROM_MODE) == 3 {
			bank := uint(mapper.prgBank) & 0x0F
			return mapper.cartridge.PrgRom[(bank*0x4000)+uint(address-0x8000)]
		} else
		// 32kb bank
		if mapper.getControlRegister(PRG_ROM_MODE) == 0 {
			bank := uint(mapper.prgBank) & 0x0E
			return mapper.cartridge.PrgRom[(bank*0x4000)+uint(address-0x8000)]
		}
	case address >= 0xC000:
		// switchable 16kb bank
		if mapper.getControlRegister(PRG_ROM_MODE) == 2 {
			bank := uint(mapper.prgBank) & 0x0F
			return mapper.cartridge.PrgRom[(bank*0x4000)+uint(address-0xC000)]
		} else
		// fixed last 16kb bank at 0xC000
		if mapper.getControlRegister(PRG_ROM_MODE) == 3 {
			return mapper.cartridge.PrgRom[((mapper.totalPrgBanks-1)*0x4000)+uint(address-0xC000)]
		} else
		// 32kb bank
		if mapper.getControlRegister(PRG_ROM_MODE) == 0 {
			bank := uint(mapper.prgBank) & 0x0E
			return mapper.cartridge.PrgRom[(bank*0x4000)+uint(address-0x8000)]
		}
	default:
		fmt.Println("flopou")
	}
	return 0
}

func (mapper *Mapper1) Write(address uint16, val uint8) {
	switch {
	case address <= 0x0FFF:
		// 8kb bank
		if mapper.getControlRegister(CHR_ROM_MODE) == 0 {
			bank := uint(mapper.chrBank0) & 0x0E
			address := (bank * 0x2000) + uint(address)
			if mapper.cartridge.ChrRamSize > 0 {
				mapper.cartridge.ChrRam[address] = val
			} else {
				fmt.Printf("Warning: cannot write to address %04x with mapper1 and no CHR-RAM\n", address)
			}
		} else
		// 4kb bank each
		if mapper.getControlRegister(CHR_ROM_MODE) == 1 {
			bank := uint(mapper.chrBank0) & 0x0F
			address := (bank * 0x1000) + uint(address)
			if mapper.cartridge.ChrRamSize > 0 {
				mapper.cartridge.ChrRam[address] = val
			} else {
				fmt.Printf("Warning: cannot write to address %04x with mapper1 and no CHR-RAM\n", address)
			}
		}

	case address >= 0x1000 && address <= 0x1FFF:
		// 8kb bank
		if mapper.getControlRegister(CHR_ROM_MODE) == 0 {
			bank := uint(mapper.chrBank0) & 0x0E
			address := (bank * 0x2000) + uint(address)
			if mapper.cartridge.ChrRamSize > 0 {
				mapper.cartridge.ChrRam[address] = val
			} else {
				fmt.Printf("Warning: cannot write to address %04x with mapper1 and no CHR-RAM\n", address)
			}
		} else
		// 4kb bank each
		if mapper.getControlRegister(CHR_ROM_MODE) == 1 {
			bank := uint(mapper.chrBank1) & 0x0F
			address := (bank * 0x1000) + uint(address-0x1000)
			if mapper.cartridge.ChrRamSize > 0 {
				mapper.cartridge.ChrRam[address] = val
			} else {
				fmt.Printf("Warning: cannot write to address %04x with mapper1 and no CHR-RAM\n", address)
			}
		}

	case address >= 0x6000 && address <= 0x7FFF:
		mapper.cartridge.SRam[address-0x6000] = val

	case address >= 0x8000:
		mapper.writeToLoadRegister(address, val)
	default:
		fmt.Printf("Warning: cannot write to address %04x with mapper1\n", address)
	}
}

func (mapper *Mapper1) Mirroring() string {
	switch mapper.getControlRegister(MIRRORING) {
	case 0b00:
		return cartridge.MirroringSingle0
	case 0b01:
		return cartridge.MirroringSingle1
	case 0b10:
		return cartridge.VerticalMirroring
	case 0b11:
		return cartridge.HorizontalMirroring
	default:
		fmt.Println("Mapper1 mirroring does not exist!")
		return ""
	}
}

func (mapper *Mapper1) writeToLoadRegister(address uint16, val uint8) {
	// has msb as 1
	if val&0x80 == 0x80 {
		mapper.loadRegister = 0
		mapper.counter = 0
		mapper.control |= 0xC0
	} else {
		mapper.loadRegister = uint8(SetBitToVal(mapper.loadRegister, mapper.counter, val&0b1))
		mapper.counter += 1
		// write 5 bits, on fifth bit write to internal registers
		if mapper.counter == 5 {
			mapper.writeLoadToInternal(address)
			mapper.counter = 0
			mapper.loadRegister = 0
		}
	}
}

func (mapper *Mapper1) writeLoadToInternal(address uint16) {
	switch {
	case address >= 0x8000 && address <= 0x9FFF:
		mapper.control = mapper.loadRegister
	case address >= 0xA000 && address <= 0xBFFF:
		mapper.chrBank0 = mapper.loadRegister
	case address >= 0xC000 && address <= 0xDFFF:
		mapper.chrBank1 = mapper.loadRegister
	case address >= 0xE000:
		mapper.prgBank = mapper.loadRegister
	}
}

func (mapper *Mapper1) getControlRegister(param string) uint8 {
	switch param {
	case MIRRORING:
		return mapper.control & 0b11
	case PRG_ROM_MODE:
		return (mapper.control >> 2) & 0b11
	case CHR_ROM_MODE:
		return (mapper.control >> 4) & 0b1
	default:
		fmt.Println("Mapper1 Control Register param not valid: ", param)
		return 0
	}
}

func SetBitToVal(n uint8, pos uint, val uint8) uint8 {
	if val == 1 {
		return SetBitToOne(n, pos)
	} else {
		return SetBitToZero(n, pos)
	}
}

func SetBitToOne(n uint8, pos uint) uint8 {
	return n | (1 << pos)
}

func SetBitToZero(n uint8, pos uint) uint8 {
	return n & ^(1 << pos)
}
