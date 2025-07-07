package controller

const A = 0
const B = 1
const SELECT = 2
const START = 3
const UP = 4
const DOWN = 5
const LEFT = 6
const RIGHT = 7

type JoyPad struct {
	strobe       bool
	buttonShift  uint
	buttonStatus [8]uint
}

func NewJoypad() *JoyPad {
	var joyPad JoyPad
	return &joyPad
}

func (joyPad *JoyPad) ReceiveWrite(val uint8) {
	if val&0b1 == 1 {
		joyPad.strobe = true
		joyPad.buttonShift = 0
	} else {
		joyPad.strobe = false
	}
}

func (joyPad *JoyPad) ReceiveRead() uint8 {
	if joyPad.buttonShift > 7 {
		return 1
	}
	result := joyPad.buttonStatus[joyPad.buttonShift]
	if !joyPad.strobe {
		joyPad.buttonShift++
	}
	return uint8(result)
}

func (joyPad *JoyPad) SetButtonStatus(button uint, val uint) {
	joyPad.buttonStatus[button] = val
}
