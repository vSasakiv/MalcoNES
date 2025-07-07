package ppu

import (
	"math/bits"
)

const XSIZE = 256
const YSIZE = 240
const DEFAULT_BG_PALETTE_ADDRESS = 0x3F00
const DEFAULT_OAM_PALETTE_ADDRESS = 0x3F11
const NAMETABLE_SIZE = 0x03C0

type Frame struct {
	PixelData [XSIZE * YSIZE * 3]uint8
	Ready     bool
}

func NewFrame() *Frame {
	var frame Frame
	frame.Ready = false
	return &frame
}

func (frame *Frame) setPixel(x uint, y uint, rgb [3]uint8) {
	address := x*3 + y*3*XSIZE
	frame.PixelData[address] = rgb[0]
	frame.PixelData[address+1] = rgb[1]
	frame.PixelData[address+2] = rgb[2]
}

func (frame *Frame) GetPixelDataAndUpdateStatus() [XSIZE * YSIZE * 3]uint8 {
	frame.Ready = false
	return frame.PixelData
}

func (frame *Frame) renderTile(tile [16]uint8, tileN uint, palette map[uint8][3]uint8) {
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
		rgb := palette[pix]
		frame.setPixel(tileX+uint(i%8), tileY+uint(i/8), rgb)
	}
}

func (frame *Frame) renderOamTile(tile [16]uint8, tileX uint, tileY uint, palette map[uint8][3]uint8) {
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

func (frame *Frame) RenderRomBank(bank uint) {
	address := uint16(bank) * 0x1000
	for i := range uint(256) {
		palette := map[uint8][3]uint8{
			0b00: ppu.systemPalette[0x00],
			0b01: ppu.systemPalette[0x17],
			0b10: ppu.systemPalette[0x21],
			0b11: ppu.systemPalette[0x0F],
		}
		frame.renderTile(PpuMemReadTile(address), i, palette)
		address += 0x10
	}
}

func (frame *Frame) RenderNameTable(nameTableAddress uint16, tileBank uint) {
	tileAddress := uint16(tileBank) * 0x1000
	for i := range uint16(NAMETABLE_SIZE) {
		nameTableEntry := PpuMemRead(nameTableAddress + i)
		tile := PpuMemReadTile(tileAddress + 0x10*uint16(nameTableEntry))
		palette := GetBackgroundPalette(nameTableAddress, i)
		frame.renderTile(tile, uint(i), palette)
	}
}

func (frame *Frame) RenderOam(tileBank uint) {
	tileAddress := uint16(tileBank) * 0x1000
	for i := range uint8(64) {
		idx := i * 4
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
		palette := getOamSpritePallete(paletteIdx)
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

func getOamSpritePallete(paletteIdx uint8) map[uint8][3]uint8 {
	paletteStart := uint16(DEFAULT_OAM_PALETTE_ADDRESS) + uint16(paletteIdx)*4
	return map[uint8][3]uint8{
		0b00: ppu.systemPalette[PpuMemRead(DEFAULT_OAM_PALETTE_ADDRESS)],
		0b01: ppu.systemPalette[PpuMemRead(paletteStart)],
		0b10: ppu.systemPalette[PpuMemRead(paletteStart+1)],
		0b11: ppu.systemPalette[PpuMemRead(paletteStart+2)],
	}
}

func GetBackgroundPalette(nameTableAddress uint16, tileN uint16) map[uint8][3]uint8 {
	tileX := tileN % 32
	tileY := tileN / 32
	// divided in meta-tiles of 4 tiles each, divide everything by 4
	attributeAddress := nameTableAddress + NAMETABLE_SIZE + (tileX / 4) + (tileY/4)*8
	attribute := PpuMemRead(attributeAddress)

	subTileX := (tileX % 4) / 2
	subTileY := (tileY % 4) / 2
	paletteStart := uint16(DEFAULT_BG_PALETTE_ADDRESS) + 1
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
	return map[uint8][3]uint8{
		0b00: ppu.systemPalette[PpuMemRead(DEFAULT_BG_PALETTE_ADDRESS)],
		0b01: ppu.systemPalette[PpuMemRead(paletteStart)],
		0b10: ppu.systemPalette[PpuMemRead(paletteStart+1)],
		0b11: ppu.systemPalette[PpuMemRead(paletteStart+2)],
	}
}
