package apu

// ==================================================================== //
// ||                                                                   ||
// ||                      APU TRIANGLE PULSES                          ||
// ||                                                                   ||
// ==================================================================== //
//

var triangleSequencerTable = []uint8{
	15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
}

type TrianglePulse struct {
	channelEnable bool

	linearCounterValue  uint
	linearCounterPeriod uint
	linearCounterReload bool

	// timerPeriod uint
	// timerValue  uint

	// lengthCounterHalt bool
	// lengthCounter     uint
	sequencerStep uint

	lengthCounter LengthCounter
	timer         RawTimer
}

// write to register 0x4008 of triangle pulse
func (triangle *TrianglePulse) WriteToLinearCounter(val uint8) {
	// val = CRRR.RRRR
	// length counter halt = C
	// linear counter reload = RRR.RRRR

	triangle.lengthCounter.halted = (val>>7)&0b1 == 1
	triangle.linearCounterPeriod = uint(val & 0b111_1111)
}

// write to register 0x400A of triangle pulse
func (triangle *TrianglePulse) WriteToTimerLow(val uint8) {
	// val = LLLL.LLLL
	// LLLL.LLLL -> lower 8 bits of sequencer timer
	// triangle.timerPeriod = uint(val) | (triangle.timerPeriod & 0xFF00)
	triangle.timer.setTimerLow(val)
}

// write to register 0x400B of triangle pulse
func (triangle *TrianglePulse) WriteToTimerHigh(val uint8) {
	// val = llll.lHHH
	// llll.l -> length counter load
	// HHH -> high 3 bits of sequencer timer
	// also set the start envelope flag
	// triangle.timerPeriod = uint(val&7)<<8 | (triangle.timerPeriod & 0x00FF)
	// triangle.lengthCounter = uint(lengthLookUpTable[(val >> 3)])
	// triangle.timerValue = triangle.timerPeriod

	triangle.lengthCounter.setValue(uint(val) >> 3)
	triangle.timer.setTimerHigh(val)
	triangle.timer.value = triangle.timer.period
	triangle.linearCounterReload = true
}

// clocks the 11 bit timer every cpu cycle
func (triangle *TrianglePulse) clockTimer() {
	triangle.timer.Clock(triangle.clockSequencer)

	// if triangle.timerValue == 0 {
	// 	if triangle.linearCounterValue > 0 && triangle.lengthCounter.value > 0 {
	// 		triangle.clockSequencer()
	// 	}
	// 	triangle.timerValue = triangle.timerPeriod
	// } else {
	// 	triangle.timerValue -= 1
	// }
}

// clock the sequencer
func (triangle *TrianglePulse) clockSequencer() {
	if triangle.linearCounterValue > 0 && triangle.lengthCounter.value > 0 {
		if triangle.sequencerStep == 31 {
			triangle.sequencerStep = 0
		} else {
			triangle.sequencerStep += 1
		}
	}
}

// clock linear counter
func (triangle *TrianglePulse) clockLinearCounter() {
	if triangle.linearCounterReload {
		triangle.linearCounterValue = triangle.linearCounterPeriod
		// if the control flag is clear, clear counter reload
		if !triangle.lengthCounter.halted {
			triangle.linearCounterReload = false
		}
	} else if triangle.linearCounterValue > 0 {
		triangle.linearCounterValue -= 1
	}
}

func (triangle *TrianglePulse) clockHalfFrame() {
	triangle.lengthCounter.Clock(triangle.channelEnable)
	triangle.clockLinearCounter()
}

func (triangle *TrianglePulse) clockQuarterFrame() {
	triangle.clockLinearCounter()
}

func (triangle *TrianglePulse) getSample() uint {
	if !triangle.channelEnable || triangle.lengthCounter.halted || triangle.linearCounterValue == 0 || triangle.lengthCounter.value == 0 {
		return 0
	}
	//
	// hyper frequency mutes triangle
	if triangle.timer.period < 3 {
		return 0
	}

	return uint(triangleSequencerTable[triangle.sequencerStep])

}
