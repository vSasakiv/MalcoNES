package apu

// using ppu clock, since it is the one that clocks the apu, even if it does nothing
// makes it so that the timing will be more precise !
const timePerClock float64 = 1.00 / 5369319.00

type Apu struct {
	clockCounter uint
	apuCycle     uint
	realTime     float64
	sampleBuffer []float64
	Pulse1       Pulse
}

func NewApu() *Apu {
	var apu Apu
	return &apu
}

func (apu *Apu) Reset() {
	apu.clockCounter = 0
	apu.apuCycle = 0
	apu.realTime = 0
}

var apu Apu = *NewApu()

func Clock() {
	apu.clockCounter++
	apu.realTime += timePerClock
	if apu.clockCounter == 6 {
		apu.apuCycle++
		apu.Pulse1.clockSequencer()
		apu.clockCounter = 0
	}

	if apu.apuCycle == 3728 {
		apu.Pulse1.clockEnvelope()
	}
	if apu.apuCycle == 7456 {
		apu.Pulse1.clockEnvelope()
	}
	if apu.apuCycle == 11185 {
		apu.Pulse1.clockEnvelope()
	}
	if apu.apuCycle == 14914 {
		apu.Pulse1.clockEnvelope()
	}
	if apu.apuCycle == 14915 {
		apu.apuCycle = 0
	}

}

func GetSample() {
	apu.sampleBuffer = append(apu.sampleBuffer, apu.Pulse1.getSample())
}

func GetApu() *Apu {
	return &apu
}

func (apu *Apu) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range apu.sampleBuffer {
		samples[i][0] = apu.sampleBuffer[i]
		samples[i][1] = apu.sampleBuffer[i]
	}
	nSamples := len(apu.sampleBuffer)
	apu.sampleBuffer = apu.sampleBuffer[:0]
	return nSamples, true
}

func (apu *Apu) Err() error {
	return nil
}

// standard start volume when reseting envelope
const envelopeStartVolume = 15

// lookup table for value of pulse given duty cycle and sequencerStep
var pulseLookUpTable = [4][8]uint{
	{0, 1, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 0, 0, 0},
	{1, 0, 0, 1, 1, 1, 1, 1},
}

type Pulse struct {
	dutyCycle         uint
	lengthCounterHalt bool
	constantEnvelope  bool
	startEnvelope     bool

	decayCounter           uint
	envelopeConstantVolume uint
	envelopeDividerPeriod  uint
	envelopeDividerValue   uint

	sequencerStep uint

	sweepEnabled       uint
	sweepDividerPeriod uint
	sweepNegate        uint
	sweepShiftCount    uint
	pulseTimer         uint
	pulseTimerPeriod   uint

	lengthCounterLoad uint
}

// Write to register 0x4000 / 0x4004 of pulse registers
func (pulse *Pulse) WriteToDutyCycleAndVolume(val uint8) {
	// val = DDLC VVVV
	// DD -> dutyCycle
	// L -> lenghtEnable : 1 = infinite ; 0 = enable counter
	// C -> constantEnvelope : 1 volume = constant ; 0 = use the envelope
	// VVVV -> constant volume if C = 1 or envelope decay if C = 0

	pulse.dutyCycle = uint((val >> 6) & 0b11)

	if ((val >> 5) & 0b1) == 1 {
		pulse.lengthCounterHalt = true
	} else {
		pulse.lengthCounterHalt = false
	}

	if ((val >> 4) & 0b1) == 1 {
		pulse.constantEnvelope = true
	} else {
		pulse.constantEnvelope = false
	}
	pulse.envelopeConstantVolume = uint(val & 0x0F)
	pulse.envelopeDividerPeriod = uint(val & 0x0F)
}

// write to register 0x4002 / 0x4006 of pulse registers
func (pulse *Pulse) WriteToTimerLow(val uint8) {
	// val = LLLL.LLLL
	// LLLL.LLLL -> lower 8 bits of sequencer timer
	pulse.pulseTimerPeriod = uint(val) | pulse.pulseTimerPeriod&0xFF00
}

// write to register 0x4003 / 0x4007 of pulse registers
func (pulse *Pulse) WriteToTimerHigh(val uint8) {
	// val = llll.lHHH
	// llll.l -> length counter load
	// HHH -> high 3 bits of sequencer timer
	// also set the start envelope flag
	pulse.pulseTimerPeriod = uint(val)<<8 | pulse.pulseTimerPeriod&0x00FF
	pulse.lengthCounterLoad = uint(val >> 3)
	pulse.startEnvelope = true
}

// clocks only when the frame counter hits quarter frame
func (pulse *Pulse) clockEnvelope() {
	// if the start flag is set, load the decay counter and the divider with the respective values
	if pulse.startEnvelope {
		pulse.decayCounter = envelopeStartVolume
		pulse.envelopeDividerValue = pulse.envelopeDividerPeriod
		pulse.startEnvelope = false
	} else {
		// envelope clocked while 0, we reload the period
		if pulse.envelopeDividerValue == 0 {
			pulse.envelopeDividerValue = pulse.envelopeDividerPeriod
			// now we clock the decay counter
			// if the length counter halt flag is active, we just load decay with 15
			if pulse.lengthCounterHalt {
				pulse.decayCounter = 15
			} else
			// if it is not set, we decrement if it is not already 0
			if pulse.decayCounter > 0 {
				pulse.decayCounter -= 1
			}
		} else {
			// if it is not zero, we decrement the divider
			pulse.envelopeDividerValue -= 1
		}
	}
}

// clocks the sequencer every 2 cpu cycles or 6 ppu cycles
func (pulse *Pulse) clockSequencer() {
	// if the timer is 0, load the timer period and clock the sequencer
	if pulse.pulseTimer == 0 {
		pulse.pulseTimer = pulse.pulseTimerPeriod
		// we clock the sequencer
		if pulse.sequencerStep == 0 {
			pulse.sequencerStep = 7
		} else {
			pulse.sequencerStep -= 1
		}
	} else {
		pulse.pulseTimer -= 1
	}
}

// return the current envelope volume, if it is constant, return the value
// loaded from register, if it is not constant, return the decay counter
// of the envelope
func (pulse *Pulse) getEnvelopeVolume() uint {
	if pulse.constantEnvelope {
		return pulse.envelopeConstantVolume
	} else {
		return pulse.decayCounter
	}
}

func (pulse *Pulse) getSample() float64 {
	return float64(pulseLookUpTable[pulse.dutyCycle][pulse.sequencerStep]) * float64(pulse.getEnvelopeVolume())
}
