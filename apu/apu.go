package apu

// the max samples per frame is actually 89341 / cyclePerSample which is approximately
// 734 samples, so we use 1024 for safety
const samplesPerFrame uint = 1024

var lengthLookUpTable = []byte{
	10, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30,
}

var pulseLookUpTable []float64 = BuildPulseLookupTable()
var mixerLookUpTable []float64 = BuildMixerLookupTable()

type Apu struct {
	clockCounter  uint
	apuCycle      uint
	currentSample []byte
	Pulse1        Pulse
	Pulse2        Pulse
	Triangle      TrianglePulse
	Noise         NoiseChannel
	filterchain   FilterChain
}

func NewApu() *Apu {
	var apu Apu
	// 16 bit sample
	apu.currentSample = make([]byte, 2)
	apu.Pulse1.channel = 1
	apu.Pulse2.channel = 2
	apu.Noise.shiftRegister = 1
	apu.filterchain = FilterChain{
		HighPassFilter(float32(44100), 90),
		HighPassFilter(float32(44100), 440),
		LowPassFilter(float32(44100), 14000),
	}
	return &apu
}

func (apu *Apu) Reset() {
	apu.clockCounter = 0
	apu.apuCycle = 0
}

var apu Apu = *NewApu()

func Clock() {
	apu.clockCounter++
	if apu.clockCounter == 3 {
		// triangle clocks at cpu speed
		apu.Triangle.clockTimer()
		apu.Noise.clockTimer()
	}
	if apu.clockCounter == 6 {
		apu.apuCycle++
		apu.Pulse1.clockSequencer()
		apu.Pulse2.clockSequencer()
		apu.Triangle.clockTimer()
		apu.Noise.clockTimer()
		apu.clockCounter = 0
	}

	if apu.apuCycle == 3728 && apu.clockCounter == 0 {
		apu.Pulse1.clockEnvelope()
		apu.Pulse2.clockEnvelope()

		apu.Noise.clockEnvelope()
		apu.Triangle.clockLinearCounter()
	}
	// half frame
	if apu.apuCycle == 7456 && apu.clockCounter == 0 {
		apu.Pulse1.clockEnvelope()
		apu.Pulse1.clockLengthCounter()
		apu.Pulse1.clockSweep()

		apu.Pulse2.clockEnvelope()
		apu.Pulse2.clockLengthCounter()
		apu.Pulse2.clockSweep()

		apu.Triangle.clockLengthCounter()
		apu.Triangle.clockLinearCounter()

		apu.Noise.clockEnvelope()
		apu.Noise.clockLengthCounter()
	}
	if apu.apuCycle == 11185 && apu.clockCounter == 0 {
		apu.Pulse1.clockEnvelope()
		apu.Pulse2.clockEnvelope()

		apu.Triangle.clockLinearCounter()

		apu.Noise.clockEnvelope()
	}
	// half frame
	if apu.apuCycle == 18640 && apu.clockCounter == 0 {
		apu.Pulse1.clockEnvelope()
		apu.Pulse1.clockLengthCounter()
		apu.Pulse1.clockSweep()

		apu.Pulse2.clockEnvelope()
		apu.Pulse2.clockLengthCounter()
		apu.Pulse2.clockSweep()

		apu.Triangle.clockLengthCounter()
		apu.Triangle.clockLinearCounter()

		apu.Noise.clockEnvelope()
		apu.Noise.clockLengthCounter()
	}
	if apu.apuCycle == 18641 {
		apu.apuCycle = 0
	}

}

func GenSample() float32 {
	pulse1Sample := apu.Pulse1.getSample()
	pulse2Sample := apu.Pulse2.getSample()
	// triangleSample := apu.Triangle.getSample()
	// noiseSample := apu.Noise.getSample()

	mixedSample := apu.filterchain.Step(
		float32(pulseLookUpTable[pulse1Sample+pulse2Sample]))
	// float32(mixerLookUpTable[3*triangleSample+2*noiseSample]))
	// mixedSample := apu.filterchain.Step(float32(pulseLookUpTable[pulse1Sample+pulse2Sample]))

	return mixedSample

	// sample := int16((mixedSample*2 - 1) * 32767)
	// return sample
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

	if ((val >> 2) & 0b1) == 1 {
		apu.Triangle.channelEnable = true
	} else {
		apu.Triangle.channelEnable = false
	}

	if ((val >> 3) & 0b1) == 1 {
		apu.Noise.channelEnable = true
	} else {
		apu.Noise.channelEnable = false
	}

}

