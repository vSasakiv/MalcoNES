package apu

// ==================================================================== //
// ||                                                                   ||
// ||                      APU SQUARE PULSES                            ||
// ||                                                                   ||
// ==================================================================== //
//

// lookup table for value of pulse given duty cycle and sequencerStep
var dutyCycleLookUpTable = [4][8]uint{
	{0, 1, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 0, 0, 0},
	{1, 0, 0, 1, 1, 1, 1, 1},
}

type Pulse struct {
	channelEnable bool
	channel       uint

	dutyCycle uint

	sequencerStep uint

	sweepEnabled       bool
	sweepDividerPeriod uint
	sweepNegate        bool
	sweepShiftCount    uint
	sweepReload        bool
	sweepValue         uint
	sweepSilence       bool

	envelope      Envelope
	lengthCounter LengthCounter
	timer         RawTimer
}

// Write to register 0x4000 / 0x4004 of pulse registers
func (pulse *Pulse) WriteToDutyCycleAndVolume(val uint8) {
	// val = DDLC VVVV
	// DD -> dutyCycle
	// L -> lenghtEnable : 1 = infinite ; 0 = enable counter
	// C -> constantEnvelope : 1 volume = constant ; 0 = use the envelope
	// VVVV -> constant volume if C = 1 or envelope decay if C = 0
	pulse.dutyCycle = uint((val >> 6) & 0b11)
	pulse.envelope.loop = (val>>5)&0b1 == 1
	pulse.envelope.isConstant = (val>>4)&0b1 == 1
	pulse.envelope.constVolume = uint(val & 0x0F)
	pulse.envelope.period = uint(val & 0x0f)
	pulse.lengthCounter.halted = (val>>5)&0b1 == 1
}

// write to register 0x4001 / 0x4005 of pulse registers
func (pulse *Pulse) WriteToSweep(val uint8) {
	// val = EPPP.NSSS
	// E -> pulse sweep enable
	// PPP -> sweep divider period
	// N -> sweep negate
	// SSS - > shift ammount
	pulse.sweepEnabled = (val>>7)&0b1 == 1
	pulse.sweepDividerPeriod = uint((val>>4)&0b111) + 1
	pulse.sweepNegate = (val>>3)&0b1 == 1
	pulse.sweepShiftCount = uint(val & 0b111)
	pulse.sweepReload = true
}

// write to register 0x4002 / 0x4006 of pulse registers
func (pulse *Pulse) WriteToTimerLow(val uint8) {
	// val = LLLL.LLLL
	// LLLL.LLLL -> lower 8 bits of sequencer timer
	pulse.timer.setTimerLow(val)
}

// write to register 0x4003 / 0x4007 of pulse registers
func (pulse *Pulse) WriteToTimerHigh(val uint8) {
	// val = llll.lHHH
	// llll.l -> length counter load
	// HHH -> high 3 bits of sequencer timer
	// also set the start envelope flag
	pulse.envelope.reload = true
	pulse.lengthCounter.setValue(uint(val) >> 3)
	pulse.timer.setTimerHigh(val)
	pulse.sequencerStep = 0
}

// clock the sweep unit
func (pulse *Pulse) clockSweep() {
	result := pulse.calculateTargetPeriod()
	// if a write to the register was done
	if pulse.sweepReload {
		// if the current value is 0, we do a last sweep
		if pulse.sweepEnabled && pulse.sweepValue == 0 {
			pulse.sweep(result)
		}
		// reload the value with the period set in the register and clear sweep reload
		pulse.sweepValue = pulse.sweepDividerPeriod
		pulse.sweepReload = false
	} else if pulse.sweepValue > 0 {
		// decrement the sweep counter
		pulse.sweepValue -= 1
	} else {
		// if sweep counter is 0, sweep the channel and reload the period
		if pulse.sweepEnabled {
			pulse.sweep(result)
		}
		pulse.sweepValue = pulse.sweepDividerPeriod
	}
}

func (pulse *Pulse) calculateTargetPeriod() uint {
	var result uint
	changeAmount := pulse.timer.period >> pulse.sweepShiftCount
	// if negate flag is true, subtract instead of adding the change ammount
	if pulse.sweepNegate {
		result = pulse.timer.period - changeAmount
		// if is pulse1, subtract one more since it is one's complement for some reason
		if pulse.channel == 1 {
			result -= 1
		}

	} else {
		result = pulse.timer.period + changeAmount
	}

	if result > 0x7FF || pulse.timer.period < 8 {
		pulse.sweepSilence = true
	} else {
		pulse.sweepSilence = false
	}
	return result
}

// sweeps the current timer
func (pulse *Pulse) sweep(targetPeriod uint) {
	if pulse.sweepEnabled && pulse.sweepShiftCount > 0 && !pulse.sweepSilence {
		pulse.timer.period = targetPeriod
	}
}

// clocks the sequencer every timer clock
func (pulse *Pulse) clockSequencer() {
	if pulse.sequencerStep == 0 {
		pulse.sequencerStep = 7
	} else {
		pulse.sequencerStep -= 1
	}
}

func (pulse *Pulse) clockTimer() {
	pulse.timer.Clock(pulse.clockSequencer)
}

func (pulse *Pulse) clockHalfFrame() {
	pulse.envelope.Clock()
	pulse.clockSweep()
	pulse.lengthCounter.Clock(pulse.channelEnable)
}

func (pulse *Pulse) clockQuarterFrame() {
	pulse.envelope.Clock()
}

func (pulse *Pulse) setChannelEnabled(enabled bool) {
	if !enabled {
		pulse.lengthCounter.value = 0
	}
	pulse.channelEnable = enabled
}

func (pulse *Pulse) getSample() uint {
	// channel disabled
	if !pulse.channelEnable {
		return 0
	}
	// length counter finished
	if pulse.lengthCounter.value == 0 {
		return 0
	}

	// mute if sweep is silencing
	if pulse.sweepSilence {
		return 0
	}

	if dutyCycleLookUpTable[pulse.dutyCycle][pulse.sequencerStep] == 0 {
		return 0
	}

	return pulse.envelope.getVolume()
}
