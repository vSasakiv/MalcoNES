package main

import (
	"fmt"
	"vsasakiv/nesemulator/cpu"
	"vsasakiv/nesemulator/memory"
)

func main() {
	// MainMemory.memWrite(3, 0x09)
	// cpu := NewCpu()
	//
	// cpu.executeNext()
	// cpu.printStatus()
	// cpu.executeNext()
	// cpu.printStatus()

	memory.MemWrite(0, 0x8E)
	memory.MemWrite(1, 0x00)
	memory.MemWrite(2, 0x02)
	mainCpu := cpu.GetCpu()
	trace := mainCpu.TraceStatus()
	cpu.ExecuteNext()
	cpu.ExecuteNext()
	cpu.ExecuteNext()
	fmt.Printf("%s\n", trace)
}
