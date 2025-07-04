package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"vsasakiv/nesemulator/cartridge"
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
	// cpu.GetCpu().Pc = 0xC000
	// nestest := cartridge.ReadFromFile("./testFiles/nestest.nes")
	// memory.LoadFromCartridge(nestest)
	// cpu.GetCpu().RunAndTraceToFile("mylog.log")

	NesTestLineByLine()
}

func NesTestLineByLine() {

	file, err := os.Open("./testFiles/nestest.log")
	if err != nil {
		fmt.Println("Error opening nestest.log!")
		return
	}
	defer file.Close()

	nestest := cartridge.ReadFromFile("./testFiles/nestest.nes")
	memory.LoadFromCartridge(nestest)
	cpu.GetCpu().Pc = 0xC000

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		logTrace := line[:73]
		cpuTrace := cpu.GetCpu().TraceStatus()

		if logTrace != cpuTrace {
			errorLog := generateErrorLog(logTrace, cpuTrace)
			fmt.Printf("%s", errorLog)
			return
		}
		cpu.ExecuteNext()
	}
}

func generateErrorLog(logTrace string, cpuTrace string) string {
	var diff []int
	errorLog := ""

	for i := range cpuTrace {
		if logTrace[i] != cpuTrace[i] {
			diff = append(diff, i)
		}
	}

	errorLog += logTrace + "\n"
	for i := range cpuTrace {
		if slices.Contains(diff, i) {
			errorLog += "^"
		} else {
			errorLog += " "
		}
	}
	errorLog += "\n"
	errorLog += cpuTrace + "\n"
	return errorLog
}
