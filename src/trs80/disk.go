package main

import (
	"fmt"
)

// http://www.trs-80.com/trs80-zaps-internals.htm#memmapio

func (cpu *cpu) readDisk(addr word) byte {
	switch addr {
		/*
	case 0x37EA:
		// I don't know.
		return 0
	case 0x37EC:
		// Disk status register.
		return 0
	case 0x37F6:
		// I don't know.
		return 3 // ?? makes it not loop
		*/
	}

	panic(fmt.Sprintf("Tried to read from unknown cassette/disk at %04X", addr))
}
