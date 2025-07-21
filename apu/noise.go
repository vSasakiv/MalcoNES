package apu

// ==================================================================== //
// ||                                                                   ||
// ||                        APU NOISE CHANNEL                          ||
// ||                                                                   ||
// ==================================================================== //

// in CPU CYCLES !!!!!
var noiseTable = []uint{
	4, 8, 16, 32, 64, 96, 128, 160, 202, 254, 380, 508, 762, 1016, 2034, 4068,
}

type NoiseChannel struct {
	channelEnable bool
	mode          uint

	timerPeriod uint
	timerValue  uint

	// starts as 1
	shiftRegister uint

	envelope      Envelope
	lengthCounter LengthCounter
	timer         RawTimer
}

// Write to register 0x400C of noise channel
func (noise *NoiseChannel) WriteToVolume(val uint8) {
	// val = __LC VVVV
	// L -> lenghtEnable : 1 = infinite ; 0 = enable counter
	// C -> constantEnvelope : 1 volume = constant ; 0 = use the envelope
	// VVVV -> constant volume if C = 1 or envelope decay if C = 0
	noise.envelope.loop = (val>>5)&0b1 == 0
	noise.envelope.isConstant = (val>>4)&0b1 == 1
	noise.envelope.constVolume = uint(val & 0x0F)
	noise.envelope.period = uint(val & 0x0f)

	noise.lengthCounter.halted = (val>>5)&0b1 == 1
}

// write to register 0x400E of noise channel
func (noise *NoiseChannel) WriteToModeAndPeriod(val uint8) {
	// val = M___.PPPP
	// PPPP -> timer period
	noise.mode = (uint(val) >> 7) & 0b1
	noise.timer.period = noiseTable[uint(val)&0x0F]
}

// write to register 0x400F of noise channel
func (noise *NoiseChannel) WriteToLengthCounter(val uint8) {
	// val = llll.l___
	// llll.l -> length counter load

	noise.envelope.reload = true
	noise.lengthCounter.setValue(uint(val >> 3))
}

// clocks the timer
func (noise *NoiseChannel) clockTimer() {
	noise.timer.Clock(noise.updateShiftRegister)
}

func (noise *NoiseChannel) updateShiftRegister() {
	var feedback uint
	if noise.mode == 1 {
		feedback = noise.shiftRegister&0b1 ^ (noise.shiftRegister>>6)&0b1
	} else {
		feedback = noise.shiftRegister&0b1 ^ (noise.shiftRegister>>1)&0b1
	}
	noise.shiftRegister >>= 1
	noise.shiftRegister |= (feedback << 14)

}

func (noise *NoiseChannel) clockHalfFrame() {
	noise.envelope.Clock()
	noise.lengthCounter.Clock(noise.channelEnable)
}

func (noise *NoiseChannel) clockQuarterFrame() {
	noise.envelope.Clock()
}

func (noise *NoiseChannel) setChannelEnabled(enabled bool) {
	if !enabled {
		noise.lengthCounter.value = 0
	}
	noise.channelEnable = enabled
}

func (noise *NoiseChannel) getSample() uint {
	if noise.shiftRegister&0b1 == 1 || noise.lengthCounter.value == 0 {
		return 0
	}
	return noise.envelope.getVolume()
}
