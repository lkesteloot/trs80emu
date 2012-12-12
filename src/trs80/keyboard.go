package main

import (
	"fmt"
)

// http://www.trs-80.com/trs80-zaps-internals.htm#keyboard13
const keyboardBegin = 0x3800
const keyboardEnd = keyboardBegin + 256
var keyboard [8]byte

func (cpu *cpu) readKeyboard(addr word) byte {
	addr -= keyboardBegin

	var b byte

	for i := uint(0); i < 8; i++ {
		if addr & (1 << i) != 0 {
			b |= keyboard[i]
		}
	}
	// fmt.Printf("Reading keyboard %04X (%02X)\n", addr, b)

	return b
}

func keyEvent(key int, isPressed bool) {
    // http://www.trs-80.com/trs80-zaps-internals.htm#keyboard13
    // 0 = @
    // 1 = A
    // 8 = H
    // 48 = enter
    // 56 = shift

	fmt.Printf("Key %d is %s\n", key, isPressed)
	bit := byte(1 << uint(key%8))
	if isPressed {
		keyboard[key/8] |= bit
	} else {
		keyboard[key/8] &^= bit
	}
}
