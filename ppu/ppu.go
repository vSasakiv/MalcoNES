package ppu

import (
	"fmt"
	"math/bits"
	"vsasakiv/nesemulator/mappers"
)

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
	// register mapped to cpu memory
	ppuCtrl    uint8
	ppuMask    uint8
	ppuStatus  uint8
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
	CurrentFrame Frame
	// sprites
	spriteLine     [8][8]uint8
	spritePosition [8]uint8
	spritePalette  [8][4][3]uint8
	spriteNumber   [8]uint8
	spritePriority [8]uint8
	spriteCount    uint
	// ppu internal registers
	loopyV uint16
	loopyT uint16
	fineX  uint8
	write  uint8
	// output buffers
	outputPixelBuffer   [][3]uint8
	outputBackgroundRgb [][3]uint8
	outputBackgroundVal []uint8
}

// Initialize ppu with corret parameters, also initialize system palette
func NewPpu() *Ppu {
	var ppu Ppu

	ppu.outputPixelBuffer = make([][3]uint8, 16)
	ppu.outputBackgroundRgb = make([][3]uint8, 16)
	ppu.outputBackgroundVal = make([]uint8, 16)

	ppu.cycles = 340
	ppu.scanlines = 240
	ppu.systemPalette = GenerateFromPalFile("./ppu/palettes/2C02.pal")
	return &ppu
}

func (ppu *Ppu) Reset() {
	ppu.ppuCtrl = 0
	ppu.ppuMask = 0
	ppu.write = 0
	ppu.readBuffer = 0
	ppu.loopyV = 0
	ppu.loopyT = 0
	ppu.fineX = 0
}

func GetPpu() *Ppu {
	return &ppu
}

var ppu Ppu = *NewPpu()

