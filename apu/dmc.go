package apu

import (
	"vsasakiv/nesemulator/mappers"
)

// ==================================================================== //
// ||                                                                   ||
// ||                             APU DMC                               ||
// ||                                                                   ||
// ==================================================================== //
//

var dmcTable = []uint{
	214, 190, 170, 160, 143, 127, 113, 107, 95, 80, 71, 64, 53, 42, 36, 27,
}

type DMC struct {
	channelEnable  bool
	loop           bool
	value          uint8
	sampleAddress  uint16
	sampleLength   uint16
	currentAddress uint16
	currentLength  uint16
	shiftRegister  uint8
	bitCount       uint8
	mapper         mappers.Mapper

	cpuStall bool
	timer    RawTimer
}

func (dmc *DMC) WriteToControl(val uint8) {
	dmc.loop = (val>>6)&0b1 == 1
	dmc.timer.period = dmcTable[val&0x0F]
}

func (dmc *DMC) WriteToValue(val uint8) {
	dmc.value = val & 0x7F
}

func (dmc *DMC) WriteAddress(val uint8) {
	dmc.sampleAddress = 0xC000 | (uint16(val) << 6)
}

func (dmc *DMC) WriteLength(val uint8) {
	dmc.sampleLength = (uint16(val) << 4) | 1
}

func (dmc *DMC) restart() {
	dmc.currentAddress = dmc.sampleAddress
	dmc.currentLength = dmc.sampleLength
}

func (dmc *DMC) clockTimer() {
	if !dmc.channelEnable {
		return
	}
	dmc.clockReader()
	dmc.timer.Clock(dmc.clockShifter)
}

func (dmc *DMC) clockReader() {
	if dmc.currentLength > 0 && dmc.bitCount == 0 {
		dmc.cpuStall = true
		dmc.shiftRegister = dmc.mapper.Read(dmc.currentAddress)
		dmc.bitCount = 8
		dmc.currentAddress++
		// wraps around back
		if dmc.currentAddress == 0 {
			dmc.currentAddress = 0x8000
		}
		dmc.currentLength--
		if dmc.currentLength == 0 && dmc.loop {
			dmc.restart()
		}
	}
}

func (dmc *DMC) clockShifter() {
	if dmc.bitCount == 0 {
		return
	}
	if dmc.shiftRegister&0b1 == 1 {
		if dmc.value <= 125 {
			dmc.value += 2
		}
	} else {
		if dmc.value >= 2 {
			dmc.value -= 2
		}
	}
	dmc.shiftRegister >>= 1
	dmc.bitCount--
}

func (dmc *DMC) setChannelEnabled(enabled bool) {
	if !enabled {
		dmc.currentLength = 0
	} else {
		if dmc.currentLength == 0 {
			dmc.restart()
		}
	}
	dmc.channelEnable = enabled
}

func (dmc *DMC) PollReadInterrupt() bool {
	if dmc.cpuStall {
		dmc.cpuStall = false
		return true
	}
	return false
}

func (dmc *DMC) getSample() uint {
	return uint(dmc.value)
}
