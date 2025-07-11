package ppu

import "fmt"

// ppu control settings
const NAMETABLE_ADDRESS = "NAMETABLE_ADDRESS"
const INCREMENT = "INCREMENT"
const SPRITE_TABLE_ADDRESS = "SPRITE_TABLE_ADDRESS"
const BACKGROUND_TABLE_ADDRESS = "BACKGROUND_TABLE_ADDRESS"
const SPRITE_SIZE = "SPRITE_SIZE"
const VBLANK_NMI_ENABLE = "VBLANK_NMI_ENABLE"

// ppu mask settings
const GREYSCALE = "GREYSCALE"
const SHOW_BACKGROUND_LEFT = "SHOW_BACKGROUND_LEFT"
const SHOW_SPRITES_LEFT = "SHOW_SPRITES_LEFT"
const ENABLE_BACKGROUND = "ENABLE_BACKGROUND"
const ENABLE_SPRITE = "ENABLE_SPRITE"
const RED = "RED"
const GREEN = "GREEN"
const BLUE = "BLUE"

type Ppu struct {
	// internal registers
	currentAddress uint16
	tempAddress    uint16
	fineXScroll    uint8
	writeToggle    bool
	// register mapped to cpu memory
	ppuAddr    uint16
	ppuCtrl    uint8
	ppuMask    uint8
	ppuStatus  uint8
	ppuScrollX uint8
	ppuScrollY uint8
	ppuOamAddr uint8
	ppuOamData uint8
	// internal buffers
	readBuffer uint8
	// simulation clock cycles and scanlines
	cycles    uint
	scanlines uint
	// nmi interrupt signal
	NmiInterrupt bool
	// color palette
	systemPalette [64][3]uint8
	// frame control
	vramAddress        uint16
	CurrentFrame       Frame
	CurrentPixelBuffer [XSIZE * YSIZE * 3]uint8
	// sprite zero collision state
	hasSpriteZeroCollision            bool
	collisionCycle, collisionScanline uint
}

// Initialize ppu with corret parameters, also initialize system palette
func NewPpu() *Ppu {
	var ppu Ppu
	ppu.hasSpriteZeroCollision = false
	ppu.systemPalette = GenerateFromPalFile("./ppu/palettes/2C02.pal")
	return &ppu
}

func GetPpu() *Ppu {
	return &ppu
}

var ppu Ppu = *NewPpu()

func GetPixelBuffer() [XSIZE * YSIZE * 3]uint8 {
	return ppu.CurrentPixelBuffer
}

func Execute(cycles uint) {

	for range cycles {
		ppu.cycles += 1
		ppu.setSpriteZeroHit()
		if ppu.cycles >= 341 {
			ppu.cycles = 0
			ppu.scanlines += 1
			// when activating vblank, disable spriteZeroCollision
			if ppu.scanlines == 241 {
				ppu.clearSpriteZeroHit()
				ppu.hasSpriteZeroCollision = false
				ppu.setVblankStatus(1)

				if ppu.getControlSetting(VBLANK_NMI_ENABLE) == 1 {
					ppu.NmiInterrupt = true
				}
			}
			if ppu.scanlines >= 262 {
				ppu.scanlines = 0

				ppu.clearSpriteZeroHit()

				ppu.setVblankStatus(0)
				ppu.NmiInterrupt = false
				backgroundRomBank := ppu.getControlSetting(BACKGROUND_TABLE_ADDRESS)
				oamRomBank := ppu.getControlSetting(SPRITE_TABLE_ADDRESS)

				greyscale := ppu.getMaskSetting(GREYSCALE)

				if ppu.getMaskSetting(ENABLE_BACKGROUND) == 1 {
					nameTableControl := ppu.getControlSetting(NAMETABLE_ADDRESS)
					var baseNameTable uint16
					switch nameTableControl {
					case 0b00:
						baseNameTable = 0x2000
					case 0b01:
						baseNameTable = 0x2400
					case 0b10:
						baseNameTable = 0x2800
					case 0b11:
						baseNameTable = 0x2C00
					}
					ppu.CurrentFrame.RenderBackground(baseNameTable, uint(backgroundRomBank), ppu.ppuScrollX, ppu.ppuScrollY, greyscale)
					// ppu.CurrentFrame.RenderNameTable(0x2000, uint(backgroundRomBank), greyscale)
				} else {
					ppu.CurrentFrame.RenderEmptyBackground()
				}

				// renders sprites and detects sprite zero collision
				if ppu.getMaskSetting(ENABLE_SPRITE) == 1 {
					ppu.CurrentFrame.RenderOam(uint(oamRomBank), greyscale)
					ppu.hasSpriteZeroCollision, ppu.collisionCycle, ppu.collisionScanline = ppu.CurrentFrame.spriteZeroCollision(uint(oamRomBank))
				}
			}
		}
	}
}

func (ppu *Ppu) PollForNmiInterrupt() bool {
	if ppu.NmiInterrupt {
		ppu.NmiInterrupt = false
		return true
	}
	return false
}

// ----- PPUCTRL 0x2000 REGISTER -----

func (ppu *Ppu) WriteToPpuControl(val uint8) {
	if (val>>7)&0b1 == 1 && (ppu.ppuCtrl>>7)&0b1 == 0 && (ppu.ppuStatus>>7)&0b1 == 1 {
		ppu.NmiInterrupt = true
	}
	ppu.ppuCtrl = val
}

