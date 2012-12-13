package main

import (
	"fmt"
)

// http://www.trs-80.com/trs80-zaps-internals.htm#portsm3
// http://www.trs-80.com/trs80-zaps-internals.htm#ports
var ports map[byte]string = map[byte]string{
	0xE0: "maskable interrupt",
	0xE4: "NMI options/status",
	0xEC: "various controls",
	0xF0: "FDC command/status",
	0xF4: "select drive and options",
	0xFF: "cassette port",
}

var modeImage byte = 0x80

func readPort(port byte) byte {
	switch port {
	case 0xE4:
		// NMI latch read.
		return ^byte(0x01)
	case 0xF0:
		// No controller.
		return 0xFF
	case 0xFF:
		cassetteStatus := byte(0)
		return (modeImage & 0x7E) | cassetteStatus
	}

	panic(fmt.Sprintf("Can't read from unknown port %02X", port))
}

func writePort(port byte, value byte) {
	switch port {
	case 0xE0:
		// Set interrupt mask.
	case 0xE4, 0xE5, 0xE6, 0xE7:
		// NMI state.
		/// nmi_mask = value | M3_RESET_BIT
		/// z80_state.nmi = (nmi_latch & nmi_mask) != 0
		/// if (!z80_state.nmi) z80_state.nmi_seen = 0
	case 0xEC, 0xED, 0xEE, 0xEF:
		// Various controls.
		modeImage = value
		/// trs_cassette_motor((value & 0x02) >> 1)
		/// trs_screen_expanded((value & 0x04) >> 2)
		/// trs_screen_alternate(!((value & 0x08) >> 3))
		/// trs_timer_speed((value & 0x40) >> 6)
	case 0xF0:
		// Disk command.
	case 0xF4, 0xF5, 0xF6, 0xF7:
		// Disk select.
	default:
		panic(fmt.Sprintf("Can't write %02X to unknown port %02X", value, port))
	}
}
