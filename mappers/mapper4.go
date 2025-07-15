package mappers

import (
	"fmt"
	"vsasakiv/nesemulator/cartridge"
)

type Mapper4 struct {
	cartridge  *cartridge.Cartridge
	totalBanks int
	bankSelect uint8
	// bank data registers
	chrR0 uint8
	chrR1 uint8
	chrR2 uint8
	chrR3 uint8
	chrR4 uint8
	chrR5 uint8
	prgR6 uint8
	prgR7 uint8
	// mirroring
	mirroring uint8
	// IRQ interrupt
	irqEnabled   bool
	irqLatch     uint8
	reload       uint8
	irqInterrupt bool
}

func NewMapper4(cartridge *cartridge.Cartridge) Mapper {
	// each prg rom bank has 8kB
	totalBanks := int(cartridge.PrgRomSize) / 0x2000
	return &Mapper4{
		cartridge:  cartridge,
		totalBanks: totalBanks,
		irqEnabled: false,
	}
}

func (mapper *Mapper4) Read(address uint16) uint8 {
	switch {
	case address <= 0x1FFF:
		address := mapper.getChrAddress(address)
		if mapper.cartridge.ChrRamSize > 0 {
			return mapper.cartridge.ChrRam[address]
		}
		return mapper.cartridge.ChrRom[address]

	case address >= 0x6000 && address <= 0x7FFF:
		return mapper.cartridge.SRam[address-0x6000]
	case address >= 0x8000:
		address := mapper.getPrgAddress(address)
		return mapper.cartridge.PrgRom[address]
	default:
		fmt.Printf("Warning: cannot access address %04x with mapper3\n", address)
	}
	return 0
}

func (mapper *Mapper4) Write(address uint16, val uint8) {
	switch {
	case address <= 0x1FFF:
		address := mapper.getChrAddress(address)
		if mapper.cartridge.ChrRamSize > 0 {
			mapper.cartridge.ChrRam[address] = val
		}
	case address >= 0x6000 && address <= 0x7FFF:
		mapper.cartridge.SRam[address-0x6000] = val
		// write to bank select register
	case address >= 0x8000 && address <= 0x9FFF:
		mapper.writeToMemoryMappingRegisters(address, val)
	case address >= 0xA000 && address <= 0xBFFF:
		// if address is even, save mirroring
		if address%2 == 0 {
			mapper.mirroring = val
		}
	case address >= 0xC000 && address <= 0xDFFF:
		if address%2 == 0 {
			mapper.reload = val
		} else {
			mapper.irqLatch = 0
		}
	case address >= 0xE000:
		if address%2 == 0 {
			mapper.irqEnabled = false
		} else {
			mapper.irqEnabled = true
		}

	default:
		fmt.Printf("Warning: cannot write to address %04x with mapper3\n", address)
	}
}

func (mapper *Mapper4) writeToMemoryMappingRegisters(address uint16, val uint8) {
	// if address is even
	if address%2 == 0 {
		mapper.bankSelect = val
	} else {
		// write to bank data registers
		switch mapper.bankSelect & 0b111 {
		// R0 and R1 index larger 2kB banks, so the last bit is ignored
		case 0:
			mapper.chrR0 = val & 0xFE
		case 1:
			mapper.chrR1 = val & 0xFE
		case 2:
			mapper.chrR2 = val
		case 3:
			mapper.chrR3 = val
		case 4:
			mapper.chrR4 = val
		case 5:
			mapper.chrR5 = val
		case 6:
			mapper.prgR6 = val & 0x3F
		case 7:
			mapper.prgR7 = val & 0x3F
		default:
			fmt.Printf("Mapper4 Cannot write to register %03b\n", address&0b111)
		}
	}
}

