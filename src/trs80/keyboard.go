package main

import (
	"log"
)

// http://www.trs-80.com/trs80-zaps-internals.htm#keyboard13
const keyboardBegin = 0x3800
const keyboardEnd = keyboardBegin + 256

func (cpu *cpu) clearKeyboard() {
	for i := 0; i < len(cpu.keyboard); i++ {
		cpu.keyboard[i] = 0
	}
}

func (cpu *cpu) readKeyboard(addr word) byte {
	addr -= keyboardBegin

	var b byte

	for i, keys := range cpu.keyboard {
		if addr&(1<<uint(i)) != 0 {
			b |= keys
		}
	}

	return b
}

func (cpu *cpu) keyEvent(key int, isPressed bool) {
	// http://www.trs-80.com/trs80-zaps-internals.htm#keyboard13
	// 0 = @
	// 1 = A
	// 8 = H
	// 48 = enter
	// 56 = shift

	log.Printf("Key %d is %v\n", key, isPressed)
	bit := byte(1 << uint(key%8))
	if isPressed {
		cpu.keyboard[key/8] |= bit
	} else {
		cpu.keyboard[key/8] &^= bit
	}
}
