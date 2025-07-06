package ppu

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func GenerateFromPalFile(path string) map[uint8][3]uint8 {
	palette := make(map[uint8][3]uint8, 64)

	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening file", err)
		panic("Error reading from file")
	}

	defer file.Close()
	reader := bufio.NewReader(file)

	for i := range uint8(64) {
		color := make([]byte, 3)
		_, err = io.ReadFull(reader, color)
		if err != nil {
			fmt.Println("Error reading palette", err)
			panic("Error reading from file")
		}
		rgb := [3]uint8{color[0], color[1], color[2]}
		palette[i] = rgb
	}

	return palette
}
