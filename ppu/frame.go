package ppu

import (
	"math/bits"
)

const XSIZE = 256
const YSIZE = 240
const DEFAULT_BG_PALETTE_ADDRESS = 0x3F00
const DEFAULT_OAM_PALETTE_ADDRESS = 0x3F10
const NAMETABLE_SIZE = 0x03C0

type Frame struct {
	PixelData [XSIZE * YSIZE * 3]uint8
	// stores whether or not the background pixel is transparent, 0 = transparent ; 1 = opaque
	TransparencyMatrix [XSIZE][YSIZE]uint8
}

func NewFrame() *Frame {
	var frame Frame
	return &frame
}

func (frame *Frame) GetPixelData() []uint8 {
	return frame.PixelData[:]
}

func (frame *Frame) setPixel(x uint, y uint, rgb [3]uint8) {
	address := x*3 + y*3*XSIZE
	frame.PixelData[address] = rgb[0]
	frame.PixelData[address+1] = rgb[1]
	frame.PixelData[address+2] = rgb[2]
}

func (frame *Frame) renderTile(tile [16]uint8, tileN uint, palette [4][3]uint8) {
	// rendering full tile
	var fullTile [64]uint8
	for i := range 8 {
		for j := range 8 {
			lsb := tile[i] >> (8 - j - 1) & 0b1
			msb := tile[i+8] >> (8 - j - 1) & 0b1
			fullTile[j+8*i] = lsb | (msb << 1)
		}
	}
	tileX := (tileN % 32) * 8
	tileY := (tileN / 32) * 8

	for i, pix := range fullTile {
		pixX := tileX + uint(i%8)
		pixY := tileY + uint(i/8)
		// transparency matrix
		if pix == 0b00 {
			frame.TransparencyMatrix[pixX][pixY] = 0
		} else {
			frame.TransparencyMatrix[pixX][pixY] = 1
		}
		rgb := palette[pix]
		frame.setPixel(pixX, pixY, rgb)
	}
}

func (frame *Frame) renderOamTile(tile [16]uint8, tileX uint, tileY uint, palette [4][3]uint8) {
	var fullTile [64]uint8
	for i := range 8 {
		for j := range 8 {
			lsb := tile[i] >> (8 - j - 1) & 0b1
			msb := tile[i+8] >> (8 - j - 1) & 0b1
			fullTile[j+8*i] = lsb | (msb << 1)
		}
	}

	for i, pix := range fullTile {
		if pix == 0b00 {
			continue
		}
		rgb := palette[pix]
		pixX := tileX + uint(i%8)
		pixY := tileY + uint(i/8)
		// out of bonunds, dont render
		if pixX > XSIZE || pixY > YSIZE {
			continue
		}
		frame.setPixel(pixX, pixY, rgb)
	}
}

type View struct {
	x, y, width, height int
}

func (frame *Frame) RenderBackground(baseNameTableAddress uint16, tileBank uint, scrollX uint8, scrollY uint8, greyscale uint8) {
	nextNameTableAddress := GetNextNameTableAddress(baseNameTableAddress)
	baseTableView := View{x: int(scrollX), y: int(scrollY), width: XSIZE - int(scrollX), height: YSIZE - int(scrollY)}
	nextTableView := View{x: 0, y: 0, width: int(scrollX), height: int(scrollY)}

	frame.renderNameTable(baseNameTableAddress, tileBank, baseTableView, -int(scrollX), -int(scrollY), greyscale)
	frame.renderNameTable(nextNameTableAddress, tileBank, nextTableView, XSIZE-int(scrollX), YSIZE-int(scrollY), greyscale)
}

