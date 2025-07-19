package apu

// using ppu clock, since it is the one that clocks the apu, even if it does nothing
// makes it so that the timing will be more precise !
const timePerClock float64 = 1.00 / 5369319.00

// the max samples per frame is actually 89341 / cyclePerSample which is approximately
// 734 samples, so we use 1024 for safety
const samplesPerFrame uint = 1024

var lengthLookUpTable = []byte{
	10, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30,
}

var pulseLookUpTable []float64 = BuildPulseLookupTable()

type Apu struct {
	clockCounter  uint
	apuCycle      uint
	realTime      float64
	currentSample []byte
	Pulse1        Pulse
	Pulse2        Pulse
}

func NewApu() *Apu {
	var apu Apu
	// 16 bit sample
	apu.currentSample = make([]byte, 2)
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
		apu.Pulse2.clockSequencer()
		apu.clockCounter = 0
	}

	if apu.apuCycle == 3728 {
		apu.Pulse1.clockEnvelope()
		apu.Pulse2.clockEnvelope()
	}
	// half frame
	if apu.apuCycle == 7456 {
		apu.Pulse1.clockEnvelope()
		apu.Pulse1.clockLengthCounter()
		apu.Pulse2.clockEnvelope()
		apu.Pulse2.clockLengthCounter()
	}
	if apu.apuCycle == 11185 {
		apu.Pulse1.clockEnvelope()
		apu.Pulse2.clockEnvelope()
	}
	// half frame
	if apu.apuCycle == 14914 {
		apu.Pulse1.clockEnvelope()
		apu.Pulse1.clockLengthCounter()
		apu.Pulse2.clockEnvelope()
		apu.Pulse2.clockLengthCounter()
	}
	if apu.apuCycle == 14915 {
		apu.apuCycle = 0
	}

}

func GenSample() int16 {
	pulseSample := pulseLookUpTable[apu.Pulse1.getSample()+apu.Pulse2.getSample()]
	sample := int16((pulseSample*2 - 1) * 32767)
	return sample
}

func GetApu() *Apu {
	return &apu
}

// Write to status 0x4015 register
func (apu *Apu) WriteToStatusRegister(val uint8) {
	if (val & 0b1) == 1 {
		apu.Pulse1.channelEnable = true
	} else {
		apu.Pulse1.channelEnable = false
	}

	if ((val >> 1) & 0b1) == 1 {
		apu.Pulse2.channelEnable = true
	} else {
		apu.Pulse2.channelEnable = false
	}

}

// standard start volume when reseting envelope
const envelopeStartVolume = 15

// lookup table for value of pulse given duty cycle and sequencerStep
var dutyCycleLookUpTable = [4][8]uint{
	{0, 1, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 0, 0, 0},
	{1, 0, 0, 1, 1, 1, 1, 1},
}

type Pulse struct {
	channelEnable bool

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

	lengthCounter uint
}

// Build the mixer pulse value lookUp table for faster processing
func BuildPulseLookupTable() []float64 {
	// lookUpTable approximation for the mixer output
	lookupTable := make([]float64, 31)
	lookupTable[0] = 0
	for i := range 30 {
		lookupTable[i+1] = 95.52 / ((8128.0 / float64(i)) + 100.0)
	}
	return lookupTable
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
	pulse.pulseTimerPeriod = uint(val) | (pulse.pulseTimerPeriod & 0xFF00)
}

// write to register 0x4003 / 0x4007 of pulse registers
func (pulse *Pulse) WriteToTimerHigh(val uint8) {
	// val = llll.lHHH
	// llll.l -> length counter load
	// HHH -> high 3 bits of sequencer timer
	// also set the start envelope flag
	pulse.pulseTimerPeriod = uint(val&7)<<8 | (pulse.pulseTimerPeriod & 0x00FF)
	pulse.lengthCounter = uint(lengthLookUpTable[(val >> 3)])
	pulse.sequencerStep = 0
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

// clock the length counter
func (pulse *Pulse) clockLengthCounter() {
	// disabling the channel via status also halts length counter
	if !pulse.lengthCounterHalt && pulse.lengthCounter > 0 && pulse.channelEnable {
		pulse.lengthCounter -= 1
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

func (pulse *Pulse) getSample() uint {
	// channel disabled
	if !pulse.channelEnable {
		return 0
	}
	// length counter finished
	if pulse.lengthCounter == 0 {
		return 0
	}

	if pulse.pulseTimerPeriod < 8 || pulse.pulseTimerPeriod > 0x7FF {
		return 0
	}

	if dutyCycleLookUpTable[pulse.dutyCycle][pulse.sequencerStep] == 1 && pulse.lengthCounter > 0 {
		return pulse.getEnvelopeVolume()
	} else {
		return 0
	}
}