func (ppu *Ppu) getControlSetting(setting string) uint8 {
	switch setting {
	case NAMETABLE_ADDRESS:
		return ppu.ppuCtrl & 0b11
	case INCREMENT:
		return (ppu.ppuCtrl >> 2) & 0b1
	case SPRITE_TABLE_ADDRESS:
		return (ppu.ppuCtrl >> 3) & 0b1
	case BACKGROUND_TABLE_ADDRESS:
		return (ppu.ppuCtrl >> 4) & 0b1
	case SPRITE_SIZE:
		return (ppu.ppuCtrl >> 5) & 0b1
	case VBLANK_NMI_ENABLE:
		return (ppu.ppuCtrl >> 7) & 0b1
	}
	return 0
}

// ----- PPUMASK 0x2001 REGISTER -----

func (ppu *Ppu) WriteToPpuMask(val uint8) {
	ppu.ppuMask = val
}

func (ppu *Ppu) getMaskSetting(setting string) uint8 {
	switch setting {
	case GREYSCALE:
		return ppu.ppuMask & 0b1
	case SHOW_BACKGROUND_LEFT:
		return (ppu.ppuMask >> 1) & 0b1
	case SHOW_SPRITES_LEFT:
		return (ppu.ppuMask >> 2) & 0b1
	case ENABLE_BACKGROUND:
		return (ppu.ppuMask >> 3) & 0b1
	case ENABLE_SPRITE:
		return (ppu.ppuMask >> 4) & 0b1
	case RED:
		return (ppu.ppuMask >> 5) & 0b1
	case GREEN:
		return (ppu.ppuMask >> 6) & 0b1
	case BLUE:
		return (ppu.ppuMask >> 7) & 0b1
	}
	return 0
}

// ----- PPUSTATUS 0x2002 REGISTER -----

func (ppu *Ppu) ReadPpuStatusRegister() uint8 {
	status := ppu.ppuStatus
	ppu.writeToggle = false
	ppu.setVblankStatus(0)
	return status
}

func (ppu *Ppu) setVblankStatus(val uint8) {
	if val == 1 {
		ppu.ppuStatus |= 1 << 7
	} else {
		ppu.ppuStatus &^= 1 << 7
	}
}

func (ppu *Ppu) setSpriteZeroHit() {
	// if ppu.hasSpriteZeroCollision {
	// 	if ppu.cycles == ppu.collisionCycle && ppu.collisionScanline == ppu.scanlines {
	// 		ppu.ppuStatus |= 1 << 6
	// 	}
	// }
	ppu.ppuStatus |= 1 << 6
}

func (ppu *Ppu) clearSpriteZeroHit() {
	ppu.ppuStatus &^= 1 << 6
}

// ----- OAMADDR 0x2003 REGISTER -----

func (ppu *Ppu) WriteToOamAddrRegister(val uint8) {
	ppu.ppuOamAddr = val
}

// ----- OAMDATA 0x2004 REGISTER -----

func (ppu *Ppu) WriteToOamDataRegister(val uint8) {
	PpuOamWrite(ppu.ppuOamAddr, val)
	ppu.ppuOamAddr += 1
}

func (ppu *Ppu) ReadOamDataRegister() uint8 {
	return PpuOamRead(ppu.ppuOamAddr)
}

// ----- PPUSCROLL 0x2005 REGISTER -----

func (ppu *Ppu) WriteToPpuScroll(val uint8) {
	if !ppu.writeToggle {
		ppu.ppuScrollX = val
	} else {
		ppu.ppuScrollY = val
	}
	ppu.writeToggle = !ppu.writeToggle
}

// ----- PPUADDR 0x2006 REGISTER -----

func (ppu *Ppu) WriteToAddrRegister(val uint8) {
	// write to high/low byte
	if ppu.writeToggle {
		ppu.ppuAddr = uint16(val)&0x00FF | ppu.ppuAddr&0xFF00
	} else {
		ppu.ppuAddr = (uint16(val)<<8)&0xFF00 | ppu.ppuAddr&0x00FF
	}
	// loops value back arround to first address
	if ppu.ppuAddr > 0x3FFF {
		ppu.ppuAddr = ppu.ppuAddr & 0b1111111_1111111
	}
	ppu.writeToggle = !ppu.writeToggle
}

func (ppu *Ppu) incrementAddrRegister() {
	var increment uint16
	if ppu.getControlSetting(INCREMENT) == 0 {
		increment = 1
	} else {
		increment = 32
	}
	ppu.ppuAddr += increment
	// loops value back arround to first address
	if ppu.ppuAddr > 0x3FFF {
		ppu.ppuAddr = ppu.ppuAddr & 0b1111111_1111111
	}
}

// ----- PPUDATA 0x2007 REGISTER -----

func (ppu *Ppu) ReadPpuDataRegister() uint8 {
	// if it is pallete ram, return the value instantly
	if ppu.ppuAddr >= 0x3F00 {
		result := PpuMemRead(ppu.ppuAddr)
		ppu.incrementAddrRegister()
		return result
	}
	result := ppu.readBuffer
	ppu.readBuffer = PpuMemRead(ppu.ppuAddr)
	// increment address after everything
	ppu.incrementAddrRegister()
	return result
}

func (ppu *Ppu) WriteToPpuDataRegister(val uint8) {
	// increment address after everything
	PpuMemWrite(ppu.ppuAddr, val)
	ppu.incrementAddrRegister()
}

// ----- DEBUG -----

func (ppu *Ppu) TracePpuStatus() string {
	return fmt.Sprintf("PPU:%03d, %03d  ADDR: %04X CTRL:%08b STATUS: %08b", ppu.scanlines, ppu.cycles, ppu.ppuAddr, ppu.ppuCtrl, ppu.ppuStatus)
}