func (frame *Frame) renderNameTable(nameTableAddress uint16, tileBank uint, view View, shiftx int, shifty int, greyscale uint8) {

	tileAddress := uint16(tileBank) * 0x1000

	tileStartX := view.x / 8
	tileStartY := view.y / 8
	tileEndX := (view.width + view.x) / 8
	tileEndY := (view.height + view.y) / 8
	// fmt.Printf("startX: %03d startY: %03d endx: %03d endt: %03d\n", tileStartX, tileStartY, tileEndX, tileEndY)

	for y := tileStartY; y < tileEndY; y++ {
		for x := tileStartX; x < tileEndX; x++ {
			idx := uint16(x) + 32*uint16(y)
			nameTableEntry := PpuMemRead(nameTableAddress + idx)
			tile := PpuMemReadTile(tileAddress + 0x10*uint16(nameTableEntry))
			palette := GetBackgroundPalette(nameTableAddress, idx, greyscale)
			frame.conditionalRenderScrollTile(tile, uint(idx), view, shiftx, shifty, palette)
		}
	}
	// 30 x 32 tiles
	// TODO
}

func (frame *Frame) conditionalRenderScrollTile(tile [16]uint8, tileN uint, view View, shiftx int, shifty int, palette [4][3]uint8) {
	// rendering full tile
	var fullTile [64]uint8
	for i := range 8 {
		for j := range 8 {
			lsb := tile[i] >> (8 - j - 1) & 0b1
			msb := tile[i+8] >> (8 - j - 1) & 0b1
			fullTile[j+8*i] = lsb | (msb << 1)
		}
	}
	tileX := (tileN % 32) * 8
	tileY := (tileN / 32) * 8

	for i, pix := range fullTile {
		pixX := int(tileX + uint(i%8))
		pixY := int(tileY + uint(i/8))
		// dont render border pixels
		if pixX < view.x || pixY < view.y || pixX >= (view.x+view.width) || pixY >= (view.y+view.height) {
			continue
		}
		// transparency matrix
		if pix == 0b00 {
			frame.TransparencyMatrix[pixX+shiftx][pixY+shifty] = 0
		} else {
			frame.TransparencyMatrix[pixX+shiftx][pixY+shifty] = 1
		}
		rgb := palette[pix]
		frame.setPixel(uint(pixX+shiftx), uint(pixY+shifty), rgb)
	}
}

func (frame *Frame) RenderNameTable(nameTableAddress uint16, tileBank uint, greyscale uint8) {
	tileAddress := uint16(tileBank) * 0x1000
	for i := range uint16(NAMETABLE_SIZE) {
		nameTableEntry := PpuMemRead(nameTableAddress + i)
		tile := PpuMemReadTile(tileAddress + 0x10*uint16(nameTableEntry))
		palette := GetBackgroundPalette(nameTableAddress, i, greyscale)
		frame.renderTile(tile, uint(i), palette)
	}
}

func (frame *Frame) RenderEmptyBackground() {
	backdrop := ppu.systemPalette[PpuMemRead(DEFAULT_BG_PALETTE_ADDRESS)]
	for i := range uint(XSIZE) {
		for j := range uint(YSIZE) {
			frame.setPixel(i, j, backdrop)
			frame.TransparencyMatrix[i][j] = 0
		}
	}
}

func (frame *Frame) RenderOam(tileBank uint, greyscale uint8) {
	tileAddress := uint16(tileBank) * 0x1000
	addr := uint8(252)
	for i := range uint8(64) {
		idx := addr - (i * 4)
		tileY := PpuOamRead(idx)
		tileX := PpuOamRead(idx + 3)
		// out of bounds
		if tileY > YSIZE {
			continue
		}
		tileIdx := PpuOamRead(idx + 1)
		attrb := PpuOamRead(idx + 2)

		tile := PpuMemReadTile(tileAddress + 0x10*uint16(tileIdx))
		flipHorizontal := (attrb>>6)&0b1 == 1
		flipVertical := (attrb>>7)&0b1 == 1
		tile = flipTile(tile, flipHorizontal, flipVertical)

		paletteIdx := attrb & 0b11
		palette := getOamSpritePallete(paletteIdx, greyscale)
		frame.renderOamTile(tile, uint(tileX), uint(tileY), palette)
	}
}

