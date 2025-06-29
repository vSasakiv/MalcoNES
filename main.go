package main

import "fmt"

func main() {
	// MainMemory.memWrite(0, 0xA9)
	// MainMemory.memWrite(1, 0x09)
	// MainMemory.memWrite(2, 0xC9)
	// MainMemory.memWrite(3, 0x09)
	// cpu := NewCpu()
	//
	// cpu.executeNext()
	// cpu.printStatus()
	// cpu.executeNext()
	// cpu.printStatus()

	var pc uint16 = 0x0608
	var mem uint8 = 0xf8
	result := uint16(int16(pc) + int16(2) + int16(int8(mem)))

	fmt.Printf("val: %x\n", result)
}
