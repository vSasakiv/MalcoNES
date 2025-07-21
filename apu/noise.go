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
	// lengthCounterHalt bool
	// envelopeLoop      bool
	// lengthCounter uint

	// constantEnvelope       bool
	// envelopeRestart        bool
	// decayCounter           uint
	// envelopeConstantVolume uint
	// envelopeDividerPeriod  uint
	// envelopeDividerValue   uint

	mode uint

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

	// noise.lengthCounterHalt = (val>>5)&0b1 == 1
	// noise.envelopeLoop = (val>>5)&0b1 == 0
	// noise.constantEnvelope = (val>>4)&0b1 == 1
	// noise.envelopeConstantVolume = uint(val & 0x0F)
	// noise.envelopeDividerPeriod = uint(val & 0x0F)

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
	// noise.envelopeRestart = true
	// noise.lengthCounter = uint(lengthLookUpTable[(uint(val) >> 3)])
	noise.lengthCounter.setValue(uint(val >> 3))
}

// // clocks only when the frame counter hits quarter frame
// func (noise *NoiseChannel) clockEnvelope() {
// 	// if the start flag is set, load the decay counter and the divider with the respective values
// 	if noise.envelopeRestart {
// 		noise.decayCounter = envelopeStartVolume
// 		noise.envelopeDividerValue = noise.envelopeDividerPeriod
// 		noise.envelopeRestart = false
// 	} else {
// 		// envelope clocked while 0, we reload the period
// 		if noise.envelopeDividerValue == 0 {
// 			noise.envelopeDividerValue = noise.envelopeDividerPeriod
// 			// now we clock the decay counter
// 			// if the length counter halt flag is active, we just load decay with 15
// 			if noise.decayCounter > 0 {
// 				noise.decayCounter -= 1
// 			} else if noise.envelopeLoop {
// 				noise.decayCounter = 15
// 			}
// 			// if it is not set, we decrement if it is not already 0
// 		} else {
// 			// if it is not zero, we decrement the divider
// 			noise.envelopeDividerValue -= 1
// 		}
// 	}
// }

// return the current envelope volume, if it is constant, return the value
// loaded from register, if it is not constant, return the decay counter
// of the envelope
// func (noise *NoiseChannel) getEnvelopeVolume() uint {
// 	if noise.constantEnvelope {
// 		return noise.envelopeConstantVolume
// 	} else {
// 		return noise.decayCounter
// 	}
// }

// clock the length counter
// func (noise *NoiseChannel) clockLengthCounter() {
// 	// disabling the channel via status also halts length counter
// 	if !noise.lengthCounterHalt && noise.lengthCounter > 0 && noise.channelEnable {
// 		noise.lengthCounter -= 1
// 	}
// }

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
