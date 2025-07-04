package cpu

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"testing"
	"vsasakiv/nesemulator/cartridge"
	"vsasakiv/nesemulator/memory"
)

func NesTestLineByLine(t *testing.T) {

	file, err := os.Open("../testFiles/nestest.log")
	if err != nil {
		fmt.Println("Error opening nestest.log!")
		return
	}
	defer file.Close()

	nestest := cartridge.ReadFromFile("./testFiles/nestest.nes")
	memory.LoadFromCartridge(nestest)
	GetCpu().Pc = 0xC000

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		logTrace := line[:74]
		cpuTrace := cpu.TraceStatus()

		if logTrace != cpuTrace {
			errorLog := generateErrorLog(logTrace, cpuTrace)
			t.Errorf("%s", errorLog)
		}
	}
}

func generateErrorLog(logTrace string, cpuTrace string) string {
	var diff []int
	errorLog := ""

	for i := range logTrace {
		if logTrace[i] != cpuTrace[i] {
			diff = append(diff, i)
		}
	}

	errorLog += logTrace + "\n"
	for i := range logTrace {
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