// ==================================================================== //
// ||                                                                   ||
// ||                      APU SQUARE PULSES                            ||
// ||                                                                   ||
// ==================================================================== //
//
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
	channel       uint

	dutyCycle         uint
	lengthCounterHalt bool
	constantEnvelope  bool
	startEnvelope     bool

	decayCounter           uint
	envelopeConstantVolume uint
	envelopeDividerPeriod  uint
	envelopeDividerValue   uint

	sequencerStep uint

	sweepEnabled       bool
	sweepDividerPeriod uint
	sweepNegate        bool
	sweepShiftCount    uint
	sweepReload        bool
	sweepValue         uint
	sweepSilence       bool

	pulseTimer       uint
	timerLow         uint8
	timerHigh        uint8
	pulseTimerPeriod uint

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

// write to register 0x4001 / 0x4005 of pulse registers
func (pulse *Pulse) WriteToSweep(val uint8) {
	// val = EPPP.NSSS
	// E -> pulse sweep enable
	// PPP -> sweep divider period
	// N -> sweep negate
	// SSS - > shift ammount

	if (val>>7)&0b1 == 1 {
		pulse.sweepEnabled = true
	} else {
		pulse.sweepEnabled = false
	}
	pulse.sweepDividerPeriod = uint((val>>4)&0b111) + 1
	if (val>>3)&0b1 == 1 {
		pulse.sweepNegate = true
	} else {
		pulse.sweepNegate = false
	}
	pulse.sweepShiftCount = uint(val & 0b111)
	pulse.sweepReload = true
}

// write to register 0x4002 / 0x4006 of pulse registers
func (pulse *Pulse) WriteToTimerLow(val uint8) {
	// val = LLLL.LLLL
	// LLLL.LLLL -> lower 8 bits of sequencer timer
	pulse.timerLow = val
	pulse.pulseTimerPeriod = uint(pulse.timerLow) | (uint(pulse.timerHigh) << 8)
	// pulse.pulseTimerPeriod = uint(val) | (pulse.pulseTimerPeriod & 0xFF00)
}

// write to register 0x4003 / 0x4007 of pulse registers
func (pulse *Pulse) WriteToTimerHigh(val uint8) {
	// val = llll.lHHH
	// llll.l -> length counter load
	// HHH -> high 3 bits of sequencer timer
	// also set the start envelope flag
	pulse.timerHigh = val & 0b111
	pulse.pulseTimerPeriod = uint(pulse.timerLow) | (uint(pulse.timerHigh) << 8)
	// pulse.pulseTimerPeriod = (uint(val&0b111) << 8) | (pulse.pulseTimerPeriod & 0x00FF)
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
		pulse.pulseTimer = pulse.pulseTimerPeriod + 1
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
	changeAmount := pulse.pulseTimerPeriod >> pulse.sweepShiftCount
	// if negate flag is true, subtract instead of adding the change ammount
	if pulse.sweepNegate {
		result = pulse.pulseTimerPeriod - changeAmount
		// if is pulse1, subtract one more since it is one's complement for some reason
		if pulse.channel == 1 {
			result -= 1
		}

	} else {
		result = pulse.pulseTimerPeriod + changeAmount
	}

	if result > 0x7FF || pulse.pulseTimerPeriod < 8 {
		pulse.sweepSilence = true
	} else {
		pulse.sweepSilence = false
	}

	return result
}

