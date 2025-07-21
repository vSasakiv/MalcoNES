package apu

// ==================================================================== //
// ||                                                                   ||
// ||                           ENVELOPE                                ||
// ||                                                                   ||
// ==================================================================== //
//

type Envelope struct {
	reload      bool
	period      uint
	constVolume uint
	loop        bool
	isConstant  bool

	// internals
	decayCounter uint
	value        uint
}

// standard start volume when reseting envelope
const envelopeStartVolume = 15

func (envelope *Envelope) Clock() {
	// if the start flag is set, load the decay counter and the divider with the respective values
	if envelope.reload {
		envelope.decayCounter = envelopeStartVolume
		envelope.value = envelope.period
		envelope.reload = false
	} else {
		// envelope clocked while 0, we reload the period
		if envelope.value == 0 {
			envelope.value = envelope.period
			// now we clock the decay counter
			// we decrement if it is not already 0
			// if the length counter halt flag is active, we just load decay with 15
			if envelope.decayCounter > 0 {
				envelope.decayCounter -= 1
			} else if envelope.loop {
				envelope.decayCounter = 15
			}

		} else {
			// if it is not zero, we decrement the divider
			envelope.value -= 1
		}
	}
}

func (envelope *Envelope) getVolume() uint {
	if envelope.isConstant {
		return envelope.constVolume
	} else {
		return envelope.decayCounter
	}
}

// ==================================================================== //
// ||                                                                   ||
// ||                       LENGTH COUNTER                              ||
// ||                                                                   ||
// ==================================================================== //
//

var lengthLookUpTable = []uint{
	10, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30,
}

type LengthCounter struct {
	value  uint
	halted bool
}

func (lengthCounter *LengthCounter) Clock(channelEnabled bool) {
	if lengthCounter.value > 0 && !lengthCounter.halted && channelEnabled {
		lengthCounter.value--
	}
}

func (LengthCounter *LengthCounter) setValue(val uint) {
	LengthCounter.value = lengthLookUpTable[val]
}

// ==================================================================== //
// ||                                                                   ||
// ||                          RAW TIMER                                ||
// ||                                                                   ||
// ==================================================================== //
//

type RawTimer struct {
	value  uint
	period uint
}

type ExecuteOnTimerZero func()

// functions for use when the timer is 11 bit
func (timer *RawTimer) setTimerHigh(val uint8) {
	timer.period = (uint(val&0b111) << 8) | (timer.period & 0x00FF)
}

func (timer *RawTimer) setTimerLow(val uint8) {
	timer.period = uint(val) | (timer.period & 0xFF00)
}

// clocks timer, when it is 0 do something
func (timer *RawTimer) Clock(fc ExecuteOnTimerZero) {
	if timer.value == 0 {
		timer.value = timer.period
		fc()
	} else {
		timer.value--
	}
}
