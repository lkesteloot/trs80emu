// Copyright 2012 Lawrence Kesteloot

package main

import (
	"fmt"
	"log"
)

// Various flags that control what kind of debugging information
// is logged by the emulator. Normally these are all false.
const (
	dumpInstructionSet = false
	diskDebug          = false
	diskSortDebug      = false
	eventDebug         = false
	warnUninitMemRead  = false
	disableTimer       = false
	wavDebug           = false
	crashOnRomWrite	   = false
	logOnRomWrite      = false
)

// Same as above but can be changed at runtime. This is for
// instruction-level debugging.
var printDebug = false

// Map from PC to the ROM routine stored there.
var romRoutines = map[word]string{
	0x02A1: "$CLKOFF: Disable clock display",
	0x0298: "$CLKON: Enable clock display",
	0x0296: "$CSHIN: Search for cassette header and sync byte",
	0x0235: "$CSIN: Input a byte from cassette",
	0x0287: "$CSHWR: Write leader and sync byte",
	0x01F8: "$CSOFF: Turn off cassette",
	0x0264: "$CSOUT: Write byte to cassette",
	0x3033: "$DATE: Get today's date to (HL) as MM/DD/YY",
	0x0060: "$DELAY: Delay about BC*14.8 us",
	0x0069: "$INITIO: Initialize all I/O drivers",
	0x002B: "$KBCHAR: Get character from keyboard into A, or 0 if none pressed",
	0x0040: "$KBLINE: Input into (HL) for max B chars, ended with 0D or 01 (Break)",
	0x0049: "$KBWAIT: Wait for a keyboard character, put into A",
	0x028D: "$KBBRK: Check for Break key only, NZ if pressed",
	0x003B: "$PRCHAR: Send A to printer",
	0x01D9: "$PRSCN: Prints screen to printer",
	0x1A19: "$READY: Print Ready prompt (jump, don't call)",
	0x0000: "$RESET: Reset computer (jump, don't call)",
	0x006C: "$ROUTE: Route device at 4222 to one at 4220 (KI, DO, RI, RO, PR)",
	0x005A: "$RSINIT: Initialize RS-232",
	0x0050: "$RSRCV: Receive RS-232 character",
	0x0055: "$RSTX: Transmit RS-232 character",
	0x3042: "$SETCAS: Prompt user to set cassette baud rate with Cass?",
	0x3036: "$TIME: Get the time to (HL) as HH:MM:SS",
	0x0033: "$VDCHAR: Display character A at current position",
	0x01C9: "$VDCLS: Clear the screen",
	0x021B: "$VDLINE: Display (HL), terminated by 03 (not printed) or 0D (printed)",
}

// Log interesting information about the instruction we're about to execute.
func (vm *vm) explainLine(pc, hl word, a byte) {
	explanation, ok := romRoutines[pc]
	if ok {
		log.Print(explanation)
		if pc == 0x021B {
			// $VDLINE.
			msg := ""
			addr := hl
			for {
				ch := vm.memory[addr]
				msg += printableChar(ch)

				// Strings are terminated by 0x03 (not printed) or 0x0D (printed).
				if ch == 0x03 || ch == 0x0D {
					break
				}

				addr++
			}
			log.Printf("(HL) = \"%s\"", msg)
		} else if pc == 0x0033 {
			// $VDCHAR.
			log.Printf("A = %02X \"%s\"", a, printableChar(a))
		}
	}
}

// Convert a byte to a string meaningful to a human.
func printableChar(ch byte) string {
	if ch == 0x0A {
		return `\n`
	} else if ch == 0x0D {
		return `\r`
	} else if ch < 0x20 || ch >= 127 {
		return fmt.Sprintf(`\x%02X`, ch)
	}

	return string(ch)
}
