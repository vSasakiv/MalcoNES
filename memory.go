package main

import (
	"fmt"
	"os"
)

type Memory [0xFFFF]uint8

var MainMemory Memory

// copy of memory, where we have a 1 where the memory was touched in some point, only for debug
// and dumping purposes
var modified Memory
var debug bool = true

func (memory *Memory) memRead(addr uint16) uint8 {
	return memory[addr]
}

func (memory *Memory) memRead16(addr uint16) uint16 {
	low := uint16(memory[addr])
	high := uint16(memory[addr+1]) << 8
	return high + low
}

func (memory *Memory) memWrite(addr uint16, val uint8) {
	if debug {
		modified[addr] = 1
	}
	memory[addr] = val
}

func (memory *Memory) memWrite16(addr uint16, val uint16) {
	if debug {
		modified[addr] = 1
		modified[addr+1] = 1
	}
	memory[addr] = uint8(val & 0xff)
	memory[addr+1] = uint8((val >> 8) & 0xff)
}

func (memory *Memory) hexDump(filename string) {

	content := ""

	for i := 0; i < 0xffff; i++ {
		if modified[i] == 1 {
			content += fmt.Sprintf("%4x : %2x\n", i, memory.memRead(uint16(i)))
		}
	}

	file, err := os.Create(filename)

	if err != nil {
		fmt.Println("Error creating the file", err)
		return
	}

	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println("Error writing to file", err)
		return
	}

	fmt.Println("Successfully dumped memory to ", filename)

}
