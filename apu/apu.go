package apu

import (
	"vsasakiv/nesemulator/mappers"
)

// the max samples per frame is actually 89341 / cyclePerSample which is approximately
// 734 samples, so we use 1024 for safety
const samplesPerFrame uint = 1024

var pulseLookUpTable []float64 = BuildPulseLookupTable()
var mixerLookUpTable []float64 = BuildMixerLookupTable()

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

type Apu struct {
	clockCounter  uint
	apuCycle      uint
	currentSample []byte
	Pulse1        Pulse
	Pulse2        Pulse
	Triangle      TrianglePulse
	Noise         NoiseChannel
	Dmc           DMC
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
		apu.Pulse1.clockTimer()
		apu.Pulse2.clockTimer()

		apu.Triangle.clockTimer()
		apu.Noise.clockTimer()
		apu.Dmc.clockTimer()
		apu.clockCounter = 0
	}

	if apu.apuCycle == 3728 && apu.clockCounter == 0 {
		clockQuarterFrame()
	}
	// half frame
	if apu.apuCycle == 7456 && apu.clockCounter == 0 {
		clockHalfFrame()
	}
	if apu.apuCycle == 11185 && apu.clockCounter == 0 {
		clockQuarterFrame()
	}
	// half frame
	if apu.apuCycle == 18640 && apu.clockCounter == 0 {
		clockHalfFrame()
	}
	if apu.apuCycle == 18641 {
		apu.apuCycle = 0
	}

}

func clockHalfFrame() {
	apu.Pulse1.clockHalfFrame()
	apu.Pulse2.clockHalfFrame()
	apu.Triangle.clockHalfFrame()
	apu.Noise.clockHalfFrame()
}

func clockQuarterFrame() {
	apu.Pulse1.clockQuarterFrame()
	apu.Pulse2.clockQuarterFrame()
	apu.Triangle.clockQuarterFrame()
	apu.Noise.clockQuarterFrame()
}

func GenSample() float32 {
	pulse1Sample := apu.Pulse1.getSample()
	pulse2Sample := apu.Pulse2.getSample()
	triangleSample := apu.Triangle.getSample()
	noiseSample := apu.Noise.getSample()
	dmcSample := apu.Dmc.getSample()

	// mixedSample := apu.filterchain.Step(
	// 	float32(pulseLookUpTable[pulse1Sample+pulse2Sample]))
	// float32(mixerLookUpTable[3*triangleSample+2*noiseSample]))
	// mixedSample := apu.filterchain.Step(float32(pulseLookUpTable[pulse1Sample+pulse2Sample]))

	mixedSample := apu.filterchain.Step(
		float32(pulseLookUpTable[pulse1Sample+pulse2Sample])) +
		float32(mixerLookUpTable[3*triangleSample+2*noiseSample+dmcSample])
	//
	return mixedSample

	// sample := int16((mixedSample*2 - 1) * 32767)
	// return sample
}

func GetApu() *Apu {
	return &apu
}

func (apu *Apu) SetMapper(mapper mappers.Mapper) {
	apu.Dmc.mapper = mapper
}

// Write to status 0x4015 register
func (apu *Apu) WriteToStatusRegister(val uint8) {
	apu.Pulse1.setChannelEnabled(val&0b1 == 1)
	apu.Pulse2.setChannelEnabled((val>>1)&0b1 == 1)
	apu.Triangle.setChannelEnabled((val>>2)&0b1 == 1)
	apu.Noise.setChannelEnabled((val>>3)&0b1 == 1)
	apu.Dmc.setChannelEnabled((val>>4)&0b1 == 1)
}