func (mapper *Mapper4) getChrAddress(address uint16) uint {
	switch {
	case address <= 0x03FF:
		// 2Kb chr rom
		if mapper.getChrInversion() == 0 {
			return (uint(mapper.chrR0) * 0x0400) + uint(address)
		} else {
			return (uint(mapper.chrR2) * 0x0400) + uint(address)
		}

	case address <= 0x07FF:
		// 2Kb chr rom
		if mapper.getChrInversion() == 0 {
			return (uint(mapper.chrR0) * 0x0400) + uint(address)
		} else {
			return (uint(mapper.chrR3) * 0x0400) + (uint(address) - 0x0400)
		}

	case address <= 0x0BFF:
		// 2Kb chr rom
		if mapper.getChrInversion() == 0 {
			return (uint(mapper.chrR1) * 0x0400) + (uint(address) - 0x0800)
		} else {
			return (uint(mapper.chrR4) * 0x0400) + (uint(address) - 0x0800)
		}

	case address <= 0x0FFF:
		// 2Kb chr rom
		if mapper.getChrInversion() == 0 {
			return (uint(mapper.chrR1) * 0x0400) + (uint(address) - 0x0800)
		} else {
			return (uint(mapper.chrR5) * 0x0400) + (uint(address) - 0x0C00)
		}

	case address <= 0x13FF:
		if mapper.getChrInversion() == 0 {
			return (uint(mapper.chrR2) * 0x0400) + (uint(address) - 0x1000)
		} else {
			// 2Kb chr rom
			return (uint(mapper.chrR0) * 0x0400) + (uint(address) - 0x1000)
		}

	case address <= 0x17FF:
		if mapper.getChrInversion() == 0 {
			return (uint(mapper.chrR3) * 0x0400) + (uint(address) - 0x1400)
		} else {
			// 2Kb chr rom
			return (uint(mapper.chrR0) * 0x0400) + (uint(address) - 0x1000)
		}

	case address <= 0x1BFF:
		if mapper.getChrInversion() == 0 {
			return (uint(mapper.chrR4) * 0x0400) + (uint(address) - 0x1800)
		} else {
			// 2Kb chr rom
			return (uint(mapper.chrR1) * 0x0400) + (uint(address) - 0x1800)
		}

	case address <= 0x1FFF:
		if mapper.getChrInversion() == 0 {
			return (uint(mapper.chrR5) * 0x0400) + (uint(address) - 0x1C00)
		} else {
			// 2Kb chr rom
			return (uint(mapper.chrR1) * 0x0400) + (uint(address) - 0x1800)
		}
	default:
		fmt.Println("Mapper4 cannot get address of chr rom: %04X", address)
		return 0
	}
}

func (mapper *Mapper4) getPrgAddress(address uint16) uint {
	switch {
	case address <= 0x9FFF:
		// use register
		if mapper.getRomBankMode() == 0 {
			return (uint(mapper.prgR6) * 0x2000) + (uint(address) - 0x8000)
		} else {
			// return second to last bank
			return (uint(mapper.totalBanks-2) * 0x2000) + (uint(address) - 0x8000)
		}

	case address <= 0xBFFF:
		// use register r7
		return (uint(mapper.prgR7) * 0x2000) + (uint(address) - 0xA000)

	case address <= 0xDFFF:
		// return second to last bank
		if mapper.getRomBankMode() == 0 {
			return (uint(mapper.totalBanks-2) * 0x2000) + (uint(address) - 0xC000)
		} else {
			// return second to last bank
			return (uint(mapper.prgR6) * 0x2000) + (uint(address) - 0xC000)
		}

	case address >= 0xE000:
		// fixed on last bank
		return (uint(mapper.totalBanks-1) * 0x2000) + (uint(address) - 0xE000)

	default:
		fmt.Println("Mapper4 cannot get address of prg rom: %04X", address)
		return 0
	}
}

func (mapper *Mapper4) getRomBankMode() uint8 {
	return (mapper.bankSelect >> 6) & 0b1
}

func (mapper *Mapper4) getChrInversion() uint8 {
	return (mapper.bankSelect >> 7) & 0b1
}

func (mapper *Mapper4) Mirroring() string {
	// if selected on header, ignore register
	if mapper.cartridge.MirroringType == cartridge.FourScreenMirroring {
		return mapper.cartridge.MirroringType
	} else {
		switch mapper.mirroring & 0b1 {
		case 0:
			return cartridge.VerticalMirroring
		case 1:
			return cartridge.HorizontalMirroring
		}
	}
	return mapper.cartridge.MirroringType
}

func (mapper *Mapper4) Step(status Status) {
	// tick counter only on cycle 260
	if status.PpuCycles != 260 {
		return
	}
	// only on visible scanlines
	if status.PpuScanlines >= 240 && status.PpuScanlines <= 260 {
		return
	}
	// only when rendering is enabled
	if !status.PpuBackgroundEnabled && !status.PpuSpriteEnabled {
		return
	}

	if mapper.irqLatch == 0 {
		mapper.irqLatch = mapper.reload
	} else {
		mapper.irqLatch -= 1
		if mapper.irqLatch == 0 && mapper.irqEnabled {
			mapper.irqInterrupt = true
		}
	}
}

func (mapper *Mapper4) PollInterrupt() bool {
	if mapper.irqInterrupt {
		mapper.irqInterrupt = false
		return true
	}
	return false
}