func flipTile(tile [16]uint8, flipHorizontal bool, flipVertical bool) [16]uint8 {
	if flipHorizontal {
		for i, v := range tile {
			tile[i] = bits.Reverse8(v)
		}
	}
	if flipVertical {
		for i := range uint8(4) {
			aux := tile[i]
			tile[i] = tile[7-i]
			tile[7-i] = aux
			aux = tile[i+8]
			tile[i+8] = tile[15-i]
			tile[15-i] = tile[i+8]
		}
	}
	return tile
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

func (frame *Frame) spriteZeroCollision(tileBank uint) (bool, uint, uint) {
	tileAddress := uint16(tileBank) * 0x1000
	tileY := PpuOamRead(0)
	tileX := PpuOamRead(3)
	// out of bounds
	if tileY > YSIZE {
		return false, 0, 0
	}

	tileIdx := PpuOamRead(1)
	attrb := PpuOamRead(2)

	tile := PpuMemReadTile(tileAddress + 0x10*uint16(tileIdx))
	flipHorizontal := (attrb>>6)&0b1 == 1
	flipVertical := (attrb>>7)&0b1 == 1
	tile = flipTile(tile, flipHorizontal, flipVertical)

	var fullTile [64]uint8
	for i := range 8 {
		for j := range 8 {
			lsb := tile[i] >> (8 - j - 1) & 0b1
			msb := tile[i+8] >> (8 - j - 1) & 0b1
			fullTile[j+8*i] = lsb | (msb << 1)
		}
	}

	for i, pix := range fullTile {
		pixX := uint(tileX) + uint(i%8)
		pixY := uint(tileY) + uint(i/8)
		if pixX > XSIZE || pixY > YSIZE {
			continue
		}
		// pix and background are opaque
		if pix != 0b00 && frame.TransparencyMatrix[pixX][pixY] == 1 {
			return true, pixX + 2, pixY + 1
		}
	}
	return false, 0, 0
}

func GetBackgroundPalette(nameTableAddress uint16, tileN uint16, greyscale uint8) [4][3]uint8 {
	tileX := tileN % 32
	tileY := tileN / 32
	// divided in meta-tiles of 4 tiles each, divide everything by 4
	attributeAddress := nameTableAddress + NAMETABLE_SIZE + (tileX / 4) + (tileY/4)*8
	attribute := PpuMemRead(attributeAddress)

	subTileX := (tileX % 4) / 2
	subTileY := (tileY % 4) / 2
	paletteStart := uint16(DEFAULT_BG_PALETTE_ADDRESS)
	switch {
	// top left tiles
	case subTileX == 0 && subTileY == 0:
		paletteStart += uint16(attribute&0b11) * 4
	// top right tiles
	case subTileX == 1 && subTileY == 0:
		paletteStart += uint16((attribute>>2)&0b11) * 4
	// bottom left tiles
	case subTileX == 0 && subTileY == 1:
		paletteStart += uint16((attribute>>4)&0b11) * 4
	// bottom right tiles
	case subTileX == 1 && subTileY == 1:
		paletteStart += uint16((attribute>>6)&0b11) * 4
	}
	// and palette with 0x30, getting first column of system palette
	if greyscale == 1 {
		return [4][3]uint8{
			0b00: ppu.systemPalette[PpuMemRead(DEFAULT_BG_PALETTE_ADDRESS)&0x30],
			0b01: ppu.systemPalette[PpuMemRead(paletteStart+1)&0x30],
			0b10: ppu.systemPalette[PpuMemRead(paletteStart+2)&0x30],
			0b11: ppu.systemPalette[PpuMemRead(paletteStart+3)&0x30],
		}
	}

	return [4][3]uint8{
		0b00: ppu.systemPalette[PpuMemRead(DEFAULT_BG_PALETTE_ADDRESS)],
		0b01: ppu.systemPalette[PpuMemRead(paletteStart+1)],
		0b10: ppu.systemPalette[PpuMemRead(paletteStart+2)],
		0b11: ppu.systemPalette[PpuMemRead(paletteStart+3)],
	}
}

func (frame *Frame) RenderRomBank(bank uint) {
	address := uint16(bank) * 0x1000
	for i := range uint(256) {
		palette := [4][3]uint8{
			0b00: ppu.systemPalette[0x00],
			0b01: ppu.systemPalette[0x17],
			0b10: ppu.systemPalette[0x21],
			0b11: ppu.systemPalette[0x0F],
		}
		frame.renderTile(PpuMemReadTile(address), i, palette)
		address += 0x10
	}
}
