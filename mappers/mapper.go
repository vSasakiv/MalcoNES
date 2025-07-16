package mappers

import (
	"vsasakiv/nesemulator/cartridge"
)

type Mapper interface {
	Read(address uint16) uint8
	Write(address uint16, val uint8)
	Clock(status Status)
	PollInterrupt() bool
	Mirroring() string
}

type Status struct {
	PpuScanlines         uint
	PpuCycles            uint
	PpuBackgroundEnabled bool
	PpuSpriteEnabled     bool
}

func NewMapper(cartridge *cartridge.Cartridge) Mapper {
	switch cartridge.MapperType {
	case 0:
		return NewMapper2(cartridge)
	case 1:
		return NewMapper1(cartridge)
	case 2:
		return NewMapper2(cartridge)
	case 4:
		return NewMapper4(cartridge)
	}
	return nil
}
