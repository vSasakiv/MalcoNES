package ppu

const XSIZE = 256
const YSIZE = 240

type Frame struct {
	PixelData [256 * 240 * 3]uint8
}

func NewFrame() *Frame {
	var frame Frame
	return &frame
}

func (frame *Frame) setPixel(x uint, y uint, rgb [3]uint8) {
	address := x*3 + y*3*XSIZE
	frame.PixelData[address] = rgb[0]
	frame.PixelData[address+1] = rgb[1]
	frame.PixelData[address+2] = rgb[2]
}

func (frame *Frame) renderTile(tile [16]uint8, tileN uint) {
	// creating full tile

	var fullTile [64]uint8
	for i := range 8 {
		for j := range 8 {
			lsb := tile[i] >> (8 - j - 1) & 0b1
			msb := tile[i+8] >> (8 - j - 1) & 0b1
			fullTile[j+8*i] = lsb | (msb << 1)
		}
	}
	tileX := (tileN % 20) * 10
	tileY := (tileN / 20) * 10

	for i, pix := range fullTile {
		var rgb [3]uint8
		switch pix {
		case 0b00:
			rgb = ppu.systemPalette[0]
		case 0b01:
			rgb = ppu.systemPalette[0x37]
		case 0b10:
			rgb = ppu.systemPalette[0x27]
		case 0b11:
			rgb = ppu.systemPalette[0x0F]
		}

		frame.setPixel(tileX+uint(i%8), tileY+uint(i/8), rgb)
	}
}

func (frame *Frame) RenderRomBank(bank uint) {
	address := uint16(bank) * 0x1000
	for i := range uint(256) {
		frame.renderTile(PpuMemReadTile(address), i)
		address += 0x10
	}
}
