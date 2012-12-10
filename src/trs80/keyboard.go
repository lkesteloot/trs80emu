package main

// http://www.trs-80.com/trs80-zaps-internals.htm#keyboard13
const keyboardFirst = 0x3800
const keyboardLast = keyboardFirst + 255

func (cpu *cpu) readKeyboard(addr word) byte {
	word -= keyboardFirst

	// No keys pressed.
	return 0
}
