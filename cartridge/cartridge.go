package cartridge

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type Cartridge struct {
	PrgRom        []uint8
	PrgRomSize    uint
	ChrRom        []uint8
	ChrRomSize    uint
	ChrRam        []uint8
	ChrRamSize    uint
	Trainer       []uint8
	SRam          []uint8
	HasTrainer    bool
	MapperType    uint8
	MirroringType string
}

const MirroringSingle0 = "S0"
const MirroringSingle1 = "S1"
const VerticalMirroring = "V"
const HorizontalMirroring = "H"
const FourScreenMirroring = "4"

func ReadFromFile(path string) Cartridge {
	var cartridge Cartridge
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening file", err)
		panic("Error reading from file")
	}

	defer file.Close()
	reader := bufio.NewReader(file)

	header := make([]byte, 16)
	_, err = io.ReadFull(reader, header)
	if err != nil {
		fmt.Println("Error reading header", err)
		panic("Error reading from file")
	}
	cartridge.readHeader(header)

	if cartridge.HasTrainer {
		trainer := make([]byte, 512)
		_, err = io.ReadFull(reader, trainer)
		if err != nil {
			fmt.Println("Error reading trainer", err)
			panic("Error reading from file")
		}
		cartridge.Trainer = trainer
	}

	prgRom := make([]byte, cartridge.PrgRomSize)
	_, err = io.ReadFull(reader, prgRom)
	if err != nil {
		fmt.Println("Error reading PRG ROM", err)
		panic("Error reading from file")
	}
	cartridge.PrgRom = prgRom

	chrRom := make([]byte, cartridge.ChrRomSize)
	_, err = io.ReadFull(reader, chrRom)
	if err != nil {
		fmt.Println("Error reading CHR ROM", err)
		panic("Error reading from file")
	}
	cartridge.ChrRom = chrRom

	cartridge.ChrRam = make([]byte, cartridge.ChrRamSize)
	fmt.Println(len(cartridge.ChrRam))

	return cartridge
}

func (cartridge *Cartridge) readHeader(header []uint8) {

	if header[0] != 0x4E || header[1] != 0x45 || header[2] != 0x53 || header[3] != 0x1A {
		fmt.Println("File is not and iNES file!")
		return
	}

	cartridge.PrgRomSize = uint(header[4]) * 0x4000
	cartridge.ChrRomSize = uint(header[5]) * 0x2000

	// chr ram
	if cartridge.ChrRomSize == 0 {
		cartridge.ChrRamSize = 0x2000
	}

	control1 := header[6]
	control2 := header[7]

	if control2&0b1 == 1 || (control2>>1)&0b1 == 1 || (control2>>2)&0b11 == 0b10 {
		fmt.Println("File is not and iNES 1.0 file! iNES 2.0 is not supported")
		return
	}

	if (control1>>3)&0b1 == 1 {
		cartridge.MirroringType = FourScreenMirroring
	} else if control1&0b1 == 1 {
		cartridge.MirroringType = VerticalMirroring
	} else {
		cartridge.MirroringType = HorizontalMirroring
	}
	cartridge.MapperType = (control1 >> 4) | (control2 & 0b1111_0000)

	// has SRAM
	if (control1>>1)&0b1 == 1 || cartridge.MapperType == 4 || cartridge.MapperType == 2 {
		cartridge.SRam = make([]uint8, 0x2000)
	}

	if (control1>>2)&0b1 == 1 {
		cartridge.HasTrainer = true
	} else {
		cartridge.HasTrainer = false
	}
}
