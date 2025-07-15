package mappers

import (
	"fmt"
	"vsasakiv/nesemulator/cartridge"
)

type Mapper2 struct {
	cartridge  *cartridge.Cartridge
	totalBanks int
	bankSelect int
	fixedBank  int
}

func NewMapper2(cartridge *cartridge.Cartridge) Mapper {
	// each prg rom bank has 16kB
	totalBanks := int(cartridge.PrgRomSize) / 0x4000
	return &Mapper2{
		cartridge:  cartridge,
		totalBanks: totalBanks,
		bankSelect: 0,
		fixedBank:  totalBanks - 1,
	}
}

func (mapper *Mapper2) Read(address uint16) uint8 {
	switch {
	// chr rom unchanged in this mapper
	case address <= 0x1FFF:
		// is chr ram
		if mapper.cartridge.ChrRamSize > 0 {
			return mapper.cartridge.ChrRam[address]
		}
		return mapper.cartridge.ChrRom[address]
	case address >= 0x6000 && address <= 0x7FFF:
		return mapper.cartridge.SRam[address-0x6000]
	// prg rom switched bank
	case address >= 0x8000 && address <= 0xBFFF:
		return mapper.cartridge.PrgRom[(0x4000*mapper.bankSelect)+int(address-0x8000)]
	// fixed prg rom bank
	case address >= 0xC000:
		return mapper.cartridge.PrgRom[(0x4000*mapper.fixedBank)+int(address-0xC000)]
	default:
		fmt.Printf("Warning: cannot access address %04x with mapper2\n", address)
	}
	return 0
}

func (mapper *Mapper2) Write(address uint16, val uint8) {
	switch {
	// chr rom unchanged in this mapper
	case address <= 0x1FFF:
		// is chr ram
		if mapper.cartridge.ChrRamSize > 0 {
			mapper.cartridge.ChrRam[address] = val
			return
		}
		fmt.Printf("Warning: cannot write to ROM address %04x with mapper2\n", address)
	case address >= 0x6000 && address <= 0x7FFF:
		mapper.cartridge.SRam[address-0x6000] = val
	// write to bank select register
	case address >= 0x8000:
		mapper.bankSelect = int(val) % mapper.totalBanks
	default:
		fmt.Printf("Warning: cannot write to address %04x with mapper2\n", address)
	}
}

func (mapper *Mapper2) Mirroring() string {
	return mapper.cartridge.MirroringType
}

func (mapper *Mapper2) Step(status Status)  {}
func (mapper *Mapper2) PollInterrupt() bool { return false }
