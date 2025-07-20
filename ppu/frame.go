package ppu

const XSIZE = 256
const YSIZE = 240
const NAMETABLE_SIZE = 0x03C0

type Frame struct {
	PixelData [XSIZE * YSIZE * 3]uint8
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
	copy(frame.PixelData[address:], rgb[:])
}