// sweeps the current timer
func (pulse *Pulse) sweep(targetPeriod uint) {
	if pulse.sweepEnabled && pulse.sweepShiftCount > 0 && !pulse.sweepSilence {
		pulse.pulseTimerPeriod = targetPeriod
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

	// mute if sweep is silencing
	if pulse.sweepSilence {
		return 0
	}

	if dutyCycleLookUpTable[pulse.dutyCycle][pulse.sequencerStep] == 1 {
		return pulse.getEnvelopeVolume()
	} else {
		return 0
	}
}

// ==================================================================== //
// ||                                                                   ||
// ||                      APU TRIANGLE PULSES                          ||
// ||                                                                   ||
// ==================================================================== //
//

// Build the mixer triangle, noise and dmc lookup table
func BuildMixerLookupTable() []float64 {
	// lookUpTable approximation for the mixer output
	lookupTable := make([]float64, 203)
	lookupTable[0] = 0
	for i := range 202 {
		lookupTable[i+1] = 163.67 / ((24329.0 / float64(i)) + 100.0)
	}
	return lookupTable
}

var triangleSequencerTable = []uint8{
	15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
}

type TrianglePulse struct {
	channelEnable bool

	linearCounterValue  uint
	linearCounterPeriod uint
	linearCounterReload bool

	timerPeriod uint
	timerValue  uint

	lengthCounterHalt bool
	lengthCounter     uint
	sequencerStep     uint
}

// write to register 0x4008 of triangle pulse
func (triangle *TrianglePulse) WriteToLinearCounter(val uint8) {
	// val = CRRR.RRRR
	// length counter halt = C
	// linear counter reload = RRR.RRRR
	if (val>>7)&0b1 == 1 {
		triangle.lengthCounterHalt = true
	} else {
		triangle.lengthCounterHalt = false
	}
	triangle.linearCounterPeriod = uint(val & 0b111_1111)
}

// write to register 0x400A of triangle pulse
func (triangle *TrianglePulse) WriteToTimerLow(val uint8) {
	// val = LLLL.LLLL
	// LLLL.LLLL -> lower 8 bits of sequencer timer
	triangle.timerPeriod = uint(val) | (triangle.timerPeriod & 0xFF00)
}

// write to register 0x400B of triangle pulse
func (triangle *TrianglePulse) WriteToTimerHigh(val uint8) {
	// val = llll.lHHH
	// llll.l -> length counter load
	// HHH -> high 3 bits of sequencer timer
	// also set the start envelope flag
	triangle.timerPeriod = uint(val&7)<<8 | (triangle.timerPeriod & 0x00FF)
	triangle.lengthCounter = uint(lengthLookUpTable[(val >> 3)])
	triangle.timerValue = triangle.timerPeriod
	triangle.linearCounterReload = true
}

// clocks the 11 bit timer every cpu cycle
func (triangle *TrianglePulse) clockTimer() {

	if triangle.timerValue == 0 {
		if triangle.linearCounterValue > 0 && triangle.lengthCounter > 0 {
			triangle.clockSequencer()
		}
		triangle.timerValue = triangle.timerPeriod
	} else {
		triangle.timerValue -= 1
	}
}

// clock the sequencer
func (triangle *TrianglePulse) clockSequencer() {
	if triangle.sequencerStep == 31 {
		triangle.sequencerStep = 0
	} else {
		triangle.sequencerStep += 1
	}
}

// clock length counter
func (triangle *TrianglePulse) clockLengthCounter() {
	if !triangle.lengthCounterHalt && triangle.lengthCounter > 0 && triangle.channelEnable {
		triangle.lengthCounter -= 1
	}
}

// clock linear counter
func (triangle *TrianglePulse) clockLinearCounter() {
	if triangle.linearCounterReload {
		triangle.linearCounterValue = triangle.linearCounterPeriod
		// if the control flag is clear, clear counter reload
		if !triangle.lengthCounterHalt {
			triangle.linearCounterReload = false
		}
	} else if triangle.linearCounterValue > 0 {
		triangle.linearCounterValue -= 1
	}
}

func (triangle *TrianglePulse) getSample() uint {
	// if !triangle.channelEnable || triangle.lengthCounterHalt || triangle.linearCounterValue == 0 || triangle.lengthCounter == 0 {
	// 	return 0
	// }
	//
	// hyper frequency mutes triangle
	if triangle.timerPeriod < 3 {
		return 0
	}

	return uint(triangleSequencerTable[triangle.sequencerStep])

}

// ==================================================================== //
// ||                                                                   ||
// ||                        APU NOISE CHANNEL                          ||
// ||                                                                   ||
// ==================================================================== //

// in CPU CYCLES !!!!!
var noiseTable = []uint16{
	4, 8, 16, 32, 64, 96, 128, 160, 202, 254, 380, 508, 762, 1016, 2034, 4068,
}

type NoiseChannel struct {
	channelEnable     bool
	lengthCounterHalt bool
	lengthCounter     uint

	constantEnvelope       bool
	envelopeRestart        bool
	decayCounter           uint
	envelopeConstantVolume uint
	envelopeDividerPeriod  uint
	envelopeDividerValue   uint

	mode uint

	timerPeriod uint
	timerValue  uint

	// starts as 1
	shiftRegister uint
}

// Write to register 0x400C of noise channel
func (noise *NoiseChannel) WriteToVolume(val uint8) {
	// val = __LC VVVV
	// L -> lenghtEnable : 1 = infinite ; 0 = enable counter
	// C -> constantEnvelope : 1 volume = constant ; 0 = use the envelope
	// VVVV -> constant volume if C = 1 or envelope decay if C = 0

	if ((val >> 5) & 0b1) == 1 {
		noise.lengthCounterHalt = true
	} else {
		noise.lengthCounterHalt = false
	}

	if ((val >> 4) & 0b1) == 1 {
		noise.constantEnvelope = true
	} else {
		noise.constantEnvelope = false
	}
	noise.envelopeConstantVolume = uint(val & 0x0F)
	noise.envelopeDividerPeriod = uint(val & 0x0F)
}

// write to register 0x400E of noise channel
func (noise *NoiseChannel) WriteToModeAndPeriod(val uint8) {
	// val = M___.PPPP
	// PPPP -> timer period

	noise.mode = (uint(val) >> 7) & 0b1
	noise.timerPeriod = uint(val) & 0x0F
}

// write to register 0x400F of noise channel
func (noise *NoiseChannel) WriteToLengthCounter(val uint8) {
	// val = llll.l___
	// llll.l -> length counter load

	noise.envelopeRestart = true
	noise.lengthCounter = uint(lengthLookUpTable[(uint(val) >> 3)])
}

// clocks only when the frame counter hits quarter frame
func (noise *NoiseChannel) clockEnvelope() {
	// if the start flag is set, load the decay counter and the divider with the respective values
	if noise.envelopeRestart {
		noise.decayCounter = envelopeStartVolume
		noise.envelopeDividerValue = noise.envelopeDividerPeriod
		noise.envelopeRestart = false
	} else {
		// envelope clocked while 0, we reload the period
		if noise.envelopeDividerValue == 0 {
			noise.envelopeDividerValue = noise.envelopeDividerPeriod
			// now we clock the decay counter
			// if the length counter halt flag is active, we just load decay with 15
			if noise.lengthCounterHalt {
				noise.decayCounter = 15
			} else
			// if it is not set, we decrement if it is not already 0
			if noise.decayCounter > 0 {
				noise.decayCounter -= 1
			}
		} else {
			// if it is not zero, we decrement the divider
			noise.envelopeDividerValue -= 1
		}
	}
}

// return the current envelope volume, if it is constant, return the value
// loaded from register, if it is not constant, return the decay counter
// of the envelope
func (noise *NoiseChannel) getEnvelopeVolume() uint {
	if noise.constantEnvelope {
		return noise.envelopeConstantVolume
	} else {
		return noise.decayCounter
	}
}

// clock the length counter
func (noise *NoiseChannel) clockLengthCounter() {
	// disabling the channel via status also halts length counter
	if !noise.lengthCounterHalt && noise.lengthCounter > 0 && noise.channelEnable {
		noise.lengthCounter -= 1
	}
}

// clocks the timer
func (noise *NoiseChannel) clockTimer() {
	if noise.timerValue == 0 {
		noise.timerValue = noise.timerPeriod
		var feedback uint
		if noise.mode == 1 {
			feedback = noise.shiftRegister&0b1 ^ (noise.shiftRegister>>6)&0b1
		} else {
			feedback = noise.shiftRegister&0b1 ^ (noise.shiftRegister>>1)&0b1
		}
		noise.shiftRegister >>= 1
		noise.shiftRegister |= (feedback << 14)
	} else {
		noise.timerValue -= 1
	}
}

func (noise *NoiseChannel) getSample() uint {
	if noise.shiftRegister&0b1 == 1 || noise.lengthCounter == 0 {
		return 0
	}
	return noise.getEnvelopeVolume()
}
