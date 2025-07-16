package apu

type Apu struct {
	clockCounter uint
	apuCycle     uint
}

func NewApu() *Apu {
	var apu Apu
	return &apu
}

func (apu *Apu) Reset() {
	apu.clockCounter = 0
	apu.apuCycle = 0
}

var apu Apu = *NewApu()

func Clock() {
	apu.clockCounter++
	if apu.clockCounter == 6 {
		apu.apuCycle++
		apu.clockCounter = 0
	}

}

func GetApu() *Apu {
	return &apu
}