func Clock() {

	ppu.runCycle()

	visibleScanlines := ppu.scanlines <= 239
	visibleCycles := ppu.cycles >= 1 && ppu.cycles <= 256
	preRenderScanline := ppu.scanlines == 261
	preRenderCopyY := ppu.cycles >= 280 && ppu.cycles <= 304
	vblank := ppu.scanlines == 241 && ppu.cycles == 1
	vblankEnd := preRenderScanline && ppu.cycles == 1
	spriteEvaluate := visibleScanlines && ppu.cycles == 257

	// rendering visible scanlines
	if ppu.getMaskSetting(ENABLE_BACKGROUND) == 1 {
		if visibleScanlines {
			if visibleCycles {
				// draw pixel from output buffer
				ppu.renderPixel()
				// if cycle is 8, 16, 24 ... reload shift register and update loopyV
				if ppu.cycles%8 == 0 {
					ppu.reloadBackgroundBuffer()
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
				ppu.reloadBackgroundBuffer()
				ppu.incrementLoopyVX()
			}
			// load second tile
			if ppu.cycles == 335 {
				for range 8 {
					// shift background buffers
					ppu.outputBackgroundVal = ShiftValBufferLeft(ppu.outputBackgroundVal)
					ppu.outputBackgroundRgb = ShiftRgbBufferLeft(ppu.outputBackgroundRgb)
				}
				ppu.reloadBackgroundBuffer()
				ppu.incrementLoopyVX()
			}
		}
		if preRenderScanline && preRenderCopyY {
			ppu.copyScrollyToLoopyV()
		}
	}

	if ppu.getMaskSetting(ENABLE_SPRITE) == 1 {
		if spriteEvaluate {
			ppu.evaluateSprites()
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
		ppu.clearSpriteZeroHit()
		ppu.setVblankStatus(0)
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

		ppu.write = 0
	}
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

// ----- Rendering

func (ppu *Ppu) renderPixel() {

	rgb := ppu.outputBackgroundRgb[ppu.fineX]

	for i := range ppu.spriteCount {
		diff := int(ppu.cycles-1) - int(ppu.spritePosition[i])
		// sprite is not at this x
		if diff < 0 || diff > 7 {
			continue
		}
		// sprite is transparent
		if ppu.spriteLine[i][diff] == 0 {
			continue
		}
		// if sprite has priority or background is transparent we render it
		if ppu.spritePriority[i] == 0 || ppu.outputBackgroundVal[ppu.fineX] == 0 {
			rgb = ppu.spritePalette[i][ppu.spriteLine[i][diff]]
		}
		// background is not transparent
		if ppu.outputBackgroundVal[ppu.fineX] != 0 && ppu.spriteNumber[i] == 0 {
			ppu.setSpriteZeroHit()
		}
		break
	}

	// if no sprite is rendered, render background instead
	ppu.CurrentFrame.setPixel(ppu.cycles-1, ppu.scanlines, rgb)
	// shift background buffers
	ppu.outputBackgroundVal = ShiftValBufferLeft(ppu.outputBackgroundVal)
	ppu.outputBackgroundRgb = ShiftRgbBufferLeft(ppu.outputBackgroundRgb)

}

// ----- Background Rendering -----

func (ppu *Ppu) reloadBackgroundBuffer() {
	valueGroup, pixelGroup := ppu.getBackgroundGroup()
	for i := range 8 {
		ppu.outputBackgroundRgb[8+i] = pixelGroup[i]
		ppu.outputBackgroundVal[8+i] = valueGroup[i]
	}
}

func (ppu *Ppu) getBackgroundGroup() ([8]uint8, [8][3]uint8) {
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

	var pixelGroup [8][3]uint8
	var valGroup [8]uint8
	// fineY to select line of 8 pixels from tile
	fineY := (ppu.loopyV >> 12) & 0b111

	for i := range 8 {
		lsb := tile[fineY] >> (8 - i - 1) & 0b1
		msb := tile[fineY+8] >> (8 - i - 1) & 0b1
		pix := lsb | (msb << 1)
		valGroup[i] = pix
		pixelGroup[i] = palette[pix] // gets rgb of pixel
	}

	return valGroup, pixelGroup
}

// ----- Sprite Rendering -----

func (ppu *Ppu) evaluateSprites() {
	tileAddress := uint16(ppu.getControlSetting(SPRITE_TABLE_ADDRESS)) * 0x1000
	count := uint(0)
	for i := range uint8(64) {
		idx := (i * 4)

		pixY := PpuOamRead(idx)
		pixX := PpuOamRead(idx + 3)
		tileIdx := PpuOamRead(idx + 1)
		attrb := PpuOamRead(idx + 2)

		// support for 8x16 sprites
		var spriteHeight int
		if ppu.getControlSetting(SPRITE_SIZE) == 1 {
			spriteHeight = 16
		} else {
			spriteHeight = 8
		}

		diff := int(ppu.scanlines) - int(pixY)
		// sprite is not in this scanline
		if diff < 0 || diff >= spriteHeight {
			continue
		}

		var tile []uint8
		if spriteHeight == 8 {
			tile = PpuMemReadTile(tileAddress + 0x10*uint16(tileIdx))
		} else {
			tile = PpuMemReadBigTile(uint16(tileIdx&0x01)*0x1000 + 0x20*uint16(tileIdx>>1))
		}

		flipHorizontal := (attrb>>6)&0b1 == 1
		flipVertical := (attrb>>7)&0b1 == 1
		tile = flipTile(tile, flipHorizontal, flipVertical)

		paletteIdx := attrb & 0b11

		if count < 8 {
			ppu.spriteLine[count] = getSpriteLine(tile[:], diff)
			ppu.spritePosition[count] = pixX
			ppu.spritePalette[count] = getOamSpritePallete(paletteIdx, ppu.getMaskSetting(GREYSCALE))
			ppu.spriteNumber[count] = i
			ppu.spritePriority[count] = (attrb >> 5) & 0b1
		}

		count += 1

		// frame.renderOamTile(tile, uint(tileX), uint(tileY), palette)
	}
	if count > 8 {
		count = 8
	}
	ppu.spriteCount = count
}

func flipTile(tile []uint8, flipHorizontal bool, flipVertical bool) []uint8 {
	if flipHorizontal {
		for i, v := range tile {
			tile[i] = bits.Reverse8(v)
		}
	}

	if flipVertical && len(tile) == 16 {
		for i := range uint8(4) {
			aux := tile[i]
			tile[i] = tile[7-i]
			tile[7-i] = aux
			aux = tile[i+8]
			tile[i+8] = tile[15-i]
			tile[15-i] = aux
		}
	} else if flipVertical && len(tile) == 32 {
		// flip both tiles independently, than flip the whole thing
		for i := range uint8(4) {
			aux := tile[i]
			tile[i] = tile[7-i]
			tile[7-i] = aux

			aux = tile[i+8]
			tile[i+8] = tile[15-i]
			tile[15-i] = aux

			aux = tile[i+16]
			tile[i+16] = tile[23-i]
			tile[23-i] = aux

			aux = tile[i+24]
			tile[i+24] = tile[31-i]
			tile[31-i] = aux
		}
		for i := range uint8(8) {
			aux := tile[i]
			tile[i] = tile[i+16]
			tile[i+16] = aux

			aux = tile[i+8]
			tile[i+8] = tile[i+24]
			tile[i+24] = aux
		}
	}

	return tile
}

func getSpriteLine(tile []uint8, diff int) [8]uint8 {
	var spriteLine [8]uint8

	for i := range 8 {
		if diff < 8 {
			lsb := tile[diff] >> (8 - i - 1) & 0b1
			msb := tile[diff+8] >> (8 - i - 1) & 0b1
			spriteLine[i] = lsb | (msb << 1)
		} else {
			lsb := tile[diff+8] >> (8 - i - 1) & 0b1
			msb := tile[diff+16] >> (8 - i - 1) & 0b1
			spriteLine[i] = lsb | (msb << 1)
		}
	}
	return spriteLine
}

func getOamSpritePallete(paletteIdx uint8, greyscale uint8) [4][3]uint8 {
	paletteStart := uint16(DEFAULT_OAM_PALETTE_ADDRESS) + uint16(paletteIdx)*4
	if greyscale == 1 {
		return [4][3]uint8{
			0b00: ppu.systemPalette[PpuMemRead(paletteStart)&0x30],
			0b01: ppu.systemPalette[PpuMemRead(paletteStart+1)&0x30],
			0b10: ppu.systemPalette[PpuMemRead(paletteStart+2)&0x30],
			0b11: ppu.systemPalette[PpuMemRead(paletteStart+3)&0x30],
		}
	}
	return [4][3]uint8{
		0b00: ppu.systemPalette[PpuMemRead(paletteStart)],
		0b01: ppu.systemPalette[PpuMemRead(paletteStart+1)],
		0b10: ppu.systemPalette[PpuMemRead(paletteStart+2)],
		0b11: ppu.systemPalette[PpuMemRead(paletteStart+3)],
	}
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

func ShiftRgbBufferLeft(slice [][3]uint8) [][3]uint8 {
	copy(slice, slice[1:])
	slice[len(slice)-1] = [3]uint8{0, 0, 0}
	return slice
}

func ShiftValBufferLeft(slice []uint8) []uint8 {
	copy(slice, slice[1:])
	slice[len(slice)-1] = 0
	return slice
}

// ----- Mapper info -----

func GetPpuStatus() mappers.Status {
	status := mappers.Status{}
	status.PpuScanlines = ppu.scanlines
	status.PpuCycles = ppu.cycles
	status.PpuBackgroundEnabled = ppu.getMaskSetting(ENABLE_BACKGROUND) == 1
	status.PpuSpriteEnabled = ppu.getMaskSetting(ENABLE_SPRITE) == 1
	return status
}

// ----- DEBUG -----

func (ppu *Ppu) TracePpuStatus() string {
	return fmt.Sprintf("PPU:%03d, %03d  ADDR: %04X CTRL:%08b STATUS: %08b", ppu.scanlines, ppu.cycles, ppu.loopyV, ppu.ppuCtrl, ppu.ppuStatus)
}
