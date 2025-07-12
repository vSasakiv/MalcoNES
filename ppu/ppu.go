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

// palette addresses
const DEFAULT_BG_PALETTE_ADDRESS = 0x3F00
const DEFAULT_OAM_PALETTE_ADDRESS = 0x3F10

type Ppu struct {
	// internal registers
	currentAddress uint16
	tempAddress    uint16
	fineXScroll    uint8
	writeToggle    bool
	// register mapped to cpu memory
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

	// ppu internal registers
	loopyV            uint16
	loopyT            uint16
	fineX             uint8
	write             uint8
	outputPixelBuffer [][3]uint8
}

// Initialize ppu with corret parameters, also initialize system palette
func NewPpu() *Ppu {
	var ppu Ppu
	ppu.hasSpriteZeroCollision = false
	ppu.outputPixelBuffer = make([][3]uint8, 16)
	ppu.cycles = 340
	ppu.scanlines = 240
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

func ExecuteLoopy(cycles uint) {

	for range cycles {
		ppu.runCycle()

		visibleScanlines := ppu.scanlines <= 239
		visibleCycles := ppu.cycles >= 1 && ppu.cycles <= 256
		preRenderScanline := ppu.scanlines == 261
		preRenderCopyY := ppu.cycles >= 280 && ppu.cycles <= 304
		vblank := ppu.scanlines == 241 && ppu.cycles == 1
		vblankEnd := preRenderScanline && ppu.cycles == 1

		// rendering visible scanlines
		if ppu.getMaskSetting(ENABLE_BACKGROUND) == 1 {
			if visibleScanlines {
				if visibleCycles {
					// draw pixel from output buffer
					ppu.CurrentFrame.setPixel(ppu.cycles-1, ppu.scanlines, ppu.outputPixelBuffer[ppu.fineX])
					// shift pixel buffer
					ppu.outputPixelBuffer = ShiftBufferLeft(ppu.outputPixelBuffer)
					// if cycle is 8, 16, 24 ... reload shift register and update loopyV
					if ppu.cycles%8 == 0 {
						ppu.reloadPixelBuffer()
						ppu.incrementLoopyVX()
					}
				}
				// increment y -> go down a pixel
				if ppu.cycles == 256 {
					ppu.incrementLoopyVY()
				}
				// get scrollx back to start
				if ppu.cycles == 257 {
					ppu.copyScrollxToLoopyV()
				}
				// load first tile
				if ppu.cycles == 327 {
					ppu.reloadPixelBuffer()
					ppu.incrementLoopyVX()
				}
				// load second tile
				if ppu.cycles == 335 {
					for range 8 {
						ppu.outputPixelBuffer = ShiftBufferLeft(ppu.outputPixelBuffer)
					}
					ppu.reloadPixelBuffer()
					ppu.incrementLoopyVX()
				}
			}
			if preRenderScanline && preRenderCopyY {
				ppu.copyScrollyToLoopyV()
			}
		}

		if vblank {
			if ppu.getControlSetting(VBLANK_NMI_ENABLE) == 1 {
				ppu.NmiInterrupt = true
			}
			ppu.setVblankStatus(1)
		}
		if vblankEnd {
			ppu.NmiInterrupt = false
			ppu.setVblankStatus(0)
		}

		// if visibleScanlines {
		// 	// idle cycle
		// 	if ppu.cycles == 0 {
		// 		ppu.cycles += 1
		// 	} else
		// 	// render scanline 1, 2, 3, 4
		// 	if visibleCycles {
		// 		// draw pixel from output buffer
		// 		ppu.CurrentFrame.setPixel(ppu.cycles-1, ppu.scanlines, ppu.outputPixelBuffer[ppu.fineX])
		// 		// shift pixel buffer
		// 		ppu.outputPixelBuffer = ShiftBufferLeft(ppu.outputPixelBuffer)
		// 		// if cycle is 8, 16, 24 ... reload shift register and update loopyV
		// 		if ppu.cycles%8 == 0 {
		// 			ppu.reloadPixelBuffer()
		// 			ppu.incrementLoopyVX()
		// 		}
		// 		ppu.cycles += 1
		// 	} else if ppu.cycles >= 257 && ppu.cycles <= 340 {
		//
		// 		if ppu.cycles == 257 {
		// 			ppu.incrementLoopyVY()
		// 		}
		// 		if ppu.cycles == 258 {
		// 			ppu.copyScrollxToLoopyV()
		// 		}
		// 		if ppu.cycles == 327 {
		// 			ppu.reloadPixelBuffer()
		// 			ppu.incrementLoopyVX()
		// 		}
		// 		if ppu.cycles == 335 {
		// 			for range 8 {
		// 				ppu.outputPixelBuffer = ShiftBufferLeft(ppu.outputPixelBuffer)
		// 			}
		// 			ppu.reloadPixelBuffer()
		// 			ppu.incrementLoopyVX()
		// 		}
		// 		ppu.cycles += 1
		// 	} else
		// 	// incremenet scanline
		// 	if ppu.cycles >= 341 {
		// 		ppu.cycles = 0
		// 		ppu.scanlines += 1
		// 	}
		// } else
		// // idle scanline
		// if ppu.scanlines == 240 {
		// 	if ppu.cycles >= 341 {
		// 		ppu.cycles = 0
		// 		ppu.scanlines += 1
		// 	} else {
		// 		ppu.cycles += 1
		// 	}
		// } else
		// // vblank
		// if ppu.scanlines >= 241 && ppu.scanlines <= 260 {
		// 	// vblank enable at tick 2 of scanline 241
		// 	if ppu.scanlines == 241 && ppu.cycles == 2 {
		// 		if ppu.getControlSetting(VBLANK_NMI_ENABLE) == 1 {
		// 			ppu.NmiInterrupt = true
		// 		}
		// 		ppu.setVblankStatus(1)
		// 		ppu.cycles += 1
		// 	} else if ppu.cycles >= 341 {
		// 		ppu.cycles = 0
		// 		ppu.scanlines += 1
		// 	} else {
		// 		ppu.cycles += 1
		// 	}
		// } else
		// // pre render
		// if ppu.scanlines == 261 {
		// 	if ppu.cycles == 0 {
		// 		ppu.setVblankStatus(0)
		// 		ppu.NmiInterrupt = false
		// 		ppu.cycles += 1
		// 	} else
		// 	// TODO
		// 	if ppu.cycles == 258 {
		// 		ppu.copyScrollxToLoopyV()
		// 		ppu.cycles += 1
		// 	} else if ppu.cycles >= 281 && ppu.cycles <= 305 {
		// 		ppu.copyScrollyToLoopyV()
		// 		ppu.cycles += 1
		// 	} else if ppu.cycles == 327 {
		// 		ppu.reloadPixelBuffer()
		// 		ppu.incrementLoopyVX()
		// 		ppu.cycles += 1
		// 	} else if ppu.cycles == 335 {
		// 		for range 8 {
		// 			ppu.outputPixelBuffer = ShiftBufferLeft(ppu.outputPixelBuffer)
		// 		}
		// 		ppu.reloadPixelBuffer()
		// 		ppu.incrementLoopyVX()
		// 		ppu.cycles += 1
		// 	} else if ppu.cycles >= 341 {
		// 		ppu.CurrentFrame.RenderOam(uint(ppu.getControlSetting(SPRITE_TABLE_ADDRESS)), ppu.getMaskSetting(GREYSCALE))
		// 		ppu.cycles = 0
		// 		ppu.scanlines = 0
		// 	} else {
		// 		ppu.cycles += 1
		// 	}
		// }
	}
}

func (ppu *Ppu) runCycle() {
	ppu.cycles += 1
	if ppu.cycles == 341 {
		ppu.cycles = 0
		ppu.scanlines += 1
		if ppu.scanlines == 262 {
			ppu.scanlines = 0
		}
	}
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
	// t: ...GH.. ........ <- d: ......GH
	ppu.loopyT = SetBitToVal(ppu.loopyT, 10, val&0b1)
	ppu.loopyT = SetBitToVal(ppu.loopyT, 11, (val>>1)&0b1)
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
	ppu.write = 0
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
	if ppu.write == 0 {
		// sets internal registers
		ppu.loopyT = SetBitToVal(ppu.loopyT, 0, (val>>3)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 1, (val>>4)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 2, (val>>5)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 3, (val>>6)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 4, (val>>7)&0b1)
		ppu.fineX = val & 0b111

		ppu.ppuScrollX = val
		ppu.write = 1
	} else {
		ppu.loopyT = SetBitToVal(ppu.loopyT, 5, (val>>3)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 6, (val>>4)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 7, (val>>5)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 8, (val>>6)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 9, (val>>7)&0b1)

		ppu.loopyT = SetBitToVal(ppu.loopyT, 12, (val)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 13, (val>>1)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 14, (val>>2)&0b1)

		ppu.ppuScrollY = val
		ppu.write = 0
	}
	ppu.writeToggle = !ppu.writeToggle
}

// ----- PPUADDR 0x2006 REGISTER -----

func (ppu *Ppu) WriteToAddrRegister(val uint8) {
	// write to high/low byte
	if ppu.write == 0 {

		ppu.loopyT = SetBitToVal(ppu.loopyT, 8, (val)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 9, (val>>1)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 10, (val>>2)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 11, (val>>3)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 12, (val>>4)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 13, (val>>5)&0b1)

		// clear bit Z
		ppu.loopyT = SetBitToVal(ppu.loopyT, 14, 0)
		ppu.write = 1
	} else {

		ppu.loopyT = SetBitToVal(ppu.loopyT, 0, (val)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 1, (val>>1)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 2, (val>>2)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 3, (val>>3)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 4, (val>>4)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 5, (val>>5)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 6, (val>>6)&0b1)
		ppu.loopyT = SetBitToVal(ppu.loopyT, 7, (val>>7)&0b1)

		ppu.loopyV = ppu.loopyT
		ppu.write = 0
	}
	ppu.writeToggle = !ppu.writeToggle
}

func (ppu *Ppu) incrementAddrRegister() {
	if ppu.getControlSetting(INCREMENT) == 0 {
		ppu.loopyV += 1
	} else {
		ppu.loopyV += 32
	}
}

// ----- PPUDATA 0x2007 REGISTER -----

func (ppu *Ppu) ReadPpuDataRegister() uint8 {
	// if it is pallete ram, return the value instantly
	val := PpuMemRead(ppu.loopyV)
	if ppu.loopyV%0x4000 >= 0x3F00 {
		ppu.readBuffer = val
	} else {
		tmp := ppu.readBuffer
		ppu.readBuffer = val
		val = tmp
	}
	ppu.incrementAddrRegister()
	return val
}

func (ppu *Ppu) WriteToPpuDataRegister(val uint8) {
	PpuMemWrite(ppu.loopyV, val)
	ppu.incrementAddrRegister()
}

// ----- Rendering -----

func (ppu *Ppu) reloadPixelBuffer() {
	pixelGroup := ppu.getPixelGroup()
	ppu.outputPixelBuffer[8] = pixelGroup[0]
	ppu.outputPixelBuffer[9] = pixelGroup[1]
	ppu.outputPixelBuffer[10] = pixelGroup[2]
	ppu.outputPixelBuffer[11] = pixelGroup[3]
	ppu.outputPixelBuffer[12] = pixelGroup[4]
	ppu.outputPixelBuffer[13] = pixelGroup[5]
	ppu.outputPixelBuffer[14] = pixelGroup[6]
	ppu.outputPixelBuffer[15] = pixelGroup[7]
}

func (ppu *Ppu) getPixelGroup() [8][3]uint8 {
	// coarseX := ppu.loopyV & 0x001F
	// coarseY := (ppu.loopyV >> 5) & 0x001F

	tileAddress := uint16(ppu.getControlSetting(BACKGROUND_TABLE_ADDRESS)) * 0x1000
	nameTableEntry := PpuMemRead((ppu.loopyV & 0x0FFF) | 0x2000) // address = 10NNYYYYYXXXXX
	tile := PpuMemReadTile(tileAddress + 0x10*uint16(nameTableEntry))

	// black magic dont touch
	attributeAddress := 0x23C0 | (ppu.loopyV & 0x0C00) | ((ppu.loopyV >> 4) & 0x38) | ((ppu.loopyV >> 2) & 0x07)
	attribute := PpuMemRead(attributeAddress)

	// fetch the palette
	paletteStart := uint16(DEFAULT_BG_PALETTE_ADDRESS)
	subX := (ppu.loopyV & 0x02) == 0
	subY := (ppu.loopyV & 0x40) == 0
	// subX := coarseX % 2 // if is even x tile, is left of every 4 tile block
	// subY := coarseY % 2 // if is even y tile, is top of every 4 tile block
	switch {
	// top left tiles
	case subX && subY:
		paletteStart += uint16(attribute&0b11) * 4
	// top right tiles
	case !subX && subY:
		paletteStart += uint16((attribute>>2)&0b11) * 4
	// bottom left tiles
	case subX && !subY:
		paletteStart += uint16((attribute>>4)&0b11) * 4
	// bottom right tiles
	case !subX && !subY:
		paletteStart += uint16((attribute>>6)&0b11) * 4
	}

	var palette [4][3]uint8
	// and palette with 0x30, getting first column of system palette
	if ppu.getControlSetting(GREYSCALE) == 1 {
		palette = [4][3]uint8{
			0b00: ppu.systemPalette[PpuMemRead(DEFAULT_BG_PALETTE_ADDRESS)&0x30],
			0b01: ppu.systemPalette[PpuMemRead(paletteStart+1)&0x30],
			0b10: ppu.systemPalette[PpuMemRead(paletteStart+2)&0x30],
			0b11: ppu.systemPalette[PpuMemRead(paletteStart+3)&0x30],
		}
	} else {
		palette = [4][3]uint8{
			0b00: ppu.systemPalette[PpuMemRead(DEFAULT_BG_PALETTE_ADDRESS)],
			0b01: ppu.systemPalette[PpuMemRead(paletteStart+1)],
			0b10: ppu.systemPalette[PpuMemRead(paletteStart+2)],
			0b11: ppu.systemPalette[PpuMemRead(paletteStart+3)],
		}
	}

	// get tile in flat format
	var fullTile [64]uint8
	for i := range 8 {
		for j := range 8 {
			lsb := tile[i] >> (8 - j - 1) & 0b1
			msb := tile[i+8] >> (8 - j - 1) & 0b1
			fullTile[j+8*i] = lsb | (msb << 1)
		}
	}

	var pixelGroup [8][3]uint8
	// fineY to select line of 8 pixels from tile
	fineY := (ppu.loopyV >> 12) & 0b111
	for i := range uint16(8) {
		pix := fullTile[i+8*fineY]
		pixelGroup[i] = palette[pix] // gets rgb of pixel
	}
	return pixelGroup
}

// ----- Internal registers -----

func (ppu *Ppu) copyScrollxToLoopyV() {
	ppu.loopyV = SetBitToVal(ppu.loopyV, 0, uint8(ppu.loopyT&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 1, uint8((ppu.loopyT>>1)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 2, uint8((ppu.loopyT>>2)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 3, uint8((ppu.loopyT>>3)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 4, uint8((ppu.loopyT>>4)&0b1))

	ppu.loopyV = SetBitToVal(ppu.loopyV, 10, uint8((ppu.loopyT>>10)&0b1))
}

func (ppu *Ppu) copyScrollyToLoopyV() {
	ppu.loopyV = SetBitToVal(ppu.loopyV, 5, uint8((ppu.loopyT>>5)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 6, uint8((ppu.loopyT>>6)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 7, uint8((ppu.loopyT>>7)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 8, uint8((ppu.loopyT>>8)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 9, uint8((ppu.loopyT>>9)&0b1))

	ppu.loopyV = SetBitToVal(ppu.loopyV, 11, uint8((ppu.loopyT>>11)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 12, uint8((ppu.loopyT>>12)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 13, uint8((ppu.loopyT>>13)&0b1))
	ppu.loopyV = SetBitToVal(ppu.loopyV, 14, uint8((ppu.loopyT>>14)&0b1))
}

func (ppu *Ppu) incrementLoopyVY() {
	// fineY is not 7 (0b111)
	if (ppu.loopyV & 0x7000) != 0x7000 {
		ppu.loopyV += 0x1000 // increment fineY
	} else {
		ppu.loopyV &= ^uint16(0x7000) // fineY = 0
		coarseY := (ppu.loopyV >> 5) & 0x001F
		switch coarseY {
		case 29: // if coarse Y is 29
			coarseY = 0          // go back to 0
			ppu.loopyV ^= 0x0800 // go to next nameTable
		case 31:
			coarseY = 0
		default:
			coarseY += 1
		}
		ppu.loopyV = (ppu.loopyV &^ 0x03E0) | (coarseY << 5)
	}
}

func (ppu *Ppu) incrementLoopyVX() {
	// coarse X = 31, go to next horizontal nametable
	if (ppu.loopyV & 0x001F) == 31 {
		ppu.loopyV &= ^uint16(0x001F) // set coarse X = 0
		ppu.loopyV ^= 0x0400          // advance nametable
	} else {
		ppu.loopyV += 1
	}
}

// ----- Bit Utils -----

func SetBitToVal(n uint16, pos uint, val uint8) uint16 {
	if val == 1 {
		return SetBitToOne(n, pos)
	} else {
		return SetBitToZero(n, pos)
	}
}

func SetBitToOne(n uint16, pos uint) uint16 {
	return n | (1 << pos)
}

func SetBitToZero(n uint16, pos uint) uint16 {
	return n & ^(1 << pos)
}

func ShiftBufferLeft(slice [][3]uint8) [][3]uint8 {
	copy(slice, slice[1:])
	slice[len(slice)-1] = [3]uint8{0, 0, 0}
	return slice
}

// ----- DEBUG -----

func (ppu *Ppu) TracePpuStatus() string {
	return fmt.Sprintf("PPU:%03d, %03d  ADDR: %04X CTRL:%08b STATUS: %08b", ppu.scanlines, ppu.cycles, ppu.loopyV, ppu.ppuCtrl, ppu.ppuStatus)
}
